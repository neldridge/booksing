package storm

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/asdine/storm"
	"github.com/blevesearch/bleve"
	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
)

type stormDB struct {
	db *storm.DB
	in bleve.Index
}

type download = booksing.Download
type RefreshResult = booksing.RefreshResult

func New(path string) (*stormDB, error) {

	stormPath := filepath.Join(path, "booksing.db")
	blevePath := filepath.Join(path, "search.bleve")

	bleveIndex, err := bleve.Open(blevePath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		bleveIndex, err = bleve.New(blevePath, bleve.NewIndexMapping())
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		log.Fatal(err)
		return nil, err
	}

	db, err := storm.Open(stormPath)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"path": path,
		}).Error("could not open filedb")
	}

	database := stormDB{
		db: db,
		in: bleveIndex,
	}

	return &database, nil
}

func (db *stormDB) Close() {
	db.db.Close()
}

func (db *stormDB) AddDownload(dl download) error {
	return db.db.Save(&dl)
}

func (db *stormDB) GetDownloads(limit int) ([]download, error) {
	var dls []download
	err := db.db.All(&dls, storm.Limit(limit), storm.Reverse())
	return dls, err
}

func (db *stormDB) AddRefresh(rr RefreshResult) error {
	return db.db.Save(&rr)
}
func (db *stormDB) GetRefreshes(limit int) ([]RefreshResult, error) {
	//TODO: do something with limit
	var refreshes []RefreshResult
	err := db.db.All(&refreshes)
	return refreshes, err
}

func (db *stormDB) GetUsers() ([]booksing.User, error) {
	var users []booksing.User
	err := db.db.All(&users)
	return users, err
}

func (db *stormDB) GetUser(username string) (booksing.User, error) {
	var u booksing.User
	err := db.db.One("Name", username, &u)
	if err == storm.ErrNotFound {
		return u, booksing.ErrNotFound
	}
	return u, err
}

func (db *stormDB) SaveUser(u *booksing.User) error {
	return db.db.Save(u)
}

func (db *stormDB) GetBookCount() int {
	var stats dbBookCount
	err := db.db.One("ID", "total", &stats)
	if err != nil {
		return -1
	}
	return stats.Count
}

func (db *stormDB) UpdateBookCount(count int) error {
	var stats dbBookCount
	err := db.db.One("ID", "total", &stats)
	if err == storm.ErrNotFound {
		stats = dbBookCount{
			ID:    "total",
			Count: 0,
		}
	} else if err != nil {
		return fmt.Errorf("Unable to get total stats from db: %w", err)
	}
	stats.Count += count
	err = db.db.Save(&stats)
	if err != nil {
		return fmt.Errorf("Unable to get store total stats in db: %w", err)
	}
	today := time.Now().Format("2006-01-02")
	err = db.db.One("ID", today, &stats)
	if err == storm.ErrNotFound {
		stats = dbBookCount{
			ID:    today,
			Count: 0,
		}
	} else if err != nil {
		return fmt.Errorf("Unable to get %s stats from db: %w", today, err)
	}
	stats.Count += count
	err = db.db.Save(&stats)
	if err != nil {
		return fmt.Errorf("Unable to get store %s stats in db: %w", today, err)
	}

	return nil
}

func (db *stormDB) GetBookCountHistory(start, end time.Time) ([]booksing.BookCount, error) {
	//TODO implement
	return nil, nil
}

func (db *stormDB) AddHash(h string) error {
	return db.db.Set("hashes", h, true)
}

func (db *stormDB) HasHash(h string) (bool, error) {
	var b bool
	err := db.db.Get("hashes", h, &b)
	if err == storm.ErrNotFound {
		return false, nil
	}
	return b, err
}

type dbBookCount struct {
	ID    string `storm:"unique,index"`
	Count int
}

func (db *stormDB) AddBook(b booksing.Book) error {
	err := db.in.Index(b.Hash, b)
	if err != nil {
		return err
	}
	return db.db.Save(&b)
}

func (db *stormDB) GetBook(hash string) (*booksing.Book, error) {
	var b booksing.Book
	err := db.db.One("Hash", hash, &b)
	if err == storm.ErrNotFound {
		return &b, booksing.ErrNotFound
	}
	return &b, err
}

func (db *stormDB) AddBooks(books []booksing.Book, sync bool) error {
	var err error
	var errs []error

	for _, b := range books {
		err = db.AddBook(b)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (db *stormDB) DeleteBook(hash string) error {
	//todo remove from bleve
	return db.in.Delete(hash)
}

func (db *stormDB) GetBooks(q string, limit, offset int64) (*booksing.SearchResult, error) {

	var books []booksing.Book

	if q == "" {
		return db.recentBooks()
	}

	//query := bleve.NewQueryStringQuery(q)
	query := bleve.NewFuzzyQuery(q)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.From = int(offset)
	searchRequest.Size = int(limit)
	res, _ := db.in.Search(searchRequest)

	for _, hit := range res.Hits {
		b, err := db.GetBook(hit.ID)
		if err != nil {
			fmt.Println("Could not get book")
		}

		books = append(books, *b)
	}

	return &booksing.SearchResult{
		Items: books,
		Total: int64(res.Total),
	}, nil
}

func (db *stormDB) recentBooks() (*booksing.SearchResult, error) {

	var books []booksing.Book

	err := db.db.AllByIndex("Added", &books, storm.Limit(10))

	return &booksing.SearchResult{
		Items: books,
		Total: int64(len(books)),
	}, err
}
