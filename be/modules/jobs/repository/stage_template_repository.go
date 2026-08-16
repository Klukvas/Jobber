package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type StageTemplateRepository struct {
	pool PgxDB
}

func NewStageTemplateRepository(pool PgxDB) *StageTemplateRepository {
	return &StageTemplateRepository{pool: pool}
}

func (r *StageTemplateRepository) Create(ctx context.Context, template *model.StageTemplate) error {
	// phase column (kept transiently in DB with a default) is intentionally not written.
	query := `
		INSERT INTO stage_templates (id, user_id, name, "order", created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	template.ID = uuid.New().String()
	now := time.Now().UTC()
	template.CreatedAt = now
	template.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query, template.ID, template.UserID, template.Name, template.Order, template.CreatedAt, template.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.ErrStageTemplateNameExists
		}
		return err
	}
	return nil
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

	// The user's pipeline, in column order.
	query := `
		SELECT id, user_id, name, "order", created_at, updated_at
		FROM stage_templates WHERE user_id = $1
		ORDER BY "order" ASC, created_at ASC
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return model.ErrStageTemplateNameExists
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrStageTemplateNotFound
	}
	return nil
}

// Reorder sets a new column order for the user's pipeline. orderedIDs must be
// exactly the user's stage template IDs; each is assigned order = its index.
func (r *StageTemplateRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reorder tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Guard: the list must match the user's stages exactly (count check).
	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM stage_templates WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if total != len(orderedIDs) {
		return model.ErrReorderMismatch
	}

	now := time.Now().UTC()
	for i, id := range orderedIDs {
		ct, err := tx.Exec(ctx,
			`UPDATE stage_templates SET "order" = $3, updated_at = $4 WHERE id = $1 AND user_id = $2`,
			id, userID, i, now,
		)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return model.ErrReorderMismatch // an id that isn't the user's
		}
	}
	return tx.Commit(ctx)
}

func (r *StageTemplateRepository) Delete(ctx context.Context, userID, templateID string) error {
	query := `DELETE FROM stage_templates WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, templateID, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return model.ErrStageTemplateInUse
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrStageTemplateNotFound
	}
	return nil
}
