package repository

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	query := `
		INSERT INTO comments (id, user_id, application_id, stage_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	comment.ID = uuid.New().String()
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query, comment.ID, comment.UserID, comment.ApplicationID, comment.StageID, comment.Content, comment.CreatedAt, comment.UpdatedAt)
	return err
}

func (r *CommentRepository) ListByApplication(ctx context.Context, appID string) ([]*model.Comment, error) {
	query := `
		SELECT id, user_id, application_id, stage_id, content, created_at, updated_at
		FROM comments WHERE application_id = $1 ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		c := &model.Comment{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.ApplicationID, &c.StageID, &c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *CommentRepository) Delete(ctx context.Context, userID, commentID string) error {
	query := `DELETE FROM comments WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, commentID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrCommentNotFound
	}
	return nil
}
