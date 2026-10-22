//go:build integration

package interviewprep

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type localInterviewTransport struct {
	base    http.RoundTripper
	target  *url.URL
	entered chan struct{}
	allow   chan struct{}
}

func (t localInterviewTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	close(t.entered)
	select {
	case <-t.allow:
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}
	copy := request.Clone(request.Context())
	copy.URL.Scheme = t.target.Scheme
	copy.URL.Host = t.target.Host
	copy.Host = t.target.Host
	return t.base.RoundTrip(copy)
}

// This combines real pgx source locks with the production Anthropic adapter and
// its net/http body-write hook; only the destination transport is local/test-only.
func TestProductionAdapterSourceGuardThroughActualHTTPDispatch(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	_, err := f.pool.Exec(ctx, `INSERT INTO resume_section_orders(resume_builder_id,section_key,is_visible) VALUES($1,'skills',true)`, f.resume)
	require.NoError(t, err)
	snapshot, err := f.store.Snapshot(ctx, f.owner, f.job, f.resume, "en")
	require.NoError(t, err)
	body, _ := json.Marshal(syntheticPlan(snapshot.Facts))
	var calls atomic.Int32
	received := make(chan struct{})
	response := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		close(received)
		<-response
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "synthetic-message", "type": "message", "role": "assistant", "model": "claude-haiku-4-5-20251001", "stop_reason": "end_turn", "content": []map[string]string{{"type": "text", "text": string(body)}}, "usage": map[string]int{"input_tokens": 1, "output_tokens": 1}})
	}))
	defer server.Close()
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	original := http.DefaultTransport
	entered, allow := make(chan struct{}), make(chan struct{})
	http.DefaultTransport = localInterviewTransport{base: original, target: target, entered: entered, allow: allow}
	defer func() { http.DefaultTransport = original }()
	release, err := f.store.guardDispatch(ctx, Operation{Owner: f.owner, JobID: f.job}, snapshot)
	require.NoError(t, err)
	defer release()
	client := ai.NewAnthropicClient("synthetic-key-no-external-network")
	payload, _ := json.Marshal(AIInput{Snapshot: snapshot})
	finished := make(chan error, 1)
	go func() { _, e := client.Interview(ctx, "plan", payload, release); _ = release(); finished <- e }()
	<-entered // coherent source check completed, actual network body write pending
	hidden := make(chan error, 1)
	go func() {
		_, e := f.pool.Exec(ctx, `UPDATE resume_section_orders SET is_visible=false WHERE resume_builder_id=$1`, f.resume)
		hidden <- e
	}()
	select {
	case e := <-hidden:
		require.NoError(t, e)
		t.Fatal("hide committed before body dispatch")
	case <-time.After(80 * time.Millisecond):
	}
	close(allow)
	<-received
	select {
	case e := <-hidden:
		require.NoError(t, e)
	case <-time.After(2 * time.Second):
		t.Fatal("source remained locked while awaiting model response")
	}
	select {
	case <-finished:
		t.Fatal("model response should still be pending")
	default:
	}
	close(response)
	require.NoError(t, <-finished)
	require.EqualValues(t, 1, calls.Load())
	// A change committed before dispatch prevents the worker from calling even the
	// production adapter, preserving the old snapshot's privacy and refund path.
	_, err = f.pool.Exec(ctx, `UPDATE resume_section_orders SET is_visible=true WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	_, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE resume_section_orders SET is_visible=false WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	f.svc.client = client
	require.NoError(t, f.svc.RunOnce(ctx))
	require.EqualValues(t, 1, calls.Load())
}
