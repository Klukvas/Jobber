package sentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Init initializes the Sentry SDK. Returns true if Sentry was enabled.
func Init(dsn, environment, release string, tracesSampleRate float64, logger *zap.Logger) bool {
	if dsn == "" {
		logger.Info("Sentry DSN not configured, error tracking disabled")
		return false
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:           dsn,
		Environment:   environment,
		Release:       release,
		EnableTracing: tracesSampleRate > 0,
		TracesSampler: tracesSampler(tracesSampleRate),
	})
	if err != nil {
		logger.Error("Failed to initialize Sentry", zap.Error(err))
		return false
	}

	logger.Info("Sentry initialized",
		zap.String("environment", environment),
		zap.String("release", release),
		zap.Float64("traces_sample_rate", tracesSampleRate),
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

// The container healthcheck hits /health every 10 seconds; tracing it would
// spend most of the transaction quota on a request nobody needs to profile.
var untracedTransactions = map[string]struct{}{
	"GET /health":  {},
	"HEAD /health": {},
}

// tracesSampler applies one rate to every transaction except the healthcheck.
// It ignores the caller's sentry-trace sampled flag, so an outside request
// cannot force its traces to be recorded.
func tracesSampler(rate float64) sentry.TracesSampler {
	return func(ctx sentry.SamplingContext) float64 {
		if _, skip := untracedTransactions[ctx.Span.Name]; skip {
			return 0
		}
		return rate
	}
}
