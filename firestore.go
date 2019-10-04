package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
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

	_, err := db.GetBookBy("Hash", b.Hash)
	if err == nil {
		return ErrDuplicate
	}
	_, err = db.client.Collection("books").Doc(b.Hash).Set(ctx, b)

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

func (db *fireDB) GetBookBy(field, value string) (*Book, error) {
	ctx := context.Background()
	iter := db.client.Collection("books").Where(field, "==", value).Limit(5).Documents(ctx)

	var books []Book
	var b Book
	for {
		b = Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}

	if len(books) > 1 {
		return nil, ErrNonUniqueResult
	}
	if len(books) == 0 {
		return nil, ErrNotFound
	}

	return &books[0], nil
}

func (db *fireDB) DeleteBook(hash string) error {
	ctx := context.Background()
	_, err := db.client.Collection("books").Doc(hash).Delete(ctx)
	return err
}

func (db *fireDB) SetBookConverted(hash string) error {
	ctx := context.Background()
	book, _ := db.GetBookBy("Hash", hash)
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
		return []Book{}, err
	}
	if len(books) > 0 {
		return books, nil
	}

	books, err = db.searchMetaphoneKeys(q, limit)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return []Book{}, err
	}
	return books, nil
}

func (db *fireDB) searchMetaphoneKeys(q string, limit int) ([]Book, error) {
	ctx := context.Background()

	longestTermLength := 0
	longestTerm := ""

	terms := getMetaphoneKeys(q)

	for _, term := range terms {
		if len(term) > longestTermLength {
			longestTerm = term
			longestTermLength = len(term)
		}
	}

	iter := db.client.Collection("books").Where("MetaphoneKeys", "array-contains", longestTerm).Limit(limit).Documents(ctx)

	books, err := iterToBookList(iter)
	if err != nil {
		return nil, err
	}

	var retBooks []Book
	for _, book := range books {
		if book.HasMetaphoneKeys(terms) {
			retBooks = append(retBooks, book)
		}
	}
	return retBooks, nil
}

func (db *fireDB) searchExact(q string, limit int) ([]Book, error) {
	ctx := context.Background()

	longestTermLength := 0
	longestTerm := ""

	terms := strings.Split(q, " ")

	for _, term := range terms {
		if len(term) > longestTermLength {
			longestTerm = term
			longestTermLength = len(term)
		}
	}

	iter := db.client.Collection("books").Where("SearchWords", "array-contains", longestTerm).Limit(limit).Documents(ctx)

	books, err := iterToBookList(iter)
	if err != nil {
		return nil, err
	}

	var retBooks []Book
	for _, book := range books {
		if book.HasSearchWords(terms) {
			retBooks = append(retBooks, book)
		}
	}
	return retBooks, nil
}

func (db *fireDB) filterBooksBQL(q string, limit int) ([]Book, error) {
	query := db.parseQuery(q)
	var books []Book
	var b Book

	ctx := context.Background()
	iter := query.OrderBy("Hash", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		b = Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}

	return books, nil
}

func (db *fireDB) getRecentBooks(limit int) ([]Book, error) {
	var books []Book
	ctx := context.Background()
	var b Book
	iter := db.client.Collection("books").OrderBy("Added", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		b = Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}
	return books, nil
}

func (db *fireDB) AddDownload(dl download) error {
	ctx := context.Background()
	_, _, err := db.client.Collection("downloads").Add(ctx, dl)
	return err
}
func (db *fireDB) GetDownloads(limit int) ([]download, error) {
	var dls []download
	ctx := context.Background()
	var d download
	iter := db.client.Collection("refreshes").OrderBy("StartTime", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		d = download{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&d)
		if err == nil {
			dls = append(dls, d)
		}
	}
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
	ctx := context.Background()
	var r RefreshResult
	iter := db.client.Collection("refreshes").OrderBy("StartTime", firestore.Desc).Limit(limit).Documents(ctx)
	for {
		r = RefreshResult{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&r)
		if err == nil {
			refreshes = append(refreshes, r)
		}
	}

	return refreshes, nil
}

func (db *fireDB) parseQuery(s string) firestore.Query {
	col := db.client.Collection("books")
	var q firestore.Query
	params := strings.Split(s, ",")
	first := true
	for _, param := range params {
		parts := strings.Split(param, ":")
		if len(parts) != 2 {
			continue
		}

		field := strings.TrimSpace(parts[0])
		filter := strings.TrimSpace(parts[1])

		if first {
			q = col.Where(field, "==", filter)
		} else {
			q = q.Where(field, "==", filter)
		}
	}

	return q
}

func iterToBookList(iter *firestore.DocumentIterator) ([]Book, error) {
	var books []Book
	var b Book
	for {
		b = Book{}
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to iterate: %v", err)
		}
		err = doc.DataTo(&b)
		if err == nil {
			books = append(books, b)
		}
	}
	return books, nil
}
