package main

import (
	"encoding/json"
	"fmt"
	"log/syslog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo/bson"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"

	"github.com/globalsign/mgo"
	zglob "github.com/mattn/go-zglob"
	log "github.com/sirupsen/logrus"
)

type booksingApp struct {
	books          *mgo.Collection
	downloads      *mgo.Collection
	refreshResults *mgo.Collection
	allowDeletes   bool
	allowOrganize  bool
	bookDir        string
}

type bookResponse struct {
	Books      []Book `json:"books"`
	TotalCount int    `json:"total"`
	timestamp  time.Time
}

type parseResult int32

// hold all possible book parse results
const (
	OldBook       parseResult = iota
	AddedBook     parseResult = iota
	DuplicateBook parseResult = iota
	InvalidBook   parseResult = iota
)

// RefreshResult holds the result of a full refresh
type RefreshResult struct {
	StartTime time.Time
	StopTime  time.Time
	Old       int
	Added     int
	Duplicate int
	Invalid   int
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
	syslogServer := os.Getenv("SYSLOG_REMOTE")
	if syslogServer != "" {
		hook, err := lSyslog.NewSyslogHook("udp", syslogServer, syslog.LOG_INFO, "")
		if err == nil {
			log.SetFormatter(&log.JSONFormatter{})
			log.AddHook(&AddSourceHook{})
			log.AddHook(hook)
		}
	}
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
	log.WithField("address", mongoHost).Info("Connecting to mongodb")
	conn, err := mgo.Dial(mongoHost)
	if err != nil {
		log.WithField("err", err).Error("Could not connect to mongodb")
		return
	}
	session := conn.DB("booksing")
	if err != nil {
		log.WithField("err", err).Error("Could not create booksing session")
		return
	}
	app := booksingApp{
		books:          session.C("books"),
		downloads:      session.C("downloads"),
		refreshResults: session.C("refreshResults"),
		allowDeletes:   allowDeletes,
		allowOrganize:  allowOrganize,
		bookDir:        bookDir,
	}
	app.createIndices()

	http.HandleFunc("/refresh", app.refreshBooks())
	http.HandleFunc("/search", app.getBooks())
	http.HandleFunc("/duplicates.json", app.getDuplicates())
	http.HandleFunc("/book.json", app.getBook())
	http.HandleFunc("/user.json", app.getUser())
	http.HandleFunc("/exists", app.bookPresent())
	http.HandleFunc("/convert/", app.convertBook())
	http.HandleFunc("/delete/", app.deleteBook())
	http.HandleFunc("/download/", app.downloadBook())
	http.Handle("/", http.FileServer(assetFS()))

	log.Info("booksing is now running")
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
			Key:      []string{"author"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"title"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"date_added"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"language"},
			Unique:   false,
			DropDups: false,
		},
		mgo.Index{
			Key:      []string{"booksing_version"},
			Unique:   false,
			DropDups: false,
		},
	}
	for _, index := range indices {
		err := app.books.EnsureIndex(index)
		if err != nil {
			log.WithFields(log.Fields{
				"index": index.Key,
				"err":   err,
			}).Error("Could not create index")
		}
	}

	return nil

}

func (app booksingApp) downloadBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName := r.URL.Query().Get("book")
		toMobi := strings.HasSuffix(fileName, ".mobi")
		var book Book
		if toMobi {
			fileName = strings.Replace(fileName, ".mobi", ".epub", 1)
		}
		err := app.books.Find(bson.M{"filename": fileName}).One(&book)
		if err != nil {
			log.WithFields(log.Fields{
				"err":      err,
				"filename": fileName,
			}).Error("could not find book")
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
		log.WithFields(log.Fields{
			"user": r.Header.Get("x-auth-user"),
			"ip":   ip,
			"book": book.Hash,
		}).Info("book was downloaded")

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", path.Base(book.Filepath)))
		http.ServeFile(w, r, book.Filepath)
	}
}

func (app booksingApp) bookPresent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		author := r.URL.Query().Get("author")
		title := r.URL.Query().Get("title")
		title = fix(title, true, false)
		author = fix(author, true, true)
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
func (app booksingApp) getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Header.Get("x-auth-user")
		admin := false
		if user == os.Getenv("ADMIN_USER") || os.Getenv("ANONYMOUS_ADMIN") != "" {
			admin = true
		}
		log.WithFields(log.Fields{
			"x-auth-user": user,
			"admin":       admin,
			"env-user":    os.Getenv("ADMIN_USER"),
			"anon-admin":  os.Getenv("ANONYMOUS_ADMIN"),
		}).Info("getting user admin")
		json.NewEncoder(w).Encode(map[string]bool{
			"admin": admin,
		})
	}
}

