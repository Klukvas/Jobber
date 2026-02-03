package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/comments/model"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	ListByApplication(ctx context.Context, appID string, userID ...string) ([]*model.Comment, error)
	Delete(ctx context.Context, userID, commentID string) error
}
