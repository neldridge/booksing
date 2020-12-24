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
	Created   time.Time
	LastSeen  time.Time
}
