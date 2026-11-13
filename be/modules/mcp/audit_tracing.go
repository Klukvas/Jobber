package mcp

import (
	"context"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// The platform's Sentry Gin middleware buffers HTTP bodies. Remove the MCP
// request from its scope before auth/body decoding so even the parent HTTP
// transaction cannot include PAT headers or tool arguments.
func redactAuditRequest(c *gin.Context) {
	if hub := sentry.GetHubFromContext(c.Request.Context()); hub != nil {
		hub.Scope().SetRequest(nil)
		hub.Scope().SetRequestBody(nil)
	}
	c.Next()
}

// WithAuditTracing uses the enabled flag returned by the platform Sentry Init.
func WithAuditTracing(enabled bool) ServerOption {
	return func(t *tools) { t.auditTrace = enabled }
}

func startAuditSpan(ctx context.Context, enabled bool, tool string) (context.Context, func(string)) {
	noop := func(string) {}
	if !enabled {
		return ctx, noop
	}
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	client := hub.Client()
	if client == nil || !client.Options().EnableTracing {
		return ctx, noop
	}
	// Keep trace correlation, but do not copy request bodies, headers, user data,
	// breadcrumbs or other request scope fields into a standalone tool event.
	ctx = sentry.SetHubOnContext(ctx, sentry.NewHub(client, sentry.NewScope()))
	options := []sentry.SpanOption{sentry.WithDescription(tool)}
	if sentry.SpanFromContext(ctx) == nil {
		options = append(options, sentry.WithTransactionName(tool))
	}
	span := sentry.StartSpan(ctx, "mcp.tool", options...)
	return span.Context(), func(code string) {
		span.Status = auditSpanStatus(code)
		span.Finish()
	}
}

func auditSpanStatus(code string) sentry.SpanStatus {
	switch code {
	case "":
		return sentry.SpanStatusOK
	case "validation":
		return sentry.SpanStatusInvalidArgument
	case "not_found", "unknown_tool":
		return sentry.SpanStatusNotFound
	case "limit_reached", "no_credits":
		return sentry.SpanStatusResourceExhausted
	case "paid_feature":
		return sentry.SpanStatusPermissionDenied
	case "ai_unavailable", "pdf_unavailable":
		return sentry.SpanStatusUnavailable
	default:
		return sentry.SpanStatusInternalError
	}
}
