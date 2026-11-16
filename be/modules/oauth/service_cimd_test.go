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

// The real document Claude Code serves as its client_id: portless loopback
// callbacks, while the CLI listens on an ephemeral port.
const claudeCodeMetadataID = "https://claude.ai/oauth/claude-code-client-metadata"

const claudeCodeMetadata = `{"client_id":"https://claude.ai/oauth/claude-code-client-metadata","client_name":"Claude Code","client_uri":"https://claude.ai","redirect_uris":["http://localhost/callback","http://127.0.0.1/callback"],"grant_types":["authorization_code","refresh_token"],"response_types":["code"],"token_endpoint_auth_method":"none"}`

// The real document claude.ai serves for custom connectors; it lists a
// jwt-bearer grant this server does not support.
const claudeWebMetadataID = "https://claude.ai/oauth/mcp-oauth-client-metadata"

const claudeWebMetadata = `{"client_id":"https://claude.ai/oauth/mcp-oauth-client-metadata","client_name":"Claude","client_uri":"https://claude.ai","redirect_uris":["https://claude.ai/api/mcp/auth_callback"],"grant_types":["authorization_code","refresh_token","urn:ietf:params:oauth:grant-type:jwt-bearer"],"response_types":["code"],"token_endpoint_auth_method":"none"}`

type documentFetcher struct{ body string }

func (f documentFetcher) Fetch(_ context.Context, id string) (*Client, error) {
	return parseMetadata(id, []byte(f.body))
}

func TestClaudeCodeMetadataDocumentAuthorizes(t *testing.T) {
	for _, test := range []struct {
		redirect string
		allowed  bool
	}{
		{"http://localhost:54321/callback", true},
		{"http://127.0.0.1:61000/callback", true},
		{"http://localhost:54321/evil", false},
		{"https://evil.test/callback", false},
	} {
		t.Run(test.redirect, func(t *testing.T) {
			service, store := newTestService()
			store.client = nil
			service.fetcher = documentFetcher{claudeCodeMetadata}
			params := validAuthorize()
			params.ClientID, params.RedirectURI = claudeCodeMetadataID, test.redirect
			preview, err := service.Preview(t.Context(), "user", params)
			if !test.allowed {
				require.ErrorIs(t, err, ErrInvalidRequest)
				require.Zero(t, store.saved)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, preview.Client)
			require.Equal(t, "Claude Code", preview.Client.Name)
			require.Equal(t, 1, store.saved)
		})
	}
}

func TestClaudeWebMetadataDocumentAuthorizes(t *testing.T) {
	service, store := newTestService()
	store.client = nil
	service.fetcher = documentFetcher{claudeWebMetadata}
	params := validAuthorize()
	params.ClientID, params.RedirectURI = claudeWebMetadataID, "https://claude.ai/api/mcp/auth_callback"
	preview, err := service.Preview(t.Context(), "user", params)
	require.NoError(t, err)
	require.Equal(t, "Claude", preview.Client.Name)
}
