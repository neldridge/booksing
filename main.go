package main

import (
	"net/http"
	"path"

	"github.com/globalsign/mgo"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type configuration struct {
	AllowDeletes  bool
	AllowOrganize bool
	BookDir       string `default:"."`
	ImportDir     string `default:""`
	MongoHost     string `default:"localhost"`
	LogLevel      string `default:"info"`
}

func main() {
	var cfg configuration
	err := envconfig.Process("booksing", &cfg)
	if err != nil {
		log.WithField("err", err).Fatal("Could not parse full config from environment")
	}

	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err == nil {
		log.SetLevel(logLevel)
	}
	if cfg.ImportDir == "" {
		cfg.ImportDir = path.Join(cfg.BookDir, "import")
	}
	log.WithField("address", cfg.MongoHost).Info("Connecting to mongodb")
	conn, err := mgo.Dial(cfg.MongoHost)
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
		allowDeletes:   cfg.AllowDeletes,
		allowOrganize:  cfg.AllowOrganize,
		bookDir:        cfg.BookDir,
		importDir:      cfg.ImportDir,
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
