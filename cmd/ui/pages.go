package main

import (
	"math"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (app *booksingApp) search(c *gin.Context) {

	start := time.Now()
	var offset int64
	var limit int64
	var err error
	offset = 0
	limit = 100
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
			limit = 100
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
		Results:   len(books),
		TimeTaken: latency,
		Books:     books,
		Error:     err,
		Q:         q,
	})

}
