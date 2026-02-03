package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/companies/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCompanyRepository implements ports.CompanyRepository
type MockCompanyRepository struct {
	CreateFunc                            func(ctx context.Context, company *model.Company) error
	GetByIDFunc                           func(ctx context.Context, userID, companyID string) (*model.Company, error)
	GetByIDEnrichedFunc                   func(ctx context.Context, userID, companyID string) (*model.CompanyDTO, error)
	ListFunc                              func(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error)
	UpdateFunc                            func(ctx context.Context, company *model.Company) error
	DeleteFunc                            func(ctx context.Context, userID, companyID string) error
	GetRelatedJobsAndApplicationsCountFunc func(ctx context.Context, userID, companyID string) (jobsCount, appsCount int, err error)
}

func (m *MockCompanyRepository) Create(ctx context.Context, company *model.Company) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, company)
	}
	return nil
}

func (m *MockCompanyRepository) GetByID(ctx context.Context, userID, companyID string) (*model.Company, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, companyID)
	}
	return nil, nil
}

func (m *MockCompanyRepository) GetByIDEnriched(ctx context.Context, userID, companyID string) (*model.CompanyDTO, error) {
	if m.GetByIDEnrichedFunc != nil {
		return m.GetByIDEnrichedFunc(ctx, userID, companyID)
	}
	return nil, nil
}

func (m *MockCompanyRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, opts)
	}
	return nil, 0, nil
}

func (m *MockCompanyRepository) Update(ctx context.Context, company *model.Company) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, company)
	}
	return nil
}

func (m *MockCompanyRepository) Delete(ctx context.Context, userID, companyID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, companyID)
	}
	return nil
}

