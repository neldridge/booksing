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

func (s *Meili) AddBooks(books []booksing.Book, sync bool) error {

	//BUG workaround: https://github.com/meilisearch/MeiliSearch/issues/827
	uniquebooks := []booksing.Book{}
	hashes := make(map[string]bool)
	for _, b := range books {
		if _, present := hashes[b.Hash]; !present {
			uniquebooks = append(uniquebooks, b)
			hashes[b.Hash] = true
		}
	}

	id, err := s.client.Documents(s.index).AddOrReplace(uniquebooks)
	if err != nil {
		return fmt.Errorf("Unable to insert books: %w", err)
	}

	if sync {
		for {
			up, err := s.client.Updates(s.index).Get(id.UpdateID)
			if up.Status == "processed" {
				break
			}
			if err != nil {
				return fmt.Errorf("Unable to get update status for updateID %v books: %w", id.UpdateID, err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}

func (s *Meili) GetBook(hash string) (*booksing.Book, error) {
	var b booksing.Book
	err := s.client.Documents(s.index).Get(hash, &b)
	return &b, err
}

func (s *Meili) DeleteBook(hash string) error {
	_, err := s.client.Documents(s.index).Delete(hash)
	return err
}

func (s *Meili) GetBooks(q string, limit, offset int64) (*booksing.SearchResult, error) {

	var books []booksing.Book
	var hits []interface{}
	var totalHits int64

	if q == "" {
		for tDiff := 0 * time.Hour; tDiff < 720*time.Hour; tDiff += 24 * time.Hour {
			q := time.Now().Add(-1 * tDiff).Format("2006-01-02")
			var res *meilisearch.SearchResponse
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
			totalHits = res.NbHits
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
		totalHits = res.NbHits
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

	return &booksing.SearchResult{
		Items: books,
		Total: totalHits,
	}, nil
}

func (s *Meili) GetBookByHash(hash string) (*booksing.Book, error) {
	var b booksing.Book
	err := s.client.Documents(s.index).Get(hash, &b)
	return &b, err
}
