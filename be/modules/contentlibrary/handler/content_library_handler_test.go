package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
	"github.com/andreypavlenko/jobber/modules/contentlibrary/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockContentLibraryRepository implements ports.ContentLibraryRepository
type MockContentLibraryRepository struct {
	CreateFunc  func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error)
	GetByIDFunc func(ctx context.Context, id string) (*model.ContentLibraryEntry, error)
	ListFunc    func(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error)
	UpdateFunc  func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error)
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *MockContentLibraryRepository) Create(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, entry)
	}
	return entry, nil
}

func (m *MockContentLibraryRepository) GetByID(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockContentLibraryRepository) List(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockContentLibraryRepository) Update(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, entry)
	}
	return entry, nil
}

func (m *MockContentLibraryRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
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

func TestContentLibraryHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates entry successfully", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			CreateFunc: func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
				entry.ID = "entry-1"
				entry.CreatedAt = time.Now()
				entry.UpdatedAt = time.Now()
				return entry, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.POST("/content-library", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"My Snippet","content":"Some content here","category":"technical"}`
		req, _ := http.NewRequest(http.MethodPost, "/content-library", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.ContentLibraryEntryDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "My Snippet", response.Title)
		assert.Equal(t, "Some content here", response.Content)
		assert.Equal(t, "technical", response.Category)
	})

	t.Run("creates entry with default category", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			CreateFunc: func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
				entry.ID = "entry-2"
				entry.CreatedAt = time.Now()
				entry.UpdatedAt = time.Now()
				return entry, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.POST("/content-library", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"No Category","content":"Content"}`
		req, _ := http.NewRequest(http.MethodPost, "/content-library", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.ContentLibraryEntryDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "general", response.Category)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.POST("/content-library", handler.Create)

		body := `{"title":"Test","content":"Content"}`
		req, _ := http.NewRequest(http.MethodPost, "/content-library", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.POST("/content-library", mockAuthMiddleware(userID), handler.Create)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/content-library", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing required fields", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.POST("/content-library", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"","content":""}`
		req, _ := http.NewRequest(http.MethodPost, "/content-library", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			CreateFunc: func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.POST("/content-library", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"Test","content":"Some content"}`
		req, _ := http.NewRequest(http.MethodPost, "/content-library", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestContentLibraryHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns entries list", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockContentLibraryRepository{
			ListFunc: func(ctx context.Context, uid string) ([]*model.ContentLibraryEntry, error) {
				return []*model.ContentLibraryEntry{
					{ID: "entry-1", UserID: uid, Title: "First", Content: "Content 1", Category: "general", CreatedAt: now, UpdatedAt: now},
					{ID: "entry-2", UserID: uid, Title: "Second", Content: "Content 2", Category: "technical", CreatedAt: now, UpdatedAt: now},
				}, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.GET("/content-library", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/content-library", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []*model.ContentLibraryEntryDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "First", response[0].Title)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			ListFunc: func(ctx context.Context, uid string) ([]*model.ContentLibraryEntry, error) {
				return []*model.ContentLibraryEntry{}, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.GET("/content-library", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/content-library", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []*model.ContentLibraryEntryDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 0)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.GET("/content-library", handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/content-library", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			ListFunc: func(ctx context.Context, uid string) ([]*model.ContentLibraryEntry, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.GET("/content-library", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/content-library", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestContentLibraryHandler_Update(t *testing.T) {
	userID := "user-123"
	entryID := "entry-1"

	t.Run("updates entry successfully", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockContentLibraryRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
				return &model.ContentLibraryEntry{
					ID: id, UserID: userID, Title: "Old Title", Content: "Old Content", Category: "general",
					CreatedAt: now, UpdatedAt: now,
				}, nil
			},
			UpdateFunc: func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
				entry.UpdatedAt = time.Now()
				return entry, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.PATCH("/content-library/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/content-library/"+entryID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ContentLibraryEntryDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "New Title", response.Title)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.PATCH("/content-library/:id", handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/content-library/"+entryID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.PATCH("/content-library/:id", mockAuthMiddleware(userID), handler.Update)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPatch, "/content-library/"+entryID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 404 when entry not found", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
				return nil, errors.New("not found")
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.PATCH("/content-library/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/content-library/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 404 when entry belongs to another user", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockContentLibraryRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
				return &model.ContentLibraryEntry{
					ID: id, UserID: "other-user", Title: "Title", Content: "Content", Category: "general",
					CreatedAt: now, UpdatedAt: now,
				}, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.PATCH("/content-library/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/content-library/"+entryID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestContentLibraryHandler_Delete(t *testing.T) {
	userID := "user-123"
	entryID := "entry-1"

	t.Run("deletes entry successfully", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockContentLibraryRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
				return &model.ContentLibraryEntry{
					ID: id, UserID: userID, Title: "Title", Content: "Content", Category: "general",
					CreatedAt: now, UpdatedAt: now,
				}, nil
			},
			DeleteFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.DELETE("/content-library/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/content-library/"+entryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{}
		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.DELETE("/content-library/:id", handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/content-library/"+entryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 404 when entry not found", func(t *testing.T) {
		mockRepo := &MockContentLibraryRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
				return nil, errors.New("not found")
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.DELETE("/content-library/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/content-library/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 404 when entry belongs to another user", func(t *testing.T) {
		now := time.Now()
		mockRepo := &MockContentLibraryRepository{
			GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
				return &model.ContentLibraryEntry{
					ID: id, UserID: "other-user", Title: "Title", Content: "Content", Category: "general",
					CreatedAt: now, UpdatedAt: now,
				}, nil
			},
		}

		svc := service.NewContentLibraryService(mockRepo)
		handler := NewContentLibraryHandler(svc)

		router := setupTestRouter()
		router.DELETE("/content-library/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/content-library/"+entryID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestContentLibraryHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockContentLibraryRepository{
		CreateFunc: func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
			entry.ID = "entry-1"
			entry.CreatedAt = time.Now()
			entry.UpdatedAt = time.Now()
			return entry, nil
		},
		ListFunc: func(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error) {
			return []*model.ContentLibraryEntry{}, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
			return &model.ContentLibraryEntry{
				ID: id, UserID: "user-123", Title: "T", Content: "C", Category: "general",
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
		UpdateFunc: func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
			return entry, nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	svc := service.NewContentLibraryService(mockRepo)
	handler := NewContentLibraryHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/content-library"},
		{http.MethodGet, "/api/v1/content-library"},
		{http.MethodPatch, "/api/v1/content-library/test-id"},
		{http.MethodDelete, "/api/v1/content-library/test-id"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == http.MethodPost || route.method == http.MethodPatch {
				body = bytes.NewBufferString(`{"title":"Test","content":"Content"}`)
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
