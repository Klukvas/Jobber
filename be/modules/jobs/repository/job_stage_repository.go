package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type JobStageRepository struct {
	pool PgxDB
}

func NewJobStageRepository(pool PgxDB) *JobStageRepository {
	return &JobStageRepository{pool: pool}
}

func (r *JobStageRepository) Create(ctx context.Context, stage *model.JobStage) error {
	query := `
		INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	stage.ID = uuid.New().String()
	stage.CreatedAt = time.Now().UTC()

	_, err := r.pool.Exec(ctx, query,
		stage.ID, stage.JobID, stage.StageTemplateID, stage.Status, stage.Order, stage.StartedAt, stage.CompletedAt, stage.CreatedAt,
	)
	return err
}

func (r *JobStageRepository) GetByID(ctx context.Context, stageID, jobID string) (*model.JobStage, error) {
	query := `
		SELECT id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at
		FROM job_stages WHERE id = $1 AND job_id = $2
	`

	stage := &model.JobStage{}
	err := r.pool.QueryRow(ctx, query, stageID, jobID).Scan(
		&stage.ID, &stage.JobID, &stage.StageTemplateID, &stage.Status, &stage.Order, &stage.StartedAt, &stage.CompletedAt, &stage.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrJobStageNotFound
		}
		return nil, err
	}
	return stage, nil
}

func (r *JobStageRepository) ListByJob(ctx context.Context, jobID string) ([]*model.JobStage, error) {
	query := `
		SELECT id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at
		FROM job_stages WHERE job_id = $1 ORDER BY "order" ASC, created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*model.JobStage
	for rows.Next() {
		stage := &model.JobStage{}
		if err := rows.Scan(&stage.ID, &stage.JobID, &stage.StageTemplateID, &stage.Status, &stage.Order, &stage.StartedAt, &stage.CompletedAt, &stage.CreatedAt); err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, rows.Err()
}

func (r *JobStageRepository) Update(ctx context.Context, stage *model.JobStage) error {
	// Scope by job_id (the ownership FK) so a stage can never be updated
	// outside the job it belongs to, even if a caller's guard is dropped.
	query := `
		UPDATE job_stages SET status = $2, completed_at = $3
		WHERE id = $1 AND job_id = $4
	`

	result, err := r.pool.Exec(ctx, query, stage.ID, stage.Status, stage.CompletedAt, stage.JobID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrJobStageNotFound
	}
	return nil
}

func (r *JobStageRepository) Delete(ctx context.Context, stageID, jobID string) error {
	query := `DELETE FROM job_stages WHERE id = $1 AND job_id = $2`
	result, err := r.pool.Exec(ctx, query, stageID, jobID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrJobStageNotFound
	}
	return nil
}
