package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockResumeRepository implements ports.ResumeRepository
type MockResumeRepository struct {
	CreateFunc  func(ctx context.Context, resume *model.Resume) error
	GetByIDFunc func(ctx context.Context, userID, resumeID string) (*model.Resume, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error)
	UpdateFunc  func(ctx context.Context, resume *model.Resume) error
	DeleteFunc  func(ctx context.Context, userID, resumeID string) error
}

func (m *MockResumeRepository) Create(ctx context.Context, resume *model.Resume) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, resume)
	}
	return nil
}

func (m *MockResumeRepository) GetByID(ctx context.Context, userID, resumeID string) (*model.Resume, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, resumeID)
	}
	return nil, nil
}

func (m *MockResumeRepository) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset, sortBy, sortDir)
	}
	return nil, 0, nil
}

func (m *MockResumeRepository) Update(ctx context.Context, resume *model.Resume) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, resume)
	}
	return nil
}

func (m *MockResumeRepository) Delete(ctx context.Context, userID, resumeID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, resumeID)
	}
	return nil
}

func TestResumeService_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates resume successfully", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			CreateFunc: func(ctx context.Context, resume *model.Resume) error {
				resume.ID = "resume-1"
				resume.CreatedAt = time.Now()
				resume.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		req := &model.CreateResumeRequest{
			Title: "Software Engineer Resume",
		}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "resume-1", result.ID)
		assert.Equal(t, "Software Engineer Resume", result.Title)
		assert.True(t, result.IsActive)
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		mockRepo := &MockResumeRepository{}
		svc := NewResumeService(mockRepo, nil, nil, nil)
		req := &model.CreateResumeRequest{Title: "   "}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrResumeTitleRequired, err)
	})

	t.Run("creates resume with file URL", func(t *testing.T) {
		var createdResume *model.Resume
		fileURL := "https://example.com/resume.pdf"

		mockRepo := &MockResumeRepository{
			CreateFunc: func(ctx context.Context, resume *model.Resume) error {
				createdResume = resume
				resume.ID = "resume-1"
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		req := &model.CreateResumeRequest{
			Title:   "Resume with URL",
			FileURL: &fileURL,
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, &fileURL, createdResume.FileURL)
		assert.Equal(t, model.StorageTypeExternal, createdResume.StorageType)
	})

	t.Run("creates resume with isActive false", func(t *testing.T) {
		var createdResume *model.Resume
		isActive := false

		mockRepo := &MockResumeRepository{
			CreateFunc: func(ctx context.Context, resume *model.Resume) error {
				createdResume = resume
				resume.ID = "resume-1"
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		req := &model.CreateResumeRequest{
			Title:    "Inactive Resume",
			IsActive: &isActive,
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.False(t, createdResume.IsActive)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockResumeRepository{
			CreateFunc: func(ctx context.Context, resume *model.Resume) error {
				return expectedError
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		req := &model.CreateResumeRequest{Title: "Test Resume"}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestResumeService_GetByID(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("returns resume successfully", func(t *testing.T) {
		expectedResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Software Engineer Resume",
			IsActive:    true,
			StorageType: model.StorageTypeExternal,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, resumeID, rid)
				return expectedResume, nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		result, err := svc.GetByID(context.Background(), userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, expectedResume.ID, result.ID)
		assert.Equal(t, expectedResume.Title, result.Title)
	})

	t.Run("returns error when resume not found", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		result, err := svc.GetByID(context.Background(), userID, resumeID)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrResumeNotFound, err)
	})
}

func TestResumeService_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns resumes list with counts", func(t *testing.T) {
		expectedResumes := []*ports.ResumeWithCount{
			{
				Resume:            &model.Resume{ID: "resume-1", Title: "Resume A", IsActive: true, StorageType: model.StorageTypeExternal},
				ApplicationsCount: 5,
			},
			{
				Resume:            &model.Resume{ID: "resume-2", Title: "Resume B", IsActive: false, StorageType: model.StorageTypeExternal},
				ApplicationsCount: 3,
			},
		}

		mockRepo := &MockResumeRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
				return expectedResumes, 2, nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		result, total, err := svc.List(context.Background(), userID, 20, 0, "", "")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, 5, result[0].ApplicationsCount)
		assert.Equal(t, 3, result[1].ApplicationsCount)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
				return []*ports.ResumeWithCount{}, 0, nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		result, total, err := svc.List(context.Background(), userID, 20, 0, "", "")

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 0, total)
	})

	t.Run("uses default sort parameters", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
				assert.Equal(t, "created_at", sortBy)
				assert.Equal(t, "desc", sortDir)
				return []*ports.ResumeWithCount{}, 0, nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		_, _, err := svc.List(context.Background(), userID, 20, 0, "", "")

		require.NoError(t, err)
	})
}

