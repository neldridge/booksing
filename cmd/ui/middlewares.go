package main

import (
	"errors"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gnur/booksing"
	"github.com/sirupsen/logrus"
)

// Logger is the logrus logger handler
func Logger(log *logrus.Entry) gin.HandlerFunc {

	return func(c *gin.Context) {
		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := log.WithFields(logrus.Fields{
			"statusCode": statusCode,
			"latency":    latency, // time to process
			"clientIP":   clientIP,
			"method":     c.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": dataLength,
			"userAgent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := "GIN handled request"
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}

func (app *booksingApp) BearerTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetHeader(app.cfg.UserHeader)
		if username == "" {
			username = "unknown"
		}

		user, err := app.db.GetUser(username)
		if err == booksing.ErrNotFound {
			user = booksing.User{
				Name:      username,
				IsAdmin:   username == app.adminUser,
				IsAllowed: username == app.adminUser || app.cfg.AllowAllusers,
				Created:   time.Now(),
				LastSeen:  time.Now(),
			}
			err = app.db.SaveUser(&user)
			if err != nil {
				app.logger.WithField("err", err).Error("could not save new user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else if err == nil {
			user.LastSeen = time.Now()
			err = app.db.SaveUser(&user)
			if err != nil {
				app.logger.Error("could not update user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else {
			app.logger.WithField("err", err).Error("could not get user")
			c.JSON(500, gin.H{
				"msg": "internal server error",
			})
			c.Abort()
			return
		}
		if !user.IsAllowed {
			c.HTML(403, "error.html", V{
				Error: errors.New("User is not allowed to perform this action"),
			})
			c.Abort()
			return
		}

		c.Set("id", &user)
		c.Set("isAdmin", user.IsAdmin)
	}
}

func (app *booksingApp) mustBeAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool("isAdmin") {
			c.HTML(403, "error.html", V{
				Error: errors.New("User is not allowed to perform this action"),
			})
			c.Abort()
		}
	}
}
