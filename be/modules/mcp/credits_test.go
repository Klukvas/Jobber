package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2/proto"
	"github.com/andreypavlenko/jobber/internal/platform/ai"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	importService "github.com/andreypavlenko/jobber/modules/jobimport/service"
	subHandler "github.com/andreypavlenko/jobber/modules/subscriptions/handler"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type creditTransport func(*http.Request) (*http.Response, error)

func (f creditTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type guardedAttempt struct {
	operation  subModel.CreditOperation
	stage      string
	dispatched bool
	settled    bool
}
type guardedLedger struct {
	*aiLedger
	attempts    map[string]*guardedAttempt
	reject      bool
	rejectStage int
}

func (l *guardedLedger) reserveOperation(op subModel.CreditOperation) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.reserved[op.Key]; ok {
		if existing != op {
			return subModel.ErrAIOperationConflict
		}
		return nil
	}
	if l.balance[op.UserID] < op.Credits {
		return subModel.ErrCreditUnavailable
	}
	l.operations = append(l.operations, op)
	l.reserved[op.Key] = op
	l.balance[op.UserID] -= op.Credits
	return nil
}
func (l *guardedLedger) ReserveCreditAttempt(ctx context.Context, attempt subModel.AIAttempt) (string, error) {
	op, ok := ai.CreditOperationFromContext(ctx, attempt.Operation.UserID, "")
	if !ok || op != attempt.Operation {
		return "", subModel.ErrCreditUnavailable
	}
	if err := l.reserveOperation(op); err != nil {
		return "", err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	id := uuid.NewString()
	l.attempts[id] = &guardedAttempt{operation: op, stage: attempt.Stage}
	return id, nil
}
func (l *guardedLedger) DispatchCreditAttempt(_ context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.reject || l.rejectStage == len(l.attempts) {
		return subModel.ErrAIBudgetReached
	}
	l.attempts[id].dispatched = true
	l.completed[l.attempts[id].operation.Key] = true
	return nil
}
func (l *guardedLedger) SettleCreditAttempt(_ context.Context, id string, _ int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[id].settled = true
	return nil
}
func (l *guardedLedger) ReleaseCreditAttempt(_ context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	attempt := l.attempts[id]
	if attempt.dispatched {
		return subModel.ErrAIOperationConflict
	}
	delete(l.attempts, id)
	return nil
}
func (l *guardedLedger) BlockAIProvider(context.Context, string) error {
	return errors.New("unexpected pricing contract breach")
}
func (l *guardedLedger) AbortUnsentCreditAttempt(ctx context.Context, id string) error {
	return l.ReleaseCreditAttempt(ctx, id)
}

func providerResponse(t *testing.T, r *http.Request, fail bool) *http.Response {
	t.Helper()
	require.Equal(t, "/v1/messages", r.URL.Path)
	if fail {
		return &http.Response{StatusCode: 529, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"overloaded_error","message":"private provider detail"}}`)), Request: r}
	}
	payload := map[string]any{"id": "msg_fake", "type": "message", "role": "assistant", "model": "claude-haiku-4-5-20251001", "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 10, "output_tokens": 20}, "content": []map[string]string{{"type": "text", "text": `{"title":"Engineer","description":"Posting data"}`}}}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(encoded))), Request: r}
}

func TestCreditScopeRealJobParserWebParity(t *testing.T) {
	cases := []struct {
		name                   string
		fail, reject           bool
		balance, providerCalls int
	}{
		{"success", false, false, 98, 1},
		{"rejected before dispatch refunds", false, true, 100, 0},
		// Existing web policy retains dispatched liability, even on provider failure.
		// MCP must not introduce a refund that the web budget guard does not allow.
		{"provider failure after dispatch retains liability", true, false, 98, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, surface := range []string{"web", "mcp"} {
				t.Run(surface, func(t *testing.T) {
					f := newFakeAITools()
					store := &guardedLedger{aiLedger: f.ledger, attempts: map[string]*guardedAttempt{}, reject: tc.reject}
					previous := http.DefaultTransport
					calls := 0
					http.DefaultTransport = creditTransport(func(r *http.Request) (*http.Response, error) { calls++; return providerResponse(t, r, tc.fail), nil })
					t.Cleanup(func() { http.DefaultTransport = previous })
					provider := ai.NewAnthropicClient("fake-test-key")
					provider.SetCreditStore(store)
					parser := importService.NewImportService(provider, f.credits)
					if surface == "mcp" {
						deps := f.deps()
						deps.Parser = parser
						jobs := newFakeJobs()
						client := toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(deps)), userA)
						result := callTool(t, client, "parse_job_page", ParseJobPageInput{PageURL: "https://example.com/job", PageText: "A valid posting"})
						require.Equal(t, tc.fail || tc.reject, result.IsError)
					} else {
						runWebParser(t, parser, f)
					}
					require.Equal(t, tc.providerCalls, calls)
					require.Equal(t, tc.balance, f.ledger.balance[userA])
					require.Equal(t, 100, f.ledger.balance[userB])
					require.Equal(t, 1, f.ledger.finished)
				})
			}
		})
	}
}
func runWebParser(t *testing.T, parser *importService.ImportService, f *fakeAITools) {
	t.Helper()
	router := gin.New()
	router.Use(subHandler.AIRequestScope(f.credits))
	router.POST("/api/v1/jobs/parse", func(c *gin.Context) {
		c.Set("user_id", userA)
		_, err := parser.ParseJobPage(c.Request.Context(), userA, &importModel.ParseJobRequest{PageURL: "https://example.com/job", PageText: "A valid posting"})
		if err != nil {
			c.Status(503)
			return
		}
		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/jobs/parse", strings.NewReader(`{"page_url":"https://example.com/job","page_text":"A valid posting"}`)))
}

// WithTimeout copies the Redis client without its command hooks. Use an
// in-memory RESP connection so the production limiters execute successfully.
func creditRateLimitRedis(t *testing.T) (*redis.Client, <-chan string) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	keys := make(chan string, 2)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(keys)
		defer serverConn.Close()
		reader := bufio.NewReader(serverConn)
		for {
			wire, err := proto.Read(reader)
			if err != nil {
				return
			}
			args, err := proto.ReadStrings(wire)
			if err != nil || len(args) == 0 {
				return
			}
			reply := "-ERR unexpected command\r\n"
			switch {
			case args[0] == "hello":
				reply = "-ERR unknown command 'hello'\r\n"
			case args[0] == "evalsha" && len(args) == 5:
				keys <- args[3]
				reply = ":1\r\n"
			}
			if _, err := io.WriteString(serverConn, reply); err != nil {
				return
			}
		}
	}()
	client := redis.NewClient(&redis.Options{
		Addr: "in-memory", Protocol: 2, DisableIdentity: true, PoolSize: 1, MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) { return clientConn, nil },
	})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
		_ = serverConn.Close()
		<-done
	})
	return client, keys
}

type creditPATAuthenticator func(context.Context, string) (*patModel.AuthenticationInfo, error)

func (f creditPATAuthenticator) AuthenticateWithInfo(ctx context.Context, token string) (*patModel.AuthenticationInfo, error) {
	return f(ctx, token)
}

func TestCreditScopeHTTPMCPChargesAndFinishesOnce(t *testing.T) {
	f := newFakeAITools()
	store := &guardedLedger{aiLedger: f.ledger, attempts: map[string]*guardedAttempt{}}
	previous := http.DefaultTransport
	providerCalls := 0
	http.DefaultTransport = creditTransport(func(r *http.Request) (*http.Response, error) {
		providerCalls++
		return providerResponse(t, r, false), nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	provider := ai.NewAnthropicClient("fake-test-key")
	provider.SetCreditStore(store)
	deps := f.deps()
	deps.Parser = importService.NewImportService(provider, f.credits)
	jobs := newFakeJobs()
	resumes := newFakeResumeStore()
	server := NewServer(jobs, nil, jobs, nil,
		WithResumeTools(ResumeTools{Builder: resumes, Uploaded: &fakeUploadedResumes{}, Snapshots: &fakeResumeSnapshots{store: resumes}}),
		WithAITools(deps))
	patB := "jbr_pat_" + strings.Repeat("B", 43)
	patOwners := map[string]patModel.AuthenticationInfo{
		validPAT: {UserID: userA, TokenID: tokenA},
		patB:     {UserID: userB, TokenID: uuid.NewString()},
	}
	authCalls := 0
	tokens := creditPATAuthenticator(func(_ context.Context, token string) (*patModel.AuthenticationInfo, error) {
		authCalls++
		info, found := patOwners[token]
		if !found {
			return nil, patModel.ErrInvalidToken
		}
		return &info, nil
	})
	redisClient, rateLimitKeys := creditRateLimitRedis(t)
	router := gin.New()
	router.Use(httpPlatform.RequestIDMiddleware())
	router.Use(subHandler.AIRequestScope(f.credits))
	RegisterRoutes(router, server, tokens, f.credits, redisClient, zap.NewNop(), "")
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"parse_job_page","arguments":{"page_url":"https://example.com/job","page_text":"A valid posting"}}}`
	wantOperations := make([]subModel.CreditOperation, 0, 3)
	wantReserved := make(map[string]subModel.CreditOperation)
	// Keep one router/server alive across PAT switches to catch cached owners.
	for _, request := range []struct {
		name, token, owner, other string
	}{
		{"A first", validPAT, userA, userB},
		{"B", patB, userB, userA},
		{"A again", validPAT, userA, userB},
	} {
		if !t.Run(request.name, func(t *testing.T) {
			beforeProviderCalls, beforeAuthCalls := providerCalls, authCalls
			beforeReceipts, beforeAttempts := len(store.operations), len(store.attempts)
			beforeFinished := store.finished
			ownerBalance := store.balance[request.owner]
			otherBalance := store.balance[request.other]
			response := rpcHTTP(router, http.MethodPost, "Bearer "+request.token, body)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var envelope struct {
				Result *sdk.CallToolResult `json:"result"`
				Error  json.RawMessage     `json:"error"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			require.Empty(t, envelope.Error)
			require.NotNil(t, envelope.Result)
			output := decodeOutput[ParseJobPageOutput](t, envelope.Result)
			require.Equal(t, "Engineer", output.Draft.Title)
			require.Equal(t, beforeAuthCalls+1, authCalls)
			require.Len(t, rateLimitKeys, 2)
			require.Equal(t, "ratelimit:mcp_ip:192.0.2.1", <-rateLimitKeys)
			require.Equal(t, "ratelimit:mcp:user:"+request.owner, <-rateLimitKeys)
			require.Equal(t, beforeProviderCalls+1, providerCalls)
			require.Len(t, store.operations, beforeReceipts+1)
			operation := store.operations[beforeReceipts]
			require.Equal(t, request.owner, operation.UserID)
			require.Equal(t, "job_parse", operation.Kind)
			require.Equal(t, ai.OperationCosts["job_parse"], operation.Credits)
			_, err := uuid.Parse(operation.Key)
			require.NoError(t, err)
			require.NotEmpty(t, operation.Fingerprint)
			require.NotContains(t, wantReserved, operation.Key, "each call needs a new receipt")
			wantReserved[operation.Key] = operation
			wantOperations = append(wantOperations, operation)
			require.Equal(t, wantOperations, store.operations)
			require.Equal(t, ownerBalance-operation.Credits, store.balance[request.owner])
			require.Equal(t, otherBalance, store.balance[request.other])
			require.Equal(t, wantReserved, store.reserved)
			require.Len(t, store.attempts, beforeAttempts+1)
			currentAttempts := 0
			for _, attempt := range store.attempts {
				if attempt.operation.Key != operation.Key {
					continue
				}
				currentAttempts++
				require.Equal(t, operation, attempt.operation)
				require.True(t, attempt.dispatched)
				require.True(t, attempt.settled)
			}
			require.Equal(t, 1, currentAttempts)
			require.Equal(t, beforeFinished+1, store.finished)
			require.Equal(t, wantOperations, store.finishedOperations)
		}) {
			return
		}
	}
}