func TestResumeService_Update(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("updates resume successfully", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Old Title",
			IsActive:    true,
			StorageType: model.StorageTypeExternal,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error {
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		newTitle := "New Title"
		req := &model.UpdateResumeRequest{Title: &newTitle}

		result, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:     resumeID,
			UserID: userID,
			Title:  "Old Title",
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		emptyTitle := "   "
		req := &model.UpdateResumeRequest{Title: &emptyTitle}

		result, err := svc.Update(context.Background(), userID, resumeID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrResumeTitleRequired, err)
	})

	t.Run("updates file URL", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:     resumeID,
			UserID: userID,
			Title:  "Resume",
		}

		var updatedResume *model.Resume

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error {
				updatedResume = resume
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		newURL := "https://example.com/new-resume.pdf"
		req := &model.UpdateResumeRequest{FileURL: &newURL}

		_, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.Equal(t, &newURL, updatedResume.FileURL)
	})

	t.Run("clears file URL when empty string", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:     resumeID,
			UserID: userID,
			Title:  "Resume",
		}

		var updatedResume *model.Resume

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error {
				updatedResume = resume
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		emptyURL := ""
		req := &model.UpdateResumeRequest{FileURL: &emptyURL}

		_, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.Nil(t, updatedResume.FileURL)
	})

	t.Run("returns error when resume not found", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		newTitle := "New Title"
		req := &model.UpdateResumeRequest{Title: &newTitle}

		result, err := svc.Update(context.Background(), userID, resumeID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrResumeNotFound, err)
	})
}

func TestResumeService_Delete(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("deletes resume successfully", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Test Resume",
			StorageType: model.StorageTypeExternal,
		}

		var deletedResumeID string

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error {
				deletedResumeID = rid
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		err := svc.Delete(context.Background(), userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, resumeID, deletedResumeID)
	})

	t.Run("returns error when resume not found", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		err := svc.Delete(context.Background(), userID, resumeID)

		assert.Equal(t, model.ErrResumeNotFound, err)
	})
}

