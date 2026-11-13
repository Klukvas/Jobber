package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/jackc/pgx/v5"
)

const tokenColumns = `id, user_id, name, token_prefix, token_hash, scopes,
 last_used_at, expires_at, revoked_at, created_at`

type APITokenRepository struct{ pool PgxDB }

func NewAPITokenRepository(pool PgxDB) *APITokenRepository {
	return &APITokenRepository{pool: pool}
}

func (r *APITokenRepository) Create(ctx context.Context, token *model.APIToken) (createdToken *model.APIToken, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin api token creation: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback api token creation: %w", rollbackErr))
		}
	}()
	if err := checkActiveLimit(ctx, tx, token.UserID); err != nil {
		return nil, err
	}
	created := *token
	err = tx.QueryRow(ctx, `INSERT INTO api_tokens
 (id, user_id, name, token_prefix, token_hash, scopes, expires_at)
 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`,
		token.ID, token.UserID, token.Name, token.TokenPrefix, token.TokenHash,
		token.Scopes, token.ExpiresAt).Scan(&created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create api token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit api token creation: %w", err)
	}
	committed = true
	return &created, nil
}

func checkActiveLimit(ctx context.Context, tx pgx.Tx, userID string) error {
	// Lock a stable parent row so even concurrent first-token requests serialize.
	var id string
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id); err != nil {
		return fmt.Errorf("lock api token owner: %w", err)
	}
	var count int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM api_tokens WHERE user_id = $1
 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now())`, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count active api tokens: %w", err)
	}
	if count >= model.MaxActiveTokens {
		return model.ErrTokenLimit
	}
	return nil
}

func (r *APITokenRepository) List(ctx context.Context, userID string) ([]*model.APIToken, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+tokenColumns+` FROM api_tokens
 WHERE user_id = $1 ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	defer rows.Close()
	tokens := make([]*model.APIToken, 0)
	for rows.Next() {
		token, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api tokens: %w", err)
	}
	return tokens, nil
}

func (r *APITokenRepository) GetByHash(ctx context.Context, hash string) (*model.APIToken, error) {
	return scanToken(r.pool.QueryRow(ctx, `SELECT `+tokenColumns+` FROM api_tokens WHERE token_hash = $1`, hash))
}

func scanToken(row pgx.Row) (*model.APIToken, error) {
	var token model.APIToken
	err := row.Scan(&token.ID, &token.UserID, &token.Name, &token.TokenPrefix,
		&token.TokenHash, &token.Scopes, &token.LastUsedAt, &token.ExpiresAt,
		&token.RevokedAt, &token.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan api token: %w", err)
	}
	return &token, nil
}

func (r *APITokenRepository) Revoke(ctx context.Context, userID, tokenID string) error {
	result, err := r.pool.Exec(ctx, `UPDATE api_tokens SET revoked_at = COALESCE(revoked_at, now())
 WHERE id = $1 AND user_id = $2`, tokenID, userID)
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrTokenNotFound
	}
	return nil
}

func (r *APITokenRepository) TouchLastUsed(ctx context.Context, tokenID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_tokens SET last_used_at = now() WHERE id = $1
 AND (last_used_at IS NULL OR last_used_at <= now() - interval '1 minute')`, tokenID)
	if err != nil {
		return fmt.Errorf("update api token usage: %w", err)
	}
	return nil
}
