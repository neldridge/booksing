package meili

import (
	"fmt"
	"time"

	"github.com/gnur/booksing"
)

type Meili struct {
}

func New() *Meili {
	return &Meili{}
}

func (s *Meili) AddBook(b *booksing.Book) error {
	fmt.Println("Adding ", b.Title)
	return nil
}

func (s *Meili) AddBooks(b []booksing.Book) error {
	return nil
}

func (s *Meili) BookCount() int {
	return 4
}

func (s *Meili) GetBook(q string) (*booksing.Book, error) {
	return &booksing.Book{
		ID:          1,
		Author:      "auteur 1",
		Title:       "titel 1",
		Added:       time.Now(),
		Description: "",
	}, nil
}

func (s *Meili) DeleteBook(hash string) error {
	return nil
}

func (s *Meili) GetBooks(string, int) ([]booksing.Book, error) {
	return []booksing.Book{
		{
			ID:          1,
			Author:      "auteur 1",
			Title:       "titel 1",
			Added:       time.Now(),
			Description: "",
		},
		{
			ID:          2,
			Author:      "auteur 2",
			Title:       "titel 2",
			Added:       time.Now().Add(-5 * time.Second),
			Description: "",
		},
	}, nil
}

func (s *Meili) GetBookBy(string, string) (*booksing.Book, error) {
	return nil, nil
}
