package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/comments/model"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	ListByJob(ctx context.Context, jobID, userID string) ([]*model.Comment, error)
	// Update rewrites the body of a comment the user authored. Ownership is
	// part of the write itself, not a separate check.
	Update(ctx context.Context, userID, commentID, content string) (*model.Comment, error)
	Delete(ctx context.Context, userID, commentID string) error
}
