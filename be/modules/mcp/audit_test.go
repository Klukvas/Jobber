package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type auditRecorderFunc func(context.Context, AuditEntry) error

func (f auditRecorderFunc) Record(ctx context.Context, entry AuditEntry) error { return f(ctx, entry) }

type auditProbeInput struct {
	Text string `json:"text"`
}

const auditProbeCall = `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"audit_probe","arguments":{"text":"private job and resume text"}}}`

func auditTestWriter(t *testing.T, recorder AuditRecorder, logger *zap.Logger) *AuditWriter {
	t.Helper()
	writer := NewAuditWriter(recorder, logger)
	writer.Start(t.Context())
	t.Cleanup(func() { require.NoError(t, writer.Close()) })
	return writer
}

func auditProbeServer(t *testing.T, recorder AuditRecorder, logger *zap.Logger, handle func(context.Context, string, auditProbeInput) (auditProbeInput, error), options ...ServerOption) *sdk.Server {
	t.Helper()
	server := sdk.NewServer(&sdk.Implementation{Name: "audit-test", Version: "1"}, nil)
	handlers := &tools{audit: auditTestWriter(t, recorder, logger), logger: logger}
	for _, option := range options {
		option(handlers)
	}
	register(server, handlers, "audit_probe", "test", true, true, handle)
	server.AddReceivingMiddleware(handlers.auditMiddleware)
	return server
}

func auditHTTPRouter(server *sdk.Server, paidErr error, config httpConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerHTTP(router, server, &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}},
		&fakePaid{err: paidErr}, config, zap.NewNop())
	return router
}

func TestAuditWrapperRecordsOnce(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{"success", nil, ""},
		{"not found", jobModel.ErrJobNotFound, "not_found"},
		{"validation", validationError("private invalid input"), "validation"},
		{"limit", subModel.ErrLimitReached, "limit_reached"},
		{"credits", subModel.ErrCreditUnavailable, "no_credits"},
		{"paid", subModel.ErrPaidFeature, "paid_feature"},
		{"AI unavailable", errAIUnavailable, "ai_unavailable"},
		{"PDF unavailable", errPDFUnavailable, "pdf_unavailable"},
		{"internal", errors.New("private SQL and resume text"), "internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var entries []AuditEntry
				recorder := auditRecorderFunc(func(ctx context.Context, entry AuditEntry) error {
					require.NoError(t, ctx.Err())
					deadline, ok := ctx.Deadline()
					require.True(t, ok)
					require.WithinDuration(t, time.Now().Add(auditWriteTimeout), deadline, time.Millisecond)
					entries = append(entries, entry)
					return nil
				})
				server := auditProbeServer(t, recorder, zap.NewNop(), func(_ context.Context, owner string, input auditProbeInput) (auditProbeInput, error) {
					require.Equal(t, userA, owner)
					require.Equal(t, "private job and resume text", input.Text)
					time.Sleep(12 * time.Millisecond)
					return auditProbeInput{Text: "private generated output"}, test.err
				})
				router := auditHTTPRouter(server, nil, testHTTPConfig())
				response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, auditProbeCall)
				require.Equal(t, 200, response.Code, response.Body.String())
				require.Equal(t, test.err != nil, strings.Contains(response.Body.String(), `"isError":true`))
				synctest.Wait()
				require.Len(t, entries, 1)
				entry := entries[0]
				_, err := uuid.Parse(entry.ID)
				require.NoError(t, err)
				want := AuditEntry{ID: entry.ID, UserID: userA, APITokenID: optionalText(tokenA), Tool: "audit_probe", OK: test.err == nil, DurationMS: 12}
				if test.code != "" {
					want.ErrorCode = &test.code
				}
				require.Equal(t, want, entry, "only the allowed metadata may reach persistence")
				encoded, err := json.Marshal(entry)
				require.NoError(t, err)
				require.NotContains(t, string(encoded), "private")
				require.NotContains(t, string(encoded), validPAT)
				// Protocol discovery does not create activity.
				require.Equal(t, 200, rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listToolsCall).Code)
				synctest.Wait()
				require.Len(t, entries, 1)
			})
		})
	}
}

func TestAuditSDKInputValidation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var entries []AuditEntry
		server := auditProbeServer(t, auditRecorderFunc(func(_ context.Context, entry AuditEntry) error {
			entries = append(entries, entry)
			return nil
		}), zap.NewNop(), func(context.Context, string, auditProbeInput) (auditProbeInput, error) {
			t.Error("schema rejection must precede the handler")
			return auditProbeInput{}, nil
		})
		router := auditHTTPRouter(server, nil, testHTTPConfig())
		body := strings.Replace(auditProbeCall, `"text":"private job and resume text"`, `"text":42`, 1)
		response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, body)
		require.Equal(t, 200, response.Code)
		require.Contains(t, response.Body.String(), `"isError":true`)
		synctest.Wait()
		require.Len(t, entries, 1)
		require.False(t, entries[0].OK)
		require.Equal(t, optionalText("validation"), entries[0].ErrorCode)
	})
}

