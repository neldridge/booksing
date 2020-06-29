package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go"
	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	zglob "github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

const (
	stateUnlocked uint32 = iota
	stateLocked
)

var (
	locker = stateUnlocked
)

func (app *booksingApp) refreshLoop() {
	for {
		app.refresh()
		time.Sleep(time.Minute)
	}
}

func (app *booksingApp) downloadBook(c *gin.Context) {

	hash := c.Query("hash")

	book, err := app.s.GetBookByHash(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"err":  err,
			"hash": hash,
		}).Error("could not find book")
		return
	}
	/* TODO: actually load user
	user := c.MustGet("id")
	username := user.(*booksing.User).Username
	*/
	username := "erwin@gnur.nl"

	ip := c.ClientIP()
	dl := booksing.Download{
		User:      username,
		IP:        ip,
		Book:      book.Hash,
		Timestamp: time.Now(),
	}
	err = app.db.AddDownload(dl)
	if err != nil {
		app.logger.WithField("err", err).Error("could not store download")
	}

	fName := path.Base(book.Path)
	c.Header("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", fName))
	c.File(book.Path)
	app.logger.WithField("path", book.Path).Info("Starting download")
}

func (app *booksingApp) getStats(c *gin.Context) {
	count := app.db.GetBookCount()
	c.JSON(200, gin.H{
		"total": count,
	})
}

func (app *booksingApp) updateUser(c *gin.Context) {
	id := c.Param("username")
	dbUser, err := app.db.GetUser(id)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not get user")
		c.JSON(500, gin.H{
			"text": "could not find user",
		})
		c.Abort()
	}
	var u booksing.User
	if err := c.ShouldBind(&u); err != nil {
		app.logger.WithField("err", err).Warning("could not get values from post")
		c.JSON(400, gin.H{
			"text": err.Error(),
		})
		return
	}

	dbUser.IsAllowed = u.IsAllowed
	err = app.db.SaveUser(&dbUser)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not update user")
		c.JSON(500, gin.H{
			"text": "could not find user",
		})
		c.Abort()
	}

	c.JSON(200, gin.H{"text": "ok"})
}

func (app *booksingApp) getBooks(c *gin.Context) {

	var resp bookResponse
	numString := c.DefaultQuery("results", "100")
	filter := strings.ToLower(c.Query("filter"))
	filter = strings.TrimSpace(filter)
	var limit int64
	limit = 1000

	app.logger.WithFields(logrus.Fields{
		//"user":   r.Header.Get("x-auth-user"),
		"filter": filter,
	}).Info("user initiated search")

	if a, err := strconv.ParseInt(numString, 10, 64); err == nil {
		if a > 0 && a < 1000 {
			limit = a
		}
	}
	resp.TotalCount = app.db.GetBookCount()

	books, err := app.s.GetBooks(filter, 0, limit)
	if err != nil {
		app.logger.WithField("err", err).Error("error retrieving books")
	}
	resp.Books = books

	c.JSON(200, resp)
}

