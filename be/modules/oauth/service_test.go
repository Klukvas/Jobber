package oauth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/require"
)

type stubStore struct {
	Store
	client                                         *Client
	code                                           Code
	pair                                           TokenPair
	exchangeErr                                    error
	authHash, audience, revokedHash, revokedClient string
	saved                                          int
}

func (s *stubStore) GetClient(context.Context, string) (*Client, error) {
	if s.client == nil {
		return nil, ErrNotFound
	}
	return s.client, nil
}
func (s *stubStore) SaveClient(_ context.Context, c Client) error {
	s.client = &c
	s.saved++
	return nil
}
func (s *stubStore) CreateCode(_ context.Context, _, _ string, c Code) error { s.code = c; return nil }
func (s *stubStore) Exchange(_ context.Context, _ ExchangeRequest, p TokenPair) error {
	s.pair = p
	return s.exchangeErr
}
func (s *stubStore) Cleanup(context.Context) error { return nil }
func (s *stubStore) Authenticate(_ context.Context, h, a string) (*Identity, error) {
	s.authHash, s.audience = h, a
	return &Identity{UserID: "user", GrantID: "grant", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (s *stubStore) RevokeToken(_ context.Context, h, c string) error {
	s.revokedHash, s.revokedClient = h, c
	return nil
}

type paidStub struct{ err error }

func (p paidStub) RequirePaidPlan(context.Context, string) error { return p.err }

type fetchStub struct{ calls int }

func (f *fetchStub) Fetch(_ context.Context, id string) (*Client, error) {
	f.calls++
	now := time.Now()
	return &Client{ID: id, RedirectURIs: []string{callback}, GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"}, IsMetadata: true, FetchedAt: &now}, nil
}

func newTestService() (*Service, *stubStore) {
	store := &stubStore{client: &Client{ID: "client", Name: "Assistant", RedirectURIs: []string{callback}, GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"}}}
	return NewService(store, paidStub{}, "https://cursopus.com/", nil), store
}
func validAuthorize() AuthorizeParams {
	return AuthorizeParams{ClientID: "client", RedirectURI: callback, ResponseType: "code", Challenge: testChallenge, ChallengeMethod: "S256", Scope: "mcp", State: "opaque&state"}
}

func TestPreviewErrorSplit(t *testing.T) {
	for _, test := range []struct {
		name  string
		edit  func(*Service, *stubStore, AuthorizeParams) AuthorizeParams
		local bool
		code  string
	}{
		{"unknown client", func(_ *Service, s *stubStore, p AuthorizeParams) AuthorizeParams { s.client = nil; return p }, true, ""},
		{"untrusted redirect wins", func(_ *Service, _ *stubStore, p AuthorizeParams) AuthorizeParams {
			p.RedirectURI = "https://evil.test/cb"
			p.ChallengeMethod = "plain"
			return p
		}, true, ""},
		{"response type", func(_ *Service, _ *stubStore, p AuthorizeParams) AuthorizeParams { p.ResponseType = "token"; return p }, false, "unsupported_response_type"},
		{"plain PKCE", func(_ *Service, _ *stubStore, p AuthorizeParams) AuthorizeParams {
			p.ChallengeMethod = "plain"
			return p
		}, false, "invalid_request"},
		{"missing PKCE", func(_ *Service, _ *stubStore, p AuthorizeParams) AuthorizeParams { p.Challenge = ""; return p }, false, "invalid_request"},
		{"scope", func(_ *Service, _ *stubStore, p AuthorizeParams) AuthorizeParams { p.Scope = "admin"; return p }, false, "invalid_scope"},
		{"resource", func(_ *Service, _ *stubStore, p AuthorizeParams) AuthorizeParams {
			p.Resource = "https://evil.test/api/mcp"
			return p
		}, false, "invalid_target"},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, store := newTestService()
			params := test.edit(service, store, validAuthorize())
			result, err := service.Preview(t.Context(), "user", params)
			if test.local {
				require.Error(t, err)
				require.Nil(t, result)
				return
			}
			require.NoError(t, err)
			target, err := url.Parse(result.RedirectTo)
			require.NoError(t, err)
			require.Equal(t, test.code, target.Query().Get("error"))
			require.Equal(t, params.State, target.Query().Get("state"))
			require.Equal(t, "https://cursopus.com", target.Query().Get("iss"))
		})
	}
}

func TestAuthorizationDecisions(t *testing.T) {
	for _, decision := range []string{"allow", "deny", "unknown"} {
		t.Run(decision, func(t *testing.T) {
			service, store := newTestService()
			params := validAuthorize()
			params.Decision = decision
			redirect, err := service.Authorize(t.Context(), "user", params)
			if decision == "unknown" {
				require.ErrorIs(t, err, ErrInvalidRequest)
				return
			}
			require.NoError(t, err)
			target, err := url.Parse(redirect)
			require.NoError(t, err)
			if decision == "deny" {
				require.Equal(t, "access_denied", target.Query().Get("error"))
				require.Empty(t, store.code.Hash)
				return
			}
			require.Equal(t, hashSecret(target.Query().Get("code")), store.code.Hash)
			require.Equal(t, testChallenge, store.code.Challenge)
			require.Equal(t, service.Resource(), store.code.Resource)
			require.WithinDuration(t, time.Now().Add(10*time.Minute), store.code.ExpiresAt, time.Second)
		})
	}
	service, _ := newTestService()
	service.paid = paidStub{subModel.ErrPaidFeature}
	preview, err := service.Preview(t.Context(), "user", validAuthorize())
	require.NoError(t, err)
	require.False(t, preview.PlanAllowed)
	params := validAuthorize()
	params.Decision = "allow"
	_, err = service.Authorize(t.Context(), "user", params)
	require.ErrorIs(t, err, subModel.ErrPaidFeature)
	service.paid = paidStub{errors.New("db unavailable")}
	_, err = service.Preview(t.Context(), "user", params)
	require.Error(t, err)
}

func TestClientMetadataCache(t *testing.T) {
	service, store := newTestService()
	fetcher := &fetchStub{}
	service.fetcher = fetcher
	store.client = nil
	params := validAuthorize()
	params.ClientID = "https://client.test/meta"
	_, err := service.Preview(t.Context(), "user", params)
	client := store.client
	require.NoError(t, err)
	require.True(t, client.IsMetadata)
	require.Equal(t, 1, fetcher.calls)
	_, err = service.Preview(t.Context(), "user", params)
	require.NoError(t, err)
	require.Equal(t, 1, fetcher.calls)
	old := time.Now().Add(-25 * time.Hour)
	store.client.FetchedAt = &old
	_, err = service.Preview(t.Context(), "user", params)
	require.NoError(t, err)
	require.Equal(t, 2, fetcher.calls)
	registered, err := service.Register(t.Context(), Client{RedirectURIs: []string{callback}})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(registered.ID, "jbr_client_"))
	require.Len(t, registered.ID, len("jbr_client_")+32)
	require.False(t, registered.IsMetadata)
}

func TestTokenIssuanceAndAuthentication(t *testing.T) {
	service, store := newTestService()
	response, err := service.Token(t.Context(), ExchangeRequest{Kind: "authorization_code", ClientID: "client", Credential: strings.Repeat("A", 43), Verifier: testVerifier, RedirectURI: callback})
	require.NoError(t, err)
	require.Equal(t, 3600, response.ExpiresIn)
	require.Equal(t, "Bearer", response.TokenType)
	require.True(t, validSecret(response.AccessToken, "jbr_oat_"))
	require.True(t, validSecret(response.RefreshToken, "jbr_ort_"))
	require.Equal(t, hashSecret(response.AccessToken), store.pair.AccessHash)
	require.Equal(t, hashSecret(response.RefreshToken), store.pair.RefreshHash)
	require.WithinDuration(t, time.Now().Add(time.Hour), store.pair.AccessExpiry, time.Second)
	require.WithinDuration(t, time.Now().Add(30*24*time.Hour), store.pair.RefreshExpiry, time.Second)
	_, err = service.Authenticate(t.Context(), response.AccessToken)
	require.NoError(t, err)
	require.Equal(t, service.Resource(), store.audience)
	require.Equal(t, hashSecret(response.AccessToken), store.authHash)
	require.NoError(t, service.Revoke(t.Context(), response.RefreshToken, "client"))
	require.Equal(t, hashSecret(response.RefreshToken), store.revokedHash)
	_, err = service.Authenticate(t.Context(), response.RefreshToken)
	require.ErrorIs(t, err, ErrInvalidGrant)
}

func TestTokenRequestValidation(t *testing.T) {
	service, _ := newTestService()
	for _, test := range []struct {
		name    string
		request ExchangeRequest
		err     error
	}{
		{"no client", ExchangeRequest{}, ErrInvalidClient},
		{"unsupported", ExchangeRequest{ClientID: "client", Kind: "password"}, ErrUnsupportedGrant},
		{"bad code", ExchangeRequest{ClientID: "client", Kind: "authorization_code", Credential: "bad"}, ErrInvalidGrant},
		{"bad audience", ExchangeRequest{ClientID: "client", Resource: "https://evil.test/api/mcp"}, ErrInvalidGrant},
		{"refresh wrong scope", ExchangeRequest{ClientID: "client", Kind: "refresh_token", Credential: "jbr_ort_" + strings.Repeat("A", 43), Scope: "mcp admin"}, ErrInvalidRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Token(t.Context(), test.request)
			require.ErrorIs(t, err, test.err)
		})
	}
}