func TestAuditHTTPRejectionsAreNotRecorded(t *testing.T) {
	tests := []struct {
		name, header, body string
		paidErr            error
		status             int
		limited            bool
	}{
		{name: "unauthorized", body: auditProbeCall, status: 401},
		{name: "free", header: "Bearer " + validPAT, body: auditProbeCall, paidErr: subModel.ErrPaidFeature, status: 403},
		{name: "rate limited", header: "Bearer " + validPAT, body: auditProbeCall, status: 429, limited: true},
		{name: "oversized", header: "Bearer " + validPAT, body: strings.Repeat(" ", maxBodyBytes+1), status: 413},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var entries []AuditEntry
				server := auditProbeServer(t, auditRecorderFunc(func(_ context.Context, entry AuditEntry) error {
					entries = append(entries, entry)
					return nil
				}), zap.NewNop(), func(context.Context, string, auditProbeInput) (auditProbeInput, error) {
					t.Error("HTTP rejection reached handler")
					return auditProbeInput{}, nil
				})
				config := testHTTPConfig()
				if test.limited {
					config.userLimiter = func(c *gin.Context) { c.AbortWithStatus(429) }
				}
				response := rpcHTTP(auditHTTPRouter(server, test.paidErr, config), http.MethodPost, test.header, test.body)
				require.Equal(t, test.status, response.Code)
				synctest.Wait()
				require.Empty(t, entries)
			})
		})
	}
}

func TestAuditFailureAndTimeoutDoNotDelayResult(t *testing.T) {
	for _, slow := range []bool{false, true} {
		name := "failure"
		if slow {
			name = "timeout"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				core, logs := observer.New(zap.WarnLevel)
				var completed atomic.Bool
				recorder := auditRecorderFunc(func(ctx context.Context, _ AuditEntry) error {
					defer completed.Store(true)
					if slow {
						<-ctx.Done()
						return ctx.Err()
					}
					return errors.New("private user, SQL, and token details")
				})
				server := auditProbeServer(t, recorder, zap.New(core), func(context.Context, string, auditProbeInput) (auditProbeInput, error) {
					return auditProbeInput{Text: "expected tool output"}, nil
				})
				started := time.Now()
				response := rpcHTTP(auditHTTPRouter(server, nil, testHTTPConfig()), http.MethodPost, "Bearer "+validPAT, auditProbeCall)
				require.Equal(t, 200, response.Code)
				require.Contains(t, response.Body.String(), "expected tool output")
				require.NotContains(t, response.Body.String(), "isError")
				require.Less(t, time.Since(started), auditWriteTimeout)
				if slow {
					synctest.Wait()
					require.False(t, completed.Load())
					time.Sleep(auditWriteTimeout)
				}
				synctest.Wait()
				require.True(t, completed.Load())
				require.Len(t, logs.All(), 1)
				require.Equal(t, "MCP audit write failed", logs.All()[0].Message)
				require.Equal(t, map[string]any{"tool": "audit_probe"}, logs.All()[0].ContextMap())
			})
		})
	}
}

func TestAuditDetachedContextAndNullableToken(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var entries []AuditEntry
		recorder := auditRecorderFunc(func(ctx context.Context, entry AuditEntry) error {
			require.NoError(t, ctx.Err(), "request cancellation must not cancel audit")
			entries = append(entries, entry)
			return nil
		})
		handlers := &tools{audit: auditTestWriter(t, recorder, zap.NewNop()), logger: zap.NewNop(), auditTools: map[string]bool{"audit_probe": true}}
		ctx, cancel := context.WithCancel(t.Context())
		call := &sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{Name: "audit_probe"}, Extra: &sdk.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: userA}}}
		next := handlers.auditMiddleware(func(ctx context.Context, _ string, _ sdk.Request) (sdk.Result, error) {
			ctx.Value(auditOutcomeKey{}).(*auditOutcome).code = ""
			cancel()
			return &sdk.CallToolResult{}, nil
		})
		_, err := next(ctx, "tools/call", call)
		require.NoError(t, err)
		synctest.Wait()
		require.Len(t, entries, 1)
		require.Nil(t, entries[0].APITokenID)
	})
}

func TestAuditUnknownToolUsesSentinel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var entries []AuditEntry
		server := auditProbeServer(t, auditRecorderFunc(func(_ context.Context, entry AuditEntry) error {
			entries = append(entries, entry)
			return nil
		}), zap.NewNop(), func(context.Context, string, auditProbeInput) (auditProbeInput, error) {
			t.Error("unknown tool invoked registered handler")
			return auditProbeInput{}, nil
		})
		response := rpcHTTP(auditHTTPRouter(server, nil, testHTTPConfig()), http.MethodPost, "Bearer "+validPAT,
			strings.Replace(auditProbeCall, "audit_probe", "private_user_supplied_name", 1))
		require.Equal(t, 200, response.Code)
		require.Contains(t, response.Body.String(), `"error"`)
		synctest.Wait()
		require.Len(t, entries, 1)
		require.Equal(t, unknownAuditTool, entries[0].Tool)
		require.False(t, entries[0].OK)
		require.Equal(t, optionalText("unknown_tool"), entries[0].ErrorCode)
		encoded, err := json.Marshal(entries)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), "private")
	})
}
