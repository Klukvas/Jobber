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

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
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

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.DELETE("/resume-builder/:id", handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// --- Get Handler Tests ---

func TestResumeBuilderHandler_Get(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 with resume", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{
						ID:         resumeID,
						Title:      "My Resume",
						TemplateID: "00000000-0000-0000-0000-000000000001",
					},
				}, nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.GET("/resume-builder/:id", authMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.GET("/resume-builder/:id", handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrResumeBuilderNotFound
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.GET("/resume-builder/:id", authMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 403 when not owner", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.GET("/resume-builder/:id", authMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/resume-builder/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// --- Update Handler Tests ---

func TestResumeBuilderHandler_Update(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 on successful update", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
				return &model.ResumeBuilder{
					ID:         resumeID,
					UserID:     userID,
					Title:      "Old Title",
					TemplateID: "00000000-0000-0000-0000-000000000001",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}, nil
			},
			UpdateFunc: func(_ context.Context, _ *model.ResumeBuilder) error { return nil },
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.PATCH("/resume-builder/:id", authMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/resume-builder/"+resumeID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PATCH("/resume-builder/:id", handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/resume-builder/"+resumeID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PATCH("/resume-builder/:id", authMiddleware(userID), handler.Update)

		req, _ := http.NewRequest(http.MethodPatch, "/resume-builder/"+resumeID, bytes.NewBufferString("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrResumeBuilderNotFound
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.PATCH("/resume-builder/:id", authMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/resume-builder/"+resumeID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- Duplicate Handler Tests ---

func TestResumeBuilderHandler_Duplicate(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 201 on successful duplicate", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{
						ID:         resumeID,
						Title:      "My Resume",
						TemplateID: "00000000-0000-0000-0000-000000000001",
					},
				}, nil
			},
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "rb-dup-1"
				rb.CreatedAt = time.Now()
				rb.UpdatedAt = time.Now()
				return nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/duplicate", authMiddleware(userID), handler.Duplicate)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/duplicate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/duplicate", handler.Duplicate)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/duplicate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, limiter)
		router := setupRouter()
		router.POST("/resume-builder/:id/duplicate", authMiddleware(userID), handler.Duplicate)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/duplicate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrResumeBuilderNotFound
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/duplicate", authMiddleware(userID), handler.Duplicate)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/duplicate", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// --- List service error ---

func TestResumeBuilderHandler_List_ServiceError(t *testing.T) {
	userID := "user-123"

	repo := &mockResumeBuilderRepository{
		ListFunc: func(_ context.Context, _ string) ([]*model.ResumeBuilderDTO, error) {
			return nil, errors.New("db error")
		},
	}

	handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
	router := setupRouter()
	router.GET("/resume-builder", authMiddleware(userID), handler.List)

	req, _ := http.NewRequest(http.MethodGet, "/resume-builder", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Create invalid JSON ---

func TestResumeBuilderHandler_Create_InvalidJSON(t *testing.T) {
	userID := "user-123"

	handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
	router := setupRouter()
	router.POST("/resume-builder", authMiddleware(userID), handler.Create)

	req, _ := http.NewRequest(http.MethodPost, "/resume-builder", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Section Handler Tests ---

func TestResumeBuilderHandler_UpsertContact(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 on success", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertContactFunc:   func(_ context.Context, _ *model.Contact) error { return nil },
			GetContactFunc: func(_ context.Context, _ string) (*model.Contact, error) {
				return &model.Contact{ResumeBuilderID: resumeID, FullName: "John"}, nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/contact", authMiddleware(userID), handler.UpsertContact)

		body := `{"full_name":"John","email":"john@test.com"}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/contact", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/contact", handler.UpsertContact)

		body := `{"full_name":"John"}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/contact", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/contact", authMiddleware(userID), handler.UpsertContact)

		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/contact", bytes.NewBufferString("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrResumeBuilderNotFound
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/contact", authMiddleware(userID), handler.UpsertContact)

		body := `{"full_name":"John"}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/contact", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestResumeBuilderHandler_UpsertSummary(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 on success", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertSummaryFunc:   func(_ context.Context, _ *model.Summary) error { return nil },
			GetSummaryFunc: func(_ context.Context, _ string) (*model.Summary, error) {
				return &model.Summary{ResumeBuilderID: resumeID, Content: "Summary"}, nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/summary", authMiddleware(userID), handler.UpsertSummary)

		body := `{"content":"My summary"}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/summary", handler.UpsertSummary)

		body := `{"content":"x"}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/summary", authMiddleware(userID), handler.UpsertSummary)

		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/summary", bytes.NewBufferString("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResumeBuilderHandler_UpdateSectionOrder(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 on success", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertSectionOrderFunc: func(_ context.Context, _ string, _ []*model.SectionOrder) error {
				return nil
			},
			ListSectionOrdersFunc: func(_ context.Context, _ string) ([]*model.SectionOrder, error) {
				return []*model.SectionOrder{}, nil
			},
		}

		handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/section-order", authMiddleware(userID), handler.UpdateSectionOrder)

		body := `{"sections":[{"section_key":"experience","sort_order":1,"is_visible":true}]}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/section-order", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
		router := setupRouter()
		router.PUT("/resume-builder/:id/section-order", handler.UpdateSectionOrder)

		body := `{"sections":[{"section_key":"experience","sort_order":1}]}`
		req, _ := http.NewRequest(http.MethodPut, "/resume-builder/"+resumeID+"/section-order", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestResumeBuilderHandler_SectionCRUD tests create/update/delete for all 1:N section types
// using a table-driven approach since all section handlers follow the same pattern.
func TestResumeBuilderHandler_SectionCRUD(t *testing.T) {
	userID := "user-123"
	resumeID := "00000000-0000-0000-0000-000000000001"
	entryID := "00000000-0000-0000-0000-000000000002"

	sections := []struct {
		name       string
		createPath string
		updatePath string
		deletePath string
		createBody string
		updateBody string
		createFn   func(h *ResumeBuilderHandler) func(*testing.T)
		updateFn   func(h *ResumeBuilderHandler) func(*testing.T)
		deleteFn   func(h *ResumeBuilderHandler) func(*testing.T)
	}{
		{
			name:       "Experience",
			createPath: "/resume-builder/:id/experiences",
			updatePath: "/resume-builder/:id/experiences/:entryId",
			deletePath: "/resume-builder/:id/experiences/:entryId",
			createBody: `{"job_title":"Engineer","company":"Acme"}`,
			updateBody: `{"job_title":"Senior Engineer"}`,
		},
		{
			name:       "Education",
			createPath: "/resume-builder/:id/educations",
			updatePath: "/resume-builder/:id/educations/:entryId",
			deletePath: "/resume-builder/:id/educations/:entryId",
			createBody: `{"institution":"MIT","degree":"BS"}`,
			updateBody: `{"degree":"MS"}`,
		},
		{
			name:       "Skill",
			createPath: "/resume-builder/:id/skills",
			updatePath: "/resume-builder/:id/skills/:entryId",
			deletePath: "/resume-builder/:id/skills/:entryId",
			createBody: `{"name":"Go"}`,
			updateBody: `{"name":"Golang"}`,
		},
		{
			name:       "Language",
			createPath: "/resume-builder/:id/languages",
			updatePath: "/resume-builder/:id/languages/:entryId",
			deletePath: "/resume-builder/:id/languages/:entryId",
			createBody: `{"name":"English"}`,
			updateBody: `{"name":"French"}`,
		},
		{
			name:       "Certification",
			createPath: "/resume-builder/:id/certifications",
			updatePath: "/resume-builder/:id/certifications/:entryId",
			deletePath: "/resume-builder/:id/certifications/:entryId",
			createBody: `{"name":"AWS Cert"}`,
			updateBody: `{"name":"Azure Cert"}`,
		},
		{
			name:       "Project",
			createPath: "/resume-builder/:id/projects",
			updatePath: "/resume-builder/:id/projects/:entryId",
			deletePath: "/resume-builder/:id/projects/:entryId",
			createBody: `{"name":"MyProject"}`,
			updateBody: `{"name":"Updated Project"}`,
		},
		{
			name:       "Volunteering",
			createPath: "/resume-builder/:id/volunteering",
			updatePath: "/resume-builder/:id/volunteering/:entryId",
			deletePath: "/resume-builder/:id/volunteering/:entryId",
			createBody: `{"organization":"Red Cross"}`,
			updateBody: `{"organization":"Habitat"}`,
		},
		{
			name:       "CustomSection",
			createPath: "/resume-builder/:id/custom-sections",
			updatePath: "/resume-builder/:id/custom-sections/:entryId",
			deletePath: "/resume-builder/:id/custom-sections/:entryId",
			createBody: `{"title":"Awards"}`,
			updateBody: `{"title":"Honors"}`,
		},
	}

	type sectionHandlers struct {
		createHandler gin.HandlerFunc
		updateHandler gin.HandlerFunc
		deleteHandler gin.HandlerFunc
	}

	getSectionHandlers := func(h *ResumeBuilderHandler, name string) sectionHandlers {
		switch name {
		case "Experience":
			return sectionHandlers{h.CreateExperience, h.UpdateExperience, h.DeleteExperience}
		case "Education":
			return sectionHandlers{h.CreateEducation, h.UpdateEducation, h.DeleteEducation}
		case "Skill":
			return sectionHandlers{h.CreateSkill, h.UpdateSkill, h.DeleteSkill}
		case "Language":
			return sectionHandlers{h.CreateLanguage, h.UpdateLanguage, h.DeleteLanguage}
		case "Certification":
			return sectionHandlers{h.CreateCertification, h.UpdateCertification, h.DeleteCertification}
		case "Project":
			return sectionHandlers{h.CreateProject, h.UpdateProject, h.DeleteProject}
		case "Volunteering":
			return sectionHandlers{h.CreateVolunteering, h.UpdateVolunteering, h.DeleteVolunteering}
		case "CustomSection":
			return sectionHandlers{h.CreateCustomSection, h.UpdateCustomSection, h.DeleteCustomSection}
		default:
			t.Fatalf("unknown section: %s", name)
			return sectionHandlers{}
		}
	}

	for _, sec := range sections {
		sec := sec
		t.Run(sec.name+"_Create_Success", func(t *testing.T) {
			repo := &mockResumeBuilderRepository{
				VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			}

			handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.POST(sec.createPath, authMiddleware(userID), sh.createHandler)

			req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/"+extractSectionPath(sec.createPath), bytes.NewBufferString(sec.createBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code, "Create %s should return 201", sec.name)
		})

		t.Run(sec.name+"_Create_Unauthorized", func(t *testing.T) {
			handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.POST(sec.createPath, sh.createHandler)

			req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/"+extractSectionPath(sec.createPath), bytes.NewBufferString(sec.createBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run(sec.name+"_Create_InvalidJSON", func(t *testing.T) {
			handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.POST(sec.createPath, authMiddleware(userID), sh.createHandler)

			req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+resumeID+"/"+extractSectionPath(sec.createPath), bytes.NewBufferString("bad"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run(sec.name+"_Update_Success", func(t *testing.T) {
			repo := &mockResumeBuilderRepository{
				VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
				GetExperienceByIDFunc: func(_ context.Context, _, _ string) (*model.Experience, error) {
					return &model.Experience{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetEducationByIDFunc: func(_ context.Context, _, _ string) (*model.Education, error) {
					return &model.Education{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetSkillByIDFunc: func(_ context.Context, _, _ string) (*model.Skill, error) {
					return &model.Skill{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetLanguageByIDFunc: func(_ context.Context, _, _ string) (*model.Language, error) {
					return &model.Language{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetCertificationByIDFunc: func(_ context.Context, _, _ string) (*model.Certification, error) {
					return &model.Certification{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetProjectByIDFunc: func(_ context.Context, _, _ string) (*model.Project, error) {
					return &model.Project{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetVolunteeringByIDFunc: func(_ context.Context, _, _ string) (*model.Volunteering, error) {
					return &model.Volunteering{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
				GetCustomSectionByIDFunc: func(_ context.Context, _, _ string) (*model.CustomSection, error) {
					return &model.CustomSection{ID: entryID, ResumeBuilderID: resumeID}, nil
				},
			}

			handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.PATCH(sec.updatePath, authMiddleware(userID), sh.updateHandler)

			path := "/resume-builder/" + resumeID + "/" + extractSectionPath(sec.createPath) + "/" + entryID
			req, _ := http.NewRequest(http.MethodPatch, path, bytes.NewBufferString(sec.updateBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Update %s should return 200", sec.name)
		})

		t.Run(sec.name+"_Update_Unauthorized", func(t *testing.T) {
			handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.PATCH(sec.updatePath, sh.updateHandler)

			path := "/resume-builder/" + resumeID + "/" + extractSectionPath(sec.createPath) + "/" + entryID
			req, _ := http.NewRequest(http.MethodPatch, path, bytes.NewBufferString(sec.updateBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run(sec.name+"_Update_InvalidJSON", func(t *testing.T) {
			handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.PATCH(sec.updatePath, authMiddleware(userID), sh.updateHandler)

			path := "/resume-builder/" + resumeID + "/" + extractSectionPath(sec.createPath) + "/" + entryID
			req, _ := http.NewRequest(http.MethodPatch, path, bytes.NewBufferString("bad"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})

		t.Run(sec.name+"_Delete_Success", func(t *testing.T) {
			repo := &mockResumeBuilderRepository{
				VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			}

			handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.DELETE(sec.deletePath, authMiddleware(userID), sh.deleteHandler)

			path := "/resume-builder/" + resumeID + "/" + extractSectionPath(sec.createPath) + "/" + entryID
			req, _ := http.NewRequest(http.MethodDelete, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNoContent, w.Code, "Delete %s should return 204", sec.name)
		})

		t.Run(sec.name+"_Delete_Unauthorized", func(t *testing.T) {
			handler := newTestResumeBuilderHandler(&mockResumeBuilderRepository{}, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.DELETE(sec.deletePath, sh.deleteHandler)

			path := "/resume-builder/" + resumeID + "/" + extractSectionPath(sec.createPath) + "/" + entryID
			req, _ := http.NewRequest(http.MethodDelete, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run(sec.name+"_Delete_NotFound", func(t *testing.T) {
			repo := &mockResumeBuilderRepository{
				VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
				DeleteExperienceFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteEducationFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteSkillFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteLanguageFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteCertificationFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteProjectFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteVolunteeringFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
				DeleteCustomSectionFunc: func(_ context.Context, _, _ string) error {
					return model.ErrSectionEntryNotFound
				},
			}

			handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
			sh := getSectionHandlers(handler, sec.name)
			router := setupRouter()
			router.DELETE(sec.deletePath, authMiddleware(userID), sh.deleteHandler)

			path := "/resume-builder/" + resumeID + "/" + extractSectionPath(sec.createPath) + "/" + entryID
			req, _ := http.NewRequest(http.MethodDelete, path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code, "Delete %s not found should return 404", sec.name)
		})
	}
}

// extractSectionPath extracts the section name from a route pattern like "/resume-builder/:id/experiences"
func extractSectionPath(routePattern string) string {
	// Pattern: /resume-builder/:id/experiences or /resume-builder/:id/custom-sections
	parts := []string{
		"experiences", "educations", "skills", "languages",
		"certifications", "projects", "volunteering", "custom-sections",
	}
	for _, p := range parts {
		if len(routePattern) > len(p) && routePattern[len(routePattern)-len(p)-len("/:entryId"):] == p+"/:entryId" || routePattern[len(routePattern)-len(p):] == p {
			return p
		}
	}
	return ""
}
