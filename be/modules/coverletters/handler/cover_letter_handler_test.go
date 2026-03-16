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

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers ---

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

func parseHandlerErrorResponse(t *testing.T, w *httptest.ResponseRecorder) httpPlatform.ErrorResponse {
	t.Helper()
	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err, "failed to parse error response body")
	return errResp
}

// --- Mock repository ---

type mockCoverLetterRepo struct {
	CreateFunc  func(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	GetByIDFunc func(ctx context.Context, id string) (*model.CoverLetter, error)
	ListFunc    func(ctx context.Context, userID string) ([]*model.CoverLetter, error)
	UpdateFunc  func(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *mockCoverLetterRepo) Create(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, cl)
	}
	return cl, nil
}

func (m *mockCoverLetterRepo) GetByID(ctx context.Context, id string) (*model.CoverLetter, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockCoverLetterRepo) List(ctx context.Context, userID string) ([]*model.CoverLetter, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockCoverLetterRepo) Update(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, cl)
	}
	return cl, nil
}

func (m *mockCoverLetterRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// --- Mock limit checker ---

type mockLimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *mockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

// --- Helper builders ---

func newHandlerTestService(repo *mockCoverLetterRepo) *service.CoverLetterService {
	return service.NewCoverLetterService(repo, &mockLimitChecker{})
}

func newHandlerTestServiceWithLimit(repo *mockCoverLetterRepo, lc *mockLimitChecker) *service.CoverLetterService {
	return service.NewCoverLetterService(repo, lc)
}

func newTestCoverLetterEntity() *model.CoverLetter {
	return &model.CoverLetter{
		ID:             "cl-1",
		UserID:         "user-1",
		Title:          "Test Cover Letter",
		Template:       "professional",
		RecipientName:  "Jane Smith",
		RecipientTitle: "Hiring Manager",
		CompanyName:    "Acme Corp",
		CompanyAddress: "123 Main St",
		Greeting:       "Dear Hiring Manager,",
		Paragraphs:     []string{"First paragraph.", "Second paragraph."},
		Closing:        "Sincerely,",
		FontFamily:     "Georgia",
		FontSize:       12,
		PrimaryColor:   "#2563eb",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// --- Duplicate endpoint tests ---

func TestDuplicate_Success(t *testing.T) {
	original := newTestCoverLetterEntity()
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return original, nil
		},
		CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
			cl.ID = "cl-copy"
			cl.CreatedAt = time.Now()
			cl.UpdatedAt = time.Now()
			return cl, nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.POST("/cover-letters/:id/duplicate", mockAuthMiddleware("user-1"), handler.Duplicate)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/duplicate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var body model.CoverLetterDTO
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "cl-copy", body.ID)
	assert.Contains(t, body.Title, "(Copy)")
}

func TestDuplicate_Unauthorized(t *testing.T) {
	repo := &mockCoverLetterRepo{}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.POST("/cover-letters/:id/duplicate", handler.Duplicate) // no auth middleware

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/duplicate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "UNAUTHORIZED", errResp.ErrorCode)
}

func TestDuplicate_PlanLimitReached(t *testing.T) {
	repo := &mockCoverLetterRepo{}
	lc := &mockLimitChecker{
		CheckLimitFunc: func(_ context.Context, _, _ string) error {
			return subModel.ErrLimitReached
		},
	}
	svc := newHandlerTestServiceWithLimit(repo, lc)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.POST("/cover-letters/:id/duplicate", mockAuthMiddleware("user-1"), handler.Duplicate)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/duplicate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "PLAN_LIMIT_REACHED", errResp.ErrorCode)
}

func TestDuplicate_NotFound(t *testing.T) {
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return nil, model.ErrCoverLetterNotFound
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.POST("/cover-letters/:id/duplicate", mockAuthMiddleware("user-1"), handler.Duplicate)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/nonexistent/duplicate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "COVER_LETTER_NOT_FOUND", errResp.ErrorCode)
}

