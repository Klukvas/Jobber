package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	query := `
		INSERT INTO stage_templates (id, user_id, name, "order", kind, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	template.ID = uuid.New().String()
	if !template.Kind.IsValid() {
		template.Kind = model.StageKindOther
	}
	now := time.Now().UTC()
	template.CreatedAt = now
	template.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query, template.ID, template.UserID, template.Name, template.Order, string(template.Kind), template.CreatedAt, template.UpdatedAt)
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
		SELECT id, user_id, name, "order", kind, created_at, updated_at
		FROM stage_templates WHERE id = $1 AND user_id = $2
	`

	template := &model.StageTemplate{}
	// kind is scanned as a plain string and converted, so the model keeps its
	// own enum type without depending on driver-side type mapping.
	var kind string
	err := r.pool.QueryRow(ctx, query, templateID, userID).Scan(
		&template.ID, &template.UserID, &template.Name, &template.Order, &kind, &template.CreatedAt, &template.UpdatedAt,
	)
	template.Kind = model.StageKind(kind)

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
		SELECT id, user_id, name, "order", kind, created_at, updated_at
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
		var kind string
		if err := rows.Scan(&template.ID, &template.UserID, &template.Name, &template.Order, &kind, &template.CreatedAt, &template.UpdatedAt); err != nil {
			return nil, 0, err
		}
		template.Kind = model.StageKind(kind)
		templates = append(templates, template)
	}
	return templates, total, rows.Err()
}

// Update writes exactly the fields the patch carries and returns the resulting
// row.
//
// Field-scoped on purpose. The old whole-row UPDATE wrote name, "order" and
// kind from a copy read moments earlier, so a rename and a reclassification
// that overlapped lost one of the two: whichever statement committed second
// carried the other's pre-edit value back over the column. Name, order and kind
// are three independent things — the user's label, the user's position and the
// meaning every analytics rule reads — and each is written only when asked for.
//
// Column names come from compile-time literals; every value is a bound
// parameter.
func (r *StageTemplateRepository) Update(ctx context.Context, userID, templateID string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
	// $1/$2 scope the row to its owner; updated_at is always written, so the
	// SET list is never empty.
	args := []any{templateID, userID, time.Now().UTC()}
	sets := []string{"updated_at = $3"}

	set := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if patch.Name != nil {
		set("name", *patch.Name)
	}
	if patch.Order != nil {
		set(`"order"`, *patch.Order)
	}
	if patch.Kind != nil {
		// Same guard as Create: an unrecognized kind is stored as the
		// unclassified default rather than as a value no rule matches.
		kind := *patch.Kind
		if !kind.IsValid() {
			kind = model.StageKindOther
		}
		set("kind", string(kind))
	}

	query := fmt.Sprintf(`
		UPDATE stage_templates SET %s
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, "order", kind, created_at, updated_at
	`, strings.Join(sets, ", "))

	template := &model.StageTemplate{}
	var kind string
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&template.ID, &template.UserID, &template.Name, &template.Order, &kind, &template.CreatedAt, &template.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, model.ErrStageTemplateNameExists
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrStageTemplateNotFound
		}
		return nil, err
	}
	template.Kind = model.StageKind(kind)
	return template, nil
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
