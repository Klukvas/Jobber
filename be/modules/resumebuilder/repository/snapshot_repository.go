package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ ports.SnapshotRepository = (*SnapshotRepository)(nil)
var _ ports.SnapshotTransaction = (*snapshotTransaction)(nil)

type SnapshotRepository struct{ *ResumeBuilderRepository }

func NewSnapshotRepository(pool *pgxpool.Pool) *SnapshotRepository {
	return &SnapshotRepository{NewResumeBuilderRepository(pool)}
}

func (r *SnapshotRepository) RunSnapshotTransaction(ctx context.Context, fn func(ports.SnapshotTransaction) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin snapshot transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	repo := &snapshotTransaction{&ResumeBuilderRepository{pool: r.pool, q: tx}}
	if err := fn(repo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit snapshot transaction: %w", err)
	}
	return nil
}

func (r *SnapshotRepository) ListSnapshots(ctx context.Context, userID, resumeID string) ([]*model.ResumeSnapshotDTO, error) {
	// Filter ownership even for an empty list, without disclosing another user's resume.
	if _, err := r.GetByID(ctx, userID, resumeID); err != nil {
		return nil, fmt.Errorf("load snapshot resume: %w", err)
	}
	rows, err := r.q.Query(ctx, `SELECT s.id, s.source, s.created_at
		FROM resume_builder_snapshots s JOIN resume_builders r ON r.id = s.resume_builder_id
		WHERE s.resume_builder_id = $1 AND s.user_id = $2 AND r.user_id = $2
		ORDER BY s.created_at DESC, s.id DESC`, resumeID, userID)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	items := make([]*model.ResumeSnapshotDTO, 0)
	for rows.Next() {
		item := &model.ResumeSnapshotDTO{}
		if err := rows.Scan(&item.ID, &item.Source, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot metadata: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read snapshot metadata: %w", err)
	}
	return items, nil
}

type snapshotTransaction struct{ *ResumeBuilderRepository }

func (r *snapshotTransaction) GetFullResume(ctx context.Context, userID, resumeID string) (*model.FullResumeDTO, error) {
	return r.loadFullResume(ctx, userID, resumeID)
}

func (r *snapshotTransaction) GetSnapshot(ctx context.Context, userID, snapshotID string) (*model.ResumeSnapshot, error) {
	snapshot := &model.ResumeSnapshot{}
	err := r.q.QueryRow(ctx, `SELECT s.id, s.resume_builder_id, s.user_id, s.source, s.content, s.created_at
		FROM resume_builder_snapshots s JOIN resume_builders r ON r.id = s.resume_builder_id
		WHERE s.id = $1 AND s.user_id = $2 AND r.user_id = $2`, snapshotID, userID).
		Scan(&snapshot.ID, &snapshot.ResumeBuilderID, &snapshot.UserID, &snapshot.Source, &snapshot.Content, &snapshot.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrResumeBuilderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}
	return snapshot, nil
}

func (r *snapshotTransaction) HasRecentSnapshot(ctx context.Context, resumeID, source string, since time.Time) (bool, error) {
	var exists bool
	err := r.q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM resume_builder_snapshots
		WHERE resume_builder_id = $1 AND source = $2 AND created_at > $3)`, resumeID, source, since).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check snapshot window: %w", err)
	}
	return exists, nil
}

func (r *snapshotTransaction) InsertSnapshot(ctx context.Context, snapshot *model.ResumeSnapshot) error {
	_, err := r.q.Exec(ctx, `INSERT INTO resume_builder_snapshots
		(id, resume_builder_id, user_id, source, content, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		snapshot.ID, snapshot.ResumeBuilderID, snapshot.UserID, snapshot.Source, snapshot.Content, snapshot.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

func (r *snapshotTransaction) PruneSnapshots(ctx context.Context, resumeID string, keep int) error {
	_, err := r.q.Exec(ctx, `DELETE FROM resume_builder_snapshots WHERE id IN (
		SELECT id FROM resume_builder_snapshots WHERE resume_builder_id = $1
		ORDER BY created_at DESC, id DESC OFFSET $2)`, resumeID, keep)
	if err != nil {
		return fmt.Errorf("prune snapshots: %w", err)
	}
	return nil
}
