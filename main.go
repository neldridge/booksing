package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type bookResponse struct {
	Books     *BookList `json:"books"`
	timestamp time.Time
}

type bookConvertRequest struct {
	BookID   string `json:"bookid"`
	Receiver string `json:"email"`
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
			go convertAndSendBook(convertBook, convert.Receiver)
		}
		fmt.Println(convert.BookID)
	})
	http.HandleFunc("/refresh", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		ts := BookCache.timestamp
		if now.After(ts.Add(30 * time.Second)) {
			log.Printf("Refreshing cache...")
			a, err := NewBookListFromDir(bookDir, "tmp", false)
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
		json.NewEncoder(w).Encode(resp)
	})
	http.Handle("/", http.FileServer(assetFS()))
	log.Fatal(http.ListenAndServe(":7132", nil))
}

func convertAndSendBook(c *Book, receiver string) {
	fmt.Println("-----------------------------------")
	if !c.HasMobi {
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
	if c.HasMobi {
		fmt.Println("send the book")
	}

	fmt.Println(c)
	fmt.Println("-----------------------------------")
}
