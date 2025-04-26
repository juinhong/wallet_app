package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		l := logger.WithFields(logrus.Fields{
			"status":    c.Writer.Status(),
			"method":    c.Request.Method,
			"path":      path,
			"query":     query,
			"ip":        c.ClientIP(),
			"userAgent": c.Request.UserAgent(),
			"latency":   latency,
		})

		if len(c.Errors) > 0 {
			l.Error(c.Errors.String())
		} else {
			l.Info("Request handled")
		}
	}
}
