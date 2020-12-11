package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
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
		time.Sleep(time.Minute)
	}
}

func (app *booksingApp) downloadBook(c *gin.Context) {

	hash := c.Query("hash")

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

	//there can be some race conditions, but these will be moved to the failed dir in the worst case scenario

	//we can assume all books have been processed here, lets move all files to failed dir
	matches, err = zglob.Glob(filepath.Join(app.importDir, "/**/*.*"))
	if err != nil {
		app.logger.WithError(err).Error("Failed globbing remaining files")
		return
	}
	for _, file := range matches {
		if strings.HasSuffix(file, ".epub") {
			continue
		}
		app.logger.WithField("file", file).Info("moving file to failed dir")
		app.moveBookToFailed(file)
	}

	//only empty directories remain, let's delete them to make it easier on the filesystem
	matches, err = zglob.Glob(filepath.Join(app.importDir, "/**"))
	if err != nil {
		app.logger.WithError(err).Error("Failed globbing remaining empty dirs")
		return
	}
	for _, dir := range matches {
		app.logger.WithField("file", dir).Info("deleting")
		err := os.RemoveAll(dir)
		if err != nil {
			app.logger.WithError(err).Error("Could not delete dir")
		}
	}

}

func (app *booksingApp) resultParser() {
	a := 0
	results := booksing.RefreshResult{
		StartTime: time.Now(),
	}
	ticker := time.NewTicker(app.saveInterval)

	for {
		select {
		case <-ticker.C:
			app.logger.Debug("Storing results from ticker")
			if time.Since(results.StartTime) < app.saveInterval {
				continue
			}
			err := app.storeResult(results)
			if err != nil {
				app.logger.WithFields(logrus.Fields{
					"err": err,
				}).Warning("Failed to update book count in database")
			}
			results = booksing.RefreshResult{
				StartTime: time.Now(),
			}
		case r := <-app.resultQ:
			app.logger.Debug("Got result from resultQ")
			a++

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
			if a > 0 && a%app.cfg.BatchSize == 0 {
				err := app.storeResult(results)
				if err != nil {
					app.logger.WithFields(logrus.Fields{
						"err": err,
					}).Warning("Failed to update book count in database")
				}
				results = booksing.RefreshResult{
					StartTime: time.Now(),
				}
			}
		}
		currentTotal := app.db.GetBookCount()
		totalBooksGauge.Set(float64(currentTotal))
	}
}

func (app *booksingApp) storeResult(res booksing.RefreshResult) error {
	if res.Added <= 0 {
		return nil
	}
	processingTime := time.Since(res.StartTime)
	app.logger.WithFields(logrus.Fields{
		"invalid":   res.Invalid,
		"duplicate": res.Duplicate,
		"added":     res.Added,
		"errors":    res.Errors,
		"timetaken": processingTime.String(),
	}).Info("finished another batch")

	return app.db.UpdateBookCount(res.Added)
}

func (app *booksingApp) refreshBooks(c *gin.Context) {
	app.refresh()
}

func (app *booksingApp) bookParser() {
	epubParseProccessed := booksProcessed.WithLabelValues("parse")
	epubParseTime := booksProcessedTime.WithLabelValues("parse")

	indexProccessed := booksProcessed.WithLabelValues("index")
	indexTime := booksProcessedTime.WithLabelValues("index")
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
			}).Info("Deleting book")
			app.resultQ <- InvalidBook
			app.moveBookToFailed(filename)
			continue
		}
		exists, err := app.db.HasHash(book.Hash)
		if err != nil {
			app.logger.WithFields(logrus.Fields{
				"hash": book.Hash,
				"err":  err,
			}).Warning("Unable to get hash from db")
			app.resultQ <- DBErrorBook
			app.moveBookToFailed(filename)
			dbErrors.WithLabelValues("read").Inc()
			continue
		}

		//all books get added to search, even duplicates
		app.searchQ <- *book

		if exists {
			app.resultQ <- DuplicateBook
			continue
		}

		start = time.Now()
		err = app.db.AddHash(book.Hash)
		if err != nil {
			dbErrors.WithLabelValues("write").Inc()
			app.logger.WithError(err).Error("failed storing hash in db")
		} else {
			duration := time.Since(start).Microseconds()
			indexProccessed.Inc()
			indexTime.Add(float64(duration) / 10000000)
		}
		app.resultQ <- AddedBook
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
