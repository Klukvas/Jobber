package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/tags/model"
)

// TagRepository defines the data-access contract for tags and their relations.
type TagRepository interface {
	Create(ctx context.Context, tag *model.Tag) error
	List(ctx context.Context, userID string) ([]*model.Tag, error)
	Delete(ctx context.Context, userID, tagID string) error
	AddRelation(ctx context.Context, userID string, rel *model.TagRelation) error
	RemoveRelation(ctx context.Context, userID, tagID, entityType, entityID string) error
	ListByEntity(ctx context.Context, userID, entityType, entityID string) ([]*model.Tag, error)
}
