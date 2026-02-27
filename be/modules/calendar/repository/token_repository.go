package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TokenRepository implements ports.CalendarTokenRepository
type TokenRepository struct {
	pool *pgxpool.Pool
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

// Upsert inserts or updates a calendar token for a user
func (r *TokenRepository) Upsert(ctx context.Context, token *model.CalendarToken) error {
	now := time.Now().UTC()
	id := token.ID
	if id == "" {
		id = uuid.New().String()
	}

	query := `
		INSERT INTO google_calendar_tokens (id, user_id, token_blob, token_nonce, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id)
		DO UPDATE SET token_blob = $3, token_nonce = $4, updated_at = $6
	`

	_, err := r.pool.Exec(ctx, query,
		id,
		token.UserID,
		token.TokenBlob,
		token.TokenNonce,
		now,
		now,
	)
	return err
}

// GetByUserID retrieves the calendar token for a user
func (r *TokenRepository) GetByUserID(ctx context.Context, userID string) (*model.CalendarToken, error) {
	query := `
		SELECT id, user_id, token_blob, token_nonce, created_at, updated_at
		FROM google_calendar_tokens
		WHERE user_id = $1
	`

	token := &model.CalendarToken{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenBlob,
		&token.TokenNonce,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotConnected
		}
		return nil, err
	}
	return token, nil
}

// Delete removes the calendar token for a user
func (r *TokenRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM google_calendar_tokens WHERE user_id = $1`
	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrNotConnected
	}
	return nil
}
