package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/auth/model"
)

// RefreshTokenRepository defines the interface for refresh token data access
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	// RevokeIfValid atomically revokes a token only if it is not already revoked.
	// Returns true if the token was revoked by this call, false if already revoked/expired.
	RevokeIfValid(ctx context.Context, tokenHash string) (bool, error)
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}
