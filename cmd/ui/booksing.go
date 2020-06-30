package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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
	user := c.MustGet("id")
	username := user.(*booksing.User).Name

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
}

func (app *booksingApp) updateUser(c *gin.Context) {
	id := c.Param("username")
	dbUser, err := app.db.GetUser(id)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not get user")
		c.HTML(500, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}
	var u booksing.User
	if err := c.ShouldBind(&u); err != nil {
		app.logger.WithField("err", err).Warning("could not get values from post")
		c.HTML(400, "error.html", V{
			Error: err,
		})
		c.Abort()
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

	c.Redirect(302, c.Request.Referer())
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
			resultQ <- InvalidBook
			app.moveBookToFailed(filename)
			continue
		}
		exists, err := app.db.HasHash(book.Hash)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"hash": book.Hash,
				"err":  err,
			}).Warning("Unable to get hash from db")
			resultQ <- DBErrorBook
			app.moveBookToFailed(filename)
			continue
		}
		if exists {
			os.Remove(filename)
			resultQ <- DuplicateBook
			continue
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

			app.moveBookToFailed(filename)
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

func (app *booksingApp) moveBookToFailed(bookpath string) {
	err := os.MkdirAll(app.cfg.FailDir, 0755)
	if err != nil {
		app.logger.WithError(err).Error("unable to create fail dir")
		return
	}
	filename := path.Base(bookpath)
	newBookPath := path.Join(app.cfg.FailDir, filename)
	err = os.Rename(bookpath, newBookPath)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"faildir": app.cfg.FailDir,
			"book":    bookpath,
		}).WithError(err).Error("unable to move book to faildir")
		return
	}
}

func (app *booksingApp) addUser(c *gin.Context) {
	var u booksing.User
	if err := c.ShouldBind(&u); err != nil {
		app.logger.WithField("err", err).Warning("could not get values from post")
		c.HTML(400, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	u.Created = time.Now().In(app.timezone)
	err := app.db.SaveUser(&u)
	if err != nil {
		app.logger.WithField("err", err).Error("Could not update user")
		c.JSON(500, gin.H{
			"text": "could not find user",
		})
		c.Abort()
	}

	c.Redirect(302, c.Request.Referer())
}
