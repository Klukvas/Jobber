package service

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/andreypavlenko/jobber/modules/users/ports"
)

// UserService reads the signed-in person's own account.
type UserService struct {
	repo ports.UserRepository
}

// NewUserService creates a new user service.
func NewUserService(repo ports.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// GetProfile returns the caller's own profile.
func (s *UserService) GetProfile(ctx context.Context, userID string) (*model.UserDTO, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		// Wrapped so a storage failure carries the id it was looking up.
		// %w keeps ErrUserNotFound matchable, which is what the handler maps
		// onto 404 rather than 500.
		return nil, fmt.Errorf("get profile for user %s: %w", userID, err)
	}
	return user.ToDTO(), nil
}
