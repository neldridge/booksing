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

func main() {
	BookCache := bookResponse{
		Books:     &BookList{},
		timestamp: time.Now().AddDate(0, 0, -1),
	}

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
