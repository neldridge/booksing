package main

import (
	"time"

	"firebase.google.com/go/auth"
	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

type booksingApp struct {
	db            database
	allowDeletes  bool
	allowOrganize bool
	bookDir       string
	importDir     string
	logger        *logrus.Entry
	authClient    *auth.Client
	timezone      *time.Location
	FQDN          string
	adminUser     string
	cfg           configuration
}

type bookResponse struct {
	Books      []booksing.Book `json:"books"`
	TotalCount int             `json:"total"`
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

type database interface {
	AddBook(*booksing.Book) error
	AddBooks([]booksing.Book) error

	BookCount() int
	GetBook(string) (*booksing.Book, error)
	DeleteBook(string) error
	GetBooks(string, int) ([]booksing.Book, error)

	GetBookBy(string, string) (*booksing.Book, error)

	AddDownload(booksing.Download) error
	GetDownloads(int) ([]booksing.Download, error)

	SaveUser(*booksing.User) error
	GetUser(string) (booksing.User, error)

	GetUsers() ([]booksing.User, error)

	Close()
}
