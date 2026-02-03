package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUserRepository implements userPorts.UserRepository
type MockUserRepository struct {
	CreateFunc     func(ctx context.Context, user *userModel.User) error
	GetByIDFunc    func(ctx context.Context, userID string) (*userModel.User, error)
	GetByEmailFunc func(ctx context.Context, email string) (*userModel.User, error)
	UpdateFunc     func(ctx context.Context, user *userModel.User) error
	DeleteFunc     func(ctx context.Context, userID string) error
}

func (m *MockUserRepository) Create(ctx context.Context, user *userModel.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, userID string) (*userModel.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*userModel.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *userModel.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, userID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID)
	}
	return nil
}

// MockRefreshTokenRepository implements authPorts.RefreshTokenRepository
type MockRefreshTokenRepository struct {
	CreateFunc           func(ctx context.Context, token *authModel.RefreshToken) error
	GetByTokenHashFunc   func(ctx context.Context, tokenHash string) (*authModel.RefreshToken, error)
	RevokeFunc           func(ctx context.Context, tokenHash string) error
	RevokeAllForUserFunc func(ctx context.Context, userID string) error
	DeleteExpiredFunc    func(ctx context.Context) error
}

func (m *MockRefreshTokenRepository) Create(ctx context.Context, token *authModel.RefreshToken) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token)
	}
	return nil
}

func (m *MockRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*authModel.RefreshToken, error) {
	if m.GetByTokenHashFunc != nil {
		return m.GetByTokenHashFunc(ctx, tokenHash)
	}
	return nil, nil
}

func (m *MockRefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	if m.RevokeFunc != nil {
		return m.RevokeFunc(ctx, tokenHash)
	}
	return nil
}

func (m *MockRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	if m.RevokeAllForUserFunc != nil {
		return m.RevokeAllForUserFunc(ctx, userID)
	}
	return nil
}

func (m *MockRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	if m.DeleteExpiredFunc != nil {
		return m.DeleteExpiredFunc(ctx)
	}
	return nil
}

func createTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager(
		"test-access-secret-key-32chars!!",
		"test-refresh-secret-key-32chars!",
		15*time.Minute,
		7*24*time.Hour,
	)
}

