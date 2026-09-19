package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		attributes := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"request_id", requestIDFromContext(c),
			"client_ip", c.ClientIP(),
		}
		if role, exists := c.Get("role"); exists {
			attributes = append(attributes, "role", role)
		}
		if c.Writer.Status() >= 500 {
			log.Error("http_request", attributes...)
		} else {
			log.Info("http_request", attributes...)
		}
	}
}

func requestIDFromContext(c *gin.Context) string {
	value, _ := c.Get("request_id")
	requestID, _ := value.(string)
	return requestID
}