func TestDuplicate_WrongOwner(t *testing.T) {
	original := newTestCoverLetterEntity() // UserID = "user-1"
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return original, nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.POST("/cover-letters/:id/duplicate", mockAuthMiddleware("user-2"), handler.Duplicate)

	req, _ := http.NewRequest(http.MethodPost, "/cover-letters/cl-1/duplicate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseHandlerErrorResponse(t, w)
	assert.Equal(t, "NOT_AUTHORIZED", errResp.ErrorCode)
}

// --- CRUD endpoint tests ---

func TestCreate_Success(t *testing.T) {
	repo := &mockCoverLetterRepo{
		CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
			cl.ID = "new-cl-1"
			cl.CreatedAt = time.Now()
			cl.UpdatedAt = time.Now()
			return cl, nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.POST("/cover-letters", mockAuthMiddleware("user-1"), handler.Create)

	body, _ := json.Marshal(model.CreateCoverLetterRequest{Title: "New CL"})
	req, _ := http.NewRequest(http.MethodPost, "/cover-letters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp model.CoverLetterDTO
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "new-cl-1", resp.ID)
	assert.Equal(t, "New CL", resp.Title)
}

func TestList_Success(t *testing.T) {
	repo := &mockCoverLetterRepo{
		ListFunc: func(_ context.Context, _ string) ([]*model.CoverLetter, error) {
			return []*model.CoverLetter{
				{ID: "cl-1", UserID: "user-1", Title: "CL 1", Paragraphs: []string{}},
				{ID: "cl-2", UserID: "user-1", Title: "CL 2", Paragraphs: []string{}},
			}, nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.GET("/cover-letters", mockAuthMiddleware("user-1"), handler.List)

	req, _ := http.NewRequest(http.MethodGet, "/cover-letters", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []model.CoverLetterDTO
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
}

func TestGet_Success(t *testing.T) {
	cl := newTestCoverLetterEntity()
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.GET("/cover-letters/:id", mockAuthMiddleware("user-1"), handler.Get)

	req, _ := http.NewRequest(http.MethodGet, "/cover-letters/cl-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.CoverLetterDTO
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "cl-1", resp.ID)
}

func TestUpdate_Success(t *testing.T) {
	cl := newTestCoverLetterEntity()
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
		UpdateFunc: func(_ context.Context, updated *model.CoverLetter) (*model.CoverLetter, error) {
			return updated, nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.PATCH("/cover-letters/:id", mockAuthMiddleware("user-1"), handler.Update)

	newTitle := "Updated Title"
	body, _ := json.Marshal(model.UpdateCoverLetterRequest{Title: &newTitle})
	req, _ := http.NewRequest(http.MethodPatch, "/cover-letters/cl-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.CoverLetterDTO
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", resp.Title)
}

func TestDelete_Success(t *testing.T) {
	cl := newTestCoverLetterEntity()
	repo := &mockCoverLetterRepo{
		GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
			return cl, nil
		},
		DeleteFunc: func(_ context.Context, _ string) error {
			return nil
		},
	}
	svc := newHandlerTestService(repo)
	handler := NewCoverLetterHandler(svc)

	router := setupTestRouter()
	router.DELETE("/cover-letters/:id", mockAuthMiddleware("user-1"), handler.Delete)

	req, _ := http.NewRequest(http.MethodDelete, "/cover-letters/cl-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- handleError tests ---

func TestHandleError_MapsErrorCodesToStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "cover letter not found",
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
			name:           "plan limit reached",
			err:            subModel.ErrLimitReached,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "PLAN_LIMIT_REACHED",
		},
		{
			name:           "invalid font family",
			err:            errors.New("invalid font family"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "invalid color format",
			err:            errors.New("invalid color format"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "unknown error maps to 500",
			err:            errors.New("some unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &CoverLetterHandler{}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

			handler.handleError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			errResp := parseHandlerErrorResponse(t, w)
			assert.Equal(t, tt.expectedCode, errResp.ErrorCode)
		})
	}
}
