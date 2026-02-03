package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
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

func TestCompanyService_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates company successfully", func(t *testing.T) {
		expectedDTO := &model.CompanyDTO{
			ID:   "company-1",
			Name: "Test Company",
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

		svc := NewCompanyService(mockRepo)
		req := &model.CreateCompanyRequest{Name: "Test Company"}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, expectedDTO.ID, result.ID)
		assert.Equal(t, expectedDTO.Name, result.Name)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{}
		svc := NewCompanyService(mockRepo)
		req := &model.CreateCompanyRequest{Name: "   "}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrCompanyNameRequired, err)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")
		mockRepo := &MockCompanyRepository{
			CreateFunc: func(ctx context.Context, company *model.Company) error {
				return expectedError
			},
		}

		svc := NewCompanyService(mockRepo)
		req := &model.CreateCompanyRequest{Name: "Test Company"}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})

	t.Run("trims whitespace from name", func(t *testing.T) {
		var createdCompany *model.Company

		mockRepo := &MockCompanyRepository{
			CreateFunc: func(ctx context.Context, company *model.Company) error {
				createdCompany = company
				company.ID = "company-1"
				return nil
			},
			GetByIDEnrichedFunc: func(ctx context.Context, uid, companyID string) (*model.CompanyDTO, error) {
				return &model.CompanyDTO{ID: "company-1", Name: "Test Company"}, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		req := &model.CreateCompanyRequest{Name: "  Test Company  "}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "Test Company", createdCompany.Name)
	})
}

func TestCompanyService_GetByID(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("returns company successfully", func(t *testing.T) {
		expectedDTO := &model.CompanyDTO{
			ID:   companyID,
			Name: "Test Company",
		}

		mockRepo := &MockCompanyRepository{
			GetByIDEnrichedFunc: func(ctx context.Context, uid, cid string) (*model.CompanyDTO, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, companyID, cid)
				return expectedDTO, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		result, err := svc.GetByID(context.Background(), userID, companyID)

		require.NoError(t, err)
		assert.Equal(t, expectedDTO, result)
	})

	t.Run("returns error when company not found", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetByIDEnrichedFunc: func(ctx context.Context, uid, cid string) (*model.CompanyDTO, error) {
				return nil, model.ErrCompanyNotFound
			},
		}

		svc := NewCompanyService(mockRepo)
		result, err := svc.GetByID(context.Background(), userID, companyID)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrCompanyNotFound, err)
	})
}

func TestCompanyService_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns companies successfully", func(t *testing.T) {
		expectedCompanies := []*model.CompanyDTO{
			{ID: "company-1", Name: "Company A"},
			{ID: "company-2", Name: "Company B"},
		}

		mockRepo := &MockCompanyRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 20, opts.Limit)
				assert.Equal(t, 0, opts.Offset)
				return expectedCompanies, 2, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		opts := &ports.ListOptions{Limit: 20, Offset: 0}

		result, total, err := svc.List(context.Background(), userID, opts)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
				return []*model.CompanyDTO{}, 0, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		opts := &ports.ListOptions{Limit: 20, Offset: 0}

		result, total, err := svc.List(context.Background(), userID, opts)

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 0, total)
	})
}

func TestCompanyService_Update(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("updates company successfully", func(t *testing.T) {
		existingCompany := &model.Company{
			ID:        companyID,
			UserID:    userID,
			Name:      "Old Name",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		newName := "New Name"
		expectedDTO := &model.CompanyDTO{
			ID:   companyID,
			Name: newName,
		}

		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return existingCompany, nil
			},
			UpdateFunc: func(ctx context.Context, company *model.Company) error {
				return nil
			},
			GetByIDEnrichedFunc: func(ctx context.Context, uid, cid string) (*model.CompanyDTO, error) {
				return expectedDTO, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		req := &model.UpdateCompanyRequest{Name: &newName}

		result, err := svc.Update(context.Background(), userID, companyID, req)

		require.NoError(t, err)
		assert.Equal(t, newName, result.Name)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		existingCompany := &model.Company{
			ID:     companyID,
			UserID: userID,
			Name:   "Old Name",
		}

		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return existingCompany, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		emptyName := "   "
		req := &model.UpdateCompanyRequest{Name: &emptyName}

		result, err := svc.Update(context.Background(), userID, companyID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrCompanyNameRequired, err)
	})

	t.Run("returns error when company not found", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return nil, model.ErrCompanyNotFound
			},
		}

		svc := NewCompanyService(mockRepo)
		newName := "New Name"
		req := &model.UpdateCompanyRequest{Name: &newName}

		result, err := svc.Update(context.Background(), userID, companyID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrCompanyNotFound, err)
	})
}

func TestCompanyService_Delete(t *testing.T) {
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

		svc := NewCompanyService(mockRepo)
		err := svc.Delete(context.Background(), userID, companyID)

		require.NoError(t, err)
	})

	t.Run("returns error when company not found", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*model.Company, error) {
				return nil, model.ErrCompanyNotFound
			},
		}

		svc := NewCompanyService(mockRepo)
		err := svc.Delete(context.Background(), userID, companyID)

		assert.Equal(t, model.ErrCompanyNotFound, err)
	})
}

func TestCompanyService_GetRelatedJobsAndApplicationsCount(t *testing.T) {
	userID := "user-123"
	companyID := "company-1"

	t.Run("returns counts successfully", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetRelatedJobsAndApplicationsCountFunc: func(ctx context.Context, uid, cid string) (int, int, error) {
				return 5, 10, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		jobsCount, appsCount, err := svc.GetRelatedJobsAndApplicationsCount(context.Background(), userID, companyID)

		require.NoError(t, err)
		assert.Equal(t, 5, jobsCount)
		assert.Equal(t, 10, appsCount)
	})

	t.Run("returns zero counts", func(t *testing.T) {
		mockRepo := &MockCompanyRepository{
			GetRelatedJobsAndApplicationsCountFunc: func(ctx context.Context, uid, cid string) (int, int, error) {
				return 0, 0, nil
			},
		}

		svc := NewCompanyService(mockRepo)
		jobsCount, appsCount, err := svc.GetRelatedJobsAndApplicationsCount(context.Background(), userID, companyID)

		require.NoError(t, err)
		assert.Equal(t, 0, jobsCount)
		assert.Equal(t, 0, appsCount)
	})
}
