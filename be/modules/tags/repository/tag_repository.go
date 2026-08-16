package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

type TagRepository struct {
	pool PgxDB
}

func NewTagRepository(pool PgxDB) *TagRepository {
	return &TagRepository{pool: pool}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}

func (r *TagRepository) Create(ctx context.Context, tag *model.Tag) error {
	query := `INSERT INTO tags (id, user_id, name, color, created_at) VALUES ($1, $2, $3, $4, $5)`
	tag.ID = uuid.New().String()
	tag.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, tag.ID, tag.UserID, tag.Name, tag.Color, tag.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return model.ErrTagNameExists
		}
		return err
	}
	return nil
}

func (r *TagRepository) List(ctx context.Context, userID string) ([]*model.Tag, error) {
	query := `SELECT id, user_id, name, color, created_at FROM tags WHERE user_id = $1 ORDER BY name ASC`
	return r.queryTags(ctx, query, userID)
}

func (r *TagRepository) Delete(ctx context.Context, userID, tagID string) error {
	query := `DELETE FROM tags WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, tagID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrTagNotFound
	}
	return nil
}

// AddRelation attaches a tag to an entity only when BOTH the tag and the target
// entity (job or company) belong to the user — the ownership checks run inside
// the INSERT so a forged tag_id or entity_id can never cross users. Attaching an
// already-attached tag is idempotent (unique violation is swallowed). Zero rows
// affected means the tag or entity was not found for this user.
func (r *TagRepository) AddRelation(ctx context.Context, userID string, rel *model.TagRelation) error {
	// $3 (entity_type) is cast to text everywhere so Postgres deduces a single,
	// consistent type for the parameter — the bare column-insert position would
	// otherwise deduce varchar while the "= 'job'" comparisons deduce text,
	// which fails with "inconsistent types deduced for parameter $3" (42P08).
	query := `
		INSERT INTO tag_relations (id, tag_id, entity_type, entity_id, created_at)
		SELECT $1, $2, $3::text, $4, $5
		WHERE EXISTS (SELECT 1 FROM tags WHERE id = $2 AND user_id = $6)
		  AND (
		    ($3::text = 'job' AND EXISTS (SELECT 1 FROM jobs WHERE id = $4 AND user_id = $6))
		    OR ($3::text = 'company' AND EXISTS (SELECT 1 FROM companies WHERE id = $4 AND user_id = $6))
		  )
	`
	rel.ID = uuid.New().String()
	rel.CreatedAt = time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, rel.ID, rel.TagID, rel.EntityType, rel.EntityID, rel.CreatedAt, userID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil // already attached — idempotent
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrEntityNotFound
	}
	return nil
}

// RemoveRelation detaches a tag from an entity, scoped so a user can only touch
// relations of tags they own.
func (r *TagRepository) RemoveRelation(ctx context.Context, userID, tagID, entityType, entityID string) error {
	query := `
		DELETE FROM tag_relations tr
		USING tags t
		WHERE tr.tag_id = t.id
		  AND t.user_id = $1
		  AND tr.tag_id = $2
		  AND tr.entity_type = $3
		  AND tr.entity_id = $4
	`
	_, err := r.pool.Exec(ctx, query, userID, tagID, entityType, entityID)
	return err
}

// ListByEntity returns the user's tags attached to an entity. Scoped by
// t.user_id so it never leaks another user's tags.
func (r *TagRepository) ListByEntity(ctx context.Context, userID, entityType, entityID string) ([]*model.Tag, error) {
	query := `
		SELECT t.id, t.user_id, t.name, t.color, t.created_at
		FROM tags t
		INNER JOIN tag_relations tr ON t.id = tr.tag_id
		WHERE tr.entity_type = $1 AND tr.entity_id = $2 AND t.user_id = $3
		ORDER BY t.name ASC
	`
	return r.queryTags(ctx, query, entityType, entityID, userID)
}

func (r *TagRepository) queryTags(ctx context.Context, query string, args ...any) ([]*model.Tag, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]*model.Tag, 0)
	for rows.Next() {
		t := &model.Tag{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
