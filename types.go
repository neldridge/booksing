package booksing

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNonUniqueResult = errors.New("Query gave more then 1 result")
var ErrNotFound = errors.New("Query no results")
var ErrDuplicate = errors.New("Duplicate key")

// RefreshResult holds the result of a full refresh
type RefreshResult struct {
	gorm.Model
	StartTime time.Time
	StopTime  time.Time
	Old       int
	Added     int
	Duplicate int
	Invalid   int
	Errors    int
}

type Download struct {
	gorm.Model
	Book      string    `json:"hash"`
	User      string    `json:"user"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type BookCount struct {
	Date  string
	Count int
}

type SearchResult struct {
	Items []Book
	Total int64
}
