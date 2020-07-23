package booksing

import (
	"time"
)

// User
type User struct {
	ID        int    `storm:"id,increment"`
	Name      string `storm:"unique,index"`
	IsAdmin   bool
	IsAllowed bool
	Created   time.Time
	LastSeen  time.Time
	Bookmarks map[string]Bookmark //map[book_hash]shelveicon
}

type Bookmark struct {
	Icon       ShelveIcon
	LastChange time.Time
}
