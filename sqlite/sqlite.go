package sqlite

import (
	"fmt"
	"time"

	"github.com/gnur/booksing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type liteDB struct {
	db *gorm.DB
}

type dbHash struct {
	gorm.Model
	Hash string
}

type download = booksing.Download
type RefreshResult = booksing.RefreshResult

func New(path string) (*liteDB, error) {

	path = "file::memory:?cache=shared"
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&booksing.Book{},
		&download{},
		&RefreshResult{},
		&booksing.User{},
		&dbHash{},
		&dbBookCount{},
	)
	if err != nil {
		return nil, err
	}

	tx := db.Exec("CREATE VIRTUAL TABLE search USING fts5(content=books, author, title, description, hash);")
	if tx.Error != nil {
		return nil, tx.Error
	}

	tx = db.Exec(`
CREATE TRIGGER books_bu BEFORE UPDATE ON books BEGIN
  DELETE FROM search WHERE rowid=old.rowid;
END;
CREATE TRIGGER books_bd BEFORE DELETE ON books BEGIN
  DELETE FROM search WHERE rowid=old.rowid;
END;

CREATE TRIGGER books_au AFTER UPDATE ON books BEGIN
  INSERT INTO search(rowid, author, title, description) VALUES(new.rowid, new.author, new.title, new.description);
END;
CREATE TRIGGER books_ai AFTER INSERT ON books BEGIN
  INSERT INTO search(rowid, author, title, description) VALUES(new.rowid, new.author, new.title, new.description);
END;`)
	if tx.Error != nil {
		return nil, tx.Error
	}

	return &liteDB{
		db: db,
	}, nil
}

func (db *liteDB) Close() {
	//noop, gorm removed it
}

func (db *liteDB) AddDownload(dl download) error {
	tx := db.db.Create(&dl)
	return tx.Error
}

func (db *liteDB) GetDownloads(limit int) ([]download, error) {
	var dls []download
	tx := db.db.Order("Timestamp desc").Limit(limit).Find(&dls)
	return dls, tx.Error
}

func (db *liteDB) AddRefresh(rr RefreshResult) error {
	tx := db.db.Create(&rr)
	return tx.Error
}
func (db *liteDB) GetRefreshes(limit int) ([]RefreshResult, error) {
	//TODO: do something with limit
	var refreshes []RefreshResult
	tx := db.db.Find(&refreshes)
	return refreshes, tx.Error
}

func (db *liteDB) GetUsers() ([]booksing.User, error) {
	var users []booksing.User
	tx := db.db.Find(&users)
	return users, tx.Error
}

func (db *liteDB) GetUser(username string) (booksing.User, error) {
	var u booksing.User
	tx := db.db.Where("name = ?", username).First(&u)
	if tx.Error == gorm.ErrRecordNotFound {
		return u, booksing.ErrNotFound
	}
	return u, tx.Error
}

func (db *liteDB) SaveUser(u *booksing.User) error {
	if _, err := db.GetUser(u.Name); err != booksing.ErrNotFound {
		tx := db.db.Save(&u)
		return tx.Error
	}
	tx := db.db.Create(&u)
	return tx.Error
}

func (db *liteDB) GetBookCount() int {
	var stats dbBookCount
	tx := db.db.Where("id = ?", "total").First(&stats)
	if tx.Error != nil {
		return -1
	}
	return stats.Count
}

func (db *liteDB) UpdateBookCount(count int) error {
	var stats dbBookCount
	tx := db.db.Where("id = ?", "total").First(&stats)
	if tx.Error == gorm.ErrRecordNotFound {
		stats = dbBookCount{
			ID:    "total",
			Count: count,
		}
		tx = db.db.Create(&stats)
		if tx.Error != nil {
			return fmt.Errorf("Unable to get store total stats in db: %w", tx.Error)
		}
	} else if tx.Error != nil {
		return fmt.Errorf("Unable to get total stats from db: %w", tx.Error)
	}
	stats.Count += count
	tx = db.db.Save(&stats)
	if tx.Error != nil {
		return fmt.Errorf("Unable to get store total stats in db: %w", tx.Error)
	}

	today := time.Now().Format("2006-01-02")
	tx = db.db.Where("id = ?", today).First(&stats)
	if tx.Error == gorm.ErrRecordNotFound {
		stats = dbBookCount{
			ID:    today,
			Count: count,
		}
		tx = db.db.Create(&stats)
		if tx.Error != nil {
			return fmt.Errorf("Unable to get store %s stats in db: %w", today, tx.Error)
		}
	} else if tx.Error != nil {
		return fmt.Errorf("Unable to get %s stats from db: %w", today, tx.Error)
	}
	stats.Count += count
	tx = db.db.Save(&stats)
	if tx.Error != nil {
		return fmt.Errorf("Unable to get store %s stats in db: %w", today, tx.Error)
	}

	return nil
}

func (db *liteDB) GetBookCountHistory(start, end time.Time) ([]booksing.BookCount, error) {
	var stats []dbBookCount

	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	tx := db.db.Order("id desc").Where("id BETWEEN ? AND ?", startStr, endStr).Find(&stats)
	if tx.Error != nil {
		return nil, fmt.Errorf("Unable to get stats from db: %w", tx.Error)
	}

	var retStats []booksing.BookCount
	for _, stat := range stats {
		retStats = append(retStats, booksing.BookCount{
			Date:  stat.ID,
			Count: stat.Count,
		})

	}

	return retStats, nil
}

func (db *liteDB) AddHash(h string) error {
	inHash := dbHash{
		Hash: h,
	}
	tx := db.db.Create(&inHash)
	return tx.Error
}

func (db *liteDB) HasHash(h string) (bool, error) {
	var dbH dbHash
	var b bool
	tx := db.db.First(&dbH, "Hash = ?", h)
	if tx.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	return b, tx.Error
}

type dbBookCount struct {
	gorm.Model
	ID    string `storm:"unique,index"`
	Count int
}

func (db *liteDB) AddBook(b booksing.Book) error {
	tx := db.db.Create(&b)
	return tx.Error
}

func (db *liteDB) GetBook(hash string) (*booksing.Book, error) {
	var b booksing.Book
	tx := db.db.Where("hash = ?", hash).First(&b)
	if tx.Error == gorm.ErrRecordNotFound {
		return &b, booksing.ErrNotFound
	}
	return &b, tx.Error
}

func (db *liteDB) AddBooks(books []booksing.Book, sync bool) error {
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

func (db *liteDB) DeleteBook(hash string) error {
	var h dbHash
	tx := db.db.Where("hash = ?", hash).Delete(&h)
	return tx.Error
}

func (db *liteDB) GetBooks(q string, limit, offset int64) (*booksing.SearchResult, error) {

	var books []booksing.Book

	if q == "" {
		return db.recentBooks()
	}

	tx := db.db.Raw("SELECT hash FROM search WHERE search MATCH ?", q).Scan(&books)
	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(books) > 0 {
		hashes := []string{}
		for _, b := range books {
			hashes = append(hashes, b.Hash)
		}
		tx := db.db.Where("hash IN ?", hashes).Find(&books)
		if tx.Error != nil {
			return nil, tx.Error
		}

	}

	return &booksing.SearchResult{
		Items: books,
		Total: int64(0),
	}, nil
}

func (db *liteDB) recentBooks() (*booksing.SearchResult, error) {

	var books []booksing.Book

	tx := db.db.Order("Added desc").Limit(20).Find(&books)

	return &booksing.SearchResult{
		Items: books,
		Total: int64(len(books)),
	}, tx.Error
}
