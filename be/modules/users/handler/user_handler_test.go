package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/andreypavlenko/jobber/modules/users/service"
	"github.com/gin-gonic/gin"
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

func storedUser() *model.User {
	return &model.User{
		ID:           "user-123",
		Email:        "alex@example.com",
		Name:         "Alex Jobseeker",
		PasswordHash: "hash",
		Locale:       "en",
	}
}

func newHandler(repo *MockUserRepository) *UserHandler {
	return NewUserHandler(service.NewUserService(repo))
}

func patchProfile(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(http.MethodPatch, "/profile", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
} {
	t.Helper()
	var body struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestUserHandler_GetProfile(t *testing.T) {
	t.Run("returns the caller's own profile", func(t *testing.T) {
		repo := &MockUserRepository{
			GetByIDFunc: func(context.Context, string) (*model.User, error) { return storedUser(), nil },
		}
		router := setupTestRouter()
		router.GET("/profile", mockAuthMiddleware("user-123"), newHandler(repo).GetProfile)

		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var dto model.UserDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "Alex Jobseeker", dto.Name)
	})

	t.Run("returns 401 without authentication", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/profile", newHandler(&MockUserRepository{}).GetProfile)

		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestUserHandler_RegisterRoutes(t *testing.T) {
	// There is no anonymous view of an account.
	t.Run("puts the profile route behind the auth middleware", func(t *testing.T) {
		router := setupTestRouter()
		v1 := router.Group("/api/v1")
		blocked := func(c *gin.Context) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		newHandler(&MockUserRepository{}).RegisterRoutes(v1, blocked)

		req, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// The profile is read-only. A write route here would need its own
	// validation, its own error mapping and its own audit story; none of that
	// exists, so the method must not be routed at all rather than reaching a
	// handler by accident.
	t.Run("routes no write method on the profile", func(t *testing.T) {
		router := setupTestRouter()
		v1 := router.Group("/api/v1")
		newHandler(&MockUserRepository{}).RegisterRoutes(v1, mockAuthMiddleware("user-123"))

		for _, method := range []string{
			http.MethodPatch,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		} {
			req, _ := http.NewRequest(method, "/api/v1/profile", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNotFound, w.Code, "method %s", method)
		}
	})
}
