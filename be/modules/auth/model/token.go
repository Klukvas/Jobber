package model

import (
	"time"
)

// RefreshToken represents a refresh token in the database
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// NewRefreshToken creates a new refresh token
func NewRefreshToken(userID, tokenHash string, expiresAt time.Time) *RefreshToken {
	return &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
}

// IsValid checks if the token is valid
func (t *RefreshToken) IsValid() bool {
	return t.RevokedAt == nil && time.Now().UTC().Before(t.ExpiresAt)
}

// AuthTokens represents access and refresh tokens
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}
