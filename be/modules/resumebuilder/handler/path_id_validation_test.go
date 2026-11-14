package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/docx"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	pathResumeID = "5d000000-0000-4000-8000-000000000001"
	pathEntryID  = "5d000000-0000-4000-8000-000000000002"
)

// pathIDRouter mounts a module's real routes behind the production path-id
// middleware, so the test also proves the routes use UUID param names.
func pathIDRouter(register func(*gin.RouterGroup)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(httpPlatform.CanonicalUUIDParams())
	register(router.Group("/api/v1"))
	return router
}

func noLimit(c *gin.Context) { c.Next() }

// repoCalls records every id the resume builder repository receives.
type repoCalls struct {
	mu  sync.Mutex
	ids []string
}

func (r *repoCalls) add(ids ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = append(r.ids, ids...)
}

func (r *repoCalls) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.ids...)
}

func sectionRouter(calls *repoCalls) *gin.Engine {
	repo := &mockResumeBuilderRepository{
		VerifyOwnershipFunc: func(_ context.Context, _, id string) error {
			calls.add(id)
			return nil
		},
		DeleteExperienceFunc: func(_ context.Context, resumeID, entryID string) error {
			calls.add(resumeID, entryID)
			return nil
		},
	}
	handler := newTestResumeBuilderHandler(repo, &mockLimitChecker{})
	return pathIDRouter(func(g *gin.RouterGroup) { handler.RegisterRoutes(g, authMiddleware("user-1")) })
}

func exportRouter(calls *repoCalls) *gin.Engine {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, id string) error {
			calls.add(id)
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, id string) (*model.FullResumeDTO, error) {
			calls.add(id)
			return newTestFullResumeDTO(), nil
		},
	}
	handler := NewExportHandler(newTestService(repo), nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())
	return pathIDRouter(func(g *gin.RouterGroup) { handler.RegisterRoutes(g, authMiddleware("user-1"), noLimit) })
}

func TestSectionRoutesForwardCanonicalIDs(t *testing.T) {
	resumeForms, entryForms := idstest.NonCanonicalForms(pathResumeID), idstest.NonCanonicalForms(pathEntryID)
	for name := range resumeForms {
		t.Run(name, func(t *testing.T) {
			calls := &repoCalls{}
			path := "/api/v1/resume-builder/" + url.PathEscape(resumeForms[name]) +
				"/experiences/" + url.PathEscape(entryForms[name])
			w := httptest.NewRecorder()
			sectionRouter(calls).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, path, nil))

			require.Equal(t, http.StatusNoContent, w.Code)
			assert.Equal(t, []string{pathResumeID, pathResumeID, pathEntryID}, calls.all())
		})
	}
}

func TestExportRoutesForwardCanonicalIDs(t *testing.T) {
	for name, form := range idstest.NonCanonicalForms(pathResumeID) {
		t.Run(name, func(t *testing.T) {
			calls := &repoCalls{}
			w := httptest.NewRecorder()
			path := "/api/v1/resume-builder/" + url.PathEscape(form) + "/export-docx"
			exportRouter(calls).ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.NotEmpty(t, calls.all())
			for _, id := range calls.all() {
				assert.Equal(t, pathResumeID, id)
			}
		})
	}
}

// Malformed path ids used to reach Postgres and come back as a 500.
func TestResumeBuilderRoutesRejectMalformedIDs(t *testing.T) {
	tests := []struct {
		name, method, path string
		router             func(*repoCalls) *gin.Engine
	}{
		{"section resume id", http.MethodDelete, "/api/v1/resume-builder/not-a-uuid/experiences/" + pathEntryID, sectionRouter},
		{"section entry id", http.MethodDelete, "/api/v1/resume-builder/" + pathResumeID + "/experiences/42", sectionRouter},
		{"export resume id", http.MethodPost, "/api/v1/resume-builder/rb-1/export-docx", exportRouter},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := &repoCalls{}
			w := httptest.NewRecorder()
			tt.router(calls).ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "VALIDATION_ERROR", parseErrorResponse(t, w).ErrorCode)
			assert.Empty(t, calls.all(), "repository must not be reached")
		})
	}
}
