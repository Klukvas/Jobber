package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type auditTraceTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *auditTraceTransport) Configure(sentry.ClientOptions)        {}
func (t *auditTraceTransport) Flush(time.Duration) bool              { return true }
func (t *auditTraceTransport) FlushWithContext(context.Context) bool { return true }
func (t *auditTraceTransport) Close()                                {}
func (t *auditTraceTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func auditTraceHub(t *testing.T, enabled bool) (*sentry.Hub, *auditTraceTransport) {
	t.Helper()
	transport := &auditTraceTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn: "https://public@example.invalid/1", EnableTracing: enabled, TracesSampleRate: 1,
		Transport: transport, DisableTelemetryBuffer: true,
	})
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return sentry.NewHub(client, sentry.NewScope()), transport
}

func TestAuditTracingOutcomeAndPrivacy(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		tool   string
		status sentry.SpanStatus
	}{
		{"success", nil, "audit_probe", sentry.SpanStatusOK},
		{"validation", validationError("private invalid input"), "audit_probe", sentry.SpanStatusInvalidArgument},
		{"missing", jobModel.ErrJobNotFound, "audit_probe", sentry.SpanStatusNotFound},
		{"unknown", nil, "private_user_supplied_name", sentry.SpanStatusNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hub, transport := auditTraceHub(t, true)
			server := auditProbeServer(t, auditRecorderFunc(func(context.Context, AuditEntry) error { return nil }), zap.NewNop(),
				func(ctx context.Context, _ string, _ auditProbeInput) (auditProbeInput, error) {
					span := sentry.SpanFromContext(ctx)
					require.NotNil(t, span, "span must cover tool execution")
					require.Equal(t, "mcp.tool", span.Op)
					return auditProbeInput{Text: "private output"}, test.err
				}, WithAuditTracing(true))
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Request = c.Request.WithContext(sentry.SetHubOnContext(c.Request.Context(), hub.Clone()))
				c.Next()
			})
			router.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
			registerHTTP(router, server, &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}}, &fakePaid{}, testHTTPConfig(), zap.NewNop())
			response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, strings.Replace(auditProbeCall, "audit_probe", test.tool, 1))
			require.Equal(t, 200, response.Code)
			transport.mu.Lock()
			defer transport.mu.Unlock()
			require.Len(t, transport.events, 1)
			event := transport.events[0]
			require.Nil(t, event.Request, "parent HTTP transaction must omit the request too")
			require.Len(t, event.Spans, 1)
			span := event.Spans[0]
			require.Equal(t, "mcp.tool", span.Op)
			wantTool := "audit_probe"
			if test.name == "unknown" {
				wantTool = unknownAuditTool
			}
			require.Equal(t, wantTool, span.Description)
			require.Equal(t, test.status, span.Status)
			require.Empty(t, span.Data)
			require.Empty(t, span.Tags)
			encoded, err := json.Marshal(event)
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "private")
			require.NotContains(t, string(encoded), validPAT)
		})
	}
}

func TestAuditTracingDisabledIsNoop(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "feature disabled"
		if enabled {
			name = "client tracing disabled"
		}
		t.Run(name, func(t *testing.T) {
			hub, transport := auditTraceHub(t, false)
			ctx := sentry.SetHubOnContext(t.Context(), hub)
			got, finish := startAuditSpan(ctx, enabled, "list_jobs")
			require.Equal(t, ctx, got)
			require.Nil(t, sentry.SpanFromContext(got))
			finish("internal")
			require.Empty(t, transport.events)
		})
	}
}

func TestAuditSpanStatus(t *testing.T) {
	for _, test := range []struct {
		code   string
		status sentry.SpanStatus
	}{
		{"", sentry.SpanStatusOK}, {"validation", sentry.SpanStatusInvalidArgument},
		{"not_found", sentry.SpanStatusNotFound}, {"unknown_tool", sentry.SpanStatusNotFound},
		{"no_credits", sentry.SpanStatusResourceExhausted}, {"limit_reached", sentry.SpanStatusResourceExhausted},
		{"paid_feature", sentry.SpanStatusPermissionDenied}, {"ai_unavailable", sentry.SpanStatusUnavailable},
		{"pdf_unavailable", sentry.SpanStatusUnavailable}, {"internal", sentry.SpanStatusInternalError},
	} {
		name := test.code
		if name == "" {
			name = "success"
		}
		t.Run(name, func(t *testing.T) { require.Equal(t, test.status, auditSpanStatus(test.code)) })
	}
}
