package main

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/globalsign/mgo"
	log "github.com/sirupsen/logrus"
)

func main() {
	envDeletes := os.Getenv("ALLOW_DELETES")
	allowDeletes := envDeletes != "" && strings.ToLower(envDeletes) == "true"
	envOrganize := os.Getenv("REORGANIZE_BOOKS")
	allowOrganize := envOrganize != "" && strings.ToLower(envOrganize) == "true"
	bookDir := os.Getenv("BOOK_DIR")
	if bookDir == "" {
		bookDir = "."
	}
	importDir := path.Join(bookDir, "import")
	mongoHost := os.Getenv("MONGO_HOST")
	if mongoHost == "" {
		mongoHost = "localhost"
	}
	log.WithField("address", mongoHost).Info("Connecting to mongodb")
	conn, err := mgo.Dial(mongoHost)
	if err != nil {
		log.WithField("err", err).Error("Could not connect to mongodb")
		return
	}
	session := conn.DB("booksing")
	if err != nil {
		log.WithField("err", err).Error("Could not create booksing session")
		return
	}
	app := booksingApp{
		books:          session.C("books"),
		downloads:      session.C("downloads"),
		refreshResults: session.C("refreshResults"),
		allowDeletes:   allowDeletes,
		allowOrganize:  allowOrganize,
		bookDir:        bookDir,
		importDir:      importDir,
	}
	go app.refreshLoop()

	http.HandleFunc("/refresh", app.refreshBooks())
	http.HandleFunc("/search", app.getBooks())
	http.HandleFunc("/duplicates.json", app.getDuplicates())
	http.HandleFunc("/book.json", app.getBook())
	http.HandleFunc("/user.json", app.getUser())
	http.HandleFunc("/downloads.json", app.getDownloads())
	http.HandleFunc("/refreshes.json", app.getRefreshes())
	http.HandleFunc("/exists", app.bookPresent())
	http.HandleFunc("/convert/", app.convertBook())
	http.HandleFunc("/delete/", app.deleteBook())
	http.HandleFunc("/download/", app.downloadBook())
	http.Handle("/", http.FileServer(assetFS()))

	log.Info("booksing is now running")
	log.Fatal(http.ListenAndServe(":7132", nil))
}
