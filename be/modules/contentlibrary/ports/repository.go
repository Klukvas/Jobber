package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
)

// ContentLibraryRepository defines data access for content library.
type ContentLibraryRepository interface {
	Create(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error)
	GetByID(ctx context.Context, id string) (*model.ContentLibraryEntry, error)
	List(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error)
	Update(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error)
	Delete(ctx context.Context, id string) error
}
