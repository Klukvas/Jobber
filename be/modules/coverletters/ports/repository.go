package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/coverletters/model"
)

// CoverLetterRepository defines data access for cover letters.
type CoverLetterRepository interface {
	Create(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	GetByID(ctx context.Context, id string) (*model.CoverLetter, error)
	List(ctx context.Context, userID string) ([]*model.CoverLetter, error)
	Update(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	Delete(ctx context.Context, id string) error
}
