package sentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Init initializes the Sentry SDK. Returns true if Sentry was enabled.
func Init(dsn, environment, release string, logger *zap.Logger) bool {
	if dsn == "" {
		logger.Info("Sentry DSN not configured, error tracking disabled")
		return false
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Release:          release,
		TracesSampleRate: 0.2,
	})
	if err != nil {
		logger.Error("Failed to initialize Sentry", zap.Error(err))
		return false
	}

	logger.Info("Sentry initialized",
		zap.String("environment", environment),
		zap.String("release", release),
	)
	return true
}

// RecoveryMiddleware returns sentrygin recovery if enabled, else gin.Recovery().
func RecoveryMiddleware(enabled bool) gin.HandlerFunc {
	if enabled {
		return sentrygin.New(sentrygin.Options{Repanic: true})
	}
	return gin.Recovery()
}

// CaptureError sends a non-panic error to Sentry with optional tags.
func CaptureError(err error, tags map[string]string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		sentry.CaptureException(err)
	})
}

// Flush waits for pending events to be sent before shutdown.
func Flush() {
	sentry.Flush(2 * time.Second)
}
