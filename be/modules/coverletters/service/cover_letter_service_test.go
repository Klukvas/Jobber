package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type MockCoverLetterRepository struct {
	CreateFunc  func(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	GetByIDFunc func(ctx context.Context, id string) (*model.CoverLetter, error)
	ListFunc    func(ctx context.Context, userID string) ([]*model.CoverLetter, error)
	UpdateFunc  func(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error)
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *MockCoverLetterRepository) Create(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, cl)
	}
	return cl, nil
}

func (m *MockCoverLetterRepository) GetByID(ctx context.Context, id string) (*model.CoverLetter, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockCoverLetterRepository) List(ctx context.Context, userID string) ([]*model.CoverLetter, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockCoverLetterRepository) Update(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, cl)
	}
	return cl, nil
}

func (m *MockCoverLetterRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

type MockLimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *MockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

// --- Helpers ---

func strPtr(s string) *string   { return &s }
func intPtr(i int) *int         { return &i }
func strSlicePtr(s []string) *[]string { return &s }

func newTestCoverLetter() *model.CoverLetter {
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

func newService(repo *MockCoverLetterRepository) *CoverLetterService {
	return NewCoverLetterService(repo, &MockLimitChecker{})
}

// --- Tests ---

func TestCreate(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	t.Run("success", func(t *testing.T) {
		var createdCL *model.CoverLetter
		repo := &MockCoverLetterRepository{
			CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				cl.ID = "new-cl-1"
				cl.CreatedAt = time.Now()
				cl.UpdatedAt = time.Now()
				createdCL = cl
				return cl, nil
			},
		}
		svc := newService(repo)

		result, err := svc.Create(ctx, userID, &model.CreateCoverLetterRequest{
			Title: "My Cover Letter",
		})

		require.NoError(t, err)
		assert.Equal(t, "new-cl-1", result.ID)
		assert.Equal(t, "My Cover Letter", result.Title)
		assert.Equal(t, "professional", result.Template)
		assert.Equal(t, "Georgia", createdCL.FontFamily)
		assert.Equal(t, 12, createdCL.FontSize)
		assert.Equal(t, "#2563eb", createdCL.PrimaryColor)
		assert.Empty(t, createdCL.Paragraphs)
	})

	t.Run("limit reached", func(t *testing.T) {
		limitErr := errors.New("limit exceeded")
		svc := NewCoverLetterService(
			&MockCoverLetterRepository{},
			&MockLimitChecker{
				CheckLimitFunc: func(_ context.Context, _, _ string) error { return limitErr },
			},
		)

		_, err := svc.Create(ctx, userID, &model.CreateCoverLetterRequest{})
		assert.ErrorIs(t, err, limitErr)
	})

	t.Run("default title when empty", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				cl.ID = "new-cl"
				return cl, nil
			},
		}
		svc := newService(repo)

		result, err := svc.Create(ctx, userID, &model.CreateCoverLetterRequest{})
		require.NoError(t, err)
		assert.Equal(t, "Untitled Cover Letter", result.Title)
	})

	t.Run("default template when empty", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				cl.ID = "new-cl"
				return cl, nil
			},
		}
		svc := newService(repo)

		result, err := svc.Create(ctx, userID, &model.CreateCoverLetterRequest{
			Title: "My CL",
		})
		require.NoError(t, err)
		assert.Equal(t, "professional", result.Template)
	})

	t.Run("with job_id", func(t *testing.T) {
		jobID := "job-123"
		var createdCL *model.CoverLetter
		repo := &MockCoverLetterRepository{
			CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				cl.ID = "new-cl"
				createdCL = cl
				return cl, nil
			},
		}
		svc := newService(repo)

		result, err := svc.Create(ctx, userID, &model.CreateCoverLetterRequest{
			Title: "Job CL",
			JobID: &jobID,
		})
		require.NoError(t, err)
		assert.Equal(t, &jobID, result.JobID)
		assert.Equal(t, &jobID, createdCL.JobID)
	})

	t.Run("returns error when repo Create fails", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			CreateFunc: func(_ context.Context, _ *model.CoverLetter) (*model.CoverLetter, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newService(repo)

		_, err := svc.Create(ctx, userID, &model.CreateCoverLetterRequest{})
		assert.Error(t, err)
	})
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cl := newTestCoverLetter()
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, id string) (*model.CoverLetter, error) {
				assert.Equal(t, "cl-1", id)
				return cl, nil
			},
		}
		svc := newService(repo)

		result, err := svc.Get(ctx, "user-1", "cl-1")
		require.NoError(t, err)
		assert.Equal(t, "cl-1", result.ID)
		assert.Equal(t, "Test Cover Letter", result.Title)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return nil, model.ErrCoverLetterNotFound
			},
		}
		svc := newService(repo)

		_, err := svc.Get(ctx, "user-1", "nonexistent")
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
	})

	t.Run("not authorized - wrong user", func(t *testing.T) {
		cl := newTestCoverLetter() // UserID = "user-1"
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return cl, nil
			},
		}
		svc := newService(repo)

		_, err := svc.Get(ctx, "user-2", "cl-1")
		assert.ErrorIs(t, err, model.ErrNotAuthorized)
	})
}

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			ListFunc: func(_ context.Context, _ string) ([]*model.CoverLetter, error) {
				return []*model.CoverLetter{
					{ID: "cl-1", UserID: "user-1", Title: "CL 1", Paragraphs: []string{}},
					{ID: "cl-2", UserID: "user-1", Title: "CL 2", Paragraphs: []string{}},
				}, nil
			},
		}
		svc := newService(repo)

		results, err := svc.List(ctx, "user-1")
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "CL 1", results[0].Title)
		assert.Equal(t, "CL 2", results[1].Title)
	})

	t.Run("empty list", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			ListFunc: func(_ context.Context, _ string) ([]*model.CoverLetter, error) {
				return []*model.CoverLetter{}, nil
			},
		}
		svc := newService(repo)

		results, err := svc.List(ctx, "user-1")
		require.NoError(t, err)
		assert.Len(t, results, 0)
	})
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	clID := "cl-1"

	defaultRepo := func() *MockCoverLetterRepository {
		return &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return newTestCoverLetter(), nil
			},
			UpdateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				return cl, nil
			},
		}
	}

	t.Run("success - updates title", func(t *testing.T) {
		svc := newService(defaultRepo())

		result, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			Title: strPtr("Updated Title"),
		})

		require.NoError(t, err)
		assert.Equal(t, "Updated Title", result.Title)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return nil, model.ErrCoverLetterNotFound
			},
		}
		svc := newService(repo)

		_, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			Title: strPtr("New Title"),
		})
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
	})

	t.Run("not authorized - wrong user", func(t *testing.T) {
		svc := newService(defaultRepo())

		_, err := svc.Update(ctx, "user-2", clID, &model.UpdateCoverLetterRequest{
			Title: strPtr("Hacked"),
		})
		assert.ErrorIs(t, err, model.ErrNotAuthorized)
	})

	t.Run("updates job_id", func(t *testing.T) {
		jobID := "job-456"
		var updatedCL *model.CoverLetter
		repo := defaultRepo()
		repo.UpdateFunc = func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
			updatedCL = cl
			return cl, nil
		}
		svc := newService(repo)

		result, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			JobID: &jobID,
		})

		require.NoError(t, err)
		assert.Equal(t, &jobID, result.JobID)
		assert.Equal(t, &jobID, updatedCL.JobID)
	})

	t.Run("rejects invalid font family", func(t *testing.T) {
		svc := newService(defaultRepo())

		_, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			FontFamily: strPtr("ComicSansXYZ"),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid font family")
	})

	t.Run("rejects invalid color format", func(t *testing.T) {
		svc := newService(defaultRepo())

		_, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			PrimaryColor: strPtr("not-a-color"),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid color format")
	})

	t.Run("rejects font size below 8", func(t *testing.T) {
		svc := newService(defaultRepo())

		_, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			FontSize: intPtr(5),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "font size must be between 8 and 18")
	})

	t.Run("rejects font size above 18", func(t *testing.T) {
		svc := newService(defaultRepo())

		_, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			FontSize: intPtr(20),
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "font size must be between 8 and 18")
	})

	t.Run("accepts valid font size", func(t *testing.T) {
		svc := newService(defaultRepo())

		result, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			FontSize: intPtr(14),
		})
		require.NoError(t, err)
		assert.Equal(t, 14, result.FontSize)
	})

	t.Run("accepts valid font family", func(t *testing.T) {
		svc := newService(defaultRepo())

		result, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			FontFamily: strPtr("Inter"),
		})
		require.NoError(t, err)
		assert.Equal(t, "Inter", result.FontFamily)
	})

	t.Run("accepts valid color", func(t *testing.T) {
		svc := newService(defaultRepo())

		result, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			PrimaryColor: strPtr("#FF5733"),
		})
		require.NoError(t, err)
		assert.Equal(t, "#FF5733", result.PrimaryColor)
	})

	t.Run("updates all fields at once", func(t *testing.T) {
		svc := newService(defaultRepo())
		jobID := "job-all"
		paragraphs := []string{"New paragraph 1", "New paragraph 2"}

		result, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			Title:          strPtr("All Fields"),
			JobID:          &jobID,
			Template:       strPtr("modern"),
			RecipientName:  strPtr("John Doe"),
			RecipientTitle: strPtr("CEO"),
			CompanyName:    strPtr("Big Corp"),
			CompanyAddress: strPtr("456 Elm St"),
			Greeting:       strPtr("Hello,"),
			Paragraphs:     strSlicePtr(paragraphs),
			Closing:        strPtr("Best regards,"),
			FontFamily:     strPtr("Inter"),
			FontSize:       intPtr(14),
			PrimaryColor:   strPtr("#e11d48"),
		})

		require.NoError(t, err)
		assert.Equal(t, "All Fields", result.Title)
		assert.Equal(t, &jobID, result.JobID)
		assert.Equal(t, "modern", result.Template)
		assert.Equal(t, "John Doe", result.RecipientName)
		assert.Equal(t, "CEO", result.RecipientTitle)
		assert.Equal(t, "Big Corp", result.CompanyName)
		assert.Equal(t, "456 Elm St", result.CompanyAddress)
		assert.Equal(t, "Hello,", result.Greeting)
		assert.Equal(t, paragraphs, result.Paragraphs)
		assert.Equal(t, "Best regards,", result.Closing)
		assert.Equal(t, "Inter", result.FontFamily)
		assert.Equal(t, 14, result.FontSize)
		assert.Equal(t, "#e11d48", result.PrimaryColor)
	})

	t.Run("returns error when repo Update fails", func(t *testing.T) {
		repo := defaultRepo()
		repo.UpdateFunc = func(_ context.Context, _ *model.CoverLetter) (*model.CoverLetter, error) {
			return nil, errors.New("db error")
		}
		svc := newService(repo)

		_, err := svc.Update(ctx, userID, clID, &model.UpdateCoverLetterRequest{
			Title: strPtr("New Title"),
		})
		assert.Error(t, err)
	})
}

