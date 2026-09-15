package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUserRepository implements ports.UserRepository.
type MockUserRepository struct {
	GetByIDFunc func(ctx context.Context, userID string) (*model.User, error)
	UpdateFunc  func(ctx context.Context, user *model.User) error
}

func (m *MockUserRepository) Create(context.Context, *model.User) error { return nil }

func (m *MockUserRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(context.Context, string) (*model.User, error) {
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) Delete(context.Context, string) error { return nil }

func (m *MockUserRepository) SetEmailVerified(context.Context, string) error { return nil }

func (m *MockUserRepository) UpdatePasswordHash(context.Context, string, string) error {
	return nil
}

func storedUser() *model.User {
	return &model.User{
		ID:            "user-123",
		Email:         "alex@example.com",
		Name:          "Alex Jobseeker",
		PasswordHash:  "hash",
		Locale:        "en",
		EmailVerified: true,
	}
}

func TestUserService_GetProfile(t *testing.T) {
	t.Run("returns the caller's own profile", func(t *testing.T) {
		svc := NewUserService(&MockUserRepository{
			GetByIDFunc: func(context.Context, string) (*model.User, error) { return storedUser(), nil },
		})

		dto, err := svc.GetProfile(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Equal(t, "Alex Jobseeker", dto.Name)
		assert.Equal(t, "alex@example.com", dto.Email)
	})

	t.Run("propagates a missing user", func(t *testing.T) {
		svc := NewUserService(&MockUserRepository{
			GetByIDFunc: func(context.Context, string) (*model.User, error) {
				return nil, model.ErrUserNotFound
			},
		})

		_, err := svc.GetProfile(context.Background(), "gone")

		assert.ErrorIs(t, err, model.ErrUserNotFound)
	})

	// Wrapping must add the id without breaking the sentinel the handler maps
	// onto 404.
	t.Run("wraps a read failure with the user id", func(t *testing.T) {
		boom := errors.New("boom")
		svc := NewUserService(&MockUserRepository{
			GetByIDFunc: func(context.Context, string) (*model.User, error) { return nil, boom },
		})

		_, err := svc.GetProfile(context.Background(), "user-123")

		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
		assert.Contains(t, err.Error(), "user-123")
	})
}
