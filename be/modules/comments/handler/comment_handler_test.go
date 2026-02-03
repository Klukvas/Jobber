package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/comments/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommentRepository implements ports.CommentRepository
type MockCommentRepository struct {
	CreateFunc            func(ctx context.Context, comment *model.Comment) error
	ListByApplicationFunc func(ctx context.Context, appID string, userID ...string) ([]*model.Comment, error)
	DeleteFunc            func(ctx context.Context, userID, commentID string) error
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) ListByApplication(ctx context.Context, appID string, userID ...string) ([]*model.Comment, error) {
	if m.ListByApplicationFunc != nil {
		return m.ListByApplicationFunc(ctx, appID, userID...)
	}
	return nil, nil
}

func (m *MockCommentRepository) Delete(ctx context.Context, userID, commentID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, commentID)
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

func TestCommentHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates comment successfully", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			CreateFunc: func(ctx context.Context, comment *model.Comment) error {
				comment.ID = "comment-1"
				comment.CreatedAt = time.Now()
				comment.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.POST("/comments", mockAuthMiddleware(userID), handler.Create)

		body := `{"application_id":"app-1","content":"This is a comment"}`
		req, _ := http.NewRequest(http.MethodPost, "/comments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.CommentDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "This is a comment", response.Content)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.POST("/comments", handler.Create) // No auth middleware

		body := `{"application_id":"app-1","content":"Comment"}`
		req, _ := http.NewRequest(http.MethodPost, "/comments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.POST("/comments", mockAuthMiddleware(userID), handler.Create)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/comments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for empty content", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.POST("/comments", mockAuthMiddleware(userID), handler.Create)

		body := `{"application_id":"app-1","content":"   "}`
		req, _ := http.NewRequest(http.MethodPost, "/comments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCommentHandler_ListByApplication(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("returns comments list", func(t *testing.T) {
		expectedComments := []*model.Comment{
			{ID: "comment-1", ApplicationID: appID, Content: "First", CreatedAt: time.Now()},
			{ID: "comment-2", ApplicationID: appID, Content: "Second", CreatedAt: time.Now()},
		}

		mockRepo := &MockCommentRepository{
			ListByApplicationFunc: func(ctx context.Context, aid string, uid ...string) ([]*model.Comment, error) {
				return expectedComments, nil
			},
		}

		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.GET("/applications/:id/comments", mockAuthMiddleware(userID), handler.ListByApplication)

		req, _ := http.NewRequest(http.MethodGet, "/applications/"+appID+"/comments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []model.CommentDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.GET("/applications/:id/comments", handler.ListByApplication)

		req, _ := http.NewRequest(http.MethodGet, "/applications/"+appID+"/comments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCommentHandler_Delete(t *testing.T) {
	userID := "user-123"
	commentID := "comment-1"

	t.Run("deletes comment successfully", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			DeleteFunc: func(ctx context.Context, uid, cid string) error {
				return nil
			},
		}

		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.DELETE("/comments/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/comments/"+commentID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when comment not found", func(t *testing.T) {
		mockRepo := &MockCommentRepository{
			DeleteFunc: func(ctx context.Context, uid, cid string) error {
				return model.ErrCommentNotFound
			},
		}

		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.DELETE("/comments/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/comments/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockCommentRepository{}
		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.DELETE("/comments/:id", handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/comments/"+commentID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCommentHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockCommentRepository{
		CreateFunc: func(ctx context.Context, comment *model.Comment) error {
			comment.ID = "comment-1"
			return nil
		},
		ListByApplicationFunc: func(ctx context.Context, aid string, uid ...string) ([]*model.Comment, error) {
			return []*model.Comment{}, nil
		},
		DeleteFunc: func(ctx context.Context, uid, cid string) error {
			return nil
		},
	}

	svc := service.NewCommentService(mockRepo)
	handler := NewCommentHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/comments"},
		{http.MethodDelete, "/api/v1/comments/test-id"},
		{http.MethodGet, "/api/v1/applications/test-id/comments"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == http.MethodPost {
				body = bytes.NewBufferString(`{"application_id":"app-1","content":"Test"}`)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(route.method, route.path, body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be registered", route.method, route.path)
		})
	}
}
