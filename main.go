package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	http.HandleFunc("/books.json", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		ts := BookCache.timestamp
		if now.After(ts.Add(time.Minute)) {
			fmt.Println("Refreshing cache...")
			a, err := NewBookListFromDir("/home/erwin/Downloads/drive-download-20171002T115616Z-001", "tmp", false)
			if err != nil {
				fmt.Println(err)
			} else {
				BookCache.timestamp = time.Now()
				BookCache.Books = a
			}
		}

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
	fmt.Println(c)
	fmt.Println("-----------------------------------")
}