func (m *MockCompanyRepository) GetRelatedJobsAndApplicationsCount(ctx context.Context, userID, companyID string) (jobsCount, appsCount int, err error) {
	if m.GetRelatedJobsAndApplicationsCountFunc != nil {
		return m.GetRelatedJobsAndApplicationsCountFunc(ctx, userID, companyID)
	}
	return 0, 0, nil
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

func TestCompanyHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates company successfully", func(t *testing.T) {
		expectedDTO := &model.CompanyDTO{
			ID:        "company-1",
			Name:      "Test Company",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo := &MockCompanyRepository{
			CreateFunc: func(ctx context.Context, company *model.Company) error {
				company.ID = "company-1"
				return nil
			},
			GetByIDEnrichedFunc: func(ctx context.Context, uid, companyID string) (*model.CompanyDTO, error) {
				return expectedDTO, nil
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.POST("/companies", mockAuthMiddleware(userID), handler.Create)

		body := `{"name":"Test Company"}`
		req, _ := http.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.CompanyDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Test Company", response.Name)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{}
		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.POST("/companies", handler.Create) // No auth middleware

		body := `{"name":"Test Company"}`
		req, _ := http.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{}
		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.POST("/companies", mockAuthMiddleware(userID), handler.Create)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for empty name", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{}
		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.POST("/companies", mockAuthMiddleware(userID), handler.Create)

		body := `{"name":"   "}`
		req, _ := http.NewRequest(http.MethodPost, "/companies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCompanyHandler_Get(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("returns company successfully", func(t *testing.T) {
		expectedDTO := &model.CompanyDTO{
			ID:   companyID,
			Name: "Test Company",
		}

		mockRepo := &MockCompanyRepository{
			GetByIDEnrichedFunc: func(ctx context.Context, uid, cid string) (*model.CompanyDTO, error) {
				return expectedDTO, nil
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.GET("/companies/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/companies/"+companyID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.CompanyDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedDTO.Name, response.Name)
	})

	t.Run("returns 404 when company not found", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetByIDEnrichedFunc: func(ctx context.Context, uid, cid string) (*model.CompanyDTO, error) {
				return nil, model.ErrCompanyNotFound
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.GET("/companies/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/companies/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCompanyHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns companies list", func(t *testing.T) {
		expectedCompanies := []*model.CompanyDTO{
			{ID: "company-1", Name: "Company A"},
			{ID: "company-2", Name: "Company B"},
		}

		mockRepo := &MockCompanyRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
				return expectedCompanies, 2, nil
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.GET("/companies", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/companies?limit=20&offset=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestCompanyHandler_Update(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("updates company successfully", func(t *testing.T) {
		existingCompany := &model.Company{
			ID:     companyID,
			UserID: userID,
			Name:   "Old Name",
		}

		updatedDTO := &model.CompanyDTO{
			ID:   companyID,
			Name: "New Name",
		}

		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return existingCompany, nil
			},
			UpdateFunc: func(ctx context.Context, company *model.Company) error {
				return nil
			},
			GetByIDEnrichedFunc: func(ctx context.Context, uid, cid string) (*model.CompanyDTO, error) {
				return updatedDTO, nil
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.PATCH("/companies/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"name":"New Name"}`
		req, _ := http.NewRequest(http.MethodPatch, "/companies/"+companyID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when company not found", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return nil, model.ErrCompanyNotFound
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.PATCH("/companies/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"name":"New Name"}`
		req, _ := http.NewRequest(http.MethodPatch, "/companies/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCompanyHandler_Delete(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("deletes company successfully", func(t *testing.T) {
		existingCompany := &model.Company{
			ID:     companyID,
			UserID: userID,
			Name:   "Test Company",
		}

		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return existingCompany, nil
			},
			DeleteFunc: func(ctx context.Context, uid, cid string) error {
				return nil
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.DELETE("/companies/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/companies/"+companyID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when company not found", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return nil, model.ErrCompanyNotFound
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.DELETE("/companies/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/companies/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCompanyHandler_GetRelatedCounts(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("returns related counts", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetRelatedJobsAndApplicationsCountFunc: func(ctx context.Context, uid, cid string) (int, int, error) {
				return 3, 5, nil
			},
		}

		svc := service.NewCompanyService(mockRepo)
		handler := NewCompanyHandler(svc)

		router := setupTestRouter()
		router.GET("/companies/:id/related-counts", mockAuthMiddleware(userID), handler.GetRelatedCounts)

		req, _ := http.NewRequest(http.MethodGet, "/companies/"+companyID+"/related-counts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]int
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 3, response["jobs_count"])
		assert.Equal(t, 5, response["applications_count"])
	})
}

func TestCompanyHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockCompanyRepository{
		CreateFunc: func(ctx context.Context, company *model.Company) error {
			company.ID = "company-1"
			return nil
		},
		GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
			return &model.Company{ID: cid, UserID: uid, Name: "Test"}, nil
		},
		GetByIDEnrichedFunc: func(ctx context.Context, uid, companyID string) (*model.CompanyDTO, error) {
			return &model.CompanyDTO{ID: "company-1", Name: "Test"}, nil
		},
		ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
			return []*model.CompanyDTO{}, 0, nil
		},
		UpdateFunc: func(ctx context.Context, company *model.Company) error {
			return nil
		},
		DeleteFunc: func(ctx context.Context, uid, cid string) error {
			return nil
		},
		GetRelatedJobsAndApplicationsCountFunc: func(ctx context.Context, uid, cid string) (int, int, error) {
			return 0, 0, nil
		},
	}

	svc := service.NewCompanyService(mockRepo)
	handler := NewCompanyHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/companies"},
		{http.MethodGet, "/api/v1/companies"},
		{http.MethodGet, "/api/v1/companies/test-id"},
		{http.MethodGet, "/api/v1/companies/test-id/related-counts"},
		{http.MethodPatch, "/api/v1/companies/test-id"},
		{http.MethodDelete, "/api/v1/companies/test-id"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == http.MethodPost || route.method == http.MethodPatch {
				body = bytes.NewBufferString(`{"name":"Test"}`)
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
