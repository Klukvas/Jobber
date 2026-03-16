package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/users/model"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, userID string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, userID string) error
	SetEmailVerified(ctx context.Context, userID string) error
	UpdatePasswordHash(ctx context.Context, userID, hash string) error
}
