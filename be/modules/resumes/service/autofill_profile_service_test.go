package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAutofillProfileRepository implements ports.AutofillProfileRepository.
type MockAutofillProfileRepository struct {
	GetFunc                func(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error)
	UpsertFunc             func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error)
	InvalidateByResumeFunc func(ctx context.Context, resumeID string) error
}

func (m *MockAutofillProfileRepository) Get(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, userID, resumeID)
	}
	return nil, nil
}

func (m *MockAutofillProfileRepository) Upsert(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, userID, resumeID, profile, parserVersion)
	}
	return true, nil
}

func (m *MockAutofillProfileRepository) InvalidateByResume(ctx context.Context, resumeID string) error {
	if m.InvalidateByResumeFunc != nil {
		return m.InvalidateByResumeFunc(ctx, resumeID)
	}
	return nil
}

type mockParser struct {
	ParseFunc func(ctx context.Context, text string) (*ai.ParsedResume, error)
	called    bool
}

func (m *mockParser) ParseResumeText(ctx context.Context, text string) (*ai.ParsedResume, error) {
	m.called = true
	if m.ParseFunc != nil {
		return m.ParseFunc(ctx, text)
	}
	return &ai.ParsedResume{FullName: "Test User"}, nil
}

type mockPlanChecker struct {
	RequirePaidPlanErr error
	CheckLimitErr      error
	RecordErr          error
	paidChecked        bool
	limitChecked       bool
	usageRecorded      bool
}

func (m *mockPlanChecker) RequirePaidPlan(ctx context.Context, userID string) error {
	m.paidChecked = true
	return m.RequirePaidPlanErr
}

func (m *mockPlanChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	m.limitChecked = true
	return m.CheckLimitErr
}

func (m *mockPlanChecker) RecordResumeAutofillUsage(ctx context.Context, userID string) error {
	m.usageRecorded = true
	return m.RecordErr
}

const (
	testUserID   = "user-123"
	testResumeID = "resume-1"
)

func s3Resume(key string) *model.Resume {
	return &model.Resume{
		ID:          testResumeID,
		UserID:      testUserID,
		StorageType: model.StorageTypeS3,
		StorageKey:  &key,
	}
}

func resumeRepoReturning(resume *model.Resume) *MockResumeRepository {
	return &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, userID, resumeID string) (*model.Resume, error) {
			if resume == nil {
				return nil, model.ErrResumeNotFound
			}
			return resume, nil
		},
	}
}

func fullParsed() *ai.ParsedResume {
	return &ai.ParsedResume{
		FullName: "Jane Dev",
		Email:    "jane@dev.io",
		Phone:    "+1 555",
		Location: "Kyiv",
		Website:  "https://jane.dev",
		LinkedIn: "linkedin.com/in/jane",
		GitHub:   "github.com/jane",
		Summary:  "Senior engineer",
		Experiences: []ai.ParsedExperience{
			{Company: "Acme", Position: "Engineer", IsCurrent: true},
		},
		Educations: []ai.ParsedEducation{
			{Institution: "KPI", Degree: "MSc", FieldOfStudy: "CS"},
		},
		Skills: []ai.ParsedSkill{{Name: "Go", Level: "expert"}},
	}
}

