package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

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
		time.Sleep(time.Hour)
	}
}

func (app *booksingApp) downloadBook(c *gin.Context) {

	hash := c.Query("hash")

	book, err := app.s.GetBookBy("Hash", hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"err":  err,
			"hash": hash,
		}).Error("could not find book")
		return
	}
	user := c.MustGet("id")
	username := user.(*booksing.User).Username

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
	return
}

func (app *booksingApp) bookPresent(c *gin.Context) {
	author, _ := url.QueryUnescape(c.Param("author"))
	title, _ := url.QueryUnescape(c.Param("title"))

	author = booksing.Fix(author, true, true)
	title = booksing.Fix(title, true, false)

	hash := booksing.HashBook(author, title)
	app.logger.WithFields(logrus.Fields{
		"author": author,
		"title":  title,
		"hash":   hash,
	}).Info("checking if book exists")

	_, err := app.s.GetBookBy("Hash", hash)
	found := err == nil

	c.JSON(200, map[string]bool{"found": found})
}

func (app *booksingApp) getBook(c *gin.Context) {
	hash := c.Param("hash")
	book, err := app.s.GetBookBy("Hash", hash)
	if err != nil {
		return
	}
	c.JSON(200, book)
}

func (app *booksingApp) getStats(c *gin.Context) {
	count := app.s.BookCount()
	c.JSON(200, gin.H{
		"total": count,
	})
}

func (app *booksingApp) getUser(c *gin.Context) {
	id := c.MustGet("id")
	user := id.(*booksing.User)
	c.JSON(200, gin.H{
		"admin": user.IsAdmin,
	})
}

func (app *booksingApp) getDownloads(c *gin.Context) {
	downloads, _ := app.db.GetDownloads(200)
	c.JSON(200, downloads)
}

func (app *booksingApp) getUsers(c *gin.Context) {
	users, err := app.db.GetUsers()
	if err != nil {
		c.JSON(500, gin.H{
			"text": "oopsie",
		})
		c.Abort()
	}
	c.JSON(200, users)
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
	var limit int
	numString := c.DefaultQuery("results", "100")
	filter := strings.ToLower(c.Query("filter"))
	filter = strings.TrimSpace(filter)
	limit = 1000

	app.logger.WithFields(logrus.Fields{
		//"user":   r.Header.Get("x-auth-user"),
		"filter": filter,
	}).Info("user initiated search")

	if a, err := strconv.Atoi(numString); err == nil {
		if a > 0 && a < 1000 {
			limit = a
		}
	}
	resp.TotalCount = app.s.BookCount()

	books, err := app.s.GetBooks(filter, limit)
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
	app.logger.Info("starting refresh of booklist")
	results := booksing.RefreshResult{
		StartTime: time.Now(),
	}
	matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
	if err != nil {
		app.logger.WithField("err", err).Error("glob of all books failed")
		return
	}
	if len(matches) == 0 {
		app.logger.Info("finished refresh of booklist, no new books found")
		return
	}
	app.logger.WithFields(logrus.Fields{
		"total":   len(matches),
		"bookdir": app.importDir,
	}).Info("located books on filesystem")

	bookQ := make(chan string, len(matches))
	resultQ := make(chan parseResult)

	for w := 0; w < 6; w++ { //not sure yet how concurrent-proof my solution is
		go app.bookParser(bookQ, resultQ)
	}

	for _, filename := range matches {
		bookQ <- filename
	}

	for a := 0; a < len(matches); a++ {
		r := <-resultQ
		app.logger.WithField("r", r).Debug("got book result")

		switch r {
		case OldBook:
			results.Old++
		case InvalidBook:
			results.Invalid++
		case AddedBook:
			results.Added++
		case DuplicateBook:
			results.Duplicate++
		}
		if a > 0 && a%100 == 0 {
			app.logger.WithFields(logrus.Fields{
				"processed": a,
				"total":     len(matches),
			}).Info("processing books")
		}

	}
	total := app.s.BookCount()
	if err != nil {
		app.logger.WithField("err", err).Error("could not get total book count")
	}
	results.Old = total
	results.StopTime = time.Now()

	app.logger.WithFields(logrus.Fields{
		"result":  results,
		"old":     results.Old,
		"added":   results.Added,
		"invalid": results.Invalid,
		"other":   results.Duplicate,
	}).Info("finished refresh of booklist")
}
func (app *booksingApp) refreshBooks(c *gin.Context) {
	app.refresh()
}

func (app *booksingApp) bookParser(bookQ chan string, resultQ chan parseResult) {
	for filename := range bookQ {
		book, err := booksing.NewBookFromFile(filename, app.allowOrganize, app.bookDir)
		if err != nil {
			if app.allowDeletes {
				app.logger.WithFields(logrus.Fields{
					"file":   filename,
					"reason": "invalid",
				}).Info("Deleting book")
				os.Remove(filename)
			}
			resultQ <- InvalidBook
			continue
		}
		err = app.s.AddBook(book)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"file": filename,
				"err":  err,
			}).Error("could not store book")

			if err == booksing.ErrDuplicate {
				if app.allowDeletes {
					app.logger.WithFields(logrus.Fields{
						"file":   filename,
						"reason": "duplicate",
					}).Info("Deleting book")
					os.Remove(filename)
				}
				resultQ <- DuplicateBook
			}
		} else {
			resultQ <- AddedBook
		}
	}
}

func (app *booksingApp) addBook(c *gin.Context) {
	var b booksing.BookInput
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(400, gin.H{
			"text": "invalid input",
		})
		return
	}

	book := b.ToBook()

	book.Added = time.Now().In(app.timezone)

	err := app.s.AddBook(&book)
	if err != nil {
		c.JSON(500, gin.H{
			"text": err,
		})
		return
	}
	c.JSON(200, book)

}

func (app *booksingApp) addBooks(c *gin.Context) {
	var inBooks []booksing.BookInput
	if err := c.ShouldBindJSON(&inBooks); err != nil {
		c.JSON(400, gin.H{
			"text": "invalid input",
		})
		return
	}

	var books []booksing.Book

	var bo booksing.Book

	for _, b := range inBooks {
		bo = b.ToBook()
		bo.Added = time.Now().In(app.timezone)
		books = append(books, bo)
	}

	err := app.s.AddBooks(books)
	if err != nil {
		c.JSON(500, gin.H{
			"text": err,
		})
		return
	}
	c.JSON(200, gin.H{
		"ok": "yes",
	})

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

	book, err := app.s.GetBookBy("Hash", hash)
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
