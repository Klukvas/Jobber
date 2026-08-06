package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBPool is the subset of *pgxpool.Pool this repository uses; it lets tests
// exercise the real queries through pgxmock.
type DBPool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type SharingRepository struct {
	pool DBPool
}

func NewSharingRepository(pool DBPool) *SharingRepository {
	return &SharingRepository{pool: pool}
}

func (r *SharingRepository) Create(ctx context.Context, share *model.SharedStats) error {
	// Single-statement check-and-insert: the row is only written while the
	// user is under the per-user cap.
	query := `
		INSERT INTO shared_stats (id, user_id, token, snapshot, created_at)
		SELECT $1, $2, $3, $4, $5
		WHERE (SELECT COUNT(*) FROM shared_stats WHERE user_id = $2) < $6
	`
	share.ID = uuid.New().String()
	share.CreatedAt = time.Now().UTC()

	snapshot, err := json.Marshal(share.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal share snapshot: %w", err)
	}

	result, err := r.pool.Exec(ctx, query,
		share.ID, share.UserID, share.Token, snapshot, share.CreatedAt, model.MaxActiveShares)
	if err != nil {
		return fmt.Errorf("insert shared stats: %w", err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrShareLimitReached
	}
	return nil
}

func (r *SharingRepository) GetByToken(ctx context.Context, token string) (*model.SharedStats, error) {
	query := `
		SELECT id, user_id, token, snapshot, created_at
		FROM shared_stats
		WHERE token = $1
	`
	return r.scanShare(r.pool.QueryRow(ctx, query, token))
}

func (r *SharingRepository) ListByUser(ctx context.Context, userID string) ([]*model.SharedStats, error) {
	query := `
		SELECT id, user_id, token, snapshot, created_at
		FROM shared_stats
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list shared stats: %w", err)
	}
	defer rows.Close()

	var shares []*model.SharedStats
	for rows.Next() {
		share, err := r.scanShare(rows)
		if err != nil {
			return nil, err
		}
		shares = append(shares, share)
	}
	return shares, rows.Err()
}

func (r *SharingRepository) Delete(ctx context.Context, userID, shareID string) error {
	query := `DELETE FROM shared_stats WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, shareID, userID)
	if err != nil {
		return fmt.Errorf("delete shared stats: %w", err)
	}
	if result.RowsAffected() == 0 {
		return model.ErrShareNotFound
	}
	return nil
}

func (r *SharingRepository) scanShare(row pgx.Row) (*model.SharedStats, error) {
	share := &model.SharedStats{}
	var snapshot []byte
	if err := row.Scan(&share.ID, &share.UserID, &share.Token, &snapshot, &share.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrShareNotFound
		}
		return nil, fmt.Errorf("scan shared stats: %w", err)
	}
	if err := json.Unmarshal(snapshot, &share.Snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal share snapshot: %w", err)
	}
	return share, nil
}
