package main

import (
	"github.com/asdine/storm"
)

type stormDB struct {
	db storm.DB
}

func newStormDB(path string) (*stormDB, error) {

	database := stormDB{}

	return &database, nil
}

func (db *stormDB) AddBook(b *Book) error {
	return nil
}

func (db *stormDB) GetBook(q string) (*Book, error) {
	return nil, nil
}

func (db *stormDB) DeleteBook(hash string) error {
	return nil
}

func (db *stormDB) SetBookConverted(hash string) error {
	return nil
}

func (db *stormDB) GetBooks(q string, limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) searchMetaphoneKeys(q string, limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) searchExact(q string, limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) filterBooksBQL(q string, limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) getRecentBooks(limit int) ([]Book, error) {
	return nil, nil
}

func (db *stormDB) AddDownload(dl download) error {
	return nil
}
func (db *stormDB) GetDownloads(limit int) ([]download, error) {
	return nil, nil
}
func (db *stormDB) BookCount() int {
	return 0
}

func (db *stormDB) AddRefresh(rr RefreshResult) error {
	return nil
}
func (db *stormDB) GetRefreshes(limit int) ([]RefreshResult, error) {
	return nil, nil
}
