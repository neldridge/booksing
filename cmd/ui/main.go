package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/gnur/booksing/meili"
	"github.com/gnur/booksing/storm"
	"github.com/markbates/pkger"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// V is the holder struct for all possible template values
type V struct {
	Results   int
	Error     error
	Books     []booksing.Book
	Q         string
	TimeTaken int
	IsAdmin   bool
	Username  string
}

type configuration struct {
	AdminUser string `default:""`
	BookDir   string `default:"."`
	ImportDir string `default:"./import"`
	Database  string `default:"file://booksing.db"`
	Meili     struct {
		Host  string
		Index string `default:"books"`
		Key   string `required:"true"`
	}
	LogLevel    string `default:"info"`
	BindAddress string `default:"localhost:7132"`
	Timezone    string `default:"Europe/Amsterdam"`
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

	var db database
	if strings.HasPrefix(cfg.Database, "file://") {
		log.WithField("filedbpath", cfg.Database).Debug("using this file")
		db, err = storm.New(strings.TrimPrefix(cfg.Database, "file://"))
		if err != nil {
			log.WithField("err", err).Fatal("could not create fileDB")
		}
		defer db.Close()
	} else {
		log.Fatal("invalid database chosen")
	}

	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.WithField("err", err).Fatal("could not load timezone")
	}

	var s search
	s, err = meili.New(cfg.Meili.Host, cfg.Meili.Index, cfg.Meili.Key)
	if err != nil {
		log.WithField("err", err).Fatal("unable to start meili client")
	}

	tpl := template.New("")
	tpl.Funcs(template.FuncMap{
		"percent": func(a, b int) float64 {
			return float64(a) / float64(b) * 100
		},
		"safeHTML": func(s interface{}) template.HTML {
			return template.HTML(fmt.Sprint(s))
		},
		"prettyTime": func(s interface{}) template.HTML {
			t, ok := s.(time.Time)
			if !ok {
				return ""
			}
			return template.HTML(t.Format("2006-01-02 15:04:05"))
		},
		"json": func(s interface{}) template.HTML {
			json, _ := json.MarshalIndent(s, "", "  ")
			return template.HTML(string(json))
		},
		"relativeTime": func(s interface{}) template.HTML {
			t, ok := s.(time.Time)
			if !ok {
				return ""
			}
			tense := "ago"
			diff := time.Since(t)
			seconds := int64(diff.Seconds())
			if seconds < 0 {
				tense = "from now"
			}
			var quantifier string

			if seconds < 60 {
				quantifier = "s"
			} else if seconds < 3600 {
				quantifier = "m"
				seconds /= 60
			} else if seconds < 86400 {
				quantifier = "h"
				seconds /= 3600
			} else if seconds < 604800 {
				quantifier = "d"
				seconds /= 86400
			} else if seconds < 31556736 {
				quantifier = "w"
				seconds /= 604800
			} else {
				quantifier = "y"
				seconds /= 31556736
			}

			return template.HTML(fmt.Sprintf("%v%s %s", seconds, quantifier, tense))
		},
	})

	err = pkger.Walk("/cmd/ui/templates", func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".html") {
			log.WithField("path", path).Debug("loading template")
			f, err := pkger.Open(path)
			if err != nil {
				return err
			}
			sl, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			_, err = tpl.Parse(string(sl))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.WithField("err", err).Fatal("could not read templates")
		return
	}

	app := booksingApp{
		db:        db,
		s:         s,
		bookDir:   cfg.BookDir,
		importDir: cfg.ImportDir,
		timezone:  tz,
		adminUser: cfg.AdminUser,
		logger:    log.WithField("app", "booksing"),
		cfg:       cfg,
	}

	if cfg.ImportDir != "" {
		go app.refreshLoop()
	}

	r := gin.New()
	r.Use(Logger(app.logger), gin.Recovery())
	r.SetHTMLTemplate(tpl)
	r.GET("/", app.search)
	r.GET("download", app.downloadBook)

	auth := r.Group("/auth")
	auth.Use(gin.Recovery(), app.BearerTokenMiddleware())
	{
		auth.GET("search", app.getBooks)
		auth.GET("stats", app.getStats)

	}

	admin := r.Group("/admin")
	admin.Use(gin.Recovery(), app.BearerTokenMiddleware(), app.mustBeAdmin())
	{
		admin.POST("user/:username", app.updateUser)
		admin.POST("refresh", app.refreshBooks)
		admin.POST("delete", app.deleteBook)
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

func (app *booksingApp) IsUserAdmin(c *gin.Context) bool {

	return true
}
