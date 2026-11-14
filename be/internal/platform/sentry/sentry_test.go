package sentry

import (
	"context"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
)

func sampleRateFor(rate float64, transaction string) float64 {
	span := sentry.StartTransaction(context.Background(), transaction)
	return tracesSampler(rate)(sentry.SamplingContext{Span: span})
}

func TestTracesSamplerSkipsHealthcheck(t *testing.T) {
	assert.Equal(t, 0.0, sampleRateFor(0.2, "GET /health"))
	assert.Equal(t, 0.0, sampleRateFor(0.2, "HEAD /health"))
}

func TestTracesSamplerAppliesRateToEverythingElse(t *testing.T) {
	for _, name := range []string{"POST /api/mcp", "GET /api/v1/jobs", "GET /healthz"} {
		assert.Equal(t, 0.2, sampleRateFor(0.2, name), name)
	}
}
