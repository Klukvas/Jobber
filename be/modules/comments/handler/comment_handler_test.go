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
	CreateFunc    func(ctx context.Context, comment *model.Comment) error
	ListByJobFunc func(ctx context.Context, jobID string, userID ...string) ([]*model.Comment, error)
	UpdateFunc    func(ctx context.Context, userID, commentID, content string) (*model.Comment, error)
	DeleteFunc    func(ctx context.Context, userID, commentID string) error
}

func (m *MockCommentRepository) Update(ctx context.Context, userID, commentID, content string) (*model.Comment, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, userID, commentID, content)
	}
	return nil, nil
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}

func (m *MockCommentRepository) ListByJob(ctx context.Context, jobID, userID string) ([]*model.Comment, error) {
	if m.ListByJobFunc != nil {
		return m.ListByJobFunc(ctx, jobID, userID)
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

		body := `{"job_id":"job-1","content":"This is a comment"}`
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

		body := `{"job_id":"job-1","content":"Comment"}`
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

		body := `{"job_id":"job-1","content":"   "}`
		req, _ := http.NewRequest(http.MethodPost, "/comments", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCommentHandler_ListByJob(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("returns comments list", func(t *testing.T) {
		expectedComments := []*model.Comment{
			{ID: "comment-1", JobID: jobID, Content: "First", CreatedAt: time.Now()},
			{ID: "comment-2", JobID: jobID, Content: "Second", CreatedAt: time.Now()},
		}

		mockRepo := &MockCommentRepository{
			ListByJobFunc: func(ctx context.Context, jid string, uid ...string) ([]*model.Comment, error) {
				return expectedComments, nil
			},
		}

		svc := service.NewCommentService(mockRepo)
		handler := NewCommentHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs/:id/comments", mockAuthMiddleware(userID), handler.ListByJob)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/"+jobID+"/comments", nil)
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
		router.GET("/jobs/:id/comments", handler.ListByJob)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/"+jobID+"/comments", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// Comment ids are uuids in the database, and the service refuses anything else
// before it reaches storage — so the tests have to use real ones.
const (
	validCommentID   = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	missingCommentID = "11111111-2222-3333-4444-555555555555"
)

func TestCommentHandler_Update(t *testing.T) {
	const (
		userID    = "user-123"
		commentID = validCommentID
	)

	newRouter := func(repo *MockCommentRepository, withAuth bool) *gin.Engine {
		handler := NewCommentHandler(service.NewCommentService(repo))
		router := setupTestRouter()
		if withAuth {
			router.PATCH("/comments/:id", mockAuthMiddleware(userID), handler.Update)
		} else {
			router.PATCH("/comments/:id", handler.Update)
		}
		return router
	}

	patch := func(router *gin.Engine, id, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodPatch, "/comments/"+id, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("updates the comment and returns it", func(t *testing.T) {
		var sawUserID string
		repo := &MockCommentRepository{
			UpdateFunc: func(_ context.Context, uid, cid, content string) (*model.Comment, error) {
				sawUserID = uid
				return &model.Comment{ID: cid, UserID: uid, JobID: "job-1", Content: content}, nil
			},
		}

		w := patch(newRouter(repo, true), commentID, `{"content":"Edited body"}`)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Edited body")
		assert.Equal(t, userID, sawUserID)
	})

	t.Run("returns 404 for another user's comment", func(t *testing.T) {
		repo := &MockCommentRepository{
			UpdateFunc: func(context.Context, string, string, string) (*model.Comment, error) {
				// The UPDATE is scoped to the author, so a foreign id matches
				// no row — indistinguishable from a comment that never existed.
				return nil, model.ErrCommentNotFound
			},
		}

		w := patch(newRouter(repo, true), "someone-elses-comment", `{"content":"Edited"}`)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.NotContains(t, w.Body.String(), "Edited")
	})

	// One resource, two write endpoints, one taxonomy: an unparseable body is a
	// VALIDATION_ERROR and an empty comment is CONTENT_REQUIRED, on both. This
	// endpoint used to answer both with CONTENT_REQUIRED, which pointed the
	// client at a field that was never the problem.
	t.Run("returns 400 CONTENT_REQUIRED for an empty comment", func(t *testing.T) {
		called := false
		repo := &MockCommentRepository{
			UpdateFunc: func(context.Context, string, string, string) (*model.Comment, error) {
				called = true
				return nil, nil
			},
		}

		w := patch(newRouter(repo, true), commentID, `{"content":"   "}`)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "CONTENT_REQUIRED")
		assert.False(t, called)
	})

	t.Run("returns 400 VALIDATION_ERROR for a body that is not JSON", func(t *testing.T) {
		called := false
		repo := &MockCommentRepository{
			UpdateFunc: func(context.Context, string, string, string) (*model.Comment, error) {
				called = true
				return nil, nil
			},
		}

		w := patch(newRouter(repo, true), commentID, `not json at all`)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "VALIDATION_ERROR")
		assert.NotContains(t, w.Body.String(), "CONTENT_REQUIRED")
		assert.False(t, called)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		w := patch(newRouter(&MockCommentRepository{}, false), commentID, `{"content":"Edited"}`)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCommentHandler_Delete(t *testing.T) {
	userID := "user-123"
	commentID := validCommentID

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

		req, _ := http.NewRequest(http.MethodDelete, "/comments/"+missingCommentID, nil)
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

// A path parameter that is not a uuid used to reach Postgres as a cast error
// and come back as a 500. It is a 404: there is no such comment.
func TestCommentHandler_MalformedID(t *testing.T) {
	const userID = "user-123"

	for _, id := range []string{"comment-1", "not-a-uuid", "7c9e6679-7425"} {
		t.Run(id, func(t *testing.T) {
			touched := false
			handler := NewCommentHandler(service.NewCommentService(&MockCommentRepository{
				UpdateFunc: func(context.Context, string, string, string) (*model.Comment, error) {
					touched = true
					return nil, nil
				},
				DeleteFunc: func(context.Context, string, string) error {
					touched = true
					return nil
				},
			}))
			router := setupTestRouter()
			router.PATCH("/comments/:id", mockAuthMiddleware(userID), handler.Update)
			router.DELETE("/comments/:id", mockAuthMiddleware(userID), handler.Delete)

			patch, _ := http.NewRequest(http.MethodPatch, "/comments/"+id,
				bytes.NewBufferString(`{"content":"Edited"}`))
			patch.Header.Set("Content-Type", "application/json")
			patchRec := httptest.NewRecorder()
			router.ServeHTTP(patchRec, patch)

			del, _ := http.NewRequest(http.MethodDelete, "/comments/"+id, nil)
			delRec := httptest.NewRecorder()
			router.ServeHTTP(delRec, del)

			assert.Equal(t, http.StatusNotFound, patchRec.Code)
			assert.Equal(t, http.StatusNotFound, delRec.Code)
			assert.False(t, touched, "storage must not be asked about an impossible id")
		})
	}
}

func TestCommentHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockCommentRepository{
		CreateFunc: func(ctx context.Context, comment *model.Comment) error {
			comment.ID = "comment-1"
			return nil
		},
		ListByJobFunc: func(ctx context.Context, jid string, uid ...string) ([]*model.Comment, error) {
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
		{http.MethodDelete, "/api/v1/comments/" + validCommentID},
		{http.MethodGet, "/api/v1/jobs/test-id/comments"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == http.MethodPost {
				body = bytes.NewBufferString(`{"job_id":"job-1","content":"Test"}`)
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
