package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobimport/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func noopRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func TestImportHandler_ParseJobPage(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", handler.ParseJobPage)

		body := `{"page_text":"some text that is long enough","page_url":"https://example.com/job"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", mockAuthMiddleware(userID), handler.ParseJobPage)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing page_text", func(t *testing.T) {
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", mockAuthMiddleware(userID), handler.ParseJobPage)

		body := `{"page_text":"","page_url":"https://example.com/job"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for too short page_text", func(t *testing.T) {
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", mockAuthMiddleware(userID), handler.ParseJobPage)

		body := `{"page_text":"short","page_url":"https://example.com/job"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid page_url", func(t *testing.T) {
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", mockAuthMiddleware(userID), handler.ParseJobPage)

		body := `{"page_text":"some text that is long enough for validation","page_url":"not-a-url"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing page_url", func(t *testing.T) {
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", mockAuthMiddleware(userID), handler.ParseJobPage)

		body := `{"page_text":"some text that is long enough for validation"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 503 when AI is not configured", func(t *testing.T) {
		// aiClient is nil -> service returns ErrAINotConfigured
		svc := service.NewImportService(nil, nil)
		handler := NewImportHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/parse", mockAuthMiddleware(userID), handler.ParseJobPage)

		body := `{"page_text":"some text that is long enough for validation","page_url":"https://example.com/job"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "AI_NOT_CONFIGURED", response["error_code"])
	})
}

func TestImportHandler_RegisterRoutes(t *testing.T) {
	svc := service.NewImportService(nil, nil)
	handler := NewImportHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"), noopRateLimiter())

	t.Run("POST /api/v1/jobs/parse is registered", func(t *testing.T) {
		body := `{"page_text":"some text that is long enough for validation","page_url":"https://example.com/job"}`
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs/parse", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusNotFound, w.Code, "Route should be registered")
	})
}
