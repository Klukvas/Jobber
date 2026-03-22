package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type MockContentLibraryRepository struct {
	CreateFunc  func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error)
	GetByIDFunc func(ctx context.Context, id string) (*model.ContentLibraryEntry, error)
	ListFunc    func(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error)
	UpdateFunc  func(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error)
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *MockContentLibraryRepository) Create(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, entry)
	}
	return entry, nil
}

func (m *MockContentLibraryRepository) GetByID(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *MockContentLibraryRepository) List(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockContentLibraryRepository) Update(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, entry)
	}
	return entry, nil
}

func (m *MockContentLibraryRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestContentLibraryService_Create(t *testing.T) {
	userID := "user-123"

	tests := []struct {
		name        string
		req         *model.CreateContentLibraryRequest
		setupRepo   func(repo *MockContentLibraryRepository)
		wantErr     bool
		errContains string
		validate    func(t *testing.T, dto *model.ContentLibraryEntryDTO)
	}{
		{
			name: "creates entry successfully",
			req: &model.CreateContentLibraryRequest{
				Title:    "My Cover Letter Intro",
				Content:  "Dear Hiring Manager...",
				Category: "cover_letters",
			},
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.CreateFunc = func(_ context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
					entry.ID = "entry-1"
					entry.CreatedAt = time.Now()
					entry.UpdatedAt = time.Now()
					return entry, nil
				}
			},
			validate: func(t *testing.T, dto *model.ContentLibraryEntryDTO) {
				assert.Equal(t, "entry-1", dto.ID)
				assert.Equal(t, "My Cover Letter Intro", dto.Title)
				assert.Equal(t, "Dear Hiring Manager...", dto.Content)
				assert.Equal(t, "cover_letters", dto.Category)
			},
		},
		{
			name: "defaults category to general when empty",
			req: &model.CreateContentLibraryRequest{
				Title:   "Snippet",
				Content: "Some content",
			},
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.CreateFunc = func(_ context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
					assert.Equal(t, "general", entry.Category)
					entry.ID = "entry-2"
					return entry, nil
				}
			},
			validate: func(t *testing.T, dto *model.ContentLibraryEntryDTO) {
				assert.Equal(t, "general", dto.Category)
			},
		},
		{
			name: "returns error from repository",
			req: &model.CreateContentLibraryRequest{
				Title:   "Test",
				Content: "Content",
			},
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.CreateFunc = func(_ context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr:     true,
			errContains: "failed to create content library entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockContentLibraryRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := NewContentLibraryService(repo)

			result, err := svc.Create(context.Background(), userID, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// List tests
// ---------------------------------------------------------------------------

func TestContentLibraryService_List(t *testing.T) {
	userID := "user-123"

	tests := []struct {
		name      string
		setupRepo func(repo *MockContentLibraryRepository)
		wantCount int
		wantErr   bool
	}{
		{
			name: "returns entries list",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.ListFunc = func(_ context.Context, uid string) ([]*model.ContentLibraryEntry, error) {
					assert.Equal(t, userID, uid)
					return []*model.ContentLibraryEntry{
						{ID: "entry-1", UserID: userID, Title: "Entry 1", Content: "Content 1", Category: "general"},
						{ID: "entry-2", UserID: userID, Title: "Entry 2", Content: "Content 2", Category: "cover_letters"},
					}, nil
				}
			},
			wantCount: 2,
		},
		{
			name: "returns empty list",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.ListFunc = func(_ context.Context, uid string) ([]*model.ContentLibraryEntry, error) {
					return []*model.ContentLibraryEntry{}, nil
				}
			},
			wantCount: 0,
		},
		{
			name: "returns error from repository",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.ListFunc = func(_ context.Context, uid string) ([]*model.ContentLibraryEntry, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockContentLibraryRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := NewContentLibraryService(repo)

			result, err := svc.List(context.Background(), userID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Update tests
// ---------------------------------------------------------------------------

func TestContentLibraryService_Update(t *testing.T) {
	userID := "user-123"
	entryID := "entry-1"

	tests := []struct {
		name        string
		req         *model.UpdateContentLibraryRequest
		setupRepo   func(repo *MockContentLibraryRepository)
		wantErr     bool
		errContains string
		validate    func(t *testing.T, dto *model.ContentLibraryEntryDTO)
	}{
		{
			name: "updates title successfully",
			req: func() *model.UpdateContentLibraryRequest {
				title := "Updated Title"
				return &model.UpdateContentLibraryRequest{Title: &title}
			}(),
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:       entryID,
						UserID:   userID,
						Title:    "Old Title",
						Content:  "Content",
						Category: "general",
					}, nil
				}
				repo.UpdateFunc = func(_ context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
					assert.Equal(t, "Updated Title", entry.Title)
					return entry, nil
				}
			},
			validate: func(t *testing.T, dto *model.ContentLibraryEntryDTO) {
				assert.Equal(t, "Updated Title", dto.Title)
			},
		},
		{
			name: "updates content and category",
			req: func() *model.UpdateContentLibraryRequest {
				content := "New content"
				category := "skills"
				return &model.UpdateContentLibraryRequest{Content: &content, Category: &category}
			}(),
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:       entryID,
						UserID:   userID,
						Title:    "Title",
						Content:  "Old content",
						Category: "general",
					}, nil
				}
				repo.UpdateFunc = func(_ context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
					assert.Equal(t, "New content", entry.Content)
					assert.Equal(t, "skills", entry.Category)
					return entry, nil
				}
			},
			validate: func(t *testing.T, dto *model.ContentLibraryEntryDTO) {
				assert.Equal(t, "New content", dto.Content)
				assert.Equal(t, "skills", dto.Category)
			},
		},
		{
			name: "returns not authorized for wrong user",
			req: func() *model.UpdateContentLibraryRequest {
				title := "Updated"
				return &model.UpdateContentLibraryRequest{Title: &title}
			}(),
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:     entryID,
						UserID: "other-user",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "not authorized",
		},
		{
			name: "returns error when entry not found",
			req: func() *model.UpdateContentLibraryRequest {
				title := "Updated"
				return &model.UpdateContentLibraryRequest{Title: &title}
			}(),
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return nil, errors.New("not found")
				}
			},
			wantErr: true,
		},
		{
			name: "returns error when update fails",
			req: func() *model.UpdateContentLibraryRequest {
				title := "Updated"
				return &model.UpdateContentLibraryRequest{Title: &title}
			}(),
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:     entryID,
						UserID: userID,
						Title:  "Title",
					}, nil
				}
				repo.UpdateFunc = func(_ context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
					return nil, errors.New("update failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockContentLibraryRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := NewContentLibraryService(repo)

			result, err := svc.Update(context.Background(), userID, entryID, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestContentLibraryService_Delete(t *testing.T) {
	userID := "user-123"
	entryID := "entry-1"

	tests := []struct {
		name        string
		setupRepo   func(repo *MockContentLibraryRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "deletes entry successfully",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:     entryID,
						UserID: userID,
					}, nil
				}
				repo.DeleteFunc = func(_ context.Context, id string) error {
					assert.Equal(t, entryID, id)
					return nil
				}
			},
		},
		{
			name: "returns not authorized for wrong user",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:     entryID,
						UserID: "other-user",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "not authorized",
		},
		{
			name: "returns error when entry not found",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return nil, errors.New("not found")
				}
			},
			wantErr: true,
		},
		{
			name: "returns error when delete fails",
			setupRepo: func(repo *MockContentLibraryRepository) {
				repo.GetByIDFunc = func(_ context.Context, id string) (*model.ContentLibraryEntry, error) {
					return &model.ContentLibraryEntry{
						ID:     entryID,
						UserID: userID,
					}, nil
				}
				repo.DeleteFunc = func(_ context.Context, id string) error {
					return errors.New("delete failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockContentLibraryRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := NewContentLibraryService(repo)

			err := svc.Delete(context.Background(), userID, entryID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
