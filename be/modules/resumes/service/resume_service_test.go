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

		svc := NewResumeService(mockRepo, nil)
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
		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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

		svc := NewResumeService(mockRepo, nil)
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
	assert.False(t, dto.CanDelete) // Cannot delete if has applications

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

		svc := NewResumeService(mockRepo, nil)
		isActive := false
		req := &model.UpdateResumeRequest{IsActive: &isActive}

		result, err := svc.Update(context.Background(), userID, resumeID, req)

		require.NoError(t, err)
		assert.False(t, result.IsActive)
		assert.False(t, updatedResume.IsActive)
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

		svc := NewResumeService(mockRepo, nil)
		err := svc.Delete(context.Background(), userID, resumeID)

		assert.Equal(t, model.ErrResumeInUse, err)
	})
}
