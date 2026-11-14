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
// A new order is a move, not a raw write: the column takes that position and
// the others close up behind it, all in one transaction. Writing the number
// as given put "Position 1" (order 0) next to the existing first column, and
// the tie left the moved column second.
//
// Column names come from compile-time literals; every value is a bound
// parameter.
func (r *StageTemplateRepository) Update(ctx context.Context, userID, templateID string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
	if patch.Order == nil {
		return updateTemplateFields(ctx, r.pool, userID, templateID, patch)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin stage move tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	if err := moveTemplate(ctx, tx, userID, templateID, *patch.Order); err != nil {
		return nil, err
	}

	fields := *patch
	fields.Order = nil
	template, err := updateTemplateFields(ctx, tx, userID, templateID, &fields)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit stage move: %w", err)
	}
	return template, nil
}

// moveTemplate renumbers the user's columns 0..n-1 with templateID at
// position. The rows are locked first so two moves cannot interleave.
func moveTemplate(ctx context.Context, tx pgx.Tx, userID, templateID string, position int) error {
	rows, err := tx.Query(ctx,
		`SELECT id FROM stage_templates WHERE user_id = $1 ORDER BY "order" ASC, created_at ASC FOR UPDATE`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("lock stage templates: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("read stage templates: %w", err)
	}

	ordered, ok := moveToPosition(ids, templateID, position)
	if !ok {
		return model.ErrStageTemplateNotFound
	}

	now := time.Now().UTC()
	for i, id := range ordered {
		if _, err := tx.Exec(ctx,
			`UPDATE stage_templates SET "order" = $3, updated_at = $4 WHERE id = $1 AND user_id = $2`,
			id, userID, i, now,
		); err != nil {
			return fmt.Errorf("renumber stage template: %w", err)
		}
	}
	return nil
}

// moveToPosition returns ids with id moved to position, clamped to the list.
// It reports false when id is not in the list.
func moveToPosition(ids []string, id string, position int) ([]string, bool) {
	rest := make([]string, 0, len(ids))
	found := false
	for _, candidate := range ids {
		if candidate == id {
			found = true
			continue
		}
		rest = append(rest, candidate)
	}
	if !found {
		return nil, false
	}
	position = max(0, min(position, len(rest)))
	moved := make([]string, 0, len(ids))
	moved = append(moved, rest[:position]...)
	moved = append(moved, id)
	moved = append(moved, rest[position:]...)
	return moved, true
}

// rowQuerier is what a field update needs: the pool, or the move's transaction.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func updateTemplateFields(ctx context.Context, db rowQuerier, userID, templateID string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
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
	err := db.QueryRow(ctx, query, args...).Scan(
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
