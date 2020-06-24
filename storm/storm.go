package storm

import (
	"fmt"

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
	err := db.db.One("Username", username, &u)
	if err == storm.ErrNotFound {
		return u, booksing.ErrNotFound
	}
	return u, err
}

func (db *stormDB) SaveUser(u *booksing.User) error {
	fmt.Println(u)
	return db.db.Save(u)
}