func TestResume_ToDTO(t *testing.T) {
	now := time.Now()
	fileURL := "https://example.com/resume.pdf"

	resume := &model.Resume{
		ID:          "resume-1",
		UserID:      "user-123",
		Title:       "Software Engineer Resume",
		FileURL:     &fileURL,
		StorageType: model.StorageTypeExternal,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dto := resume.ToDTO()

	assert.Equal(t, resume.ID, dto.ID)
	assert.Equal(t, resume.Title, dto.Title)
	assert.Equal(t, resume.FileURL, dto.FileURL)
	assert.Equal(t, resume.StorageType, dto.StorageType)
	assert.Equal(t, resume.IsActive, dto.IsActive)
	assert.Equal(t, 0, dto.ApplicationsCount)
	assert.True(t, dto.CanDelete)
}

func TestResume_ToDTOWithCounts(t *testing.T) {
	now := time.Now()

	resume := &model.Resume{
		ID:          "resume-1",
		UserID:      "user-123",
		Title:       "Software Engineer Resume",
		StorageType: model.StorageTypeExternal,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Test with applications
	dto := resume.ToDTOWithCounts(5)
	assert.Equal(t, 5, dto.ApplicationsCount)
	assert.True(t, dto.CanDelete) // Always deletable — deletion NULLs FK references in applications

	// Test without applications
	dto = resume.ToDTOWithCounts(0)
	assert.Equal(t, 0, dto.ApplicationsCount)
	assert.True(t, dto.CanDelete) // Can delete if no applications
}

func TestResumeService_Update_IsActive(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("updates isActive status", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Resume",
			IsActive:    true,
			StorageType: model.StorageTypeExternal,
		}

		var updatedResume *model.Resume

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error {
				updatedResume = resume
				return nil
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		isActive := false
		req := &model.UpdateResumeRequest{IsActive: &isActive}

		result, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.False(t, result.IsActive)
		assert.False(t, updatedResume.IsActive)
	})
}

// MockCacheInvalidator implements CacheInvalidator for testing
type MockCacheInvalidator struct {
	InvalidateByResumeFunc func(ctx context.Context, resumeID string) error
	CalledWith             string
	CallCount              int
}

func (m *MockCacheInvalidator) InvalidateByResume(ctx context.Context, resumeID string) error {
	m.CalledWith = resumeID
	m.CallCount++
	if m.InvalidateByResumeFunc != nil {
		return m.InvalidateByResumeFunc(ctx, resumeID)
	}
	return nil
}

func TestResumeService_Update_CacheInvalidation(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	existingResume := func() *model.Resume {
		return &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Resume",
			IsActive:    true,
			StorageType: model.StorageTypeExternal,
		}
	}

	t.Run("invalidates cache when file URL changes", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume(), nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error { return nil },
		}

		svc := NewResumeService(mockRepo, nil, nil, cache)
		newURL := "https://example.com/new.pdf"
		req := &model.UpdateResumeRequest{FileURL: &newURL}

		_, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.Equal(t, 1, cache.CallCount)
		assert.Equal(t, resumeID, cache.CalledWith)
	})

	t.Run("does not invalidate cache when only title changes", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume(), nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error { return nil },
		}

		svc := NewResumeService(mockRepo, nil, nil, cache)
		newTitle := "New Title"
		req := &model.UpdateResumeRequest{Title: &newTitle}

		_, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.Equal(t, 0, cache.CallCount)
	})

	t.Run("invalidation error does not break update", func(t *testing.T) {
		cache := &MockCacheInvalidator{
			InvalidateByResumeFunc: func(ctx context.Context, resumeID string) error {
				return errors.New("cache unavailable")
			},
		}
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume(), nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error { return nil },
		}

		svc := NewResumeService(mockRepo, nil, nil, cache)
		newURL := "https://example.com/new.pdf"
		req := &model.UpdateResumeRequest{FileURL: &newURL}

		result, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestResumeService_Delete_CacheInvalidation(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("invalidates cache before delete", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return &model.Resume{
					ID:          resumeID,
					UserID:      userID,
					StorageType: model.StorageTypeExternal,
				}, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error { return nil },
		}

		svc := NewResumeService(mockRepo, nil, nil, cache)
		err := svc.Delete(context.Background(), userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, 1, cache.CallCount)
		assert.Equal(t, resumeID, cache.CalledWith)
	})

	t.Run("invalidation error does not break delete", func(t *testing.T) {
		cache := &MockCacheInvalidator{
			InvalidateByResumeFunc: func(ctx context.Context, resumeID string) error {
				return errors.New("cache unavailable")
			},
		}
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return &model.Resume{
					ID:          resumeID,
					UserID:      userID,
					StorageType: model.StorageTypeExternal,
				}, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error { return nil },
		}

		svc := NewResumeService(mockRepo, nil, nil, cache)
		err := svc.Delete(context.Background(), userID, resumeID)

		require.NoError(t, err)
	})
}

func TestResumeService_Delete_ResumeInUse(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("returns error when resume is in use", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Test Resume",
			StorageType: model.StorageTypeExternal,
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error {
				return model.ErrResumeInUse
			},
		}

		svc := NewResumeService(mockRepo, nil, nil, nil)
		err := svc.Delete(context.Background(), userID, resumeID)

		assert.Equal(t, model.ErrResumeInUse, err)
	})
}

// --- LimitChecker tests ---

// MockResumeLimitChecker implements LimitChecker for resumes
type MockResumeLimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *MockResumeLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

