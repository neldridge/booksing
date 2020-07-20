package main

import (
	"math/rand"
	"strings"
	"time"
)

var charSet = []rune("1234567890_-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randID() string {
	rand.Seed(time.Now().UnixNano())
	var output strings.Builder
	length := 16
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		output.WriteRune(randomChar)
	}
	return output.String()
}
