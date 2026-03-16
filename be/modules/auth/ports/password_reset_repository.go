package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/auth/model"
)

// PasswordResetRepository defines the interface for password reset token data access.
type PasswordResetRepository interface {
	Create(ctx context.Context, token *model.PasswordResetToken) error
	GetActiveForUser(ctx context.Context, userID string) (*model.PasswordResetToken, error)
	// IncrementAttempts atomically increments the attempts counter.
	// Returns the new attempts count. If attempts >= maxAttempts, returns ErrTooManyAttempts without incrementing.
	IncrementAttempts(ctx context.Context, id string, maxAttempts int) (int, error)
	MarkUsed(ctx context.Context, id string) error
	DeleteForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}
