package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"strings"

	"github.com/beevik/etree"
	"golang.org/x/tools/godoc/vfs/zipfs"
)

func fix(s string, capitalize, correctOrder bool) string {
	if s == "" {
		return "Unknown"
	}
	if capitalize {
		s = strings.Title(strings.ToLower(s))
		s = strings.Replace(s, "'S", "'s", -1)
	}
	if correctOrder && strings.Contains(s, ",") {
		sParts := strings.Split(s, ",")
		if len(sParts) == 2 {
			s = strings.TrimSpace(sParts[1]) + " " + strings.TrimSpace(sParts[0])
		}
	}
	return strings.Map(func(in rune) rune {
		switch in {
		case '“', '‹', '”', '›':
			return '"'
		case '_':
			return ' '
		case '‘', '’':
			return '\''
		}
		return in
	}, s)
}

// NameID represents a name and an id
type NameID struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// Series represents a book series
type Series struct {
	NameID
	Index float64 `json:"index,omitempty"`
}

// Author represents a book author
type Author struct {
	NameID
}

// Book represents a book
type Book struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      Author `json:"author"`
	Description string `json:"description,omitempty"`
	Series      Series `json:"series,omitempty"`
	Filepath    string `json:"filepath"`
	HasMobi     bool   `json:"hasmobi"`
}

// NewBookFromFile creates a book object from a file
func NewBookFromFile(path string) (bk *Book, err error) {
	defer func() {
		if r := recover(); r != nil {
			bk = nil
			err = fmt.Errorf("Unknown error parsing book. Skipping. Error: %s", r)
		}
	}()

	book := new(Book)
	book.Title = filepath.Base(path)
	book.Filepath = path

	mobiPath := strings.Replace(path, "epub", "mobi", -1)
	_, err = os.Stat(mobiPath)
	book.HasMobi = !os.IsNotExist(err)

	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	zfs := zipfs.New(zr, "epub")

	rsk, err := zfs.Open("/META-INF/container.xml")
	if err != nil {
		return nil, err
	}
	defer rsk.Close()
	container := etree.NewDocument()
	_, err = container.ReadFrom(rsk)
	if err != nil {
		return nil, err
	}
	rootfile := ""
	for _, e := range container.FindElements("//rootfiles/rootfile[@full-path]") {
		rootfile = e.SelectAttrValue("full-path", "")
	}
	if rootfile == "" {
		return nil, errors.New("Cannot parse container")
	}

	rootReadSeeker, err := zfs.Open("/" + rootfile)
	if err != nil {
		return nil, err
	}
	defer rootReadSeeker.Close()
	opf := etree.NewDocument()
	_, err = opf.ReadFrom(rootReadSeeker)
	if err != nil {
		return nil, err
	}
	book.Title = filepath.Base(path)
	for _, e := range opf.FindElements("//title") {
		book.Title = e.Text()
		break
	}
	for _, e := range opf.FindElements("//creator") {
		book.Author.Name = e.Text()
		break
	}
	for _, e := range opf.FindElements("//description") {
		book.Description = e.Text()
		break
	}
	for _, e := range opf.FindElements("//meta[@name='calibre:series']") {
		book.Series.Name = e.SelectAttrValue("content", "")
		seriesID := sha1.New()
		io.WriteString(seriesID, book.Series.Name)
		book.Series.ID = hex.EncodeToString(seriesID.Sum(nil))[:10]
		break
	}
	for _, e := range opf.FindElements("//meta[@name='calibre:series_index']") {
		i, err := strconv.ParseFloat(e.SelectAttrValue("content", "0"), 64)
		if err == nil {
			book.Series.Index = i
			break
		}
	}

	book.Title = fix(book.Title, true, false)
	book.Author.Name = fix(book.Author.Name, true, true)
	book.Description = fix(book.Description, false, false)
	book.Series.Name = fix(book.Series.Name, true, false)

	id := sha1.New()
	io.WriteString(id, book.Author.Name)
	book.Author.ID = hex.EncodeToString(id.Sum(nil))[:10]
	io.WriteString(id, book.Title)
	book.ID = hex.EncodeToString(id.Sum(nil))[:10]

	return book, nil
}

