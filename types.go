package booksing

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNonUniqueResult = errors.New("Query gave more then 1 result")
var ErrNotFound = errors.New("Query no results")
var ErrDuplicate = errors.New("Duplicate key")

type Download struct {
	gorm.Model
	Book      string    `json:"hash"`
	User      string    `json:"user"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type SearchResult struct {
	Items []Book
	Total int64
}
