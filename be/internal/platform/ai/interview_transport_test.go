package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInterviewDispatchReleasesBeforeResponseAndNeverRedirects(t *testing.T) {
	response := make(chan struct{})
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		received <- string(raw)
		<-response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	var calls atomic.Int32
	dispatched := make(chan struct{})
	ctx, cancel, client, dispatchErr := interviewHTTP(t.Context(), func() error { calls.Add(1); close(dispatched); return nil })
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, strings.NewReader("visible facts"))
	require.NoError(t, err)
	finished := make(chan error, 1)
	go func() {
		res, e := client.Do(req)
		if e == nil {
			res.Body.Close()
		}
		finished <- e
	}()
	select {
	case <-dispatched:
	case <-time.After(time.Second):
		t.Fatal("request body did not dispatch")
	}
	require.Equal(t, "visible facts", <-received)
	select {
	case <-finished:
		t.Fatal("response unexpectedly finished before release")
	default:
	}
	close(response)
	require.NoError(t, <-finished)
	require.NoError(t, dispatchErr())
	require.EqualValues(t, 1, calls.Load())
	redirectCalls := atomic.Int32{}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCalls.Add(1)
		http.Redirect(w, r, server.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, redirect.URL, strings.NewReader("do not replay"))
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	require.EqualValues(t, 1, redirectCalls.Load())
}

func TestInterviewTransportRemovesReplayBody(t *testing.T) {
	called := false
	transport := interviewTransport{base: interviewRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		require.Nil(t, r.GetBody)
		return nil, context.Canceled
	})}
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1/", strings.NewReader("body"))
	require.NoError(t, err)
	require.NotNil(t, req.GetBody)
	_, err = transport.RoundTrip(req)
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, called)
	require.NotNil(t, req.GetBody)
}

type interviewRoundTripFunc func(*http.Request) (*http.Response, error)

func (f interviewRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
