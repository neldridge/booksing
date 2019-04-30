package main

import (
	"time"
)

type bookResponse struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total"`
	timestamp  time.Time
}

type parseResult int32

// hold all possible book parse results
const (
	OldBook       parseResult = iota
	AddedBook     parseResult = iota
	DuplicateBook parseResult = iota
	InvalidBook   parseResult = iota
)

// RefreshResult holds the result of a full refresh
type RefreshResult struct {
	StartTime time.Time
	StopTime  time.Time
	Old       int
	Added     int
	Duplicate int
	Invalid   int
}

type download struct {
	Book      string    `json:"hash"`
	User      string    `json:"user"`
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

type pipelineResult struct {
	Title  string   `bson:"_id"`
	Count  int      `bson:"count"`
	Hashes []string `bson:"docs"`
}

type booksingApp struct {
	db            database
	allowDeletes  bool
	allowOrganize bool
	bookDir       string
	importDir     string
}

type database interface {
	AddBook(*Book) error
	BookCount() int
	GetBook(string) (*Book, error)
	DeleteBook(string) error
	GetBooks(string, int) ([]Book, error)
	SetBookConverted(string) error

	AddDownload(download) error
	GetDownloads(int) ([]download, error)

	AddRefresh(RefreshResult) error
	GetRefreshes(int) ([]RefreshResult, error)
}
