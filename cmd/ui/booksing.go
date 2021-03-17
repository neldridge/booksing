package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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
		time.Sleep(time.Minute)
	}
}

func (app *booksingApp) downloadBook(c *gin.Context) {

	hash := c.Query("hash")
	file := c.Query("file")

	book, err := app.db.GetBook(hash)
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

	if app.cfg.MQTTEnabled {
		e, err := newEvent("booksing", "xyz.dekeijzer.booksing.download", map[string]string{
			"user": username,
			"ip":   ip,
			"book": book.Hash,
		})
		if err != nil {
			app.logger.WithField("err", err).Error("could not create dl event")
		}
		err = app.pushEvent(e)
		if err != nil {
			app.logger.WithField("err", err).Error("could not push dl event")
		}
	}

	if file != "" {
		fName := path.Base(file)
		c.Header("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"%s\"", fName))
		c.File(file)
	} else {
		fName := path.Base(book.Path)
		c.Header("Content-Disposition",
			fmt.Sprintf("attachment; filename=\"%s\"", fName))
		c.File(book.Path)
	}
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
		statusGauge.Set(0)
	}()

	app.state = "indexing"
	statusGauge.Set(1)
	matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
	if err != nil {
		app.logger.WithField("err", err).Error("glob of all books failed")
		return
	}

	app.logger.WithFields(logrus.Fields{
		"total":     len(matches),
		"bookdir":   app.importDir,
		"batchsize": app.cfg.BatchSize,
	}).Info("located books on filesystem, processing per batchsize")

	for _, filename := range matches {
		app.bookQ <- filename
	}

}

func (app *booksingApp) refreshBooks(c *gin.Context) {
	app.refresh()
}

func (app *booksingApp) bookParser() {
	epubParseProccessed := booksProcessed.WithLabelValues("parse")
	epubParseTime := booksProcessedTime.WithLabelValues("parse")

	for filename := range app.bookQ {
		app.logger.WithField("f", filename).Debug("parsing book")
		start := time.Now()
		book, err := booksing.NewBookFromFile(filename, app.bookDir)
		duration := time.Since(start).Microseconds()
		epubParseProccessed.Inc()
		epubParseTime.Add(float64(duration) / 1000000)

		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"file":   filename,
				"reason": "invalid",
				"err":    err,
			}).Info("Moving book to failed")
			app.moveBookToFailed(filename)
			continue
		}

		//all books get added to search, even duplicates
		app.searchQ <- *book

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

func (app *booksingApp) searchUpdater() {
	lastSave := time.Now()
	ticker := time.NewTicker(app.saveInterval)
	var books []booksing.Book
	searchProcessed := booksProcessed.WithLabelValues("search")
	searchTime := booksProcessedTime.WithLabelValues("search")
	searchErrors := searchErrorsMetric.WithLabelValues("update")

	for {
		app.logger.WithField("bookstoupdate", len(books)).Debug("search books ready to update")
		select {
		case <-ticker.C:
			app.logger.Debug("Storing results from ticker")
			if time.Since(lastSave) < app.saveInterval {
				continue
			} else if len(books) == 0 {
				continue
			}
			start := time.Now()
			err := app.db.AddBooks(books, false)
			if err != nil {
				app.logger.WithFields(logrus.Fields{
					"err": err,
				}).Error("Failed updating search index")
				searchErrors.Inc()
			} else {
				//only update metrics if it succeeded
				duration := time.Since(start).Microseconds()
				searchProcessed.Add(float64(len(books)))
				searchTime.Add(float64(duration) / 1000000)
			}

			books = []booksing.Book{}
			lastSave = time.Now()
		case b := <-app.searchQ:
			books = append(books, b)

			if len(books) >= app.cfg.BatchSize {
				start := time.Now()
				err := app.db.AddBooks(books, true)
				if err != nil {
					searchErrors.Inc()
					app.logger.WithFields(logrus.Fields{
						"err": err,
					}).Error("Failed updating search index")
				} else {
					//only update metrics if it succeeded
					duration := time.Since(start).Microseconds()
					searchProcessed.Add(float64(len(books)))
					searchTime.Add(float64(duration) / 1000000)
				}
				books = []booksing.Book{}
				lastSave = time.Now()
			}
		}
	}
}
