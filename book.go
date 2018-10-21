package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"strings"

	"github.com/beevik/etree"
	"github.com/globalsign/mgo/bson"
	"github.com/kennygrant/sanitize"
	"golang.org/x/tools/godoc/vfs/zipfs"
)

var yearRemove = regexp.MustCompile(`\((1|2)[0-9]{3}\)`)
var drukRemove = regexp.MustCompile(`(?i)/ druk [0-9]+`)
var filenameSafe = regexp.MustCompile("[^a-zA-Z0-9 -]+")

// Book represents a book
type Book struct {
	ID            bson.ObjectId `json:"id"`
	Hash          string        `json:"hash"`
	Title         string        `json:"title"`
	Author        string        `json:"author"`
	Language      string        `json:"language"`
	Description   string        `json:"description"`
	Filepath      string        `json:"filepath"`
	Filename      string        `json:"filename"`
	HasMobi       bool          `json:"hasmobi"`
	MetaphoneKeys []string      `bson:"metaphone_keys"`
	SearchWords   []string      `bson:"search_keys"`
	Added         time.Time     `bson:"date_added" json:"date_added"`
}

// NewBookFromFile creates a book object from a file
func NewBookFromFile(bookpath string, rename bool, baseDir string) (bk *Book, err error) {
	defer func() {
		if r := recover(); r != nil {
			bk = nil
			err = fmt.Errorf("Unknown error parsing book. Skipping. Error: %s", r)
		}
	}()

	f, err := os.Open(bookpath)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	book := new(Book)
	book.Language = ""
	book.Title = filepath.Base(bookpath)
	book.Filename = filepath.Base(bookpath)
	book.Filepath = bookpath
	book.Added = fi.ModTime()

	mobiPath := strings.Replace(bookpath, "epub", "mobi", -1)
	_, err = os.Stat(mobiPath)
	book.HasMobi = !os.IsNotExist(err)

	zr, err := zip.OpenReader(bookpath)
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
	book.Title = filepath.Base(bookpath)
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
	for _, e := range opf.FindElements("//language") {
		book.Language = e.Text()
		break
	}

	book.Title = fix(book.Title, true, false)
	book.Author = fix(book.Author, true, true)
	book.Language = fixLang(book.Language)
	book.Description = sanitize.HTML(book.Description)

	searchWords := book.Title + " " + book.Author
	book.MetaphoneKeys = getMetaphoneKeys(searchWords)
	book.SearchWords = getLowercasedSlice(searchWords)

	book.Hash = hashBook(book.Author, book.Title)

	if rename {
		newBookPath := path.Join(baseDir, getOrganizedBookPath(book))
		if bookpath != newBookPath {
			baseDir := filepath.Dir(newBookPath)
			err := os.MkdirAll(baseDir, 0755)
			if err == nil {
				os.Rename(bookpath, newBookPath)
				book.Filepath = newBookPath
				book.Filename = filepath.Base(newBookPath)
			}
			fmt.Println(newBookPath)
		}
	}

	return book, nil
}

func getOrganizedBookPath(b *Book) string {
	title := b.Title
	author := b.Author
	author = filenameSafe.ReplaceAllString(author, "")
	title = filenameSafe.ReplaceAllString(title, "")
	if len(title) > 35 {
		title = title[:30]
	}
	title = strings.TrimSuffix(title, " ")
	firstChar := author[0:1]
	parts := strings.Split(author, " ")
	firstChar = parts[len(parts)-1][0:1]
	formatted := fmt.Sprintf("%s/%s/%s-%s.epub", firstChar, author, author, title)
	formatted = strings.Replace(formatted, " ", "_", -1)
	formatted = strings.Replace(formatted, "__", "_", -1)

	return formatted
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

func fixLang(s string) string {
	s = strings.ToLower(s)

	switch s {
	case "nld":
		s = "nl"
	case "dutch":
		s = "nl"
	case "nederlands":
		s = "nl"
	case "nederland":
		s = "nl"
	case "nl-nl":
		s = "nl"

	case "deutsch":
		s = "de"
	case "deutsche":
		s = "de"
	case "duits":
		s = "de"
	case "german":
		s = "de"
	case "de-de":
		s = "de"

	case "english":
		s = "en"
	case "engels":
		s = "en"
	case "uk":
		s = "en"
	case "en-en":
		s = "en"
	case "us":
		s = "en"
	}
	return s
}

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

	s = yearRemove.ReplaceAllString(s, "")
	s = drukRemove.ReplaceAllString(s, "")
	s = strings.Replace(s, ".", " ", -1)
	s = strings.Replace(s, "  ", " ", -1)
	s = strings.TrimSpace(s)

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
