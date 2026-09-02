package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockAutofillProfileRepository implements ports.AutofillProfileRepository.
type MockAutofillProfileRepository struct {
	GetFunc func(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error)
}

func (m *MockAutofillProfileRepository) Get(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, userID, resumeID)
	}
	return nil, nil
}

func (m *MockAutofillProfileRepository) Upsert(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
	return true, nil
}

func (m *MockAutofillProfileRepository) InvalidateByResume(ctx context.Context, resumeID string) error {
	return nil
}

type stubParser struct{}

func (stubParser) ParseResumeText(ctx context.Context, text string) (*ai.ParsedResume, error) {
	return &ai.ParsedResume{FullName: "Stub"}, nil
}

type stubPlanChecker struct {
	paidErr  error
	limitErr error
}

func (s *stubPlanChecker) RequirePaidPlan(ctx context.Context, userID string) error { return s.paidErr }
func (s *stubPlanChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	return s.limitErr
}
func (s *stubPlanChecker) RecordResumeAutofillUsage(ctx context.Context, userID string) error {
	return nil
}

func autofillGet(t *testing.T, svc *service.AutofillProfileService, userID string) *httptest.ResponseRecorder {
	t.Helper()
	router := setupTestRouter()
	hdl := NewAutofillProfileHandler(svc, zap.NewNop())
	hdl.RegisterRoutes(&router.RouterGroup, mockAuthMiddleware(userID), func(c *gin.Context) { c.Next() })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resumes/resume-1/autofill-profile", nil)
	router.ServeHTTP(w, req)
	return w
}

func s3TestResume(key string) *model.Resume {
	return &model.Resume{
		ID:          "resume-1",
		UserID:      "user-123",
		StorageType: model.StorageTypeS3,
		StorageKey:  &key,
	}
}

func TestAutofillProfileHandler_Get(t *testing.T) {
	userID := "user-123"

	resumeRepo := &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
			return s3TestResume("k"), nil
		},
	}

	t.Run("200 with cached profile", func(t *testing.T) {
		profileRepo := &MockAutofillProfileRepository{
			GetFunc: func(ctx context.Context, uid, rid string) (*ai.ParsedResume, error) {
				return &ai.ParsedResume{FullName: "Jane Dev", Email: "jane@dev.io"}, nil
			},
		}
		svc := service.NewAutofillProfileService(resumeRepo, profileRepo, nil, stubParser{}, &stubPlanChecker{})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusOK, w.Code)
		var dto model.AutofillProfileDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "Jane Dev", dto.Contact.FullName)
	})

	t.Run("403 PAID_FEATURE for free plan", func(t *testing.T) {
		svc := service.NewAutofillProfileService(resumeRepo, &MockAutofillProfileRepository{}, nil, stubParser{}, &stubPlanChecker{paidErr: subModel.ErrPaidFeature})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "PAID_FEATURE")
	})

	t.Run("403 PLAN_LIMIT_REACHED when AI quota exhausted", func(t *testing.T) {
		svc := service.NewAutofillProfileService(resumeRepo, &MockAutofillProfileRepository{}, nil, stubParser{}, &stubPlanChecker{limitErr: subModel.ErrLimitReached})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "PLAN_LIMIT_REACHED")
	})

	t.Run("404 for foreign or missing resume", func(t *testing.T) {
		notFoundRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}
		svc := service.NewAutofillProfileService(notFoundRepo, &MockAutofillProfileRepository{}, nil, stubParser{}, &stubPlanChecker{})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "RESUME_NOT_FOUND")
	})

	t.Run("422 RESUME_UNREADABLE for a garbage file", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("not a pdf at all")})
		defer cleanup()

		svc := service.NewAutofillProfileService(resumeRepo, &MockAutofillProfileRepository{}, s3Client, stubParser{}, &stubPlanChecker{})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "RESUME_UNREADABLE")
	})

	t.Run("422 RESUME_FILE_MISSING when the resume has no file", func(t *testing.T) {
		noFileRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return &model.Resume{ID: "resume-1", UserID: userID, StorageType: model.StorageTypeExternal}, nil
			},
		}
		svc := service.NewAutofillProfileService(noFileRepo, &MockAutofillProfileRepository{}, nil, stubParser{}, &stubPlanChecker{})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "RESUME_FILE_MISSING")
	})

	// Both handlers register under /resumes/:id — gin panics at registration
	// time on param-name conflicts, so co-registering here guards the real
	// main.go wiring.
	t.Run("coexists with the resume routes on one router", func(t *testing.T) {
		router := setupTestRouter()
		resumeSvc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
		NewResumeHandler(resumeSvc, zap.NewNop()).RegisterRoutes(&router.RouterGroup, mockAuthMiddleware(userID))

		svc := service.NewAutofillProfileService(resumeRepo, &MockAutofillProfileRepository{
			GetFunc: func(ctx context.Context, uid, rid string) (*ai.ParsedResume, error) {
				return &ai.ParsedResume{FullName: "Jane"}, nil
			},
		}, nil, stubParser{}, &stubPlanChecker{})
		NewAutofillProfileHandler(svc, zap.NewNop()).RegisterRoutes(&router.RouterGroup, mockAuthMiddleware(userID), func(c *gin.Context) { c.Next() })

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/resumes/resume-1/autofill-profile", nil))
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("422 RESUME_FILE_MISSING for an abandoned S3 upload slot", func(t *testing.T) {
		// GenerateUploadURL creates the row before the browser PUTs the file —
		// a missing S3 object must read as "no file", not an internal error.
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
		defer cleanup()

		svc := service.NewAutofillProfileService(resumeRepo, &MockAutofillProfileRepository{}, s3Client, stubParser{}, &stubPlanChecker{})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		assert.Contains(t, w.Body.String(), "RESUME_FILE_MISSING")
	})

	t.Run("500 without internals when the profile cache read fails", func(t *testing.T) {
		profileRepo := &MockAutofillProfileRepository{
			GetFunc: func(ctx context.Context, uid, rid string) (*ai.ParsedResume, error) {
				return nil, errors.New("connection reset by postgres at 10.0.0.5")
			},
		}
		svc := service.NewAutofillProfileService(resumeRepo, profileRepo, nil, stubParser{}, &stubPlanChecker{})

		w := autofillGet(t, svc, userID)

		require.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "AUTOFILL_PROFILE_FAILED")
		assert.NotContains(t, w.Body.String(), "postgres", "db internals must not leak to clients")
	})
}
