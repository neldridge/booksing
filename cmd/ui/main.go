package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/gnur/booksing/sqlite"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

// V is the holder struct for all possible template values
type V struct {
	HasMobi    bool
	Results    int64
	Error      error
	Books      []booksing.Book
	Book       *booksing.Book
	ExtraPaths []string
	Users      []booksing.User
	Downloads  []booksing.Download
	Q          string
	TimeTaken  int
	IsAdmin    bool
	Username   string
	TotalBooks int
	Limit      int64
	Offset     int64
	Indexing   bool
	CanConvert bool
}

type configuration struct {
	AdminUser         string   `default:"unknown"`
	UserHeader        string   `default:""`
	AllowAllusers     bool     `default:"true"`
	BookDir           string   `default:"."`
	ImportDir         string   `default:"./import"`
	FailDir           string   `default:"./failed"`
	DatabaseDir       string   `default:"./db/"`
	LogLevel          string   `default:"info"`
	BindAddress       string   `default:":7132"`
	Timezone          string   `default:"Europe/Amsterdam"`
	AcceptedLanguages []string `default:""`
	MaxSize           int64    `default:"0"`
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

	var db database
	log.WithField("dbpath", cfg.DatabaseDir).Debug("using this file")
	db, err = sqlite.New(cfg.DatabaseDir)

	if err != nil {
		log.WithField("err", err).Fatal("could not create fileDB")
	}
	defer db.Close()

	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.WithField("err", err).Fatal("could not load timezone")
	}

	tpl := template.New("")
	tpl.Funcs(templateFunctions)
	tpl, err = tpl.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		log.WithField("err", err).Fatal("could not read templates")
		return
	}

	app := booksingApp{
		db:        db,
		bookDir:   cfg.BookDir,
		importDir: cfg.ImportDir,
		timezone:  tz,
		adminUser: cfg.AdminUser,
		logger:    log.WithField("app", "booksing"),
		cfg:       cfg,
	}

	//Check if ebook-convert is present so we can provide additional functionality
	_, err = exec.LookPath("ebook-convert")
	if err == nil {
		app.canConvert = true
	}

	if cfg.ImportDir != "" {
		go app.refreshLoop()
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(Logger(app.logger), gin.Recovery())
	r.SetHTMLTemplate(tpl)

	static := r.Group("/", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=86400, immutable")
	})

	static.StaticFS("/static", http.FS(staticFiles))

	r.GET("/kill", func(c *gin.Context) {
		app.logger.Fatal("Killing so I get restarted anew")
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": app.state,
			"total":  app.db.GetBookCount(),
		})
	})

	auth := r.Group("/")
	auth.Use(app.BearerTokenMiddleware())
	{
		auth.GET("/", app.search)
		auth.GET("/detail/:hash", app.detailPage)
		auth.POST("/convert/:hash", app.convert)
		auth.GET("/download", app.downloadBook)
		auth.GET("/cover", app.cover)

	}

	admin := r.Group("/admin")
	admin.Use(gin.Recovery(), app.BearerTokenMiddleware(), app.mustBeAdmin())
	{
		admin.GET("/users", app.showUsers)
		admin.GET("/downloads", app.showDownloads)
		admin.POST("/delete/:hash", app.deleteBook)
		admin.POST("user/:username", app.updateUser)
		admin.POST("/adduser", app.addUser)
	}

	log.Info("booksing is now running")
	port := os.Getenv("PORT")

	if port == "" {
		port = cfg.BindAddress
	} else {
		port = fmt.Sprintf(":%s", port)
	}

	err = r.Run(port)
	if err != nil {
		log.WithField("err", err).Fatal("unable to start running")
	}
}

func (app *booksingApp) keepBook(b *booksing.Book) bool {
	if b == nil {
		return false
	}

	if app.cfg.MaxSize > 0 && b.Size > app.cfg.MaxSize {
		return false
	}

	if len(app.cfg.AcceptedLanguages) > 0 {
		return contains(app.cfg.AcceptedLanguages, b.Language)
	}

	return true
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.EqualFold(s, needle) {
			return true
		}
	}
	return false
}
