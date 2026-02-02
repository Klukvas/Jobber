package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplicationStageRepository struct {
	pool *pgxpool.Pool
}

func NewApplicationStageRepository(pool *pgxpool.Pool) *ApplicationStageRepository {
	return &ApplicationStageRepository{pool: pool}
}

func (r *ApplicationStageRepository) Create(ctx context.Context, stage *model.ApplicationStage) error {
	query := `
		INSERT INTO application_stages (id, application_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	stage.ID = uuid.New().String()
	stage.CreatedAt = time.Now().UTC()

	_, err := r.pool.Exec(ctx, query,
		stage.ID, stage.ApplicationID, stage.StageTemplateID, stage.Status, stage.Order, stage.StartedAt, stage.CompletedAt, stage.CreatedAt,
	)
	return err
}

func (r *ApplicationStageRepository) GetByID(ctx context.Context, stageID string) (*model.ApplicationStage, error) {
	query := `
		SELECT id, application_id, stage_template_id, status, "order", started_at, completed_at, created_at
		FROM application_stages WHERE id = $1
	`

	stage := &model.ApplicationStage{}
	err := r.pool.QueryRow(ctx, query, stageID).Scan(
		&stage.ID, &stage.ApplicationID, &stage.StageTemplateID, &stage.Status, &stage.Order, &stage.StartedAt, &stage.CompletedAt, &stage.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrApplicationStageNotFound
		}
		return nil, err
	}
	return stage, nil
}

func (r *ApplicationStageRepository) ListByApplication(ctx context.Context, appID string) ([]*model.ApplicationStage, error) {
	query := `
		SELECT id, application_id, stage_template_id, status, "order", started_at, completed_at, created_at
		FROM application_stages WHERE application_id = $1 ORDER BY "order" ASC, created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*model.ApplicationStage
	for rows.Next() {
		stage := &model.ApplicationStage{}
		if err := rows.Scan(&stage.ID, &stage.ApplicationID, &stage.StageTemplateID, &stage.Status, &stage.Order, &stage.StartedAt, &stage.CompletedAt, &stage.CreatedAt); err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, rows.Err()
}

func (r *ApplicationStageRepository) Update(ctx context.Context, stage *model.ApplicationStage) error {
	query := `
		UPDATE application_stages SET status = $2, completed_at = $3
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, stage.ID, stage.Status, stage.CompletedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrApplicationStageNotFound
	}
	return nil
}

func (r *ApplicationStageRepository) Delete(ctx context.Context, stageID string) error {
	query := `DELETE FROM application_stages WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, stageID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrApplicationStageNotFound
	}
	return nil
}
