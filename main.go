package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"

	"github.com/globalsign/mgo"
	zglob "github.com/mattn/go-zglob"
)

type booksingApp struct {
	books         *mgo.Collection
	downloads     *mgo.Collection
	allowDeletes  bool
	allowOrganize bool
	bookDir       string
}

type bookResponse struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total"`
	timestamp  time.Time
}

type download struct {
	Book      string
	User      string
	IP        string
	Timestamp time.Time
}

type bookConvertRequest struct {
	Hash          string `json:"bookhash"`
	Receiver      string `json:"email"`
	SMTPServer    string `json:"smtpserver"`
	SMTPUser      string `json:"smtpuser"`
	SMTPPassword  string `json:"smtppass"`
	ConvertToMobi bool   `json:"convert"`
}

func main() {
	envDeletes := os.Getenv("ALLOW_DELETES")
	allowDeletes := envDeletes != "" && strings.ToLower(envDeletes) == "true"
	envOrganize := os.Getenv("REORGANIZE_BOOKS")
	allowOrganize := envOrganize != "" && strings.ToLower(envOrganize) == "true"
	bookDir := os.Getenv("BOOK_DIR")
	if bookDir == "" {
		bookDir = "."
	}
	mongoHost := os.Getenv("MONGO_HOST")
	if mongoHost == "" {
		mongoHost = "localhost"
	}
	fmt.Println("Connecting to", mongoHost)
	conn, err := mgo.Dial(mongoHost)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Connected")
	session := conn.DB("booksing")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	app := booksingApp{
		books:         session.C("books"),
		downloads:     session.C("downloads"),
		allowDeletes:  allowDeletes,
		allowOrganize: allowOrganize,
		bookDir:       bookDir,
	}
	app.createIndices()

	http.HandleFunc("/refresh", app.refreshBooks())
	http.HandleFunc("/search", app.getBooks())
	http.HandleFunc("/duplicates.json", app.getDuplicates())
	http.HandleFunc("/book.json", app.getBook())
	http.HandleFunc("/exists", app.bookPresent())
	http.HandleFunc("/convert/", app.convertBook())
	http.HandleFunc("/download/", app.downloadBook())
	http.Handle("/", http.FileServer(assetFS()))

	log.Println("Please visit http://localhost:7132 to view booksing")
	log.Fatal(http.ListenAndServe(":7132", nil))
}

func (app booksingApp) createIndices() error {

	indices := []mgo.Index{
		mgo.Index{
			Key:      []string{"hash"},
			Unique:   true,
			DropDups: true,
		},
		mgo.Index{
			Key:      []string{"filepath"},
			Unique:   true,
			DropDups: true,
		},
		mgo.Index{
			Key:      []string{"metaphone_keys"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"search_keys"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"date_added"},
			Unique:   false,
			DropDups: false,
		},
	}
	for _, index := range indices {
		err := app.books.EnsureIndex(index)
		if err != nil {
			fmt.Println(index.Key)
			fmt.Println(err)
		}
	}

	return nil

}

func (app booksingApp) downloadBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName := r.URL.Query().Get("book")
		toMobi := strings.HasSuffix(fileName, ".mobi")
		fmt.Println("trying to download ", fileName)
		var book Book
		if toMobi {
			fileName = strings.Replace(fileName, ".mobi", ".epub", 1)
		}
		err := app.books.Find(bson.M{"filename": fileName}).One(&book)
		if err != nil {
			fmt.Println(err)
			return
		}
		if toMobi {
			book.Filepath = strings.Replace(book.Filepath, ".epub", ".mobi", 1)
		}
		ip := r.RemoteAddr
		if r.Header.Get("x-forwarded-for") != "" {
			ip = ip + ", " + r.Header.Get("x-forwarded-for")
		}
		dl := download{
			User:      r.Header.Get("x-auth-user"),
			IP:        ip,
			Book:      book.Hash,
			Timestamp: time.Now(),
		}
		err = app.downloads.Insert(dl)

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(book.Filepath)))
		http.ServeFile(w, r, book.Filepath)
	}
}

func (app booksingApp) bookPresent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		author := r.URL.Query().Get("author")
		title := r.URL.Query().Get("title")
		hash := hashBook(author, title)

		var book Book
		err := app.books.Find(bson.M{"hash": hash}).One(&book)
		found := err == nil

		json.NewEncoder(w).Encode(map[string]bool{"found": found})
	}
}

func (app booksingApp) getBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		var book Book
		err := app.books.Find(bson.M{"hash": hash}).One(&book)
		if err != nil {
			return
		}
		json.NewEncoder(w).Encode(book)
	}
}

