package ai

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

type interviewTransport struct{ base http.RoundTripper }

func (t interviewTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	// A nonempty POST cannot be transparently replayed by the transport.
	copy.GetBody = nil
	return t.base.RoundTrip(copy)
}

// The callback runs only after net/http has written the request body. Connect or
// write stalls cancel the HTTP call; the source guard remains until it returns.
func interviewHTTP(ctx context.Context, dispatched func() error) (context.Context, context.CancelFunc, *http.Client, func() error) {
	callCtx, cancel := context.WithCancel(ctx)
	var once sync.Once
	var mu sync.Mutex
	var callbackErr error
	timer := time.AfterFunc(5*time.Second, cancel)
	trace := &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
		if info.Err != nil {
			return
		}
		once.Do(func() {
			timer.Stop()
			if dispatched != nil {
				err := dispatched()
				mu.Lock()
				callbackErr = err
				mu.Unlock()
				if err != nil {
					cancel()
				}
			}
		})
	}}
	callCtx = httptrace.WithClientTrace(callCtx, trace)
	client := &http.Client{Transport: interviewTransport{base: http.DefaultTransport}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	cleanup := func() { timer.Stop(); cancel() }
	readError := func() error { mu.Lock(); defer mu.Unlock(); return callbackErr }
	return callCtx, cleanup, client, readError
}
