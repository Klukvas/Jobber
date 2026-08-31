package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/coverletters/model"
)

// CoverLetterRepository defines data access for cover letters.
type CoverLetterRepository interface {
	Create(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	VerifyOwnership(ctx context.Context, userID, id string) error
	GetByID(ctx context.Context, userID, id string) (*model.CoverLetter, error)
	List(ctx context.Context, userID string) ([]*model.CoverLetter, error)
	Update(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	Delete(ctx context.Context, userID, id string) error
}