func TestResumeService_Create_LimitCheckerBlocks(t *testing.T) {
	limitChecker := &MockResumeLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	mockRepo := &MockResumeRepository{}
	svc := NewResumeService(mockRepo, nil, limitChecker, nil)

	req := &model.CreateResumeRequest{Title: "Test Resume"}
	result, err := svc.Create(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

func TestResumeService_Create_LimitCheckerPasses(t *testing.T) {
	limitChecker := &MockResumeLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return nil
		},
	}

	mockRepo := &MockResumeRepository{
		CreateFunc: func(ctx context.Context, resume *model.Resume) error {
			resume.ID = "resume-1"
			return nil
		},
	}
	svc := NewResumeService(mockRepo, nil, limitChecker, nil)

	req := &model.CreateResumeRequest{Title: "Test Resume"}
	result, err := svc.Create(context.Background(), "user-123", req)

	require.NoError(t, err)
	assert.Equal(t, "resume-1", result.ID)
}

// --- GenerateUploadURL tests ---

func TestResumeService_GenerateUploadURL_S3Disabled(t *testing.T) {
	mockRepo := &MockResumeRepository{}
	svc := NewResumeService(mockRepo, nil, nil, nil) // no S3 client

	req := &model.GenerateUploadURLRequest{
		Filename:    "resume.pdf",
		ContentType: "application/pdf",
	}

	result, err := svc.GenerateUploadURL(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S3 storage is not configured")
}

func TestResumeService_GenerateUploadURL_InvalidContentType(t *testing.T) {
	// S3 check comes before content type check, so with nil S3 client
	// we get "S3 storage is not configured" error first.
	// This test verifies the early exit with no S3 and non-PDF content type.
	mockRepo := &MockResumeRepository{}
	svc := NewResumeService(mockRepo, nil, nil, nil) // no S3 client

	req := &model.GenerateUploadURLRequest{
		Filename:    "resume.docx",
		ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}

	result, err := svc.GenerateUploadURL(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	// S3 disabled check happens before content type validation
	assert.Contains(t, err.Error(), "S3 storage is not configured")
}

func TestResumeService_GenerateUploadURL_LimitCheckerBlocks(t *testing.T) {
	limitChecker := &MockResumeLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("upload limit reached")
		},
	}

	mockRepo := &MockResumeRepository{}
	svc := NewResumeService(mockRepo, nil, limitChecker, nil)

	req := &model.GenerateUploadURLRequest{
		Filename:    "resume.pdf",
		ContentType: "application/pdf",
	}

	result, err := svc.GenerateUploadURL(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, "upload limit reached", err.Error())
}

// --- GenerateDownloadURL tests ---

func TestResumeService_GenerateDownloadURL_S3Disabled(t *testing.T) {
	mockRepo := &MockResumeRepository{}
	svc := NewResumeService(mockRepo, nil, nil, nil)

	result, err := svc.GenerateDownloadURL(context.Background(), "user-123", "resume-1")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "S3 storage is not configured")
}

// --- Update edge cases ---

func TestResumeService_Update_RepoUpdateFails(t *testing.T) {
	existingResume := &model.Resume{
		ID:          "resume-1",
		UserID:      "user-123",
		Title:       "Old Title",
		StorageType: model.StorageTypeExternal,
	}

	mockRepo := &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
			return existingResume, nil
		},
		UpdateFunc: func(ctx context.Context, resume *model.Resume) error {
			return errors.New("update error")
		},
	}

	svc := NewResumeService(mockRepo, nil, nil, nil)
	newTitle := "New Title"
	req := &model.UpdateResumeRequest{Title: &newTitle}

	result, err := svc.Update(context.Background(), "user-123", "resume-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- List error path ---

func TestResumeService_List_RepoError(t *testing.T) {
	mockRepo := &MockResumeRepository{
		ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
			return nil, 0, errors.New("list error")
		},
	}

	svc := NewResumeService(mockRepo, nil, nil, nil)
	result, total, err := svc.List(context.Background(), "user-123", 20, 0, "", "")

	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
}

// --- List with custom sort ---

func TestResumeService_List_CustomSort(t *testing.T) {
	mockRepo := &MockResumeRepository{
		ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
			assert.Equal(t, "title", sortBy)
			assert.Equal(t, "asc", sortDir)
			return []*ports.ResumeWithCount{}, 0, nil
		},
	}

	svc := NewResumeService(mockRepo, nil, nil, nil)
	_, _, err := svc.List(context.Background(), "user-123", 20, 0, "title", "asc")

	require.NoError(t, err)
}