func TestDuplicate(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	clID := "cl-1"

	t.Run("success - copies all fields with (Copy) suffix", func(t *testing.T) {
		original := newTestCoverLetter()
		jobID := "job-dup"
		original.JobID = &jobID

		var createdCL *model.CoverLetter
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return original, nil
			},
			CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				cl.ID = "cl-copy"
				createdCL = cl
				return cl, nil
			},
		}
		svc := newService(repo)

		result, err := svc.Duplicate(ctx, userID, clID)
		require.NoError(t, err)
		assert.Equal(t, "cl-copy", result.ID)
		assert.Equal(t, "Test Cover Letter (Copy)", result.Title)
		assert.Equal(t, original.Template, result.Template)
		assert.Equal(t, original.RecipientName, result.RecipientName)
		assert.Equal(t, original.RecipientTitle, result.RecipientTitle)
		assert.Equal(t, original.CompanyName, result.CompanyName)
		assert.Equal(t, original.CompanyAddress, result.CompanyAddress)
		assert.Equal(t, original.Greeting, result.Greeting)
		assert.Equal(t, original.Closing, result.Closing)
		assert.Equal(t, original.FontFamily, result.FontFamily)
		assert.Equal(t, original.FontSize, result.FontSize)
		assert.Equal(t, original.PrimaryColor, result.PrimaryColor)
		assert.Equal(t, &jobID, result.JobID)
		assert.Equal(t, &jobID, createdCL.JobID)
		assert.Equal(t, original.Paragraphs, result.Paragraphs)
	})

	t.Run("paragraphs are a new slice - not same pointer", func(t *testing.T) {
		original := newTestCoverLetter()

		var createdCL *model.CoverLetter
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return original, nil
			},
			CreateFunc: func(_ context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
				cl.ID = "cl-copy"
				createdCL = cl
				return cl, nil
			},
		}
		svc := newService(repo)

		_, err := svc.Duplicate(ctx, userID, clID)
		require.NoError(t, err)

		// Verify the paragraphs slice is a copy, not the same reference
		assert.Equal(t, original.Paragraphs, createdCL.Paragraphs)
		// Modify the copy and ensure original is unchanged
		if len(createdCL.Paragraphs) > 0 {
			createdCL.Paragraphs[0] = "MODIFIED"
			assert.NotEqual(t, createdCL.Paragraphs[0], original.Paragraphs[0])
		}
	})

	t.Run("limit reached", func(t *testing.T) {
		limitErr := errors.New("limit exceeded")
		svc := NewCoverLetterService(
			&MockCoverLetterRepository{},
			&MockLimitChecker{
				CheckLimitFunc: func(_ context.Context, _, _ string) error { return limitErr },
			},
		)

		_, err := svc.Duplicate(ctx, userID, clID)
		assert.ErrorIs(t, err, limitErr)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return nil, model.ErrCoverLetterNotFound
			},
		}
		svc := newService(repo)

		_, err := svc.Duplicate(ctx, userID, clID)
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
	})

	t.Run("not authorized - wrong user", func(t *testing.T) {
		original := newTestCoverLetter() // UserID = "user-1"
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return original, nil
			},
		}
		svc := newService(repo)

		_, err := svc.Duplicate(ctx, "user-2", clID)
		assert.ErrorIs(t, err, model.ErrNotAuthorized)
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		var deletedID string
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, id string) (*model.CoverLetter, error) {
				return newTestCoverLetter(), nil
			},
			DeleteFunc: func(_ context.Context, id string) error {
				deletedID = id
				return nil
			},
		}
		svc := newService(repo)

		err := svc.Delete(ctx, "user-1", "cl-1")
		require.NoError(t, err)
		assert.Equal(t, "cl-1", deletedID)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return nil, model.ErrCoverLetterNotFound
			},
		}
		svc := newService(repo)

		err := svc.Delete(ctx, "user-1", "nonexistent")
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
	})

	t.Run("not authorized - wrong user", func(t *testing.T) {
		repo := &MockCoverLetterRepository{
			GetByIDFunc: func(_ context.Context, _ string) (*model.CoverLetter, error) {
				return newTestCoverLetter(), nil // UserID = "user-1"
			},
		}
		svc := newService(repo)

		err := svc.Delete(ctx, "user-2", "cl-1")
		assert.ErrorIs(t, err, model.ErrNotAuthorized)
	})
}
