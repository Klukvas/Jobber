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

type StageTemplateRepository struct {
	pool *pgxpool.Pool
}

func NewStageTemplateRepository(pool *pgxpool.Pool) *StageTemplateRepository {
	return &StageTemplateRepository{pool: pool}
}

func (r *StageTemplateRepository) Create(ctx context.Context, template *model.StageTemplate) error {
	query := `
		INSERT INTO stage_templates (id, user_id, name, "order", created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	template.ID = uuid.New().String()
	now := time.Now().UTC()
	template.CreatedAt = now
	template.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query, template.ID, template.UserID, template.Name, template.Order, template.CreatedAt, template.UpdatedAt)
	return err
}

func (r *StageTemplateRepository) GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
	query := `
		SELECT id, user_id, name, "order", created_at, updated_at
		FROM stage_templates WHERE id = $1 AND user_id = $2
	`

	template := &model.StageTemplate{}
	err := r.pool.QueryRow(ctx, query, templateID, userID).Scan(
		&template.ID, &template.UserID, &template.Name, &template.Order, &template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrStageTemplateNotFound
		}
		return nil, err
	}
	return template, nil
}

func (r *StageTemplateRepository) List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM stage_templates WHERE user_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, user_id, name, "order", created_at, updated_at
		FROM stage_templates WHERE user_id = $1 ORDER BY "order" ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var templates []*model.StageTemplate
	for rows.Next() {
		template := &model.StageTemplate{}
		if err := rows.Scan(&template.ID, &template.UserID, &template.Name, &template.Order, &template.CreatedAt, &template.UpdatedAt); err != nil {
			return nil, 0, err
		}
		templates = append(templates, template)
	}
	return templates, total, rows.Err()
}

func (r *StageTemplateRepository) Update(ctx context.Context, template *model.StageTemplate) error {
	query := `
		UPDATE stage_templates SET name = $3, "order" = $4, updated_at = $5
		WHERE id = $1 AND user_id = $2
	`

	template.UpdatedAt = time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, template.ID, template.UserID, template.Name, template.Order, template.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrStageTemplateNotFound
	}
	return nil
}

func (r *StageTemplateRepository) Delete(ctx context.Context, userID, templateID string) error {
	query := `DELETE FROM stage_templates WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, templateID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrStageTemplateNotFound
	}
	return nil
}
