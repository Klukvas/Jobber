package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/email"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUserRepository implements userPorts.UserRepository
type MockUserRepository struct {
	CreateFunc             func(ctx context.Context, user *userModel.User) error
	GetByIDFunc            func(ctx context.Context, userID string) (*userModel.User, error)
	GetByEmailFunc         func(ctx context.Context, email string) (*userModel.User, error)
	UpdateFunc             func(ctx context.Context, user *userModel.User) error
	DeleteFunc             func(ctx context.Context, userID string) error
	SetEmailVerifiedFunc   func(ctx context.Context, userID string) error
	UpdatePasswordHashFunc func(ctx context.Context, userID, hash string) error
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

func (m *MockUserRepository) SetEmailVerified(ctx context.Context, userID string) error {
	if m.SetEmailVerifiedFunc != nil {
		return m.SetEmailVerifiedFunc(ctx, userID)
	}
	return nil
}

func (m *MockUserRepository) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	if m.UpdatePasswordHashFunc != nil {
		return m.UpdatePasswordHashFunc(ctx, userID, hash)
	}
	return nil
}

// MockRefreshTokenRepository implements authPorts.RefreshTokenRepository
type MockRefreshTokenRepository struct {
	CreateFunc           func(ctx context.Context, token *authModel.RefreshToken) error
	GetByTokenHashFunc   func(ctx context.Context, tokenHash string) (*authModel.RefreshToken, error)
	RevokeFunc           func(ctx context.Context, tokenHash string) error
	RevokeIfValidFunc    func(ctx context.Context, tokenHash string) (bool, error)
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

func (m *MockRefreshTokenRepository) RevokeIfValid(ctx context.Context, tokenHash string) (bool, error) {
	if m.RevokeIfValidFunc != nil {
		return m.RevokeIfValidFunc(ctx, tokenHash)
	}
	return true, nil
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

// MockEmailVerificationRepository implements authPorts.EmailVerificationRepository
type MockEmailVerificationRepository struct {
	CreateFunc            func(ctx context.Context, token *authModel.EmailVerificationToken) error
	GetActiveForUserFunc  func(ctx context.Context, userID string) (*authModel.EmailVerificationToken, error)
	IncrementAttemptsFunc func(ctx context.Context, id string, maxAttempts int) (int, error)
	MarkUsedFunc          func(ctx context.Context, id string) error
	DeleteForUserFunc     func(ctx context.Context, userID string) error
	DeleteExpiredFunc     func(ctx context.Context) error
}

func (m *MockEmailVerificationRepository) Create(ctx context.Context, token *authModel.EmailVerificationToken) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token)
	}
	return nil
}

func (m *MockEmailVerificationRepository) GetActiveForUser(ctx context.Context, userID string) (*authModel.EmailVerificationToken, error) {
	if m.GetActiveForUserFunc != nil {
		return m.GetActiveForUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockEmailVerificationRepository) IncrementAttempts(ctx context.Context, id string, maxAttempts int) (int, error) {
	if m.IncrementAttemptsFunc != nil {
		return m.IncrementAttemptsFunc(ctx, id, maxAttempts)
	}
	return 1, nil
}

func (m *MockEmailVerificationRepository) MarkUsed(ctx context.Context, id string) error {
	if m.MarkUsedFunc != nil {
		return m.MarkUsedFunc(ctx, id)
	}
	return nil
}

func (m *MockEmailVerificationRepository) DeleteForUser(ctx context.Context, userID string) error {
	if m.DeleteForUserFunc != nil {
		return m.DeleteForUserFunc(ctx, userID)
	}
	return nil
}

func (m *MockEmailVerificationRepository) DeleteExpired(ctx context.Context) error {
	if m.DeleteExpiredFunc != nil {
		return m.DeleteExpiredFunc(ctx)
	}
	return nil
}

// MockPasswordResetRepository implements authPorts.PasswordResetRepository
type MockPasswordResetRepository struct {
	CreateFunc            func(ctx context.Context, token *authModel.PasswordResetToken) error
	GetActiveForUserFunc  func(ctx context.Context, userID string) (*authModel.PasswordResetToken, error)
	IncrementAttemptsFunc func(ctx context.Context, id string, maxAttempts int) (int, error)
	MarkUsedFunc          func(ctx context.Context, id string) error
	DeleteForUserFunc     func(ctx context.Context, userID string) error
	DeleteExpiredFunc     func(ctx context.Context) error
}

func (m *MockPasswordResetRepository) Create(ctx context.Context, token *authModel.PasswordResetToken) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token)
	}
	return nil
}

