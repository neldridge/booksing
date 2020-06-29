package storm

import (
	"fmt"
	"time"

	"github.com/asdine/storm"
	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
)

type stormDB struct {
	db *storm.DB
}

type download = booksing.Download
type RefreshResult = booksing.RefreshResult

func New(path string) (*stormDB, error) {
	db, err := storm.Open(path)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"path": path,
		}).Error("could not open filedb")
	}

	database := stormDB{
		db: db,
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
	//TODO: do something with limit
	var dls []download
	err := db.db.All(&dls)
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
