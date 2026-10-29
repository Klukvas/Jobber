package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/require"
)

type guardStore struct {
	reserves, dispatches, settles, releases int
	blocks                                  int
	reserved, actual                        int64
	reject                                  error
}

func (s *guardStore) ReserveCreditAttempt(_ context.Context, r subModel.AIAttempt) (string, error) {
	if s.reject != nil {
		return "", s.reject
	}
	s.reserves++
	s.reserved = r.MaxNanodollars
	return "attempt", nil
}
func (s *guardStore) DispatchCreditAttempt(context.Context, string) error { s.dispatches++; return nil }
func (s *guardStore) SettleCreditAttempt(_ context.Context, _ string, actual int64) error {
	s.settles++
	s.actual = actual
	return nil
}
func (s *guardStore) ReleaseCreditAttempt(context.Context, string) error { s.releases++; return nil }
func guardContext() context.Context {
	return CreditContext(context.Background(), func() string { return "owner" }, "operation", "improve")
}
func guardParams() anthropic.MessageNewParams {
	return anthropic.MessageNewParams{Model: anthropic.ModelClaudeHaiku4_5_20251001, MaxTokens: 1024, Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("private resume"))}}
}
func decodedUsage(t *testing.T, body string) *anthropic.Message {
	t.Helper()
	var m anthropic.Message
	require.NoError(t, json.Unmarshal([]byte(body), &m))
	return &m
}
func TestBudgetGuardPreflightAndOpenBreakerNeverDispatch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		context context.Context
		reject  error
		open    bool
	}{
		{name: "missing identity", context: context.Background()},
		{name: "global budget exhausted", context: guardContext(), reject: subModel.ErrAIBudgetReached},
		{name: "credits exhausted", context: guardContext(), reject: subModel.ErrCreditUnavailable},
		{name: "breaker open", context: guardContext(), open: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &guardStore{reject: tc.reject}
			c := NewAnthropicClient("fake")
			c.SetCreditStore(store)
			calls := 0
			c.callAPIFunc = func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				return nil, nil
			}
			if tc.open {
				c.breaker = circuitbreaker.New("test", 1, time.Hour)
				_ = c.breaker.Execute(func() error { return errors.New("unavailable") })
			}
			_, err := c.callAPI(tc.context, guardParams())
			require.Error(t, err)
			require.Zero(t, calls)
			require.Zero(t, store.dispatches)
			require.Zero(t, store.reserves)
		})
	}
}
func TestBudgetGuardRequiresAuthoritativeUsageAndExactModel(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		settles    int
		blocks     int
	}{
		{name: "known usage", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20}}`, settles: 1},
		{name: "coerced input string", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":"100","output_tokens":20}}`},
		{name: "coerced output string", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":"20"}}`},
		{name: "fractional input", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100.5,"output_tokens":20}}`},
		{name: "boolean output", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":false}}`},
		{name: "malformed present cache", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":{"unexpected":1}}}`},
		{name: "malformed cache read", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":"1"}}`},
		{name: "null cache read", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":null}}`},
		{name: "negative cache read", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":-1}}`},
		{name: "malformed cache container", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation":false}}`},
		{name: "malformed cache breakdown", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation":{"ephemeral_1h_input_tokens":"0"}}}`},
		{name: "premium cache breakdown", settles: 0, blocks: 1, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation":{"ephemeral_5m_input_tokens":1}}}`},
		{name: "malformed tool container", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"server_tool_use":[]}}`},
		{name: "malformed tool counter", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"server_tool_use":{"web_search_requests":"0"}}}`},
		{name: "malformed service tier", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"service_tier":{}}}`},
		{name: "unknown service tier", settles: 0, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"service_tier":"expedited"}}`},
		{name: "malformed and verified premium", settles: 0, blocks: 1, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_read_input_tokens":{},"service_tier":"priority"}}`},
		{name: "valid optional zero metadata", settles: 1, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":0},"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard"}}`},
		{name: "valid known batch tier", settles: 1, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"service_tier":"batch"}}`},
		{name: "unrelated usage metadata", settles: 1, blocks: 0, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"unrelated_metadata":{"future":true}}}`},
		{name: "missing output", body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100}}`},
		{name: "missing input", body: `{"model":"claude-haiku-4-5-20251001","usage":{"output_tokens":20}}`},
		{name: "partial wrong model", blocks: 1, body: `{"model":"claude-opus-4-1","usage":{"input_tokens":100}}`},
		{name: "partial excessive input", blocks: 1, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":200001}}`},
		{name: "partial excessive output", blocks: 1, body: `{"usage":{"output_tokens":1025}}`},
		{name: "partial cache premium", blocks: 1, body: `{"usage":{"cache_creation_input_tokens":1}}`},
		{name: "partial priority premium", blocks: 1, body: `{"usage":{"input_tokens":100,"service_tier":"priority"}}`},
		{name: "partial tool premium", blocks: 1, body: `{"usage":{"server_tool_use":{"web_search_requests":1}}}`},
		{name: "missing usage only", body: `{"model":"claude-haiku-4-5-20251001"}`},
		{name: "missing model only", body: `{"usage":{"input_tokens":100,"output_tokens":20}}`},
		{name: "different model", blocks: 1, body: `{"model":"claude-opus-4-1","usage":{"input_tokens":100,"output_tokens":20}}`},
		{name: "premium cache", blocks: 1, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":1}}`},
		{name: "output above cap", blocks: 1, body: `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":2000}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &guardStore{}
			c := NewAnthropicClient("fake")
			c.SetCreditStore(store)
			c.callAPIFunc = func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				return decodedUsage(t, tc.body), nil
			}
			_, err := c.callAPI(guardContext(), guardParams())
			if tc.name == "malformed present cache" {
				t.Logf("settlements=%d actual=%d reserved=%d", store.settles, store.actual, store.reserved)
			}
			if tc.settles == 1 {
				require.NoError(t, err)
				require.EqualValues(t, 200_000, store.actual)
			} else {
				require.ErrorIs(t, err, subModel.ErrAIBudgetReached)
			}
			require.Equal(t, tc.settles, store.settles)
			require.Equal(t, tc.blocks, store.blocks)
			require.Equal(t, 1, store.dispatches)
			require.Zero(t, store.releases)
			require.EqualValues(t, 205_120_000, store.reserved)
		})
	}
}
func TestBudgetGuardPaidFailureRetainsUnknownAndKnownFailureSettles(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown transport", true: "known usage with failed output"}[known], func(t *testing.T) {
			store := &guardStore{}
			c := NewAnthropicClient("fake")
			c.SetCreditStore(store)
			c.callAPIFunc = func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				if known {
					return decodedUsage(t, `{"model":"claude-haiku-4-5-20251001","stop_reason":"max_tokens","usage":{"input_tokens":100,"output_tokens":1024}}`), errors.New("output validation failed")
				}
				return nil, context.DeadlineExceeded
			}
			_, err := c.callAPI(guardContext(), guardParams())
			require.Error(t, err)
			require.Equal(t, 1, store.dispatches)
			require.Zero(t, store.releases)
			if known {
				require.Equal(t, 1, store.settles)
			} else {
				require.Zero(t, store.settles)
			}
		})
	}
}
func TestBudgetGuardStrictStageAndPremiumPolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*anthropic.MessageNewParams)
	}{
		{name: "wrong model", mutate: func(p *anthropic.MessageNewParams) { p.Model = "claude-sonnet-5-5" }},
		{name: "excess output", mutate: func(p *anthropic.MessageNewParams) { p.MaxTokens = 12000 }},
		{name: "fast mode", mutate: func(p *anthropic.MessageNewParams) { p.SetExtraFields(map[string]any{"speed": "fast"}) }},
		{name: "regional premium", mutate: func(p *anthropic.MessageNewParams) { p.SetExtraFields(map[string]any{"inference_geo": "us"}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &guardStore{}
			c := NewAnthropicClient("fake")
			c.SetCreditStore(store)
			calls := 0
			c.callAPIFunc = func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				return nil, nil
			}
			p := guardParams()
			tc.mutate(&p)
			_, err := c.callAPI(guardContext(), p)
			require.ErrorIs(t, err, subModel.ErrAIBudgetReached)
			require.Zero(t, store.reserves)
			require.Zero(t, calls)
		})
	}
}

func (s *guardStore) BlockAIProvider(context.Context, string) error { s.blocks++; return nil }

func (s *guardStore) AbortUnsentCreditAttempt(context.Context, string) error {
	s.releases++
	return nil
}

func TestBudgetGuardProviderMissingUsageTripsBreakerButAdmissionDoesNot(t *testing.T) {
	for _, admission := range []bool{false, true} {
		t.Run(map[bool]string{false: "paid missing usage", true: "admission denied"}[admission], func(t *testing.T) {
			store := &guardStore{}
			if admission {
				store.reject = subModel.ErrAIBudgetReached
			}
			c := NewAnthropicClient("fake")
			c.SetCreditStore(store)
			c.breaker = circuitbreaker.New("response-regression", 5, time.Hour)
			calls := 0
			c.callAPIFunc = func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				return decodedUsage(t, `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100}}`), nil
			}
			for i := 0; i < 5; i++ {
				_, err := c.callAPI(guardContext(), guardParams())
				require.ErrorIs(t, err, subModel.ErrAIBudgetReached)
			}
			if admission {
				require.Equal(t, circuitbreaker.StateClosed, c.breaker.State())
				require.Zero(t, calls)
				require.Zero(t, store.dispatches)
			} else {
				require.Equal(t, circuitbreaker.StateOpen, c.breaker.State())
				require.Equal(t, 5, calls)
				require.Equal(t, 5, store.dispatches)
				_, err := c.callAPI(guardContext(), guardParams())
				require.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
				require.Equal(t, 5, calls)
				require.Equal(t, 5, store.dispatches)
			}
			require.Zero(t, store.settles)
			require.Zero(t, store.releases)
			require.Zero(t, store.blocks)
		})
	}
}

func TestBudgetGuardMalformedBillingMetadataTripsBreaker(t *testing.T) {
	store := &guardStore{}
	c := NewAnthropicClient("fake")
	c.SetCreditStore(store)
	c.breaker = circuitbreaker.New("malformed-billing", 5, time.Hour)
	calls := 0
	c.callAPIFunc = func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		return decodedUsage(t, `{"model":"claude-haiku-4-5-20251001","usage":{"input_tokens":100,"output_tokens":20,"cache_creation_input_tokens":{"unexpected":1}}}`), nil
	}
	for i := 0; i < 5; i++ {
		_, err := c.callAPI(guardContext(), guardParams())
		require.ErrorIs(t, err, subModel.ErrAIBudgetReached)
	}
	require.Equal(t, circuitbreaker.StateOpen, c.breaker.State())
	_, err := c.callAPI(guardContext(), guardParams())
	require.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
	require.Equal(t, 5, calls)
	require.Equal(t, 5, store.dispatches)
	require.Zero(t, store.settles)
	require.Zero(t, store.releases)
	require.Zero(t, store.blocks)
}
