package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const MaxAuditRowsPerUser = 500

type auditDB interface {
	Begin(context.Context) (pgx.Tx, error)
}

type AuditRepository struct{ pool auditDB }

func NewAuditRepository(pool auditDB) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Record(ctx context.Context, entry AuditEntry) (err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mcp audit: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback mcp audit: %w", rollbackErr))
		}
	}()
	// Serialize each user's insert/prune, including their first call. Separate
	// statements under READ COMMITTED ensure pruning sees prior concurrent writes.
	var userID string
	if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR NO KEY UPDATE`, entry.UserID).Scan(&userID); err != nil {
		return fmt.Errorf("lock mcp audit owner: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO mcp_audit_log
 (id, user_id, api_token_id, tool, ok, error_code, duration_ms)
 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.ID, entry.UserID, entry.APITokenID, entry.Tool, entry.OK, entry.ErrorCode, entry.DurationMS)
	if err != nil {
		return fmt.Errorf("insert mcp audit: %w", err)
	}
	_, err = tx.Exec(ctx, `DELETE FROM mcp_audit_log WHERE user_id = $1 AND id IN
 (SELECT id FROM mcp_audit_log WHERE user_id = $1 ORDER BY created_at DESC, id DESC OFFSET $2)`,
		entry.UserID, MaxAuditRowsPerUser)
	if err != nil {
		return fmt.Errorf("prune mcp audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit mcp audit: %w", err)
	}
	committed = true
	return nil
}
