package http

import (
	"regexp"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const maxRequestIDLength = 128

var validRequestIDRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_\.]+$`)

// isValidRequestID checks that a client-supplied request ID is safe.
func isValidRequestID(id string) bool {
	return len(id) <= maxRequestIDLength && validRequestIDRegex.MatchString(id)
}

// RequestIDMiddleware adds a unique request ID to each request.
// Client-supplied IDs are validated; invalid values are replaced with a new UUID.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" || !isValidRequestID(requestID) {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggerMiddleware logs each request
func LoggerMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		requestID, _ := c.Get("request_id")

		c.Next()

		duration := time.Since(start).Milliseconds()
		statusCode := c.Writer.Status()

		logEntry := log.WithRequestID(requestID.(string)).
			WithAction(method + " " + path).
			WithDuration(duration)

		if statusCode >= 500 {
			logEntry.Error("request completed",
				zap.Int("status", statusCode),
			)
		} else if statusCode >= 400 {
			logEntry.Warn("request completed",
				zap.Int("status", statusCode),
			)
		} else {
			logEntry.Info("request completed",
				zap.Int("status", statusCode),
			)
		}
	}
}

// CORSMiddleware handles CORS with configurable allowed origins.
// allowedOrigins is a comma-separated list of origins (e.g. "https://jobber-app.com,https://www.jobber-app.com").
// Pass "*" to allow all origins (development only).
func CORSMiddleware(allowedOrigins string) gin.HandlerFunc {
	origins := parseOrigins(allowedOrigins)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowedOrigins == "*" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			// Do NOT set Allow-Credentials with wildcard origin — browsers reject this combination
		} else if isOriginAllowed(origin, origins) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func parseOrigins(raw string) []string {
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}
