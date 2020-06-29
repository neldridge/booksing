package meili

import (
	"fmt"
	"strings"
	"time"

	"github.com/gnur/booksing"
	"github.com/meilisearch/meilisearch-go"
)

type Meili struct {
	client *meilisearch.Client
	index  string
}

func New(host, index, key string) (*Meili, error) {
	client := meilisearch.NewClient(meilisearch.Config{
		Host:   host,
		APIKey: key,
	})
	// Create an index if your index does not already exist
	_, err := client.Indexes().Create(meilisearch.CreateIndexRequest{
		UID:        index,
		PrimaryKey: "Hash",
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return nil, fmt.Errorf("Unable to create index: %w", err)
	}
	return &Meili{
		client: client,
		index:  index,
	}, nil
}

func (s *Meili) AddBook(b *booksing.Book) error {
	_, err := s.client.Documents(s.index).AddOrUpdate([]booksing.Book{*b})
	if err != nil {
		return fmt.Errorf("Unable to insert book: %w", err)
	}
	return nil
}

func (s *Meili) AddBooks(books []booksing.Book) (*booksing.AddBooksResult, error) {
	var res booksing.AddBooksResult
	for _, b := range books {
		err := s.AddBook(&b)
		if err != nil {
			res.Errors++
		} else {
			res.Added++
		}
	}
	return &res, nil
}

func (s *Meili) GetBook(q string) (*booksing.Book, error) {
	return &booksing.Book{
		Author:      "auteur 1",
		Title:       "titel 1",
		Added:       time.Now(),
		Description: "",
	}, nil
}

func (s *Meili) DeleteBook(hash string) error {
	return nil
}

func (s *Meili) GetBooks(q string, limit, offset int64) ([]booksing.Book, error) {

	var books []booksing.Book
	var hits []interface{}

	if q == "" {
		for tDiff := 0 * time.Hour; tDiff < 720*time.Hour; tDiff += 24 * time.Hour {
			q := time.Now().Add(-1 * tDiff).Format("2006-01-02")
			res, err := s.client.Search(s.index).Search(meilisearch.SearchRequest{
				Query:  q,
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return nil, fmt.Errorf("Unable to get results from meili: %w", err)
			}
			if len(res.Hits) > 0 {
				hits = res.Hits
				break
			}
		}
	} else {

		res, err := s.client.Search(s.index).Search(meilisearch.SearchRequest{
			Query:  q,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("Unable to get results from meili: %w", err)
		}
		hits = res.Hits
	}

	for _, hit := range hits {
		m, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		var b booksing.Book
		b.Title = m["Title"].(string)
		b.Author = m["Author"].(string)
		b.Description = m["Description"].(string)
		b.Hash = m["Hash"].(string)
		b.Added, _ = time.Parse(time.RFC3339, m["Added"].(string))

		books = append(books, b)
	}

	return books, nil
}

func (s *Meili) GetBookByHash(hash string) (*booksing.Book, error) {
	var b booksing.Book
	err := s.client.Documents(s.index).Get(hash, &b)
	return &b, err
}