// BookList is a list of books
type BookList []Book

// Sorted returns a copy of the BookList sorted by the function
func (l *BookList) Sorted(sorter func(a, b Book) bool) BookList {
	// Make a copy
	sorted := make(BookList, len(*l))
	copy(sorted, *l)
	// Sort the copy
	sort.Slice(sorted, func(i, j int) bool {
		return sorter(sorted[i], sorted[j])
	})
	return sorted
}

// Filtered returns a copy of the BookList filtered by the function
func (l *BookList) Filtered(filterer func(a Book) bool) *BookList {
	filtered := BookList{}
	for _, a := range *l {
		if filterer(a) {
			filtered = append(filtered, a)
		}
	}

	return &filtered
}

// AuthorList is a list of authors
type AuthorList []Author

// SeriesList is a list of series
type SeriesList []Series

// GetAuthors gets the authors in a BookList
func (l *BookList) GetAuthors() *AuthorList {
	authors := AuthorList{}
	done := map[string]bool{}
	for _, b := range *l {
		if done[b.Author.ID] {
			continue
		}
		authors = append(authors, b.Author)
		done[b.Author.ID] = true
	}
	return authors.Filtered(func(a Author) bool {
		return a.Name != ""
	})
}

// Sorted returns a copy of the AuthorList sorted by the function
func (l *AuthorList) Sorted(sorter func(a, b Author) bool) *AuthorList {
	// Make a copy
	sorted := make(AuthorList, len(*l))
	copy(sorted, *l)
	// Sort the copy
	sort.Slice(sorted, func(i, j int) bool {
		return sorter(sorted[i], sorted[j])
	})
	return &sorted
}

// Filtered returns a copy of the AuthorList filtered by the function
func (l *AuthorList) Filtered(filterer func(a Author) bool) *AuthorList {
	filtered := AuthorList{}
	for _, a := range *l {
		if filterer(a) {
			filtered = append(filtered, a)
		}
	}

	return &filtered
}

// GetSeries gets the series in a BookList
func (l *BookList) GetSeries() *SeriesList {
	series := SeriesList{}
	done := map[string]bool{}
	for _, b := range *l {
		if done[b.Series.ID] {
			continue
		}
		series = append(series, b.Series)
		done[b.Series.ID] = true
	}
	return series.Filtered(func(a Series) bool {
		return a.Name != ""
	})
}

// Sorted returns a copy of the SeriesList sorted by the function
func (l *SeriesList) Sorted(sorter func(a, b Series) bool) *SeriesList {
	// Make a copy
	sorted := make(SeriesList, len(*l))
	copy(sorted, *l)
	// Sort the copy
	sort.Slice(sorted, func(i, j int) bool {
		return sorter(sorted[i], sorted[j])
	})
	return &sorted
}

// Filtered returns a copy of the SeriesList filtered by the function
func (l *SeriesList) Filtered(filterer func(a Series) bool) *SeriesList {
	filtered := SeriesList{}
	for _, a := range *l {
		if filterer(a) {
			filtered = append(filtered, a)
		}
	}

	return &filtered
}

// HasBook checks whether a book with an id exists
func (l *BookList) HasBook(id string) bool {
	exists := false
	for _, b := range *l {
		if b.ID == id {
			exists = true
		}
	}
	return exists
}

// HasAuthor checks whether an author with an id exists
func (l *BookList) HasAuthor(id string) bool {
	exists := false
	for _, b := range *l {
		if b.Author.ID == id {
			exists = true
		}
	}
	return exists
}

// HasSeries checks whether a series with an id exists
func (l *BookList) HasSeries(id string) bool {
	exists := false
	for _, b := range *l {
		if b.Series.ID == id {
			exists = true
		}
	}
	return exists
}
