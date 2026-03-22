package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/email"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	"github.com/andreypavlenko/jobber/modules/auth/service"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func mockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func createTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager(
		"test-access-secret-key-32chars!!",
		"test-refresh-secret-key-32chars!",
		15*time.Minute,
		7*24*time.Hour,
	)
}

func createTestAuthService(userRepo *MockUserRepository, tokenRepo *MockRefreshTokenRepository) *service.AuthService {
	return service.NewAuthService(service.AuthServiceConfig{
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

func TestAuthHandler_Register(t *testing.T) {
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

		mockTokenRepo := &MockRefreshTokenRepository{}

		svc := createTestAuthService(mockUserRepo, mockTokenRepo)
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/register", handler.Register)

		body := `{"email":"test@example.com","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "message")
	})

	t.Run("returns 400 for invalid request payload", func(t *testing.T) {
		svc := createTestAuthService(&MockUserRepository{}, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/register", handler.Register)

		body := `{"invalid": json}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 409 for existing user", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return &userModel.User{ID: "existing-user", Email: email}, nil
			},
		}

		svc := createTestAuthService(mockUserRepo, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/register", handler.Register)

		body := `{"email":"existing@example.com","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("returns 400 for invalid email", func(t *testing.T) {
		svc := createTestAuthService(&MockUserRepository{}, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/register", handler.Register)

		body := `{"email":"invalid-email","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
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

		svc := createTestAuthService(mockUserRepo, mockTokenRepo)
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/login", handler.Login)

		body := `{"email":"test@example.com","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response.User)
		assert.NotNil(t, response.Tokens)
	})

	t.Run("returns 403 for unverified email", func(t *testing.T) {
		passwordHash, _ := auth.HashPassword("password123")
		unverifiedUser := &userModel.User{
			ID:            "user-456",
			Email:         "unverified@example.com",
			Name:          "Unverified User",
			PasswordHash:  passwordHash,
			Locale:        "en",
			EmailVerified: false,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return unverifiedUser, nil
			},
		}

		svc := createTestAuthService(mockUserRepo, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/login", handler.Login)

		body := `{"email":"unverified@example.com","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("returns 401 for invalid credentials", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
		}

		svc := createTestAuthService(mockUserRepo, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/login", handler.Login)

		body := `{"email":"nonexistent@example.com","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request payload", func(t *testing.T) {
		svc := createTestAuthService(&MockUserRepository{}, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/login", handler.Login)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	t.Run("returns 400 for invalid payload", func(t *testing.T) {
		svc := createTestAuthService(&MockUserRepository{}, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/verify-email", handler.VerifyEmail)

		body := `{"email":"not-an-email","code":"12"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/verify-email", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("successfully refreshes tokens", func(t *testing.T) {
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

		svc := createTestAuthService(&MockUserRepository{}, mockTokenRepo)
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/refresh", handler.Refresh)

		body := `{"refresh_token":"` + refreshToken + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response authModel.AuthTokens
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
	})

	t.Run("returns 401 for invalid refresh token", func(t *testing.T) {
		svc := createTestAuthService(&MockUserRepository{}, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/refresh", handler.Refresh)

		body := `{"refresh_token":"invalid-token"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("successfully logs out", func(t *testing.T) {
		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeAllForUserFunc: func(ctx context.Context, userID string) error {
				return nil
			},
		}

		svc := createTestAuthService(&MockUserRepository{}, mockTokenRepo)
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/logout", mockAuthMiddleware("user-123"), handler.Logout)

		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		svc := createTestAuthService(&MockUserRepository{}, &MockRefreshTokenRepository{})
		handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

		router := setupTestRouter()
		router.POST("/auth/logout", handler.Logout) // No auth middleware

		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
	svc := createTestAuthService(
		&MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
			CreateFunc: func(ctx context.Context, user *userModel.User) error {
				user.ID = "user-123"
				return nil
			},
		},
		&MockRefreshTokenRepository{},
	)
	handler := NewAuthHandler(svc, auth.NewCookieConfig("test"), 15*time.Minute, 168*time.Hour)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	jwtManager := createTestJWTManager()
	authMiddleware := auth.AuthMiddleware(jwtManager)
	handler.RegisterRoutes(v1, AuthRouteConfig{
		AuthMiddleware: authMiddleware,
	})

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodPost, "/api/v1/auth/verify-email"},
		{http.MethodPost, "/api/v1/auth/resend-verification"},
		{http.MethodPost, "/api/v1/auth/forgot-password"},
		{http.MethodPost, "/api/v1/auth/reset-password"},
	}

	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			req, _ := http.NewRequest(route.method, route.path, bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// We expect either success or a handled error (not 404)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be registered", route.method, route.path)
		})
	}
}
