package main

import (
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

		username := "erwin@gnur.nl"

		u, err := app.db.GetUser(username)
		if err == booksing.ErrNotFound {
			err = app.db.SaveUser(&booksing.User{
				Username:  username,
				IsAdmin:   username == app.adminUser,
				IsAllowed: username == app.adminUser,
				Created:   time.Now(),
				LastSeen:  time.Now(),
			})
			if err != nil {
				app.logger.WithField("err", err).Error("could not save new user")
				c.JSON(500, gin.H{
					"msg": "internal server error",
				})
				c.Abort()
				return
			}
		} else if err == nil {
			u.LastSeen = time.Now()
			err = app.db.SaveUser(&u)
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
		if !u.IsAllowed {
			c.JSON(430, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}

		c.Set("id", &u)

	}
}

func (app *booksingApp) mustBeAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.MustGet("id")
		user := id.(*booksing.User)
		if !user.IsAdmin {
			c.JSON(403, gin.H{
				"msg": "access denied",
			})
			c.Abort()
			return
		}
	}
}
