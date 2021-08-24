package booksing

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNonUniqueResult = errors.New("query gave more then 1 result")
var ErrNotFound = errors.New("query no results")
var ErrDuplicate = errors.New("duplicate key")

type Download struct {
	gorm.Model
	Book      string    `json:"hash"`
	User      string    `json:"user" gorm:"index"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type SearchResult struct {
	Items []Book
	Total int64
}
