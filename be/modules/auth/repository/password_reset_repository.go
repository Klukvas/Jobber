package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/auth/model"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PasswordResetRepository implements ports.PasswordResetRepository.
type PasswordResetRepository struct {
	pool *pgxpool.Pool
}

// NewPasswordResetRepository creates a new password reset repository.
func NewPasswordResetRepository(pool *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{pool: pool}
}

// Create stores a new password reset token.
func (r *PasswordResetRepository) Create(ctx context.Context, token *model.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, code, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	token.ID = uuid.New().String()

	_, err := r.pool.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.Code,
		token.ExpiresAt,
		token.CreatedAt,
	)
	return err
}

// GetActiveForUser retrieves the most recent active (unused, non-expired) password reset token for a user.
func (r *PasswordResetRepository) GetActiveForUser(ctx context.Context, userID string) (*model.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, code, attempts, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	token := &model.PasswordResetToken{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&token.ID,
		&token.UserID,
		&token.Code,
		&token.Attempts,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, userModel.ErrInvalidResetToken
		}
		return nil, err
	}

	return token, nil
}

// IncrementAttempts atomically increments the attempts counter for a token.
// Returns the new attempts count. If attempts >= maxAttempts, returns ErrTooManyAttempts without incrementing.
func (r *PasswordResetRepository) IncrementAttempts(ctx context.Context, id string, maxAttempts int) (int, error) {
	query := `UPDATE password_reset_tokens SET attempts = attempts + 1 WHERE id = $1 AND attempts < $2 RETURNING attempts`
	var newAttempts int
	err := r.pool.QueryRow(ctx, query, id, maxAttempts).Scan(&newAttempts)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, userModel.ErrTooManyAttempts
		}
		return 0, err
	}
	return newAttempts, nil
}

// MarkUsed marks a password reset token as used.
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	query := `UPDATE password_reset_tokens SET used_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now().UTC())
	return err
}

// DeleteForUser removes all unused password reset tokens for a user.
func (r *PasswordResetRepository) DeleteForUser(ctx context.Context, userID string) error {
	query := `DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// DeleteExpired removes expired password reset tokens.
func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < $1`
	_, err := r.pool.Exec(ctx, query, time.Now().UTC())
	return err
}
