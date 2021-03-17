package sqlite

import (
	"fmt"

	"github.com/gnur/booksing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type liteDB struct {
	db *gorm.DB
}

type download = booksing.Download

func New(path string) (*liteDB, error) {

	path = fmt.Sprintf("file:%s/booksing.db?cache=shared", path)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&booksing.Book{},
		&download{},
		&booksing.User{},
	)
	if err != nil {
		return nil, err
	}

	tx := db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(content=books, author, title, description, hash);")
	if tx.Error != nil {
		return nil, tx.Error
	}

	db.Exec(`
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
	var count int64
	tx := db.db.Model(&booksing.Book{}).Count(&count)
	if tx.Error != nil {
		return -1
	}
	return int(count)
}

func (db *liteDB) HasHash(h string) (bool, error) {
	var b bool
	var count int64
	tx := db.db.Model(&booksing.Book{}).Where("hash = ?", h).Count(&count)
	if tx.Error == gorm.ErrRecordNotFound || count == 0 {
		return false, nil
	}
	return b, tx.Error
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
	var b booksing.Book
	tx := db.db.Where("hash = ?", hash).Delete(&b)
	return tx.Error
}

func (db *liteDB) GetBooks(q string, limit, offset int64) (*booksing.SearchResult, error) {

	var books []booksing.Book
	var total int64

	if q == "" {
		return db.recentBooks()
	}

	tx := db.db.Raw("SELECT hash FROM search WHERE search MATCH ? LIMIT ?,?", q, offset, limit).Scan(&books)
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
		var count struct {
			Count int64
		}
		tx = db.db.Raw("SELECT count(*) as count FROM search WHERE search MATCH ?", q).Scan(&count)
		if tx.Error != nil {
			return nil, tx.Error
		} else {
			total = count.Count
		}

	}

	return &booksing.SearchResult{
		Items: books,
		Total: total,
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
