package booksing

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID        int
	Name      string `gorm:"uniqueIndex"`
	IsAdmin   bool
	IsAllowed bool
	Downloads int64
	Created   time.Time
	LastSeen  time.Time
}
