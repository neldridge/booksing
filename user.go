package booksing

import (
	"time"
)

// User demo
type User struct {
	ID        int    `storm:"id,increment"`
	Username  string `storm:unique,index`
	IsAdmin   bool
	IsAllowed bool
	Created   time.Time
	LastSeen  time.Time
	APIKeys   []Apikey
}
