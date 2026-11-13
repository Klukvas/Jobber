package model

import (
	"errors"
	"slices"
	"time"
)

const MaxActiveTokens = 10

var (
	ErrTokenNotFound = errors.New("api token not found")
	ErrInvalidToken  = errors.New("invalid api token")
	ErrTokenLimit    = errors.New("active api token limit reached")
	ErrInvalidName   = errors.New("token name must contain 1 to 100 characters")
	ErrInvalidExpiry = errors.New("expiry must be 30, 90, 365 days or null")
	ErrInvalidID     = errors.New("invalid token id")
)

// APIToken is the persistence model. It must never be serialized with its hash.
type APIToken struct {
	ID          string     `json:"-"`
	UserID      string     `json:"-"`
	Name        string     `json:"-"`
	TokenPrefix string     `json:"-"`
	TokenHash   string     `json:"-"`
	Scopes      []string   `json:"-"`
	LastUsedAt  *time.Time `json:"-"`
	ExpiresAt   *time.Time `json:"-"`
	RevokedAt   *time.Time `json:"-"`
	CreatedAt   time.Time  `json:"-"`
}

type APITokenDTO struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	TokenPrefix string     `json:"token_prefix"`
	Scopes      []string   `json:"scopes"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (t *APIToken) ToDTO() *APITokenDTO {
	return &APITokenDTO{
		ID: t.ID, Name: t.Name, TokenPrefix: t.TokenPrefix, Scopes: slices.Clone(t.Scopes),
		LastUsedAt: cloneTime(t.LastUsedAt), ExpiresAt: cloneTime(t.ExpiresAt),
		RevokedAt: cloneTime(t.RevokedAt), CreatedAt: t.CreatedAt,
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