// --- Delete with cache invalidation error does not break delete ---

func TestResumeService_Delete_WithCacheInvalidator(t *testing.T) {
	t.Run("calls invalidator before delete", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		var deleteOrder []string

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return &model.Resume{
					ID:          "resume-1",
					UserID:      "user-123",
					StorageType: model.StorageTypeExternal,
				}, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error {
				deleteOrder = append(deleteOrder, "delete")
				return nil
			},
		}

		cache.InvalidateByResumeFunc = func(ctx context.Context, resumeID string) error {
			deleteOrder = append(deleteOrder, "invalidate")
			return nil
		}

		svc := NewResumeService(mockRepo, nil, nil, cache)
		err := svc.Delete(context.Background(), "user-123", "resume-1")

		require.NoError(t, err)
		assert.Equal(t, []string{"invalidate", "delete"}, deleteOrder)
	})
}

// --- GenerateDownloadURL edge cases ---

func TestResumeService_GenerateDownloadURL_ResumeNotFound(t *testing.T) {
	mockRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, _, _ string) (*model.Resume, error) {
			return nil, model.ErrResumeNotFound
		},
	}
	// Create a service with s3Enabled = true by directly setting the field
	svc := &ResumeService{repo: mockRepo, s3Enabled: true}

	result, err := svc.GenerateDownloadURL(context.Background(), "user-123", "resume-1")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, model.ErrResumeNotFound)
}

func TestResumeService_GenerateDownloadURL_NotS3Storage(t *testing.T) {
	mockRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, _, _ string) (*model.Resume, error) {
			return &model.Resume{
				ID:          "resume-1",
				UserID:      "user-123",
				StorageType: model.StorageTypeExternal,
			}, nil
		},
	}
	svc := &ResumeService{repo: mockRepo, s3Enabled: true}

	result, err := svc.GenerateDownloadURL(context.Background(), "user-123", "resume-1")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resume does not use S3 storage")
}

func TestResumeService_GenerateDownloadURL_MissingStorageKey(t *testing.T) {
	mockRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, _, _ string) (*model.Resume, error) {
			return &model.Resume{
				ID:          "resume-1",
				UserID:      "user-123",
				StorageType: model.StorageTypeS3,
				StorageKey:  nil,
			}, nil
		},
	}
	svc := &ResumeService{repo: mockRepo, s3Enabled: true}

	result, err := svc.GenerateDownloadURL(context.Background(), "user-123", "resume-1")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resume storage key is missing")
}

// --- GenerateUploadURL edge cases ---

func TestResumeService_GenerateUploadURL_InvalidContentType_S3Enabled(t *testing.T) {
	// With s3Enabled=true, we should reach the content type validation
	svc := &ResumeService{repo: &MockResumeRepository{}, s3Enabled: true}

	req := &model.GenerateUploadURLRequest{
		Filename:    "resume.docx",
		ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}

	result, err := svc.GenerateUploadURL(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only PDF files are allowed")
}

// --- Delete edge cases ---

func TestResumeService_Delete_S3StorageGetByIDError(t *testing.T) {
	mockRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, _, _ string) (*model.Resume, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewResumeService(mockRepo, nil, nil, nil)

	err := svc.Delete(context.Background(), "user-123", "resume-1")

	assert.Error(t, err)
}

// --- Create edge cases ---

func TestResumeService_Create_WhitespaceOnlyFileURL(t *testing.T) {
	mockRepo := &MockResumeRepository{
		CreateFunc: func(ctx context.Context, resume *model.Resume) error {
			resume.ID = "resume-1"
			return nil
		},
	}

	svc := NewResumeService(mockRepo, nil, nil, nil)
	whiteURL := "   "
	req := &model.CreateResumeRequest{
		Title:   "Test",
		FileURL: &whiteURL,
	}

	result, err := svc.Create(context.Background(), "user-123", req)

	require.NoError(t, err)
	// whitespace-only URL should not be set
	assert.NotNil(t, result)
}
