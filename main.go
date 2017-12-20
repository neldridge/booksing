package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/msgpack"
	"github.com/asdine/storm/q"
	"github.com/jordan-wright/email"
	zglob "github.com/mattn/go-zglob"
)

type bookResponse struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total"`
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
	envDeletes := os.Getenv("ALLOW_DELETES")
	allowDeletes := envDeletes != "" && strings.ToLower(envDeletes) == "true"
	bookDir := os.Getenv("BOOK_DIR")
	if bookDir == "" {
		bookDir = "."
	}

	dbLocation := os.Getenv("DATABASE_LOCATION")
	if dbLocation == "" {
		dbLocation = filepath.Join(bookDir, "booksing.db")
	}
	db, err := storm.Open(dbLocation, storm.Codec(msgpack.Codec), storm.Batch())
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer db.Close()

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
		var convertBook Book
		err = db.One("ID", convert.BookID, &convertBook)
		if err == nil {
			go convertAndSendBook(&convertBook, convert)
		} else {
			fmt.Println(err.Error())
		}
		fmt.Println(convert.BookID)
	})
	http.HandleFunc("/refresh", refreshBooks(db, bookDir, allowDeletes))
	http.HandleFunc("/books.json", getBooks(db))
	http.HandleFunc("/download/", getBook(db))

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
func getBook(db *storm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookId := r.URL.Query().Get("bookid")
		var book Book
		err := db.One("ID", bookId, &book)
		if err != nil {
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(book.Filepath)))
		http.ServeFile(w, r, book.Filepath)
	}
}

func getBooks(db *storm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// do stuff with db here
		var resp bookResponse
		var book Book
		var books []Book
		var limit int
		numString := r.URL.Query().Get("results")
		filter := strings.ToLower(r.URL.Query().Get("filter"))
		limit = 1000
		if a, err := strconv.Atoi(numString); err == nil {
			if a > 0 && a < 1000 {
				limit = a
			}
		}
		numResults, err := db.Count(&book)
		if err != nil {
			fmt.Println(err)
		}
		resp.TotalCount = numResults

		if filter == "" {
			err := db.All(&books, storm.Limit(limit))
			if err != nil {
				fmt.Println(err.Error())
			}
		} else {
			filter = "(?i)" + filter
			query := db.Select(q.Or(
				q.Re("Author", filter),
				q.Re("Title", filter),
			)).Limit(limit).OrderBy("Author")
			query.Find(&books)
		}
		resp.Books = books
		if len(resp.Books) == 0 {
			resp.Books = []Book{}
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func refreshBooks(db *storm.DB, bookDir string, allowDeletes bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("starting refresh of booklist")
		db.Set("status", "current", "indexing books")
		matches, err := zglob.Glob(filepath.Join(bookDir, "/**/*.epub"))
		if err != nil {
			fmt.Println("Scan could not complete: ", err.Error())
			return
		}

		bookQ := make(chan string, len(matches))
		resultQ := make(chan int)

		for w := 0; w < 5; w++ {
			go bookParser(db, bookQ, resultQ, allowDeletes)
		}

		for _, filename := range matches {
			bookQ <- filename
		}

		for a := 0; a < len(matches); a++ {
			<-resultQ
			if a > 0 && a%100 == 0 {
				fmt.Println("Scraped", a, "books so far")
			}
		}

		db.Set("status", "current", "idle")
		log.Println("started refresh of booklist")
	}
}

func bookParser(db *storm.DB, bookQ chan string, resultQ chan int, allowDeletes bool) {
	for filename := range bookQ {
		var dbBook Book
		err := db.One("Filepath", filename, &dbBook)
		if err == nil {
			resultQ <- 1
			continue
		}
		book, err := NewBookFromFile(filename)
		if err != nil {
			if allowDeletes {
				fmt.Println("Deleting ", filename)
				os.Remove(filename)
			}
			resultQ <- 1
			continue
		}
		err = db.One("ID", book.ID, &dbBook)
		if err == storm.ErrNotFound {
			err = db.Save(book)
			if err != nil {
				fmt.Println(err)
			}
		} else if allowDeletes {
			fmt.Println("Deleting ", filename)
			os.Remove(filename)
		}
		resultQ <- 1
	}
}
