package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockImportResumeTextParser implements service.ResumeTextParser for import handler tests.
type mockImportResumeTextParser struct {
	ParseResumeTextFunc func(ctx context.Context, text string) (*ai.ParsedResume, error)
}

func (m *mockImportResumeTextParser) ParseResumeText(ctx context.Context, text string) (*ai.ParsedResume, error) {
	if m.ParseResumeTextFunc != nil {
		return m.ParseResumeTextFunc(ctx, text)
	}
	return &ai.ParsedResume{}, nil
}

// --- Helpers ---

func importSetupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func importAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func defaultImportTestService() *service.ImportService {
	return service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "rb-new"
				return nil
			},
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{
						ID:         "rb-new",
						Title:      "Imported Resume",
						TemplateID: "00000000-0000-0000-0000-000000000001",
					},
				}, nil
			},
		},
		&mockImportResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return &ai.ParsedResume{FullName: "John Doe"}, nil
			},
		},
		&mockLimitChecker{},
		func(_ []byte) (string, error) { return "extracted text", nil },
	)
}

// --- ImportFromText Handler Tests ---

func TestImportFromText_Handler_Success(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"text":"John Doe\nSoftware Engineer\njohn@example.com","title":"My Resume"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result model.FullResumeDTO
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "rb-new", result.ID)
}

func TestImportFromText_Handler_Unauthorized(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", handler.ImportFromText) // No auth middleware

	body := `{"text":"some text"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "UNAUTHORIZED", errResp.ErrorCode)
}

func TestImportFromText_Handler_InvalidJSON(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp.ErrorCode)
}

func TestImportFromText_Handler_MissingText(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"title":"My Resume"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", errResp.ErrorCode)
}

func TestImportFromText_Handler_EmptyText(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"text":"   "}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "EMPTY_TEXT", errResp.ErrorCode)
}

func TestImportFromText_Handler_TextTooLarge(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	// 100KB + 1 byte to exceed the limit
	largeText := strings.Repeat("a", 100*1024+1)
	body := `{"text":"` + largeText + `"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "TEXT_TOO_LARGE", errResp.ErrorCode)
}

func TestImportFromText_Handler_PlanLimitReached(t *testing.T) {
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{},
		&mockImportResumeTextParser{},
		&mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		},
		func(_ []byte) (string, error) { return "", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"text":"some resume text"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "PLAN_LIMIT_REACHED", errResp.ErrorCode)
}

func TestImportFromText_Handler_ServiceError(t *testing.T) {
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, _ *model.ResumeBuilder) error {
				return errors.New("db error")
			},
		},
		&mockImportResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return &ai.ParsedResume{FullName: "Test"}, nil
			},
		},
		&mockLimitChecker{},
		func(_ []byte) (string, error) { return "", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"text":"some resume text"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", errResp.ErrorCode)
}

// --- ImportFromPDF Handler Tests ---

func createMultipartPDFRequest(t *testing.T, fieldName string, fileContent []byte, title string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, "resume.pdf")
	require.NoError(t, err)
	_, err = part.Write(fileContent)
	require.NoError(t, err)

	if title != "" {
		err = writer.WriteField("title", title)
		require.NoError(t, err)
	}

	err = writer.Close()
	require.NoError(t, err)

	req, _ := http.NewRequest(http.MethodPost, "/import/pdf", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestImportFromPDF_Handler_Success(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/pdf", importAuthMiddleware("user-1"), handler.ImportFromPDF)

	req := createMultipartPDFRequest(t, "file", []byte("fake-pdf-content"), "My PDF Resume")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var result model.FullResumeDTO
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "rb-new", result.ID)
}

func TestImportFromPDF_Handler_Unauthorized(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/pdf", handler.ImportFromPDF) // No auth middleware

	req := createMultipartPDFRequest(t, "file", []byte("fake-pdf"), "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestImportFromPDF_Handler_NoFile(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/pdf", importAuthMiddleware("user-1"), handler.ImportFromPDF)

	// Send request without a file
	req, _ := http.NewRequest(http.MethodPost, "/import/pdf", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=something")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "NO_FILE", errResp.ErrorCode)
}

func TestImportFromPDF_Handler_PlanLimitReached(t *testing.T) {
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{},
		&mockImportResumeTextParser{},
		&mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		},
		func(_ []byte) (string, error) { return "text", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/pdf", importAuthMiddleware("user-1"), handler.ImportFromPDF)

	req := createMultipartPDFRequest(t, "file", []byte("fake-pdf"), "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "PLAN_LIMIT_REACHED", errResp.ErrorCode)
}

func TestImportFromPDF_Handler_ServiceError(t *testing.T) {
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, _ *model.ResumeBuilder) error {
				return errors.New("db error")
			},
		},
		&mockImportResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return &ai.ParsedResume{FullName: "Test"}, nil
			},
		},
		&mockLimitChecker{},
		func(_ []byte) (string, error) { return "text", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/pdf", importAuthMiddleware("user-1"), handler.ImportFromPDF)

	req := createMultipartPDFRequest(t, "file", []byte("fake-pdf"), "Title")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestImportFromPDF_Handler_WithTitle(t *testing.T) {
	var receivedTitle string
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				receivedTitle = rb.Title
				rb.ID = "rb-new"
				return nil
			},
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "rb-new"},
				}, nil
			},
		},
		&mockImportResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return &ai.ParsedResume{FullName: "Jane"}, nil
			},
		},
		&mockLimitChecker{},
		func(_ []byte) (string, error) { return "text", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/pdf", importAuthMiddleware("user-1"), handler.ImportFromPDF)

	req := createMultipartPDFRequest(t, "file", []byte("pdf"), "Custom PDF Title")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "Custom PDF Title", receivedTitle)
}

// --- handleError Tests ---

func TestImportHandleError_NotFound(t *testing.T) {
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{},
		&mockImportResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return nil, model.ErrResumeBuilderNotFound
			},
		},
		&mockLimitChecker{},
		func(_ []byte) (string, error) { return "", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"text":"some text"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "RESUME_BUILDER_NOT_FOUND", errResp.ErrorCode)
}

func TestImportHandleError_NotOwner(t *testing.T) {
	svc := service.NewImportServiceWithDeps(
		&mockResumeBuilderRepository{},
		&mockImportResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return nil, model.ErrNotOwner
			},
		},
		&mockLimitChecker{},
		func(_ []byte) (string, error) { return "", nil },
	)
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	router.POST("/import/text", importAuthMiddleware("user-1"), handler.ImportFromText)

	body := `{"text":"some text"}`
	req, _ := http.NewRequest(http.MethodPost, "/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Equal(t, "NOT_OWNER", errResp.ErrorCode)
}

// --- RegisterRoutes Test ---

func TestImportRegisterRoutes(t *testing.T) {
	svc := defaultImportTestService()
	handler := NewImportHandler(svc)

	router := importSetupRouter()
	group := router.Group("/api")
	noopMw := func(c *gin.Context) { c.Next() }
	handler.RegisterRoutes(group, importAuthMiddleware("user-1"), noopMw)

	// Test text import route
	body := `{"text":"some resume text"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/resume-builder/import/text", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Test PDF import route
	req2 := createMultipartPDFRequest(t, "file", []byte("pdf-content"), "")
	req2.URL.Path = "/api/resume-builder/import/pdf"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusCreated, w2.Code)
}
