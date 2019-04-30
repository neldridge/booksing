package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	zglob "github.com/mattn/go-zglob"
	log "github.com/sirupsen/logrus"
)

const (
	stateUnlocked uint32 = iota
	stateLocked
)

var (
	locker    = stateUnlocked
	errLocked = errors.New("already running")
)

type booksingApp struct {
	books          *mgo.Collection
	downloads      *mgo.Collection
	refreshResults *mgo.Collection
	allowDeletes   bool
	allowOrganize  bool
	bookDir        string
	importDir      string
}

type db interface {
	AddBook(*Book) error
	GetBook(string) (*Book, error)
	GetBooks(string) ([]*Book, error)

	AddDownload(*download) error
	GetDownloads(string) ([]*download, error)

	AddRefresh(*RefreshResult) error
	GetRefreshes([]*RefreshResult) error
}

func (app *booksingApp) refreshLoop() {
	for {
		app.refresh()
		time.Sleep(time.Hour)
	}
}

func (app *booksingApp) createIndices() error {
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

func (app *booksingApp) downloadBook() http.HandlerFunc {
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

func (app *booksingApp) bookPresent() http.HandlerFunc {
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

func (app *booksingApp) getBook() http.HandlerFunc {
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
func (app *booksingApp) getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := app.userIsAdmin(r)
		json.NewEncoder(w).Encode(map[string]bool{
			"admin": admin,
		})
	}
}

func (app *booksingApp) userIsAdmin(r *http.Request) bool {
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
	return admin

}

func (app *booksingApp) getDownloads() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := app.userIsAdmin(r)
		if !admin {
			json.NewEncoder(w).Encode([]bool{})
			return
		}
		var downloads []download

		app.downloads.Find(nil).Sort("-timestamp").Iter().All(&downloads)

		json.NewEncoder(w).Encode(downloads)
	}
}
func (app *booksingApp) getRefreshes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := app.userIsAdmin(r)
		if !admin {
			json.NewEncoder(w).Encode([]bool{})
			return
		}
		var refreshes []RefreshResult

		app.refreshResults.Find(nil).Sort("-starttime").Iter().All(&refreshes)

		json.NewEncoder(w).Encode(refreshes)
	}
}

func (app *booksingApp) convertBook() http.HandlerFunc {
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

func (app *booksingApp) getDuplicates() http.HandlerFunc {
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

func (app *booksingApp) getBooks() http.HandlerFunc {
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

func (app *booksingApp) filterBooks(filter string, limit int, exact bool) []Book {
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

func (app *booksingApp) refresh() {
	if !atomic.CompareAndSwapUint32(&locker, stateUnlocked, stateLocked) {
		log.Warning("not refreshing because it is already running")
		return
	}
	defer atomic.StoreUint32(&locker, stateUnlocked)
	log.Info("starting refresh of booklist")
	results := RefreshResult{
		StartTime: time.Now(),
	}
	app.createIndices()
	matches, err := zglob.Glob(filepath.Join(app.importDir, "/**/*.epub"))
	if err != nil {
		log.WithField("err", err).Error("glob of all books failed")
		return
	}
	if len(matches) == 0 {
		log.Info("finished refresh of booklist, no new books found")
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
		if a > 0 && a%100 == 0 {
			log.WithFields(log.Fields{
				"processed": a,
				"total":     len(matches),
			}).Info("processing books")
		}

	}
	total, err := app.books.Count()
	if err != nil {
		log.WithField("err", err).Error("could not get total book count")
	}
	results.Old = total
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
func (app *booksingApp) refreshBooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app.refresh()
	}
}

func (app *booksingApp) bookParser(bookQ chan string, resultQ chan parseResult) {
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

func (app *booksingApp) deleteBook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin := app.userIsAdmin(r)
		if !admin {
			return
		}
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
		if err != nil {
			log.WithFields(log.Fields{
				"hash": hash,
				"err":  err,
			}).Error("Could not delete book from filesystem")
			return
		}

		app.books.Remove(bson.M{"hash": hash})
		if err != nil {
			log.WithFields(log.Fields{
				"hash": hash,
				"err":  err,
			}).Error("Could not delete book from database")
			return
		}
		log.WithFields(log.Fields{
			"hash": hash,
		}).Error("book was deleted")
	}
}