func (app *booksingApp) refresh() {
	if !atomic.CompareAndSwapUint32(&locker, stateUnlocked, stateLocked) {
		app.logger.Warning("not refreshing because it is already running")
		return
	}
	defer atomic.StoreUint32(&locker, stateUnlocked)
	defer func() {
		app.state = "idle"
	}()

	app.state = "indexing"
	for {
		results := booksing.RefreshResult{
			StartTime: time.Now(),
		}
		matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
		if err != nil {
			app.logger.WithField("err", err).Error("glob of all books failed")
			return
		}
		if len(matches) == 0 {
			return
		}
		app.logger.WithFields(logrus.Fields{
			"total":     len(matches),
			"bookdir":   app.importDir,
			"batchsize": app.cfg.BatchSize,
		}).Info("located books on filesystem, processing per batchsize")

		if len(matches) > app.cfg.BatchSize {
			matches = matches[0:app.cfg.BatchSize]
		}

		bookQ := make(chan string, len(matches))
		resultQ := make(chan parseResult)

		for w := 0; w < 5; w++ { //not sure yet how concurrent-proof my solution is
			go app.bookParser(bookQ, resultQ)
		}

		for _, filename := range matches {
			bookQ <- filename
		}

		for a := 0; a < len(matches); a++ {
			r := <-resultQ

			switch r {
			case OldBook:
				results.Old++
			case InvalidBook:
				results.Invalid++
			case AddedBook:
				results.Added++
			case DuplicateBook:
				results.Duplicate++
			case DBErrorBook:
				results.Errors++
			}
			if a > 0 && a%10 == 0 {
				app.logger.WithFields(logrus.Fields{
					"processed": a,
					"old":       results.Old,
					"invalid":   results.Invalid,
					"duplicate": results.Duplicate,
					"added":     results.Added,
					"errors":    results.Errors,
					"total":     len(matches),
				}).Info("processing books")
			}

		}
		total := app.db.GetBookCount()
		if err != nil {
			app.logger.WithField("err", err).Error("could not get total book count")
		}
		results.Old = total
		results.StopTime = time.Now()

		app.logger.WithFields(logrus.Fields{
			"old":       results.Old,
			"invalid":   results.Invalid,
			"duplicate": results.Duplicate,
			"added":     results.Added,
			"errors":    results.Errors,
		}).Info("finished refresh of booklist")
		if results.Added > 0 {
			err := app.db.UpdateBookCount(results.Added)
			if err != nil {
				app.logger.WithFields(logrus.Fields{
					"err": err,
				}).Warning("Failed to update book count in database")
			}
		}
	}
}
func (app *booksingApp) refreshBooks(c *gin.Context) {
	app.refresh()
}

func (app *booksingApp) bookParser(bookQ chan string, resultQ chan parseResult) {
	for filename := range bookQ {
		book, err := booksing.NewBookFromFile(filename, app.bookDir)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"file":   filename,
				"reason": "invalid",
			}).Info("Deleting book")
			app.moveBookToFailed(book)
			resultQ <- InvalidBook
			continue
		}
		exists, err := app.db.HasHash(book.Hash)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"hash": book.Hash,
				"err":  err,
			}).Warning("Unable to get hash from db")
			app.moveBookToFailed(book)
		}
		if exists {
			os.Remove(filename)
			resultQ <- DuplicateBook
		}

		err = retry.Do(func() error {
			return app.s.AddBook(book)
		},
			retry.Attempts(10),
			retry.DelayType(retry.BackOffDelay),
			retry.RetryIf(func(err error) bool {
				return err != booksing.ErrDuplicate
			}),
			retry.LastErrorOnly(true),
		)

		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"file": filename,
				"err":  err,
			}).Error("could not store book")

			app.moveBookToFailed(book)
			resultQ <- DBErrorBook
		} else {
			err = app.db.AddHash(book.Hash)
			if err != nil {
				app.logger.WithError(err).Error("failed storing hash in db")
			}
			resultQ <- AddedBook
		}
	}
}

type deleteRequest struct {
	Hash string `form:"hash"`
}

func (app *booksingApp) deleteBook(c *gin.Context) {
	var req deleteRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{
			"text": err.Error(),
		})
		return
	}
	app.logger.WithField("req", req).Info("got delete request")
	hash := req.Hash

	book, err := app.s.GetBookByHash(hash)
	if err != nil {
		return
	}

	err = os.Remove(book.Path)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
			"path": book.Path,
		}).Error("Could not delete book from filesystem")
	}

	err = app.s.DeleteBook(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("Could not delete book from database")
		return
	}
	app.logger.WithFields(logrus.Fields{
		"hash": hash,
	}).Info("book was deleted")
	c.JSON(200, gin.H{
		"text": "ok",
	})
}

func (app *booksingApp) moveBookToFailed(b *booksing.Book) {
	err := os.MkdirAll(app.cfg.FailDir, 0755)
	if err != nil {
		app.logger.WithError(err).Fatal("unable to create fail dir")
		return
	}
	bookpath := b.Path
	filename := path.Base(bookpath)
	newBookPath := path.Join(app.cfg.FailDir, filename)
	err = os.Rename(bookpath, newBookPath)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"faildir": app.cfg.FailDir,
			"book":    bookpath,
		}).WithError(err).Fatal("unable to move book to faildir")
		return
	}
}