func (app booksingApp) convertBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.WithField("err", err).Error("could not parse form data")
			return
		}
		hash := r.Form.Get("hash")
		var book Book
		err = app.books.Find(bson.M{"hash": hash}).One(&book)
		if err != nil {
			return
		}
		log.WithField("book", book.Filepath).Debug("converting to mobi")
		mobiPath := strings.Replace(book.Filepath, ".epub", ".mobi", 1)
		cmd := exec.Command("ebook-convert", book.Filepath, mobiPath)

		_, err = cmd.CombinedOutput()
		if err != nil {
			log.WithField("err", err).Error("Command finished with error")
		} else {
			book.HasMobi = true
			app.books.Update(bson.M{"hash": hash}, book)
			log.WithField("book", book.Filepath).Debug("conversion successful")
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
			log.WithField("err", err).Error("could not get total book count")
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
			log.WithField("err", err).Error("Could not get duplicates")
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
		filter = strings.TrimSpace(filter)
		limit = 1000

		log.WithFields(log.Fields{
			"user":   r.Header.Get("x-auth-user"),
			"filter": filter,
		}).Info("user initiated search")

		if a, err := strconv.Atoi(numString); err == nil {
			if a > 0 && a < 1000 {
				limit = a
			}
		}
		numResults, err := app.books.Count()
		if err != nil {
			log.WithField("err", err).Error("could not get total book count")
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
		iter = app.books.Find(bson.M{"language": "nl"}).Sort("-date_added").Limit(limit).Iter()
	} else if strings.Contains(filter, ":") {
		q := parseQuery(filter)
		iter = app.books.Find(q).Limit(limit).Sort("author", "title").Iter()
	} else if exact {
		s := strings.Split(filter, " ")
		iter = app.books.Find(bson.M{"search_keys": bson.M{"$all": s}}).Limit(limit).Sort("author", "title").Iter()
	} else {
		s := getMetaphoneKeys(filter)
		iter = app.books.Find(bson.M{"metaphone_keys": bson.M{"$all": s}}).Limit(limit).Sort("author", "title").Iter()
	}
	err := iter.All(&books)
	if err != nil {
		log.WithField("err", err).Error("filtering books failed")
		return []Book{}
	}
	return books
}

func (app booksingApp) refreshBooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("starting refresh of booklist")
		results := RefreshResult{
			StartTime: time.Now(),
		}
		clean := r.URL.Query().Get("clean")
		if clean == "force" {
			info, err := app.books.RemoveAll(nil)
			if err != nil {
				log.WithField("err", err).Warning("clearing collection failed")
			}
			log.WithField("removed", info.Removed).Warning("documents were removed")
		}
		app.createIndices()
		matches, err := zglob.Glob(filepath.Join(app.bookDir, "/**/*.epub"))
		if err != nil {
			log.WithField("err", err).Error("glob of all books failed")
			return
		}
		log.WithFields(log.Fields{
			"total":   len(matches),
			"bookdir": app.bookDir,
		}).Info("located books on filesystem")

		bookQ := make(chan string, len(matches))
		resultQ := make(chan parseResult)

		for w := 0; w < 6; w++ { //not sure yet how concurrent-proof my solution is
			go app.bookParser(bookQ, resultQ)
		}

		for _, filename := range matches {
			bookQ <- filename
		}

		for a := 0; a < len(matches); a++ {
			r := <-resultQ

			switch r {
			case OldBook:
				results.Old++
			case InvalidBook:
				results.Invalid++
			case AddedBook:
				results.Added++
			case DuplicateBook:
				results.Duplicate++
			}
			if a > 0 && a%500 == 0 {
				log.WithFields(log.Fields{
					"processed": a,
					"total":     len(matches),
				}).Info("processing books")
			}

		}
		results.StopTime = time.Now()
		err = app.refreshResults.Insert(results)
		if err != nil {
			log.WithFields(log.Fields{
				"err":     err,
				"results": results,
			}).Error("Could not save refresh results")
		}

		log.WithField("result", results).Info("finished refresh of booklist")
	}
}

func (app booksingApp) bookParser(bookQ chan string, resultQ chan parseResult) {
	for filename := range bookQ {
		var dbBook Book
		//err := db.One("Filepath", filename, &dbBook)
		err := app.books.Find(bson.M{"filepath": filename}).One(&dbBook)
		if err == nil {
			resultQ <- OldBook
			continue
		}
		book, err := NewBookFromFile(filename, app.allowOrganize, app.bookDir)
		if err != nil {
			if app.allowDeletes {
				log.WithFields(log.Fields{
					"file":   filename,
					"reason": "invalid",
				}).Info("Deleting book")
				os.Remove(filename)
			}
			resultQ <- InvalidBook
			continue
		}
		book.ID = bson.NewObjectId()
		err = app.books.Insert(book)
		if err != nil && mgo.IsDup(err) {
			if app.allowDeletes {
				log.WithFields(log.Fields{
					"file":   filename,
					"reason": "duplicate",
				}).Info("Deleting book")
				os.Remove(filename)
			}
			resultQ <- DuplicateBook
		} else {
			resultQ <- AddedBook
		}
	}
}

func (app booksingApp) deleteBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			fmt.Println(err)
			return
		}
		hash := r.Form.Get("hash")
		var book Book
		err = app.books.Find(bson.M{"hash": hash}).One(&book)
		if err != nil {
			return
		}
		if book.HasMobi {
			mobiPath := strings.Replace(book.Filepath, ".epub", ".mobi", 1)
			os.Remove(mobiPath)
		}
		os.Remove(book.Filepath)

		app.books.Remove(bson.M{"hash": hash})
	}
}
