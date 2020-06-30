package booksing

import (
	"time"
)

// User
type User struct {
	ID         int    `storm:"id,increment"`
	Name       string `storm:"unique,index"`
	IsAdmin    bool
	IsAllowed  bool
	Created    time.Time
	LastSeen   time.Time
	SavedBooks map[string]*ShelveIcon //map[book_hash]shelveicon
}
