package repository

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/google/uuid"
)

type CommentRepository struct {
	pool PgxDB
}

func NewCommentRepository(pool PgxDB) *CommentRepository {
	return &CommentRepository{pool: pool}
}

func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	// The row is inserted only if the target job belongs to the comment's author
	// and (when a stage is given) the stage belongs to that job. This closes an
	// IDOR: a caller cannot anchor a comment on another user's job by supplying
	// its UUID in the request body.
	query := `
		INSERT INTO comments (id, user_id, job_id, stage_id, content, created_at, updated_at)
		SELECT $1, $2, $3, $4, $5, $6, $7
		WHERE EXISTS (SELECT 1 FROM jobs WHERE id = $3 AND user_id = $2)
		  AND ($4::uuid IS NULL OR EXISTS (SELECT 1 FROM job_stages WHERE id = $4 AND job_id = $3))
	`
	comment.ID = uuid.New().String()
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	result, err := r.pool.Exec(ctx, query, comment.ID, comment.UserID, comment.JobID, comment.StageID, comment.Content, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrJobNotFound
	}
	return nil
}

func (r *CommentRepository) ListByJob(ctx context.Context, jobID, userID string) ([]*model.Comment, error) {
	// userID is required: comments are always scoped to the job's owner so a
	// caller can never accidentally read another user's comments.
	query := `
		SELECT c.id, c.user_id, c.job_id, c.stage_id, c.content, c.created_at, c.updated_at
		FROM comments c
		JOIN jobs j ON c.job_id = j.id AND j.user_id = $1
		WHERE c.job_id = $2 ORDER BY c.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, userID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		c := &model.Comment{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.JobID, &c.StageID, &c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
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
