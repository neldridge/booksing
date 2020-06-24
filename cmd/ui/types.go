package main

import (
	"html/template"
	"time"

	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

type booksingApp struct {
	s             search
	db            database
	allowDeletes  bool
	allowOrganize bool
	bookDir       string
	importDir     string
	logger        *logrus.Entry
	timezone      *time.Location
	FQDN          string
	adminUser     string
	cfg           configuration
	templates     *template.Template
}

type bookResponse struct {
	Books      []booksing.Book `json:"books"`
	TotalCount int             `json:"total"`
}

type parseResult int32

// hold all possible book parse results
const (
	OldBook       parseResult = iota
	AddedBook     parseResult = iota
	DuplicateBook parseResult = iota
	InvalidBook   parseResult = iota
)

type database interface {
	AddDownload(booksing.Download) error
	GetDownloads(int) ([]booksing.Download, error)

	SaveUser(*booksing.User) error
	GetUser(string) (booksing.User, error)

	GetUsers() ([]booksing.User, error)

	Close()
}

type search interface {
	AddBook(*booksing.Book) error
	AddBooks([]booksing.Book) error

	BookCount() int
	GetBook(string) (*booksing.Book, error)
	DeleteBook(string) error
	GetBooks(string, int) ([]booksing.Book, error)

	GetBookBy(string, string) (*booksing.Book, error)
}
