package main

import (
	"fmt"
)

func main() {
	fmt.Println("hoi")
	a, err := NewBookFromFile("testdata/books/test1.epub", "testdata/books/test1.jpg")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(a)
}
