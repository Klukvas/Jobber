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
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	"github.com/andreypavlenko/jobber/modules/auth/service"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
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

		mockTokenRepo := &MockRefreshTokenRepository{
			CreateFunc: func(ctx context.Context, token *authModel.RefreshToken) error {
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

		router := setupTestRouter()
		router.POST("/auth/register", handler.Register)

		body := `{"email":"test@example.com","password":"password123"}`
		req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response RegisterResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response.User)
		assert.NotNil(t, response.Tokens)
		assert.Equal(t, "test@example.com", response.User.Email)
	})

	t.Run("returns 400 for invalid request payload", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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

		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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
	t.Run("successfully logs in", func(t *testing.T) {
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
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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

	t.Run("returns 401 for invalid credentials", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{
			GetByEmailFunc: func(ctx context.Context, email string) (*userModel.User, error) {
				return nil, userModel.ErrUserNotFound
			},
		}

		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}
		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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

func TestAuthHandler_Refresh(t *testing.T) {
	t.Run("successfully refreshes tokens", func(t *testing.T) {
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

		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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
		jwtManager := createTestJWTManager()
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}

		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

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
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{
			RevokeAllForUserFunc: func(ctx context.Context, userID string) error {
				return nil
			},
		}

		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

		router := setupTestRouter()
		router.POST("/auth/logout", mockAuthMiddleware("user-123"), handler.Logout)

		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockUserRepo := &MockUserRepository{}
		mockTokenRepo := &MockRefreshTokenRepository{}

		jwtManager := createTestJWTManager()
		svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
		handler := NewAuthHandler(svc)

		router := setupTestRouter()
		router.POST("/auth/logout", handler.Logout) // No auth middleware

		req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_RegisterRoutes(t *testing.T) {
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
	svc := service.NewAuthService(mockUserRepo, mockTokenRepo, jwtManager, 15*time.Minute, 7*24*time.Hour)
	handler := NewAuthHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/auth/logout"},
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