func TestAutofillProfileService_GetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("cache hit returns profile without parser or plan checks", func(t *testing.T) {
		parser := &mockParser{}
		plan := &mockPlanChecker{}
		profileRepo := &MockAutofillProfileRepository{
			GetFunc: func(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error) {
				return fullParsed(), nil
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, nil, parser, plan)
		dto, err := svc.GetProfile(ctx, testUserID, testResumeID)

		require.NoError(t, err)
		assert.Equal(t, "Jane Dev", dto.Contact.FullName)
		assert.False(t, parser.called, "cache hit must not call the parser")
		assert.False(t, plan.paidChecked, "cache hit must skip the paid gate (ADR-0001)")
		assert.False(t, plan.usageRecorded, "cache hit must not charge quota")
	})

	t.Run("foreign resume returns not found before plan checks", func(t *testing.T) {
		plan := &mockPlanChecker{}
		svc := NewAutofillProfileService(resumeRepoReturning(nil), &MockAutofillProfileRepository{}, nil, &mockParser{}, plan)

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.ErrorIs(t, err, model.ErrResumeNotFound)
		assert.False(t, plan.paidChecked, "not-found must not leak plan state")
	})

	t.Run("free plan is rejected before any download or parse", func(t *testing.T) {
		parser := &mockParser{}
		plan := &mockPlanChecker{RequirePaidPlanErr: subModel.ErrPaidFeature}
		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), &MockAutofillProfileRepository{}, nil, parser, plan)

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.ErrorIs(t, err, subModel.ErrPaidFeature)
		assert.False(t, parser.called)
	})

	t.Run("ai limit reached is rejected", func(t *testing.T) {
		plan := &mockPlanChecker{CheckLimitErr: subModel.ErrLimitReached}
		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), &MockAutofillProfileRepository{}, nil, &mockParser{}, plan)

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.ErrorIs(t, err, subModel.ErrLimitReached)
	})

	t.Run("resume without a file returns file-missing", func(t *testing.T) {
		resume := &model.Resume{ID: testResumeID, UserID: testUserID, StorageType: model.StorageTypeExternal}
		svc := NewAutofillProfileService(resumeRepoReturning(resume), &MockAutofillProfileRepository{}, nil, &mockParser{}, &mockPlanChecker{})

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.ErrorIs(t, err, model.ErrResumeFileMissing)
	})

	t.Run("unreadable pdf is not parsed, cached, or charged", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("not a pdf")})
		defer cleanup()

		parser := &mockParser{}
		plan := &mockPlanChecker{}
		upserted := false
		profileRepo := &MockAutofillProfileRepository{
			UpsertFunc: func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
				upserted = true
				return true, nil
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, s3Client, parser, plan)
		svc.extractPDF = func([]byte) (string, error) { return "", errors.New("bad pdf") }

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.ErrorIs(t, err, model.ErrResumeUnreadable)
		assert.False(t, parser.called)
		assert.False(t, upserted)
		assert.False(t, plan.usageRecorded)
	})

	t.Run("extraction below contact threshold is rejected without caching or charging", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("pdf")})
		defer cleanup()

		plan := &mockPlanChecker{}
		upserted := false
		profileRepo := &MockAutofillProfileRepository{
			UpsertFunc: func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
				upserted = true
				return true, nil
			},
		}
		parser := &mockParser{
			ParseFunc: func(ctx context.Context, text string) (*ai.ParsedResume, error) {
				// No name and no email — junior resume WITH experiences still
				// fails only if contact is empty.
				return &ai.ParsedResume{Experiences: []ai.ParsedExperience{{Company: "Acme"}}}, nil
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, s3Client, parser, plan)
		svc.extractPDF = func([]byte) (string, error) { return "some text", nil }

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.ErrorIs(t, err, model.ErrResumeUnreadable)
		assert.False(t, upserted)
		assert.False(t, plan.usageRecorded)
	})

	t.Run("successful extraction caches before charging and maps all fields", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("pdf")})
		defer cleanup()

		plan := &mockPlanChecker{}
		var events []string
		profileRepo := &MockAutofillProfileRepository{
			UpsertFunc: func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
				events = append(events, "upsert")
				assert.Equal(t, 1, parserVersion)
				assert.Equal(t, testUserID, userID)
				assert.Equal(t, testResumeID, resumeID)
				return true, nil
			},
		}
		parser := &mockParser{
			ParseFunc: func(ctx context.Context, text string) (*ai.ParsedResume, error) {
				return fullParsed(), nil
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, s3Client, parser, plan)
		svc.extractPDF = func([]byte) (string, error) { return "resume text", nil }

		dto, err := svc.GetProfile(ctx, testUserID, testResumeID)

		require.NoError(t, err)
		assert.Equal(t, []string{"upsert"}, events)
		assert.True(t, plan.usageRecorded, "successful extraction must record usage")

		assert.Equal(t, "Jane Dev", dto.Contact.FullName)
		assert.Equal(t, "jane@dev.io", dto.Contact.Email)
		assert.Equal(t, "linkedin.com/in/jane", dto.Contact.LinkedIn)
		assert.Equal(t, "github.com/jane", dto.Contact.GitHub)
		require.NotNil(t, dto.Summary)
		assert.Equal(t, "Senior engineer", dto.Summary.Content)
		require.Len(t, dto.Experiences, 1)
		assert.Equal(t, "Acme", dto.Experiences[0].Company)
		assert.Equal(t, "Engineer", dto.Experiences[0].Position)
		assert.True(t, dto.Experiences[0].IsCurrent)
		require.Len(t, dto.Educations, 1)
		assert.Equal(t, "KPI", dto.Educations[0].Institution)
		assert.Equal(t, "CS", dto.Educations[0].FieldOfStudy)
		require.Len(t, dto.Skills, 1)
		assert.Equal(t, "Go", dto.Skills[0].Name)
	})

	t.Run("cache write failure does not charge quota", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("pdf")})
		defer cleanup()

		plan := &mockPlanChecker{}
		profileRepo := &MockAutofillProfileRepository{
			UpsertFunc: func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
				return false, errors.New("db down")
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, s3Client, &mockParser{ParseFunc: func(ctx context.Context, text string) (*ai.ParsedResume, error) {
			return fullParsed(), nil
		}}, plan)
		svc.extractPDF = func([]byte) (string, error) { return "resume text", nil }

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.Error(t, err)
		assert.False(t, plan.usageRecorded, "a profile the user never received must not be charged")
	})

	t.Run("usage-recording failure is swallowed: the user still gets the profile", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("pdf")})
		defer cleanup()

		// Deliberate: favor the user over the ledger — a ledger blip must not
		// void a successfully extracted (and cached) profile.
		plan := &mockPlanChecker{RecordErr: errors.New("ledger down")}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), &MockAutofillProfileRepository{}, s3Client, &mockParser{ParseFunc: func(ctx context.Context, text string) (*ai.ParsedResume, error) {
			return fullParsed(), nil
		}}, plan)
		svc.extractPDF = func([]byte) (string, error) { return "resume text", nil }

		dto, err := svc.GetProfile(ctx, testUserID, testResumeID)

		require.NoError(t, err)
		assert.Equal(t, "Jane Dev", dto.Contact.FullName)
		assert.True(t, plan.usageRecorded, "recording must have been attempted")
	})

	t.Run("losing a concurrent extraction race does not charge quota", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("pdf")})
		defer cleanup()

		plan := &mockPlanChecker{}
		profileRepo := &MockAutofillProfileRepository{
			// inserted=false: another request created the row first (xmax != 0).
			UpsertFunc: func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
				return false, nil
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, s3Client, &mockParser{ParseFunc: func(ctx context.Context, text string) (*ai.ParsedResume, error) {
			return fullParsed(), nil
		}}, plan)
		svc.extractPDF = func([]byte) (string, error) { return "resume text", nil }

		dto, err := svc.GetProfile(ctx, testUserID, testResumeID)

		require.NoError(t, err)
		assert.Equal(t, "Jane Dev", dto.Contact.FullName, "the loser still gets its profile")
		assert.False(t, plan.usageRecorded, "only the inserting request charges (ADR-0001: once per file)")
	})

	t.Run("cache read failure is an error, not a chargeable miss", func(t *testing.T) {
		parser := &mockParser{}
		plan := &mockPlanChecker{}
		profileRepo := &MockAutofillProfileRepository{
			GetFunc: func(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error) {
				return nil, errors.New("db blip")
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, nil, parser, plan)
		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.Error(t, err)
		assert.False(t, plan.paidChecked, "a read blip must not gate a possibly-cached profile by plan")
		assert.False(t, parser.called, "a read blip must not trigger a re-charged extraction")
	})

	t.Run("parser failure is propagated without caching", func(t *testing.T) {
		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{"k": []byte("pdf")})
		defer cleanup()

		upserted := false
		profileRepo := &MockAutofillProfileRepository{
			UpsertFunc: func(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (bool, error) {
				upserted = true
				return true, nil
			},
		}

		svc := NewAutofillProfileService(resumeRepoReturning(s3Resume("k")), profileRepo, s3Client, &mockParser{ParseFunc: func(ctx context.Context, text string) (*ai.ParsedResume, error) {
			return nil, errors.New("ai unavailable")
		}}, &mockPlanChecker{})
		svc.extractPDF = func([]byte) (string, error) { return "resume text", nil }

		_, err := svc.GetProfile(ctx, testUserID, testResumeID)

		assert.Error(t, err)
		assert.False(t, upserted)
	})
}

func TestCompositeInvalidator(t *testing.T) {
	ctx := context.Background()

	t.Run("fans out to all invalidators and joins errors", func(t *testing.T) {
		firstErr := errors.New("first failed")
		var calls []string
		first := &MockAutofillProfileRepository{InvalidateByResumeFunc: func(ctx context.Context, resumeID string) error {
			calls = append(calls, "first")
			return firstErr
		}}
		second := &MockAutofillProfileRepository{InvalidateByResumeFunc: func(ctx context.Context, resumeID string) error {
			calls = append(calls, "second")
			return nil
		}}

		inv := NewCompositeInvalidator(first, second)
		err := inv.InvalidateByResume(ctx, testResumeID)

		assert.ErrorIs(t, err, firstErr)
		assert.Equal(t, []string{"first", "second"}, calls, "one failure must not stop the fan-out")
	})

	t.Run("nil invalidators are skipped", func(t *testing.T) {
		inv := NewCompositeInvalidator(nil, &MockAutofillProfileRepository{})
		assert.NoError(t, inv.InvalidateByResume(ctx, testResumeID))
	})
}