func (app booksingApp) convertBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			fmt.Println(err)
			return
		}
		hash := r.Form.Get("hash")
		fmt.Println(hash)
		var book Book
		err = app.books.Find(bson.M{"hash": hash}).One(&book)
		if err != nil {
			return
		}
		mobiPath := strings.Replace(book.Filepath, ".epub", ".mobi", 1)
		cmd := exec.Command("ebook-convert", book.Filepath, mobiPath)
		log.Printf("Running command and waiting for it to finish...")
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Command finished with error: %v", err)
			fmt.Println(stdoutStderr)
		} else {
			book.HasMobi = true
			app.books.Update(bson.M{"hash": hash}, book)
		}
		json.NewEncoder(w).Encode(book)
	}
}

type pipelineResult struct {
	Title  string   `bson:"_id"`
	Count  int      `bson:"count"`
	Hashes []string `bson:"docs"`
}

func (app booksingApp) getDuplicates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resp bookResponse
		var book Book
		numResults, err := app.books.Count()
		if err != nil {
			log.Println(err)
		}
		resp.TotalCount = numResults

		pipe := app.books.Pipe([]bson.M{
			bson.M{
				"$group": bson.M{
					"_id":   "$title",
					"count": bson.M{"$sum": 1},
					"docs":  bson.M{"$push": "$hash"},
				},
			},
			bson.M{
				"$match": bson.M{
					"count": bson.M{"$gt": 1.0},
				},
			},
			bson.M{
				"$limit": 500,
			},
		})
		iter := pipe.Iter()
		var dupes []pipelineResult

		err = iter.All(&dupes)
		if err != nil {
			fmt.Println(err)
		}

		for _, dup := range dupes {
			for _, hash := range dup.Hashes {
				err := app.books.Find(bson.M{"hash": hash}).One(&book)
				if err != nil {
					continue
				}
				resp.Books = append(resp.Books, book)
			}
		}

		if len(resp.Books) == 0 {
			resp.Books = []Book{}
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func (app booksingApp) getBooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resp bookResponse
		var limit int
		numString := r.URL.Query().Get("results")
		filter := strings.ToLower(r.URL.Query().Get("filter"))
		limit = 1000
		if a, err := strconv.Atoi(numString); err == nil {
			if a > 0 && a < 1000 {
				limit = a
			}
		}
		numResults, err := app.books.Count()
		if err != nil {
			log.Println(err)
		}
		resp.TotalCount = numResults

		resp.Books = app.filterBooks(filter, limit, true)
		if len(resp.Books) == 0 {
			resp.Books = app.filterBooks(filter, limit, false)
		}
		if len(resp.Books) == 0 {
			resp.Books = []Book{}
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func (app booksingApp) filterBooks(filter string, limit int, exact bool) []Book {
	var books []Book
	var iter *mgo.Iter
	if filter == "" {
		iter = app.books.Find(nil).Limit(limit).Iter()
	} else if exact {
		s := strings.Split(filter, " ")
		iter = app.books.Find(bson.M{"search_keys": bson.M{"$all": s}}).Limit(limit).Sort("author", "title").Iter()
	} else {
		s := getMetaphoneKeys(filter)
		iter = app.books.Find(bson.M{"metaphone_keys": bson.M{"$all": s}}).Limit(limit).Sort("author", "title").Iter()
	}
	err := iter.All(&books)
	if err != nil {
		log.Println(err.Error())
		return []Book{}
	}
	return books
}

func (app booksingApp) refreshBooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("starting refresh of booklist")
		matches, err := zglob.Glob(filepath.Join(app.bookDir, "/**/*.epub"))
		if err != nil {
			fmt.Println("Scan could not complete: ", err.Error())
			return
		}
		log.Println("found", len(matches), "epubs in ", app.bookDir)

		bookQ := make(chan string, len(matches))
		resultQ := make(chan int)

		for w := 0; w < 6; w++ { //not sure yet how concurrent-proof my solution is
			go app.bookParser(bookQ, resultQ)
		}

		for _, filename := range matches {
			bookQ <- filename
		}

		for a := 0; a < len(matches); a++ {
			<-resultQ
			if a > 0 && a%500 == 0 {
				log.Println("Scraped", a, "books so far")
			}
		}

		log.Println("finished refresh of booklist")
	}
}

func (app booksingApp) bookParser(bookQ chan string, resultQ chan int) {
	for filename := range bookQ {
		var dbBook Book
		//err := db.One("Filepath", filename, &dbBook)
		err := app.books.Find(bson.M{"filepath": filename}).One(&dbBook)
		if err == nil {
			resultQ <- 1
			continue
		}
		book, err := NewBookFromFile(filename, app.allowOrganize, app.bookDir)
		if err != nil {
			if app.allowDeletes {
				fmt.Println("Deleting ", filename)
				os.Remove(filename)
			}
			resultQ <- 1
			continue
		}
		book.ID = bson.NewObjectId()
		err = app.books.Insert(book)
		if err != nil && mgo.IsDup(err) {
			if app.allowDeletes {
				fmt.Println("Deleting ", filename)
				os.Remove(filename)
			}
		}
		resultQ <- 1
	}
}
