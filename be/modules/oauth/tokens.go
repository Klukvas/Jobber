package oauth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

func (s *Service) Token(ctx context.Context, request ExchangeRequest) (*TokenResponse, error) {
	if err := s.validateExchange(request); err != nil {
		return nil, err
	}
	client, err := s.store.GetClient(ctx, request.ClientID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidClient
		}
		return nil, err
	}
	if !contains(client.GrantTypes, request.Kind) {
		return nil, ErrInvalidGrant
	}
	access, err := randomSecret("jbr_oat_", 32)
	if err != nil {
		return nil, err
	}
	refresh, err := randomSecret("jbr_ort_", 32)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	pair := TokenPair{AccessHash: hashSecret(access), RefreshHash: hashSecret(refresh), Resource: s.Resource(), AccessExpiry: now.Add(time.Hour), RefreshExpiry: now.Add(30 * 24 * time.Hour)}
	if err := s.store.Cleanup(ctx); err != nil {
		return nil, err
	}
	if err := s.store.Exchange(ctx, request, pair); err != nil {
		return nil, err
	}
	return &TokenResponse{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: 3600, Scope: "mcp"}, nil
}

func (s *Service) validateExchange(request ExchangeRequest) error {
	if len(request.ClientID) == 0 || len(request.ClientID) > 2048 {
		return ErrInvalidClient
	}
	if request.Resource != "" && request.Resource != s.Resource() {
		return ErrInvalidGrant
	}
	switch request.Kind {
	case "authorization_code":
		if !validSecret(request.Credential, "") || !validVerifier(request.Verifier) || ValidateRedirectURI(request.RedirectURI) != nil {
			return ErrInvalidGrant
		}
	case "refresh_token":
		if !validSecret(request.Credential, "jbr_ort_") {
			return ErrInvalidGrant
		}
		if request.Scope != "" && request.Scope != "mcp" {
			return ErrInvalidRequest
		}
	default:
		return ErrUnsupportedGrant
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (*Identity, error) {
	if !validSecret(token, "jbr_oat_") {
		return nil, ErrInvalidGrant
	}
	return s.store.Authenticate(ctx, hashSecret(token), s.Resource())
}

func (s *Service) Revoke(ctx context.Context, token, clientID string) error {
	if len(clientID) > 2048 || len(token) > 256 {
		return nil
	}
	return s.store.RevokeToken(ctx, hashSecret(token), clientID)
}

func (s *Service) ListGrants(ctx context.Context, userID string) ([]Grant, error) {
	return s.store.ListGrants(ctx, userID)
}

func (s *Service) RevokeGrant(ctx context.Context, userID, id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return ErrInvalidRequest
	}
	return s.store.RevokeGrant(ctx, userID, parsed.String())
}
