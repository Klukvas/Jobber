package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestResumeBuilderHandler creates a ResumeBuilderHandler backed by mock repo and limiter.
func newTestResumeBuilderHandler(repo *mockResumeBuilderRepository, limiter *mockLimitChecker) *ResumeBuilderHandler {
	svc := service.NewResumeBuilderService(repo, limiter)
	return NewResumeBuilderHandler(svc)
}

// --- Create Handler Tests ---

func TestResumeBuilderHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("returns 201 with created resume", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "rb-new-1"
				rb.CreatedAt = time.Now()
				rb.UpdatedAt = time.Now()
				return nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder", authMiddleware(userID), handler.Create)

		body := `{"title":"My Resume","template_id":"00000000-0000-0000-0000-000000000001"}`
		req, _ := http.NewRequest(http.MethodPost, "/resume-builder", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var result model.ResumeBuilderDTO
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "rb-new-1", result.ID)
		assert.Equal(t, "My Resume", result.Title)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", result.TemplateID)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder", handler.Create) // no auth middleware

		body := `{"title":"My Resume"}`
		req, _ := http.NewRequest(http.MethodPost, "/resume-builder", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UNAUTHORIZED", resp.ErrorCode)
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, limiter)
		router := setupRouter()
		router.POST("/resume-builder", authMiddleware(userID), handler.Create)

		body := `{"title":"My Resume"}`
		req, _ := http.NewRequest(http.MethodPost, "/resume-builder", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PLAN_LIMIT_REACHED", resp.ErrorCode)
	})
}

// --- List Handler Tests ---

func TestResumeBuilderHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns 200 with list of resumes", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			ListFunc: func(_ context.Context, _ string) ([]*model.ResumeBuilderDTO, error) {
				return []*model.ResumeBuilderDTO{
					{ID: "rb-1", Title: "Resume 1", TemplateID: "00000000-0000-0000-0000-000000000001"},
					{ID: "rb-2", Title: "Resume 2", TemplateID: "00000000-0000-0000-0000-000000000002"},
				}, nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.GET("/resume-builder", authMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/resume-builder", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result []*model.ResumeBuilderDTO
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Resume 1", result[0].Title)
		assert.Equal(t, "Resume 2", result[1].Title)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.GET("/resume-builder", handler.List) // no auth middleware

		req, _ := http.NewRequest(http.MethodGet, "/resume-builder", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UNAUTHORIZED", resp.ErrorCode)
	})
}

// --- Delete Handler Tests ---

func TestResumeBuilderHandler_Delete(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 204 on successful delete", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteFunc: func(_ context.Context, _ string) error {
				return nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.DELETE("/resume-builder/:id", authMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("returns 404 when resume not found", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrResumeBuilderNotFound
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.DELETE("/resume-builder/:id", authMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "RESUME_BUILDER_NOT_FOUND", resp.ErrorCode)
	})

	t.Run("returns 403 when not owner", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.DELETE("/resume-builder/:id", authMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NOT_OWNER", resp.ErrorCode)
	})
}
