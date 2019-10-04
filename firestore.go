package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	log "github.com/sirupsen/logrus"
)

type fireDB struct {
	client *firestore.Client
}

func newFireStore(projectID string) (*fireDB, error) {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	db := fireDB{
		client: client,
	}
	return &db, nil
}
func (db *fireDB) Close() {
	db.client.Close()
}

func (db *fireDB) AddBook(b *Book) error {
	ctx := context.Background()
	_, err := db.client.Collection("books").Doc(b.Hash).Set(ctx, b)

	return err
}

func (db *fireDB) GetBook(q string) (*Book, error) {
	results, err := db.filterBooksBQL(q, 10)
	if err != nil {
		return nil, err
	}
	if len(results) > 1 {
		return nil, ErrNonUniqueResult
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return &results[0], nil
}

func (db *fireDB) DeleteBook(hash string) error {
	ctx := context.Background()
	_, err := db.client.Collection("books").Doc(hash).Delete(ctx)
	return err
}

func (db *fireDB) SetBookConverted(hash string) error {
	ctx := context.Background()
	book, _ := db.GetBook(fmt.Sprintf("hash: %s", hash))
	book.HasMobi = true

	_, err := db.client.Collection("books").Doc(hash).Set(ctx, book)

	return err
}

func (db *fireDB) GetBooks(q string, limit int) ([]Book, error) {
	if q == "" {
		return db.getRecentBooks(limit)
	}
	if strings.Contains(q, ":") {
		return db.filterBooksBQL(q, limit)
	}

	books, err := db.searchExact(q, limit)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return nil, err
	}
	if len(books) > 0 {
		return books, nil
	}

	return db.searchMetaphoneKeys(q, limit)
}

func (db *fireDB) searchMetaphoneKeys(q string, limit int) ([]Book, error) {
	var books []Book
	_ = getMetaphoneKeys(q)
	return books, nil
}

func (db *fireDB) searchExact(q string, limit int) ([]Book, error) {
	var books []Book
	return books, nil
}

func (db *fireDB) filterBooksBQL(q string, limit int) ([]Book, error) {
	_ = parseQuery(q)
	var books []Book
	return books, nil
}

func (db *fireDB) getRecentBooks(limit int) ([]Book, error) {
	var books []Book
	return books, nil
}

func (db *fireDB) AddDownload(dl download) error {
	ctx := context.Background()
	_, _, err := db.client.Collection("downloads").Add(ctx, dl)
	return err
}
func (db *fireDB) GetDownloads(limit int) ([]download, error) {
	var dls []download

	return dls, nil
}
func (db *fireDB) BookCount() int {
	return 0
}

func (db *fireDB) AddRefresh(rr RefreshResult) error {
	ctx := context.Background()
	_, _, err := db.client.Collection("refreshes").Add(ctx, rr)
	return err
}
func (db *fireDB) GetRefreshes(limit int) ([]RefreshResult, error) {
	var refreshes []RefreshResult

	return refreshes, nil
}
