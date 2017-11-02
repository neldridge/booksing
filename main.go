package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jordan-wright/email"
	zglob "github.com/mattn/go-zglob"
)

type bookResponse struct {
	Books      *BookList `json:"books"`
	TotalCount int       `json:"total"`
	timestamp  time.Time
}

type bookConvertRequest struct {
	BookID        string `json:"bookid"`
	Receiver      string `json:"email"`
	SMTPServer    string `json:"smtpserver"`
	SMTPUser      string `json:"smtpuser"`
	SMTPPassword  string `json:"smtppass"`
	ConvertToMobi bool   `json:"convert"`
}

// BookCache is the evil global var that holds the books...
var BookCache bookResponse

func main() {
	BookCache := bookResponse{
		Books:     &BookList{},
		timestamp: time.Now().AddDate(0, 0, -1),
	}
	bookDir := os.Getenv("BOOK_DIR")
	if bookDir == "" {
		bookDir = "."
	}
	http.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		var convert bookConvertRequest
		if r.Body == nil {
			http.Error(w, "please provide body", 400)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&convert)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var convertBook *Book
		found := false
		fmt.Println(convert.BookID)
		for _, book := range *BookCache.Books {
			if book.ID == convert.BookID {
				convertBook = &book
				found = true
				break
			}
		}
		if found {
			go convertAndSendBook(convertBook, convert)
		}
		fmt.Println(convert.BookID)
	})
	http.HandleFunc("/refresh", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		ts := BookCache.timestamp
		if now.After(ts.Add(30 * time.Second)) {
			log.Printf("Refreshing cache...")
			a, err := NewBookListFromDir(bookDir, false)
			if err != nil {
				fmt.Println(err)
			} else {
				BookCache.timestamp = time.Now()
				BookCache.Books = a
			}
			log.Printf("Cache refreshed!")
		}
	})

	http.HandleFunc("/books.json", func(w http.ResponseWriter, r *http.Request) {

		resp := bookResponse{Books: BookCache.Books}
		q := strings.ToLower(r.URL.Query().Get("filter"))
		if q != "" {
			filteredList := BookCache.Books.Filtered(func(b Book) bool {
				if strings.Contains(strings.ToLower(b.Author.Name), q) {
					return true
				}
				if strings.Contains(strings.ToLower(b.Title), q) {
					return true
				}
				return false
			})
			resp.Books = filteredList

		}
		resp.TotalCount = len(*resp.Books)
		numString := r.URL.Query().Get("results")
		if a, err := strconv.Atoi(numString); err == nil {
			if a < len(*resp.Books) {
				resp.TotalCount = len(*resp.Books)
				shortedResults := *resp.Books
				shortedResults = shortedResults[:a]
				resp.Books = &shortedResults
			}
		}
		json.NewEncoder(w).Encode(resp)
	})
	http.Handle("/", http.FileServer(assetFS()))
	log.Fatal(http.ListenAndServe(":7132", nil))
}

func convertAndSendBook(c *Book, req bookConvertRequest) {
	var attachment string
	fmt.Println("-----------------------------------")
	if !c.HasMobi && req.ConvertToMobi {
		fmt.Println("first convert the book")
		cmd := exec.Command("kindlegen", c.Filepath)
		log.Printf("Running command and waiting for it to finish...")
		err := cmd.Run()
		if err != nil {
			log.Printf("Command finished with error: %v", err)
			mobiPath := strings.Replace(c.Filepath, ".epub", ".mobi", 1)
			cmd := exec.Command("ebook-convert", c.Filepath, mobiPath)
			log.Printf("Running command and waiting for it to finish...")
			err := cmd.Run()
			if err != nil {
				log.Printf("Command finished with error: %v", err)
			} else {
				c.HasMobi = true
			}
		} else {
			c.HasMobi = true
		}
	}
	if c.HasMobi && req.ConvertToMobi {
		attachment = strings.Replace(c.Filepath, ".epub", ".mobi", 1)
	} else if !req.ConvertToMobi {
		attachment = c.Filepath
	} else {
		fmt.Println("mobi not present but was requested")
		return
	}
	e := email.NewEmail()
	e.From = req.SMTPUser
	e.To = []string{req.Receiver}
	e.Subject = "A booksing book"
	e.Text = []byte("")
	e.AttachFile(attachment)
	err := e.Send(req.SMTPServer+":587", smtp.PlainAuth("", req.SMTPUser, req.SMTPPassword, req.SMTPServer))
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(c)
	fmt.Println("-----------------------------------")
}

// NewBookListFromDir creates a BookList from the books in a dir. It will still return a nil error if there are errors indexing some of the books. It will only return an error if there is a problem getting the file list.
func NewBookListFromDir(path string, verbose bool) (*BookList, error) {
	matches, err := zglob.Glob(filepath.Join(path, "/**/*.epub"))
	ids := make(map[string]bool)
	if err != nil {
		return nil, err
	}

	var books BookList
	for i, filename := range matches {
		if verbose {
			log.Printf("%.f%% Indexing %s\n", float64(i+1)/float64(len(matches))*100, filename)
		}
		book, err := NewBookFromFile(filename)
		if err != nil {
			if verbose {
				log.Printf("Error indexing %s: %s\n", filename, err)
			}
			continue
		}
		if _, ok := ids[book.ID]; !ok {
			books = append(books, *book)
			ids[book.ID] = true
		} else {
			fmt.Println(filename, "is a duplicate")
		}
	}
	b := books.Sorted(func(a, b Book) bool {
		aName := a.Author.Name
		bName := b.Author.Name
		aParts := strings.Fields(a.Author.Name)
		bParts := strings.Fields(b.Author.Name)
		if len(aParts) > 0 {
			aName = aParts[len(aParts)-1]
		}
		if len(bParts) > 0 {
			bName = bParts[len(bParts)-1]
		}
		if aName == bName {
			return a.Title < b.Title
		}
		return aName < bName
	})
	return &b, nil
}
