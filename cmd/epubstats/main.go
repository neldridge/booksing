package main

import (
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/gnur/booksing/epub"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("service", "epubstats")

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Invalid number of arguments supplied, please supply a single file to parse")
	}
	e, cover, err := epub.ParseFile(os.Args[1])
	spew.Dump(e)
	spew.Dump(err)
	fmt.Println(len(cover))

}
