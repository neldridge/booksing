package booksing

import (
	"errors"
)

var ErrInvalidShelveIcon = errors.New("Invalid shelve icon provided")

type ShelveIcon string

//next shelve icon
var shelveIcons = []ShelveIcon{
	"star-outline",
	"star",
	"book-open-outline",
	"checkmark-circle-outline",
	"close-outline",
}

func NextShelveIcon(cur ShelveIcon) (ShelveIcon, error) {
	for i, icon := range shelveIcons {
		if icon == cur {
			return shelveIcons[(i+1)%len(shelveIcons)], nil
		}
	}
	return DefaultShelveIcon(), ErrInvalidShelveIcon
}

func DefaultShelveIcon() ShelveIcon {
	return shelveIcons[0]
}
