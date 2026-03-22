package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/docx"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers (reuse from cover_letter_handler_test.go via same package) ---

func setupExportTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func exportMockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func newExportTestService(repo *mockCoverLetterRepo) *service.CoverLetterService {
	return service.NewCoverLetterService(repo, &mockLimitChecker{})
}

// --- ExportDOCX tests ---

func TestExportDOCX_Success(t *testing.T) {
	cl := newTestCoverLetterEntity()
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
	}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "Test Cover Letter.docx")
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Greater(t, w.Body.Len(), 0, "DOCX body should not be empty")
}

func TestExportDOCX_Unauthorized(t *testing.T) {
	repo := &mockCoverLetterRepo{}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-docx", handler.ExportDOCX) // no auth middleware

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "UNAUTHORIZED", errResp.ErrorCode)
}

func TestExportDOCX_NotFound(t *testing.T) {
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return nil, model.ErrCoverLetterNotFound
		},
	}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/nonexistent/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "COVER_LETTER_NOT_FOUND", errResp.ErrorCode)
}

func TestExportDOCX_SuccessWithEmptyFields(t *testing.T) {
	cl := &model.CoverLetter{
		ID:           "cl-empty",
		UserID:       "user-1",
		Title:        "",
		Template:     "professional",
		FontFamily:   "Georgia",
		FontSize:     12,
		PrimaryColor: "#2563eb",
		Paragraphs:   []string{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
	}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-empty/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Empty title should be sanitized to "cover-letter"
	assert.Contains(t, w.Header().Get("Content-Disposition"), "cover-letter.docx")
	assert.Greater(t, w.Body.Len(), 0)
}

func TestExportDOCX_WrongOwner(t *testing.T) {
	cl := newTestCoverLetterEntity() // UserID = "user-1"
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
	}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-docx", exportMockAuthMiddleware("user-2"), handler.ExportDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "NOT_AUTHORIZED", errResp.ErrorCode)
}

// --- sanitizeCLFilename tests ---

func TestSanitizeCLFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "My Cover Letter",
			expected: "My Cover Letter",
		},
		{
			name:     "removes forward slash",
			input:    "path/to/file",
			expected: "path_to_file",
		},
		{
			name:     "removes backslash",
			input:    "path\\to\\file",
			expected: "path_to_file",
		},
		{
			name:     "removes colon",
			input:    "file: name",
			expected: "file_ name",
		},
		{
			name:     "removes quotes",
			input:    `"quoted"`,
			expected: "_quoted_",
		},
		{
			name:     "empty string returns cover-letter",
			input:    "",
			expected: "cover-letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCLFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeCLFilename_LongInputTruncation(t *testing.T) {
	longName := ""
	for i := 0; i < 210; i++ {
		longName += "a"
	}
	result := sanitizeCLFilename(longName)
	require.Len(t, result, 200)
}

// --- RegisterRoutes tests ---

func TestExportRegisterRoutes_DOCXRouteRegisteredWhenServiceSet(t *testing.T) {
	repo := &mockCoverLetterRepo{}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	group := router.Group("/cover-letters-export")

	noopMiddleware := func(c *gin.Context) { c.Next() }
	handler.RegisterRoutes(group, noopMiddleware, noopMiddleware)

	// DOCX route should exist, returning 401 (not 404)
	req, _ := http.NewRequest(http.MethodPost, "/cover-letters-export/cover-letters/cl-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- ExportPDF tests ---

func TestExportPDF_Unauthorized(t *testing.T) {
	repo := &mockCoverLetterRepo{}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-pdf", handler.ExportPDF)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "UNAUTHORIZED", errResp.ErrorCode)
}

func TestExportPDF_NotFound(t *testing.T) {
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return nil, model.ErrCoverLetterNotFound
		},
	}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-pdf", exportMockAuthMiddleware("user-1"), handler.ExportPDF)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/nonexistent/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "COVER_LETTER_NOT_FOUND", errResp.ErrorCode)
}

func TestExportPDF_WrongOwner(t *testing.T) {
	cl := newTestCoverLetterEntity() // UserID = "user-1"
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
	}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)

	router := setupExportTestRouter()
	router.POST("/cover-letters/:id/export-pdf", exportMockAuthMiddleware("user-2"), handler.ExportPDF)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- ExportHandler.handleError tests ---

func TestExportHandleError_MapsAllCases(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "not found",
			err:            model.ErrCoverLetterNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "COVER_LETTER_NOT_FOUND",
		},
		{
			name:           "not authorized",
			err:            model.ErrNotAuthorized,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "NOT_AUTHORIZED",
		},
		{
			name:           "unknown error",
			err:            errors.New("some error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ExportHandler{}
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

			handler.handleError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestExportRegisterRoutes_DOCXRouteNotRegisteredWhenServiceNil(t *testing.T) {
	repo := &mockCoverLetterRepo{}
	svc := newExportTestService(repo)
	handler := NewExportHandler(svc, nil)
	// Do NOT set DOCX service

	router := setupExportTestRouter()
	group := router.Group("/cover-letters-export")

	noopMiddleware := func(c *gin.Context) { c.Next() }
	handler.RegisterRoutes(group, noopMiddleware, noopMiddleware)

	// DOCX route should not exist, returning 404
	req, _ := http.NewRequest(http.MethodPost, "/cover-letters-export/cover-letters/cl-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
