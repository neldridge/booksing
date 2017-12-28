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
	"time"

	"strings"

	"github.com/beevik/etree"
	"github.com/kennygrant/sanitize"
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

// Book represents a book
type Book struct {
	ID          string `json:"id" storm:"id"`
	Title       string `json:"title" storm:"index"`
	Author      string `json:"author" storm:"index"`
	Description string `json:"description,omitempty"`
	Filepath    string `json:"filepath" storm:"index"`
	Filename    string `json:"filename" storm:"index"`
	HasMobi     bool   `json:"hasmobi"`
	LastSeen    int64  `storm:"index"`
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
	book.LastSeen = time.Now().UnixNano()
	book.Title = filepath.Base(path)
	book.Filename = filepath.Base(path)
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
		book.Author = e.Text()
		break
	}
	for _, e := range opf.FindElements("//description") {
		book.Description = e.Text()
		break
	}

	book.Title = fix(book.Title, true, false)
	book.Author = fix(book.Author, true, true)
	book.Description = fix(book.Description, false, false)
	book.Description = sanitize.HTML(book.Description)

	id := sha1.New()
	io.WriteString(id, book.Author)
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
