package oauth

import (
	"context"
	"errors"
	"testing"
	"time"

	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/require"
)

func TestCIMDPlanAndRedirectGate(t *testing.T) {
	for _, test := range []struct {
		name                string
		planErr             error
		redirect, challenge string
		calls, saves        int
		wantErr             error
	}{
		{"free cannot fetch", subModel.ErrPaidFeature, callback, testChallenge, 0, 0, subModel.ErrPaidFeature},
		{"plan unavailable", errors.New("plan unavailable"), callback, testChallenge, 0, 0, nil},
		{"invalid redirect cannot fetch", nil, "http://evil.test/cb", testChallenge, 0, 0, ErrInvalidRequest},
		{"unregistered redirect cannot persist", nil, "https://evil.test/cb", testChallenge, 1, 0, ErrInvalidRequest},
		{"invalid PKCE cannot persist", nil, callback, "invalid", 1, 0, nil},
		{"valid request persists", nil, callback, testChallenge, 1, 1, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, store := newTestService()
			store.client = nil
			fetcher := &fetchStub{}
			service.paid, service.fetcher = paidStub{test.planErr}, fetcher
			params := validAuthorize()
			params.ClientID, params.RedirectURI, params.Challenge = "https://client.test/meta", test.redirect, test.challenge
			preview, err := service.Preview(t.Context(), "user", params)
			wantErr := test.wantErr
			if test.planErr != nil && !errors.Is(test.planErr, subModel.ErrPaidFeature) {
				wantErr = test.planErr
			}
			if wantErr != nil {
				require.ErrorIs(t, err, wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, preview)
			}
			require.Equal(t, test.calls, fetcher.calls)
			require.Equal(t, test.saves, store.saved)
		})
	}
}

func TestFreePlanCannotRefreshStaleCIMD(t *testing.T) {
	service, store := newTestService()
	fetcher := &fetchStub{}
	service.fetcher = fetcher
	params := validAuthorize()
	params.ClientID = "https://client.test/meta"
	store.client = nil
	_, err := service.Preview(t.Context(), "user", params)
	require.NoError(t, err)
	service.paid = paidStub{subModel.ErrPaidFeature}
	preview, err := service.Preview(t.Context(), "user", params)
	require.NoError(t, err)
	require.False(t, preview.PlanAllowed)
	expired := time.Now().Add(-25 * time.Hour)
	stale := *store.client
	stale.FetchedAt = &expired
	store.client = &stale
	_, err = service.Preview(t.Context(), "user", params)
	require.ErrorIs(t, err, subModel.ErrPaidFeature)
	require.Equal(t, 1, fetcher.calls)
	require.Equal(t, 1, store.saved)
}

type metadataFetchFunc func(context.Context, string) (*Client, error)

func (f metadataFetchFunc) Fetch(ctx context.Context, id string) (*Client, error) { return f(ctx, id) }

func TestCIMDConcurrentFetchLimit(t *testing.T) {
	service, store := newTestService()
	store.client = nil
	entered := make(chan struct{}, maxConcurrentMetadataFetches+1)
	release := make(chan struct{})
	service.fetcher = metadataFetchFunc(func(ctx context.Context, _ string) (*Client, error) {
		entered <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil, ErrInvalidClient
	})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	results := make(chan error, maxConcurrentMetadataFetches)
	for range maxConcurrentMetadataFetches {
		go func() { _, _, err := service.client(ctx, "https://client.test/meta", true); results <- err }()
	}
	for range maxConcurrentMetadataFetches {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("fetches did not start")
		}
	}
	_, _, err := service.client(ctx, "https://client.test/overflow", true)
	require.ErrorIs(t, err, ErrMetadataBusy)
	cancel()
	for range maxConcurrentMetadataFetches {
		require.ErrorIs(t, <-results, ErrInvalidClient)
	}
	close(release)
	// Failed/canceled fetches release their slots for subsequent requests.
	_, _, err = service.client(t.Context(), "https://client.test/retry", true)
	require.ErrorIs(t, err, ErrInvalidClient)
	require.Len(t, entered, 1)
}
