//go:build integration

package interviewprep

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func guardedInterviewFixture(t *testing.T, poolSize int32, url string) *fixture {
	t.Helper()
	f := setup(t, poolSize)
	t.Setenv("ANTHROPIC_BASE_URL", url)
	repo := subRepo.NewSubscriptionRepository(f.pool)
	require.NoError(t, repo.ConfigureAIBudget(t.Context(), 10_000_000_000))
	client := ai.NewAnthropicClient("local-fixture-only")
	client.SetCreditStore(repo)
	limits := subService.NewSubscriptionService(repo, nil, subService.BillingConfig{})
	f.svc = NewService(f.store, client, limits)
	return f
}
func localInterviewResponse(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var request struct {
		Messages []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
	require.NotEmpty(t, request.Messages)
	require.NotEmpty(t, request.Messages[0].Content)
	var input AIInput
	require.NoError(t, json.Unmarshal([]byte(request.Messages[0].Content[0].Text), &input))
	plan, err := json.Marshal(syntheticPlan(input.Snapshot.Facts))
	require.NoError(t, err)
	return map[string]any{"id": "msg_local", "type": "message", "role": "assistant", "model": "claude-haiku-4-5-20251001", "stop_reason": "end_turn", "content": []map[string]string{{"type": "text", "text": string(plan)}}, "usage": map[string]int{"input_tokens": 100, "output_tokens": 100, "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}}
}
func requireSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("bounded local operation stalled")
	}
}
func TestSubscriptionsInterviewConcurrentWorkersWithSmallPools(t *testing.T) {
	for _, size := range []int32{1, 2} {
		t.Run("pool-"+strconv.Itoa(int(size)), func(t *testing.T) {
			received := make(chan struct{}, 2)
			respond := make(chan struct{})
			defer func() {
				select {
				case <-respond:
				default:
					close(respond)
				}
			}()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := localInterviewResponse(t, r)
				calls.Add(1)
				received <- struct{}{}
				<-respond
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			f := guardedInterviewFixture(t, size, server.URL)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			for _, locale := range []string{"en", "uk"} {
				_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, locale, uuid.NewString())
				require.NoError(t, err)
			}
			done := make(chan error, 2)
			go func() { done <- f.svc.RunOnce(ctx) }()
			go func() { done <- f.svc.RunOnce(ctx) }()
			requireSignal(t, received)
			requireSignal(t, received)
			var one int
			probe, stop := context.WithTimeout(ctx, time.Second)
			require.NoError(t, f.pool.QueryRow(probe, `SELECT 1`).Scan(&one))
			stop()
			close(respond)
			require.NoError(t, <-done)
			require.NoError(t, <-done)
			require.EqualValues(t, 2, calls.Load())
			var successful int
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM interview_operations WHERE status='succeeded'`).Scan(&successful))
			require.Equal(t, 2, successful)
			b, err := subRepo.NewSubscriptionRepository(f.pool).CreditBalance(ctx, f.owner)
			require.NoError(t, err)
			require.Equal(t, 16, b.Used)
			require.EqualValues(t, 1_200_000, b.SpendAllocatedNanodollars)
		})
	}
}
func TestSubscriptionsInterviewStalePreflightRefundsWithoutProvider(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(500)
	}))
	defer server.Close()
	f := guardedInterviewFixture(t, 1, server.URL)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE resume_skills SET name='Changed source' WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	require.Zero(t, calls.Load())
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT error_code FROM interview_operations WHERE id=$1`, op.ID).Scan(&code))
	require.Equal(t, "SOURCE_CHANGED", code)
	b, err := subRepo.NewSubscriptionRepository(f.pool).CreditBalance(ctx, f.owner)
	require.NoError(t, err)
	require.Equal(t, 30, b.Remaining)
	require.Zero(t, b.SpendAllocatedNanodollars)
}
func TestSubscriptionsInterviewConnectionFailureReleasesSourceButRetainsUnknownBurn(t *testing.T) {
	f := guardedInterviewFixture(t, 1, "http://127.0.0.1:1")
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	_, err = f.pool.Exec(ctx, `UPDATE resume_skills SET name='Still editable' WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	b, err := subRepo.NewSubscriptionRepository(f.pool).CreditBalance(ctx, f.owner)
	require.NoError(t, err)
	require.Equal(t, 22, b.Remaining)
	require.EqualValues(t, 260_000_000, b.SpendAllocatedNanodollars)
}

type gateRoundTripper struct {
	base   http.RoundTripper
	before chan struct{}
	allow  <-chan struct{}
}

func (t gateRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	t.before <- struct{}{}
	select {
	case <-t.allow:
	case <-r.Context().Done():
		return nil, r.Context().Err()
	}
	return t.base.RoundTrip(r)
}
func TestSubscriptionsInterviewSourceGuardSurvivesUntilWriteThenReleasesBeforeResponse(t *testing.T) {
	before := make(chan struct{}, 1)
	allow := make(chan struct{})
	received := make(chan struct{}, 1)
	respond := make(chan struct{})
	old := http.DefaultTransport
	http.DefaultTransport = gateRoundTripper{base: old, before: before, allow: allow}
	defer func() { http.DefaultTransport = old }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := localInterviewResponse(t, r)
		received <- struct{}{}
		<-respond
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	f := guardedInterviewFixture(t, 2, server.URL)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- f.svc.RunOnce(ctx) }()
	requireSignal(t, before)
	blocked, stop := context.WithTimeout(ctx, 150*time.Millisecond)
	_, err = f.pool.Exec(blocked, `UPDATE resume_skills SET name='Must wait for dispatch' WHERE resume_builder_id=$1`, f.resume)
	stop()
	require.Error(t, err)
	close(allow)
	requireSignal(t, received)
	edit, stop := context.WithTimeout(ctx, time.Second)
	_, err = f.pool.Exec(edit, `UPDATE resume_skills SET name='Changed after dispatch' WHERE resume_builder_id=$1`, f.resume)
	stop()
	require.NoError(t, err)
	close(respond)
	require.NoError(t, <-done)
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT error_code FROM interview_operations WHERE id=$1`, op.ID).Scan(&code))
	require.Equal(t, "SOURCE_CHANGED", code)
	b, err := subRepo.NewSubscriptionRepository(f.pool).CreditBalance(ctx, f.owner)
	require.NoError(t, err)
	require.Equal(t, 22, b.Remaining)
	require.EqualValues(t, 600_000, b.SpendAllocatedNanodollars)
}
