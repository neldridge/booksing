package main

import (
	"github.com/gin-gonic/gin"
)

func (app *booksingApp) search(c *gin.Context) {

	books, err := app.s.GetBooks("hoi", 10)
	if err != nil {
		c.HTML(500, "error.html", V{
			Error: err,
		})
		return
	}

	c.HTML(200, "search.html", V{
		Results: len(books),
		Books:   books,
		Error:   err,
	})

}
