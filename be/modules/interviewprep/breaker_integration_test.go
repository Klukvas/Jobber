//go:build integration

package interviewprep

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type localBreakerTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t localBreakerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	copy.URL.Scheme = t.target.Scheme
	copy.URL.Host = t.target.Host
	copy.Host = t.target.Host
	return t.base.RoundTrip(copy)
}
func TestRealAdapterOpenBreakerRefundsOnlyNeverSentOperation(t *testing.T) {
	f := setup(t, 1)
	ctx := t.Context()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "error": map[string]string{"type": "overloaded_error", "message": "synthetic unavailable"}})
	}))
	defer server.Close()
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	original := http.DefaultTransport
	http.DefaultTransport = localBreakerTransport{base: original, target: target}
	defer func() { http.DefaultTransport = original }()
	client := ai.NewAnthropicClient("synthetic-local-only")
	for i := 0; i < 5; i++ {
		_, err = client.Interview(context.Background(), "plan", []byte(`{}`), nil)
		require.Error(t, err)
	}
	require.EqualValues(t, 5, requests.Load())
	f.svc.client = client
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	for i := 0; i < 4; i++ {
		require.NoError(t, f.svc.RunOnce(ctx))
		require.EqualValues(t, 5, requests.Load())
		var usage, attempts int
		var started, released bool
		require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usage))
		require.Zero(t, usage)
		require.NoError(t, f.pool.QueryRow(ctx, `SELECT attempts,provider_started,charge_released FROM interview_operations WHERE id=$1`, op.ID).Scan(&attempts, &started, &released))
		require.Zero(t, attempts)
		require.False(t, started)
		require.True(t, released)
		_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
		require.NoError(t, err)
		require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usage))
		require.Equal(t, 1, usage)
	}
	// Prior intent is ambiguous/sent: a later definite rejection cannot refund it.
	_, err = f.pool.Exec(ctx, `UPDATE interview_operations SET provider_started=true,attempts=1 WHERE id=$1`, op.ID)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	require.EqualValues(t, 5, requests.Load())
	var usage, attempts int
	var started, released bool
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usage))
	require.Equal(t, 1, usage)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT attempts,provider_started,charge_released FROM interview_operations WHERE id=$1`, op.ID).Scan(&attempts, &started, &released))
	require.Equal(t, 1, attempts)
	require.True(t, started)
	require.False(t, released)
	f.svc.client = f.client
	_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	var status string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT status FROM interview_operations WHERE id=$1`, op.ID).Scan(&status))
	require.Equal(t, "succeeded", status)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usage))
	require.Equal(t, 1, usage)

}