func TestAuthService_Register(t *testing.T) {
	t.Run("successfully registers a new user", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
			CreateFunc: func(ctx context.Context, user *userModel.User) error {
				user.ID = "user-123"
				return nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			Locale:   "en",
		}

		user, tokens, err := svc.Register(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, tokens)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("returns error for invalid email", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.RegisterRequest{
			Email:    "invalid-email",
			Password: "password123",
		}

		user, tokens, err := svc.Register(context.Background(), req)

		assert.Nil(t, user)
		assert.Nil(t, tokens)
		assert.Equal(t, userModel.ErrInvalidEmail, err)
	})

	t.Run("returns error for short password", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "short",
		}

		user, tokens, err := svc.Register(context.Background(), req)

		assert.Nil(t, user)
		assert.Nil(t, tokens)
		assert.Equal(t, userModel.ErrInvalidPassword, err)
	})

	t.Run("returns error if user already exists", func(t *testing.T) {
		existingUser := &userModel.User{
			ID:    "existing-user",
			Email: "test@example.com",
		}

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return existingUser, nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		user, tokens, err := svc.Register(context.Background(), req)

		assert.Nil(t, user)
		assert.Nil(t, tokens)
		assert.Equal(t, userModel.ErrUserAlreadyExists, err)
	})

	t.Run("uses default locale if not provided", func(t *testing.T) {
		var createdUser *userModel.User

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
			CreateFunc: func(ctx context.Context, user *userModel.User) error {
				createdUser = user
				user.ID = "user-123"
				return nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			Locale:   "", // Empty locale
		}

		_, _, err := svc.Register(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "en", createdUser.Locale)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("successfully logs in with valid credentials", func(t *testing.T) {
		passwordHash, _ := auth.HashPassword("password123")
		existingUser := &userModel.User{
			ID:           "user-123",
			Email:        "test@example.com",
			Name:         "Test User",
			PasswordHash: passwordHash,
			Locale:       "en",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return existingUser, nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		user, tokens, err := svc.Login(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, tokens)
		assert.Equal(t, "user-123", user.ID)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		user, tokens, err := svc.Login(context.Background(), req)

		assert.Nil(t, user)
		assert.Nil(t, tokens)
		assert.Equal(t, userModel.ErrInvalidCredentials, err)
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		passwordHash, _ := auth.HashPassword("correct-password")
		existingUser := &userModel.User{
			ID:           "user-123",
			Email:        "test@example.com",
			PasswordHash: passwordHash,
		}

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return existingUser, nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.LoginRequest{
			Email:    "test@example.com",
			Password: "wrong-password",
		}

		user, tokens, err := svc.Login(context.Background(), req)

		assert.Nil(t, user)
		assert.Nil(t, tokens)
		assert.Equal(t, userModel.ErrInvalidCredentials, err)
	})

	t.Run("normalizes email to lowercase", func(t *testing.T) {
		var queriedEmail string
		passwordHash, _ := auth.HashPassword("password123")

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				queriedEmail = email
				return &userModel.User{
					ID:           "user-123",
					Email:        email,
					PasswordHash: passwordHash,
				}, nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		req := &authModel.LoginRequest{
			Email:    "TEST@EXAMPLE.COM",
			Password: "password123",
		}

		_, _, err := svc.Login(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", queriedEmail)
	})
}

func TestAuthService_RefreshTokens(t *testing.T) {
	t.Run("successfully refreshes tokens with valid refresh token", func(t *testing.T) {
		jwtManager := createTestJWTManager()
		refreshToken, _ := jwtManager.GenerateRefreshToken("user-123")
		tokenHash := auth.HashToken(refreshToken)

		dbToken := &authModel.RefreshToken{
			ID:        "token-1",
			UserID:    "user-123",
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{
			GetByTokenHashFunc: func(ctx context.Context, hash string) (*authModel.RefreshToken, error) {
				return dbToken, nil
			},
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
			RevokeFunc: func(ctx context.Context, hash string) error {
				return nil
			},
		}

		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		tokens, err := svc.RefreshTokens(context.Background(), refreshToken)

		require.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("returns error for invalid refresh token", func(t *testing.T) {
		jwtManager := createTestJWTManager()
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}

		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		tokens, err := svc.RefreshTokens(context.Background(), "invalid-token")

		assert.Nil(t, tokens)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})

	t.Run("returns error for revoked refresh token", func(t *testing.T) {
		jwtManager := createTestJWTManager()
		refreshToken, _ := jwtManager.GenerateRefreshToken("user-123")
		tokenHash := auth.HashToken(refreshToken)
		revokedAt := time.Now()

		dbToken := &authModel.RefreshToken{
			ID:        "token-1",
			UserID:    "user-123",
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
			RevokedAt: &revokedAt,
		}

		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{
			GetByTokenHashFunc: func(ctx context.Context, hash string) (*authModel.RefreshToken, error) {
				return dbToken, nil
			},
		}

		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		tokens, err := svc.RefreshTokens(context.Background(), refreshToken)

		assert.Nil(t, tokens)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired or revoked")
	})

	t.Run("returns error for expired refresh token in database", func(t *testing.T) {
		jwtManager := createTestJWTManager()
		refreshToken, _ := jwtManager.GenerateRefreshToken("user-123")
		tokenHash := auth.HashToken(refreshToken)

		dbToken := &authModel.RefreshToken{
			ID:        "token-1",
			UserID:    "user-123",
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
			CreatedAt: time.Now().Add(-48 * time.Hour),
		}

		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{
			GetByTokenHashFunc: func(ctx context.Context, hash string) (*authModel.RefreshToken, error) {
				return dbToken, nil
			},
		}

		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		tokens, err := svc.RefreshTokens(context.Background(), refreshToken)

		assert.Nil(t, tokens)
		assert.Error(t, err)
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("successfully logs out user", func(t *testing.T) {
		var revokedUserID string

		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeAllForUserFunc: func(ctx context.Context, userID string) error {
				revokedUserID = userID
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		err := svc.Logout(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Equal(t, "user-123", revokedUserID)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeAllForUserFunc: func(ctx context.Context, userID string) error {
				return expectedError
			},
		}

		jwtManager := createTestJWTManager()
		svc := NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)

		err := svc.Logout(context.Background(), "user-123")

		assert.Equal(t, expectedError, err)
	})
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.org", true},
		{"user+tag@example.co.uk", true},
		{"invalid-email", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
		{"user@domain", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}