func (m *MockPasswordResetRepository) GetActiveForUser(ctx context.Context, userID string) (*authModel.PasswordResetToken, error) {
	if m.GetActiveForUserFunc != nil {
		return m.GetActiveForUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockPasswordResetRepository) IncrementAttempts(ctx context.Context, id string, maxAttempts int) (int, error) {
	if m.IncrementAttemptsFunc != nil {
		return m.IncrementAttemptsFunc(ctx, id, maxAttempts)
	}
	return 1, nil
}

func (m *MockPasswordResetRepository) MarkUsed(ctx context.Context, id string) error {
	if m.MarkUsedFunc != nil {
		return m.MarkUsedFunc(ctx, id)
	}
	return nil
}

func (m *MockPasswordResetRepository) DeleteForUser(ctx context.Context, userID string) error {
	if m.DeleteForUserFunc != nil {
		return m.DeleteForUserFunc(ctx, userID)
	}
	return nil
}

func (m *MockPasswordResetRepository) DeleteExpired(ctx context.Context) error {
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

func createTestService(userRepo *MockUserRepository, tokenRepo *MockRefreshTokenRepository) *AuthService {
	return NewAuthService(AuthServiceConfig{
		UserRepo:          userRepo,
		TokenRepo:         tokenRepo,
		VerificationRepo:  &MockEmailVerificationRepository{},
		PasswordResetRepo: &MockPasswordResetRepository{},
		EmailSender:       &email.NoopSender{},
		JWTManager:        createTestJWTManager(),
		AccessExpiry:      15 * time.Minute,
		RefreshExpiry:     7 * 24 * time.Hour,
	})
}

func createTestServiceFull(
	userRepo *MockUserRepository,
	tokenRepo *MockRefreshTokenRepository,
	verificationRepo *MockEmailVerificationRepository,
	passwordResetRepo *MockPasswordResetRepository,
) *AuthService {
	return NewAuthService(AuthServiceConfig{
		UserRepo:          userRepo,
		TokenRepo:         tokenRepo,
		VerificationRepo:  verificationRepo,
		PasswordResetRepo: passwordResetRepo,
		EmailSender:       &email.NoopSender{},
		JWTManager:        createTestJWTManager(),
		AccessExpiry:      15 * time.Minute,
		RefreshExpiry:     7 * 24 * time.Hour,
	})
}

func TestAuthService_Register(t *testing.T) {
	t.Run("successfully registers a new user and returns message", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
			CreateFunc: func(ctx context.Context, user *userModel.User) error {
				user.ID = "user-123"
				return nil
			},
		}

		svc := createTestService(mockUserRepo, &MockRefreshTokenRepository{})

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			Locale:   "en",
		}

		resp, err := svc.Register(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "check your email")
	})

	t.Run("returns error for invalid email", func(t *testing.T) {
		svc := createTestService(&MockUserRepository{}, &MockRefreshTokenRepository{})

		req := &authModel.RegisterRequest{
			Email:    "invalid-email",
			Password: "password123",
		}

		resp, err := svc.Register(context.Background(), req)

		assert.Nil(t, resp)
		assert.Equal(t, userModel.ErrInvalidEmail, err)
	})

	t.Run("returns error for short password", func(t *testing.T) {
		svc := createTestService(&MockUserRepository{}, &MockRefreshTokenRepository{})

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "short",
		}

		resp, err := svc.Register(context.Background(), req)

		assert.Nil(t, resp)
		assert.Equal(t, userModel.ErrInvalidPassword, err)
	})

	t.Run("returns error if user already exists", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "existing-user", Email: "test@example.com"}, nil
			},
		}

		svc := createTestService(mockUserRepo, &MockRefreshTokenRepository{})

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, err := svc.Register(context.Background(), req)

		assert.Nil(t, resp)
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

		svc := createTestService(mockUserRepo, &MockRefreshTokenRepository{})

		req := &authModel.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			Locale:   "",
		}

		_, err := svc.Register(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "en", createdUser.Locale)
	})
}

