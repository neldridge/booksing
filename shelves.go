package booksing

import (
	"errors"
)

var ErrInvalidShelveIcon = errors.New("Invalid shelve icon provided")

type ShelveIcon [2]string

//next shelve icon
var shelveIcons = []ShelveIcon{
	{"star", "text-secondary"},
	{"star-fill", "text-warning"},
	{"book-half", "text-dark"},
	{"check-circle", "text-success"},
	{"x-circle", "text-danger"},
}

func NextShelveIcon(cur string) (*ShelveIcon, error) {
	for i, icon := range shelveIcons {
		if icon[0] == cur {
			return &shelveIcons[(i+1)%len(shelveIcons)], nil
		}
	}
	return nil, ErrInvalidShelveIcon
}

func DefaultShelveIcon() *ShelveIcon {
	return &shelveIcons[0]
}
