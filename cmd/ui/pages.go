package main

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
)

func (app *booksingApp) search(c *gin.Context) {

	start := time.Now()
	var offset int64
	var limit int64
	var err error
	offset = 0
	limit = 30
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
			limit = 30
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
		IsAdmin:    app.IsUserAdmin(c),
		TotalBooks: app.db.GetBookCount(),
	})
}

func (app *booksingApp) showUsers(c *gin.Context) {

	u, ok := c.Get("id")
	if !ok {
		c.HTML(403, "error.html", V{
			Error: fmt.Errorf("Unable to retrieve user from context"),
		})
		c.Abort()
		return
	}

	user, ok := u.(*booksing.User)
	if !ok {
		c.HTML(403, "error.html", V{
			Error: fmt.Errorf("Unable to cast id into booksing.User: %+v", u),
		})
		c.Abort()
		return
	}

	if !user.IsAdmin {
		c.HTML(403, "error.html", V{
			Error: fmt.Errorf("You don't have permission to do that"),
		})
		c.Abort()
		return
	}

	users, err := app.db.GetUsers()
	if err != nil {
		c.HTML(403, "error.html", V{
			Error: err,
		})
		c.Abort()
		return
	}

	c.JSON(200, gin.H{
		"status": "ok",
		"users":  users,
	})
}
