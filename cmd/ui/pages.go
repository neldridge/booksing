package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (app *booksingApp) search(c *gin.Context) {

	start := time.Now()
	var offset int64
	var limit int64
	var err error
	offset = 0
	limit = 20
	q := c.Query("q")
	off := c.Query("o")
	if off != "" {
		offset, err = strconv.ParseInt(off, 10, 64)
		if err != nil {
			offset = 0
		}
	}
	lim := c.Query("l")
	if lim != "" {
		limit, err = strconv.ParseInt(lim, 10, 64)
		if err != nil {
			limit = 20
		}
	}

	books, err := app.s.GetBooks(q, limit, offset)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
			Q:     q,
		})
		return
	}

	stop := time.Since(start)
	latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
	c.HTML(200, "search.html", V{
		Limit:      limit,
		Offset:     offset,
		Results:    len(books),
		TimeTaken:  latency,
		Books:      books,
		Error:      err,
		Q:          q,
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.db.GetBookCount(),
	})
}

func (app *booksingApp) showUsers(c *gin.Context) {

	users, err := app.db.GetUsers()
	if err != nil {
		c.HTML(403, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	c.HTML(200, "users.html", V{
		Error:      err,
		Q:          "",
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.db.GetBookCount(),
		Users:      users,
	})

}

func (app *booksingApp) deleteBook(c *gin.Context) {
	hash := c.Param("hash")

	book, err := app.s.GetBookByHash(hash)
	if err != nil {
		c.HTML(404, "error.html", V{
			Error: errors.New("Book not found"),
		})
		return
	}

	err = os.Remove(book.Path)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
			"path": book.Path,
		}).Error("Could not delete book from filesystem")
		c.HTML(500, "error.html", V{
			Error: fmt.Errorf("Unable to delete book from filesystem: %w", err),
		})
		return
	}

	err = app.s.DeleteBook(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("Could not delete book from database")
		c.HTML(500, "error.html", V{
			Error: fmt.Errorf("Unable to delete book from database: %w", err),
		})
		return
	}
	err = app.db.UpdateBookCount(-1)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("could not update book count")
	}
	app.logger.WithFields(logrus.Fields{
		"hash": hash,
	}).Info("book was deleted")
	c.Redirect(302, c.Request.Referer())
}

func (app *booksingApp) showDownloads(c *gin.Context) {
	dls, err := app.db.GetDownloads(100)
	if err != nil {
		c.HTML(403, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	c.HTML(200, "downloads.html", V{
		Error:      err,
		Q:          "",
		IsAdmin:    c.GetBool("isAdmin"),
		TotalBooks: app.db.GetBookCount(),
		Downloads:  dls,
	})

}
