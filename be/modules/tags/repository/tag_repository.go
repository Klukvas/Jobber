package repository

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

func (r *TagRepository) Create(ctx context.Context, tag *model.Tag) error {
	query := `INSERT INTO tags (id, user_id, name, color, created_at) VALUES ($1, $2, $3, $4, $5)`
	tag.ID = uuid.New().String()
	tag.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, tag.ID, tag.UserID, tag.Name, tag.Color, tag.CreatedAt)
	return err
}

func (r *TagRepository) List(ctx context.Context, userID string) ([]*model.Tag, error) {
	query := `SELECT id, user_id, name, color, created_at FROM tags WHERE user_id = $1 ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		t := &model.Tag{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
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

func (r *TagRepository) AddRelation(ctx context.Context, rel *model.TagRelation) error {
	query := `INSERT INTO tag_relations (id, tag_id, entity_type, entity_id, created_at) VALUES ($1, $2, $3, $4, $5)`
	rel.ID = uuid.New().String()
	rel.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query, rel.ID, rel.TagID, rel.EntityType, rel.EntityID, rel.CreatedAt)
	return err
}

func (r *TagRepository) RemoveRelation(ctx context.Context, tagID, entityID string) error {
	query := `DELETE FROM tag_relations WHERE tag_id = $1 AND entity_id = $2`
	_, err := r.pool.Exec(ctx, query, tagID, entityID)
	return err
}

func (r *TagRepository) ListByEntity(ctx context.Context, entityType, entityID string) ([]*model.Tag, error) {
	query := `
		SELECT t.id, t.user_id, t.name, t.color, t.created_at
		FROM tags t
		INNER JOIN tag_relations tr ON t.id = tr.tag_id
		WHERE tr.entity_type = $1 AND tr.entity_id = $2
		ORDER BY t.name ASC
	`
	rows, err := r.pool.Query(ctx, query, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		t := &model.Tag{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