func TestAuthService_Login(t *testing.T) {
	t.Run("successfully logs in verified user", func(t *testing.T) {
		passwordHash, _ := auth.HashPassword("password123")
		existingUser := &userModel.User{
			ID:            "user-123",
			Email:         "test@example.com",
			Name:          "Test User",
			PasswordHash:  passwordHash,
			Locale:        "en",
			EmailVerified: true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
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

		svc := createTestService(mockUserRepo, mockTokenRepo)

		req := &authModel.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		user, tokens, err := svc.Login(context.Background(), req)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, tokens)
		assert.Equal(t, "user-123", user.ID)
	})

	t.Run("returns error for unverified email", func(t *testing.T) {
		passwordHash, _ := auth.HashPassword("password123")
		existingUser := &userModel.User{
			ID:            "user-123",
			Email:         "test@example.com",
			PasswordHash:  passwordHash,
			EmailVerified: false,
		}

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return existingUser, nil
			},
		}

		svc := createTestService(mockUserRepo, &MockRefreshTokenRepository{})

		req := &authModel.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		user, tokens, err := svc.Login(context.Background(), req)

		assert.Nil(t, user)
		assert.Nil(t, tokens)
		assert.Equal(t, userModel.ErrEmailNotVerified, err)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
		}

		svc := createTestService(mockUserRepo, &MockRefreshTokenRepository{})

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
			ID:            "user-123",
			Email:         "test@example.com",
			PasswordHash:  passwordHash,
			EmailVerified: true,
		}

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return existingUser, nil
			},
		}

		svc := createTestService(mockUserRepo, &MockRefreshTokenRepository{})

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
					ID:            "user-123",
					Email:         email,
					PasswordHash:  passwordHash,
					EmailVerified: true,
				}, nil
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		svc := createTestService(mockUserRepo, mockTokenRepo)

		req := &authModel.LoginRequest{
			Email:    "TEST@EXAMPLE.COM",
			Password: "password123",
		}

		_, _, err := svc.Login(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", queriedEmail)
	})
}

func TestAuthService_VerifyEmail(t *testing.T) {
	t.Run("verifies email with correct code", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		verificationRepo := &MockEmailVerificationRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.EmailVerificationToken, error) {
				return &authModel.EmailVerificationToken{
					ID:        "token-1",
					UserID:    userID,
					Code:      "123456",
					Attempts:  0,
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}, nil
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, verificationRepo, &MockPasswordResetRepository{})

		err := svc.VerifyEmail(context.Background(), "test@example.com", "123456")
		assert.NoError(t, err)
	})

	t.Run("returns error for wrong code", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		verificationRepo := &MockEmailVerificationRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.EmailVerificationToken, error) {
				return &authModel.EmailVerificationToken{
					ID:        "token-1",
					UserID:    userID,
					Code:      "123456",
					Attempts:  0,
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}, nil
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, verificationRepo, &MockPasswordResetRepository{})

		err := svc.VerifyEmail(context.Background(), "test@example.com", "999999")
		assert.Equal(t, userModel.ErrInvalidVerificationToken, err)
	})

	t.Run("returns too many attempts after 5 failures", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		verificationRepo := &MockEmailVerificationRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.EmailVerificationToken, error) {
				return &authModel.EmailVerificationToken{
					ID:        "token-1",
					UserID:    userID,
					Code:      "123456",
					Attempts:  5,
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}, nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, maxAttempts int) (int, error) {
				return 0, userModel.ErrTooManyAttempts
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, verificationRepo, &MockPasswordResetRepository{})

		err := svc.VerifyEmail(context.Background(), "test@example.com", "123456")
		assert.Equal(t, userModel.ErrTooManyAttempts, err)
	})

	t.Run("returns error for unknown email", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, &MockEmailVerificationRepository{}, &MockPasswordResetRepository{})

		err := svc.VerifyEmail(context.Background(), "unknown@example.com", "123456")
		assert.Equal(t, userModel.ErrInvalidVerificationToken, err)
	})

	t.Run("returns error when no active token exists", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		verificationRepo := &MockEmailVerificationRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.EmailVerificationToken, error) {
				return nil, errors.New("not found")
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, verificationRepo, &MockPasswordResetRepository{})

		err := svc.VerifyEmail(context.Background(), "test@example.com", "123456")
		assert.Equal(t, userModel.ErrInvalidVerificationToken, err)
	})
}

