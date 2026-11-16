package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	quota "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	"github.com/jackc/pgx/v5"
)

// MatchScoreCacheRepository implements ports.MatchScoreCacheRepository using PostgreSQL.
type MatchScoreCacheRepository struct {
	pool PgxDB
}

// NewMatchScoreCacheRepository creates a new cache repository.
func NewMatchScoreCacheRepository(pool PgxDB) *MatchScoreCacheRepository {
	return &MatchScoreCacheRepository{pool: pool}
}

// Get returns a cached match score result, or nil if not found.
func (r *MatchScoreCacheRepository) Get(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
	query := `SELECT result FROM match_score_cache WHERE user_id = $1 AND job_id = $2 AND resume_id = $3`

	var raw []byte
	err := r.pool.QueryRow(ctx, query, userID, jobID, resumeID).Scan(&raw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var result model.MatchScoreResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Upsert inserts or replaces a cached match score result.
func (r *MatchScoreCacheRepository) Upsert(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO match_score_cache (user_id, job_id, resume_id, result)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, job_id, resume_id)
		DO UPDATE SET result = EXCLUDED.result, cached_at = NOW()
	`

	op, _, found := ai.CurrentCreditOperation(ctx)
	if !found || !ai.CreditRequestAdmitted(ctx) {
		_, err = r.pool.Exec(ctx, query, userID, jobID, resumeID, raw)
		return err
	}
	beginner, ok := r.pool.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("match cache requires transactional credit storage")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin match cache result: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if _, err := tx.Exec(ctx, query, userID, jobID, resumeID, raw); err != nil {
		return fmt.Errorf("save match cache result: %w", err)
	}
	if err := quota.FinalizeCreditResultInTransaction(ctx, tx, op.UserID, op.Key, true, ""); err != nil {
		return fmt.Errorf("finalize match cache charge: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit match cache result: %w", err)
	}
	ai.MarkCreditResult(ctx)
	ai.SetDurableCreditContext(ctx)
	return nil
}

// InvalidateByJob deletes all cached results for a given job.
func (r *MatchScoreCacheRepository) InvalidateByJob(ctx context.Context, jobID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM match_score_cache WHERE job_id = $1`, jobID)
	return err
}

// InvalidateByResume deletes all cached results for a given resume.
func (r *MatchScoreCacheRepository) InvalidateByResume(ctx context.Context, resumeID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM match_score_cache WHERE resume_id = $1`, resumeID)
	return err
}
