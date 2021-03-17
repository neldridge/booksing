package main

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

// booksingApp holds all relevant global stuff for the booksing server
type booksingApp struct {
	db         database
	mqttClient mqtt.Client
	bookDir    string
	//importDir is very important
	importDir string
	logger    *logrus.Entry
	timezone  *time.Location
	adminUser string
	cfg       configuration
	state     string
}

type database interface {
	AddDownload(booksing.Download) error
	GetDownloads(int) ([]booksing.Download, error)

	SaveUser(*booksing.User) error
	GetUser(string) (booksing.User, error)

	GetUsers() ([]booksing.User, error)

	GetBookCount() int

	HasHash(string) (bool, error)

	Close()

	AddBooks([]booksing.Book) error
	AddBook(booksing.Book) error
	GetBook(string) (*booksing.Book, error)
	DeleteBook(string) error
	GetBooks(string, int64, int64) (*booksing.SearchResult, error)
}