func TestAuthService_ResetPassword(t *testing.T) {
	t.Run("resets password with correct code", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		passwordResetRepo := &MockPasswordResetRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.PasswordResetToken, error) {
				return &authModel.PasswordResetToken{
					ID:        "token-1",
					UserID:    userID,
					Code:      "654321",
					Attempts:  0,
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}, nil
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, &MockEmailVerificationRepository{}, passwordResetRepo)

		err := svc.ResetPassword(context.Background(), "test@example.com", "654321", "newpassword123")
		assert.NoError(t, err)
	})

	t.Run("returns error for wrong code", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		passwordResetRepo := &MockPasswordResetRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.PasswordResetToken, error) {
				return &authModel.PasswordResetToken{
					ID:        "token-1",
					UserID:    userID,
					Code:      "654321",
					Attempts:  0,
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}, nil
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, &MockEmailVerificationRepository{}, passwordResetRepo)

		err := svc.ResetPassword(context.Background(), "test@example.com", "000000", "newpassword123")
		assert.Equal(t, userModel.ErrInvalidResetToken, err)
	})

	t.Run("returns too many attempts", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "user-123", Email: email}, nil
			},
		}

		passwordResetRepo := &MockPasswordResetRepository{
			GetActiveForUserFunc: func(ctx context.Context, userID string) (*authModel.PasswordResetToken, error) {
				return &authModel.PasswordResetToken{
					ID:        "token-1",
					UserID:    userID,
					Code:      "654321",
					Attempts:  5,
					ExpiresAt: time.Now().Add(10 * time.Minute),
				}, nil
			},
			IncrementAttemptsFunc: func(ctx context.Context, id string, maxAttempts int) (int, error) {
				return 0, userModel.ErrTooManyAttempts
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, &MockEmailVerificationRepository{}, passwordResetRepo)

		err := svc.ResetPassword(context.Background(), "test@example.com", "654321", "newpassword123")
		assert.Equal(t, userModel.ErrTooManyAttempts, err)
	})

	t.Run("returns error for short password", func(t *testing.T) {
		svc := createTestServiceFull(&MockUserRepository{}, &MockRefreshTokenRepository{}, &MockEmailVerificationRepository{}, &MockPasswordResetRepository{})

		err := svc.ResetPassword(context.Background(), "test@example.com", "654321", "short")
		assert.Equal(t, userModel.ErrInvalidPassword, err)
	})

	t.Run("returns error for unknown email", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
		}

		svc := createTestServiceFull(mockUserRepo, &MockRefreshTokenRepository{}, &MockEmailVerificationRepository{}, &MockPasswordResetRepository{})

		err := svc.ResetPassword(context.Background(), "unknown@example.com", "654321", "newpassword123")
		assert.Equal(t, userModel.ErrInvalidResetToken, err)
	})
}

func TestAuthService_RefreshTokens(t *testing.T) {
	t.Run("successfully refreshes tokens with valid refresh token", func(t *testing.T) {
		jwtManager := createTestJWTManager()
		refreshToken, _ := jwtManager.GenerateRefreshToken("user-123")

		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeIfValidFunc: func(ctx context.Context, hash string) (bool, error) {
				return true, nil
			},
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		svc := createTestService(&MockUserRepository{}, mockTokenRepo)

		tokens, err := svc.RefreshTokens(context.Background(), refreshToken)

		require.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("returns error for invalid refresh token", func(t *testing.T) {
		svc := createTestService(&MockUserRepository{}, &MockRefreshTokenRepository{})

		tokens, err := svc.RefreshTokens(context.Background(), "invalid-token")

		assert.Nil(t, tokens)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})

	t.Run("returns error for revoked refresh token", func(t *testing.T) {
		jwtManager := createTestJWTManager()
		refreshToken, _ := jwtManager.GenerateRefreshToken("user-123")

		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeIfValidFunc: func(ctx context.Context, hash string) (bool, error) {
				return false, nil
			},
		}

		svc := createTestService(&MockUserRepository{}, mockTokenRepo)

		tokens, err := svc.RefreshTokens(context.Background(), refreshToken)

		assert.Nil(t, tokens)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired or revoked")
	})
}

func TestAuthService_Logout(t *testing.T) {
	t.Run("successfully logs out user", func(t *testing.T) {
		var revokedUserID string

		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeAllForUserFunc: func(ctx context.Context, userID string) error {
				revokedUserID = userID
				return nil
			},
		}

		svc := createTestService(&MockUserRepository{}, mockTokenRepo)

		err := svc.Logout(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Equal(t, "user-123", revokedUserID)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeAllForUserFunc: func(ctx context.Context, userID string) error {
				return expectedError
			},
		}

		svc := createTestService(&MockUserRepository{}, mockTokenRepo)

		err := svc.Logout(context.Background(), "user-123")

		assert.Equal(t, expectedError, err)
	})
}

func TestGenerateCode(t *testing.T) {
	t.Run("generates 6-digit code", func(t *testing.T) {
		code, err := generateCode()
		require.NoError(t, err)
		assert.Len(t, code, 6)
	})

	t.Run("generates codes with leading zeros", func(t *testing.T) {
		// Run multiple times to check format is always 6 digits
		for i := 0; i < 100; i++ {
			code, err := generateCode()
			require.NoError(t, err)
			assert.Len(t, code, 6, "code should always be 6 characters")
		}
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
