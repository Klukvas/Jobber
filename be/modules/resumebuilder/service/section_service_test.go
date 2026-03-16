package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func boolPtr(b bool) *bool { return &b }

var errDB = errors.New("db error")

// --- Contact ---

func TestUpsertContact(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.UpsertContactRequest{
		FullName: "Jane Doe",
		Email:    "jane@example.com",
		Phone:    "+1234567890",
		Location: "NYC",
		Website:  "https://jane.dev",
		LinkedIn: "https://linkedin.com/in/jane",
		GitHub:   "https://github.com/jane",
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.Contact
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertContactFunc: func(_ context.Context, c *model.Contact) error {
				saved = c
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.UpsertContact(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "Jane Doe", result.FullName)
		assert.Equal(t, "jane@example.com", result.Email)
		assert.Equal(t, "+1234567890", result.Phone)
		assert.Equal(t, "NYC", result.Location)
		assert.Equal(t, "https://jane.dev", result.Website)
		assert.Equal(t, "https://linkedin.com/in/jane", result.LinkedIn)
		assert.Equal(t, "https://github.com/jane", result.GitHub)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpsertContact(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertContactFunc:   func(_ context.Context, _ *model.Contact) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpsertContact(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Summary ---

func TestUpsertSummary(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.UpsertSummaryRequest{Content: "Experienced software engineer"}

	t.Run("success", func(t *testing.T) {
		var saved *model.Summary
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertSummaryFunc: func(_ context.Context, s *model.Summary) error {
				saved = s
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.UpsertSummary(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "Experienced software engineer", result.Content)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpsertSummary(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertSummaryFunc:   func(_ context.Context, _ *model.Summary) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpsertSummary(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Experience ---

func TestCreateExperience(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateExperienceRequest{
		Company:     "Acme Corp",
		Position:    "Engineer",
		Location:    "Remote",
		StartDate:   "2023-01",
		EndDate:     "2024-01",
		IsCurrent:   false,
		Description: "Built things",
		SortOrder:   0,
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.Experience
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateExperienceFunc: func(_ context.Context, e *model.Experience) error {
				e.ID = "exp-1"
				saved = e
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateExperience(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "exp-1", result.ID)
		assert.Equal(t, "Acme Corp", result.Company)
		assert.Equal(t, "Engineer", result.Position)
		assert.Equal(t, "Remote", result.Location)
		assert.Equal(t, "2023-01", result.StartDate)
		assert.Equal(t, "2024-01", result.EndDate)
		assert.False(t, result.IsCurrent)
		assert.Equal(t, "Built things", result.Description)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateExperience(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:  func(_ context.Context, _, _ string) error { return nil },
			CreateExperienceFunc: func(_ context.Context, _ *model.Experience) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateExperience(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateExperience(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "exp-1"

	existing := &model.Experience{
		ID:              entryID,
		ResumeBuilderID: rbID,
		Company:         "Old Corp",
		Position:        "Junior",
		Location:        "Office",
		StartDate:       "2022-01",
		EndDate:         "2023-01",
		IsCurrent:       false,
		Description:     "Old stuff",
		SortOrder:       0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Experience
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetExperienceByIDFunc: func(_ context.Context, _, _ string) (*model.Experience, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateExperienceFunc: func(_ context.Context, e *model.Experience) error {
				updated = e
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateExperienceRequest{
			Company:     strPtr("New Corp"),
			Position:    strPtr("Senior"),
			Location:    strPtr("Remote"),
			StartDate:   strPtr("2023-06"),
			EndDate:     strPtr("2024-06"),
			IsCurrent:   boolPtr(true),
			Description: strPtr("New stuff"),
			SortOrder:   intPtr(1),
		}
		result, err := svc.UpdateExperience(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "New Corp", result.Company)
		assert.Equal(t, "Senior", result.Position)
		assert.Equal(t, "Remote", result.Location)
		assert.Equal(t, "2023-06", result.StartDate)
		assert.Equal(t, "2024-06", result.EndDate)
		assert.True(t, result.IsCurrent)
		assert.Equal(t, "New stuff", result.Description)
		assert.Equal(t, 1, result.SortOrder)
		assert.Equal(t, "New Corp", updated.Company)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateExperience(ctx, userID, rbID, entryID, &model.UpdateExperienceRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetExperienceByIDFunc: func(_ context.Context, _, _ string) (*model.Experience, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateExperience(ctx, userID, rbID, entryID, &model.UpdateExperienceRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetExperienceByIDFunc: func(_ context.Context, _, _ string) (*model.Experience, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateExperienceFunc: func(_ context.Context, _ *model.Experience) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateExperience(ctx, userID, rbID, entryID, &model.UpdateExperienceRequest{Company: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteExperience(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "exp-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteExperienceFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteExperience(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteExperience(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:  func(_ context.Context, _, _ string) error { return nil },
			DeleteExperienceFunc: func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteExperience(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Education ---

func TestCreateEducation(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateEducationRequest{
		Institution:  "MIT",
		Degree:       "BS",
		FieldOfStudy: "CS",
		StartDate:    "2018-09",
		EndDate:      "2022-06",
		IsCurrent:    false,
		GPA:          "3.9",
		Description:  "Graduated with honors",
		SortOrder:    0,
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.Education
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateEducationFunc: func(_ context.Context, e *model.Education) error {
				e.ID = "edu-1"
				saved = e
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateEducation(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "edu-1", result.ID)
		assert.Equal(t, "MIT", result.Institution)
		assert.Equal(t, "BS", result.Degree)
		assert.Equal(t, "CS", result.FieldOfStudy)
		assert.Equal(t, "3.9", result.GPA)
		assert.Equal(t, "Graduated with honors", result.Description)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateEducation(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateEducationFunc: func(_ context.Context, _ *model.Education) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateEducation(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateEducation(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "edu-1"

	existing := &model.Education{
		ID:              entryID,
		ResumeBuilderID: rbID,
		Institution:     "Old Uni",
		Degree:          "BA",
		FieldOfStudy:    "Math",
		StartDate:       "2015-09",
		EndDate:         "2019-06",
		IsCurrent:       false,
		GPA:             "3.5",
		Description:     "Old desc",
		SortOrder:       0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Education
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetEducationByIDFunc: func(_ context.Context, _, _ string) (*model.Education, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateEducationFunc: func(_ context.Context, e *model.Education) error {
				updated = e
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateEducationRequest{
			Institution:  strPtr("MIT"),
			Degree:       strPtr("MS"),
			FieldOfStudy: strPtr("AI"),
			StartDate:    strPtr("2020-09"),
			EndDate:      strPtr("2022-06"),
			IsCurrent:    boolPtr(true),
			GPA:          strPtr("4.0"),
			Description:  strPtr("New desc"),
			SortOrder:    intPtr(1),
		}
		result, err := svc.UpdateEducation(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "MIT", result.Institution)
		assert.Equal(t, "MS", result.Degree)
		assert.Equal(t, "AI", result.FieldOfStudy)
		assert.Equal(t, "2020-09", result.StartDate)
		assert.Equal(t, "2022-06", result.EndDate)
		assert.True(t, result.IsCurrent)
		assert.Equal(t, "4.0", result.GPA)
		assert.Equal(t, "New desc", result.Description)
		assert.Equal(t, 1, result.SortOrder)
		assert.Equal(t, "MIT", updated.Institution)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateEducation(ctx, userID, rbID, entryID, &model.UpdateEducationRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetEducationByIDFunc: func(_ context.Context, _, _ string) (*model.Education, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateEducation(ctx, userID, rbID, entryID, &model.UpdateEducationRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetEducationByIDFunc: func(_ context.Context, _, _ string) (*model.Education, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateEducationFunc: func(_ context.Context, _ *model.Education) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateEducation(ctx, userID, rbID, entryID, &model.UpdateEducationRequest{Institution: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteEducation(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "edu-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteEducationFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteEducation(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteEducation(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteEducationFunc: func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteEducation(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Skill ---

func TestCreateSkill(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateSkillRequest{Name: "Go", Level: "Expert", SortOrder: 0}

	t.Run("success", func(t *testing.T) {
		var saved *model.Skill
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateSkillFunc: func(_ context.Context, s *model.Skill) error {
				s.ID = "skill-1"
				saved = s
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateSkill(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "skill-1", result.ID)
		assert.Equal(t, "Go", result.Name)
		assert.Equal(t, "Expert", result.Level)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateSkill(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateSkillFunc:     func(_ context.Context, _ *model.Skill) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateSkill(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateSkill(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "skill-1"

	existing := &model.Skill{
		ID: entryID, ResumeBuilderID: rbID,
		Name: "Go", Level: "Intermediate", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Skill
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetSkillByIDFunc: func(_ context.Context, _, _ string) (*model.Skill, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateSkillFunc: func(_ context.Context, s *model.Skill) error {
				updated = s
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateSkillRequest{Name: strPtr("Rust"), Level: strPtr("Expert"), SortOrder: intPtr(2)}
		result, err := svc.UpdateSkill(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "Rust", result.Name)
		assert.Equal(t, "Expert", result.Level)
		assert.Equal(t, 2, result.SortOrder)
		assert.Equal(t, "Rust", updated.Name)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateSkill(ctx, userID, rbID, entryID, &model.UpdateSkillRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetSkillByIDFunc: func(_ context.Context, _, _ string) (*model.Skill, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateSkill(ctx, userID, rbID, entryID, &model.UpdateSkillRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetSkillByIDFunc: func(_ context.Context, _, _ string) (*model.Skill, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateSkillFunc: func(_ context.Context, _ *model.Skill) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateSkill(ctx, userID, rbID, entryID, &model.UpdateSkillRequest{Name: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteSkill(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "skill-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteSkillFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteSkill(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteSkill(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteSkillFunc:     func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteSkill(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Language ---

func TestCreateLanguage(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateLanguageRequest{Name: "English", Proficiency: "Native", SortOrder: 0}

	t.Run("success", func(t *testing.T) {
		var saved *model.Language
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateLanguageFunc: func(_ context.Context, l *model.Language) error {
				l.ID = "lang-1"
				saved = l
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateLanguage(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "lang-1", result.ID)
		assert.Equal(t, "English", result.Name)
		assert.Equal(t, "Native", result.Proficiency)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateLanguage(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateLanguageFunc:  func(_ context.Context, _ *model.Language) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateLanguage(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateLanguage(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "lang-1"

	existing := &model.Language{
		ID: entryID, ResumeBuilderID: rbID,
		Name: "English", Proficiency: "Intermediate", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Language
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetLanguageByIDFunc: func(_ context.Context, _, _ string) (*model.Language, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateLanguageFunc: func(_ context.Context, l *model.Language) error {
				updated = l
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateLanguageRequest{Name: strPtr("Spanish"), Proficiency: strPtr("Native"), SortOrder: intPtr(1)}
		result, err := svc.UpdateLanguage(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "Spanish", result.Name)
		assert.Equal(t, "Native", result.Proficiency)
		assert.Equal(t, 1, result.SortOrder)
		assert.Equal(t, "Spanish", updated.Name)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateLanguage(ctx, userID, rbID, entryID, &model.UpdateLanguageRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetLanguageByIDFunc: func(_ context.Context, _, _ string) (*model.Language, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateLanguage(ctx, userID, rbID, entryID, &model.UpdateLanguageRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetLanguageByIDFunc: func(_ context.Context, _, _ string) (*model.Language, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateLanguageFunc: func(_ context.Context, _ *model.Language) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateLanguage(ctx, userID, rbID, entryID, &model.UpdateLanguageRequest{Name: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteLanguage(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "lang-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteLanguageFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteLanguage(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteLanguage(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteLanguageFunc:  func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteLanguage(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Certification ---

func TestCreateCertification(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateCertificationRequest{
		Name: "AWS Solutions Architect", Issuer: "Amazon",
		IssueDate: "2023-06", ExpiryDate: "2026-06",
		URL: "https://aws.amazon.com/cert/123", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.Certification
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateCertificationFunc: func(_ context.Context, c *model.Certification) error {
				c.ID = "cert-1"
				saved = c
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateCertification(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "cert-1", result.ID)
		assert.Equal(t, "AWS Solutions Architect", result.Name)
		assert.Equal(t, "Amazon", result.Issuer)
		assert.Equal(t, "2023-06", result.IssueDate)
		assert.Equal(t, "2026-06", result.ExpiryDate)
		assert.Equal(t, "https://aws.amazon.com/cert/123", result.URL)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateCertification(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:     func(_ context.Context, _, _ string) error { return nil },
			CreateCertificationFunc: func(_ context.Context, _ *model.Certification) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateCertification(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateCertification(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "cert-1"

	existing := &model.Certification{
		ID: entryID, ResumeBuilderID: rbID,
		Name: "Old Cert", Issuer: "Old Issuer",
		IssueDate: "2020-01", ExpiryDate: "2023-01",
		URL: "https://old.com", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Certification
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetCertificationByIDFunc: func(_ context.Context, _, _ string) (*model.Certification, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateCertificationFunc: func(_ context.Context, c *model.Certification) error {
				updated = c
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateCertificationRequest{
			Name: strPtr("New Cert"), Issuer: strPtr("New Issuer"),
			IssueDate: strPtr("2024-01"), ExpiryDate: strPtr("2027-01"),
			URL: strPtr("https://new.com"), SortOrder: intPtr(1),
		}
		result, err := svc.UpdateCertification(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "New Cert", result.Name)
		assert.Equal(t, "New Issuer", result.Issuer)
		assert.Equal(t, "2024-01", result.IssueDate)
		assert.Equal(t, "2027-01", result.ExpiryDate)
		assert.Equal(t, "https://new.com", result.URL)
		assert.Equal(t, 1, result.SortOrder)
		assert.Equal(t, "New Cert", updated.Name)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateCertification(ctx, userID, rbID, entryID, &model.UpdateCertificationRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetCertificationByIDFunc: func(_ context.Context, _, _ string) (*model.Certification, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateCertification(ctx, userID, rbID, entryID, &model.UpdateCertificationRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetCertificationByIDFunc: func(_ context.Context, _, _ string) (*model.Certification, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateCertificationFunc: func(_ context.Context, _ *model.Certification) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateCertification(ctx, userID, rbID, entryID, &model.UpdateCertificationRequest{Name: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteCertification(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "cert-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteCertificationFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteCertification(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteCertification(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:     func(_ context.Context, _, _ string) error { return nil },
			DeleteCertificationFunc: func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteCertification(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Project ---

func TestCreateProject(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateProjectRequest{
		Name: "Jobber", URL: "https://github.com/jobber",
		StartDate: "2023-01", EndDate: "2024-01",
		Description: "Job search platform", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.Project
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateProjectFunc: func(_ context.Context, p *model.Project) error {
				p.ID = "proj-1"
				saved = p
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateProject(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "proj-1", result.ID)
		assert.Equal(t, "Jobber", result.Name)
		assert.Equal(t, "https://github.com/jobber", result.URL)
		assert.Equal(t, "2023-01", result.StartDate)
		assert.Equal(t, "2024-01", result.EndDate)
		assert.Equal(t, "Job search platform", result.Description)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateProject(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateProjectFunc:   func(_ context.Context, _ *model.Project) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateProject(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateProject(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "proj-1"

	existing := &model.Project{
		ID: entryID, ResumeBuilderID: rbID,
		Name: "Old Project", URL: "https://old.com",
		StartDate: "2020-01", EndDate: "2021-01",
		Description: "Old desc", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Project
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetProjectByIDFunc: func(_ context.Context, _, _ string) (*model.Project, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateProjectFunc: func(_ context.Context, p *model.Project) error {
				updated = p
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateProjectRequest{
			Name: strPtr("New Project"), URL: strPtr("https://new.com"),
			StartDate: strPtr("2024-01"), EndDate: strPtr("2025-01"),
			Description: strPtr("New desc"), SortOrder: intPtr(1),
		}
		result, err := svc.UpdateProject(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "New Project", result.Name)
		assert.Equal(t, "https://new.com", result.URL)
		assert.Equal(t, "2024-01", result.StartDate)
		assert.Equal(t, "2025-01", result.EndDate)
		assert.Equal(t, "New desc", result.Description)
		assert.Equal(t, 1, result.SortOrder)
		assert.Equal(t, "New Project", updated.Name)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateProject(ctx, userID, rbID, entryID, &model.UpdateProjectRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetProjectByIDFunc: func(_ context.Context, _, _ string) (*model.Project, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateProject(ctx, userID, rbID, entryID, &model.UpdateProjectRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetProjectByIDFunc: func(_ context.Context, _, _ string) (*model.Project, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateProjectFunc: func(_ context.Context, _ *model.Project) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateProject(ctx, userID, rbID, entryID, &model.UpdateProjectRequest{Name: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteProject(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "proj-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteProjectFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteProject(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteProject(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteProjectFunc:   func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteProject(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Volunteering ---

func TestCreateVolunteering(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateVolunteeringRequest{
		Organization: "Red Cross", Role: "Coordinator",
		StartDate: "2022-01", EndDate: "2023-01",
		Description: "Organized events", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.Volunteering
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateVolunteeringFunc: func(_ context.Context, v *model.Volunteering) error {
				v.ID = "vol-1"
				saved = v
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateVolunteering(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "vol-1", result.ID)
		assert.Equal(t, "Red Cross", result.Organization)
		assert.Equal(t, "Coordinator", result.Role)
		assert.Equal(t, "2022-01", result.StartDate)
		assert.Equal(t, "2023-01", result.EndDate)
		assert.Equal(t, "Organized events", result.Description)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateVolunteering(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:    func(_ context.Context, _, _ string) error { return nil },
			CreateVolunteeringFunc: func(_ context.Context, _ *model.Volunteering) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateVolunteering(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateVolunteering(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "vol-1"

	existing := &model.Volunteering{
		ID: entryID, ResumeBuilderID: rbID,
		Organization: "Old Org", Role: "Helper",
		StartDate: "2020-01", EndDate: "2021-01",
		Description: "Old desc", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.Volunteering
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetVolunteeringByIDFunc: func(_ context.Context, _, _ string) (*model.Volunteering, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateVolunteeringFunc: func(_ context.Context, v *model.Volunteering) error {
				updated = v
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateVolunteeringRequest{
			Organization: strPtr("New Org"), Role: strPtr("Lead"),
			StartDate: strPtr("2024-01"), EndDate: strPtr("2025-01"),
			Description: strPtr("New desc"), SortOrder: intPtr(1),
		}
		result, err := svc.UpdateVolunteering(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "New Org", result.Organization)
		assert.Equal(t, "Lead", result.Role)
		assert.Equal(t, "2024-01", result.StartDate)
		assert.Equal(t, "2025-01", result.EndDate)
		assert.Equal(t, "New desc", result.Description)
		assert.Equal(t, 1, result.SortOrder)
		assert.Equal(t, "New Org", updated.Organization)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateVolunteering(ctx, userID, rbID, entryID, &model.UpdateVolunteeringRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetVolunteeringByIDFunc: func(_ context.Context, _, _ string) (*model.Volunteering, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateVolunteering(ctx, userID, rbID, entryID, &model.UpdateVolunteeringRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetVolunteeringByIDFunc: func(_ context.Context, _, _ string) (*model.Volunteering, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateVolunteeringFunc: func(_ context.Context, _ *model.Volunteering) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateVolunteering(ctx, userID, rbID, entryID, &model.UpdateVolunteeringRequest{Organization: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteVolunteering(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "vol-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteVolunteeringFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteVolunteering(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteVolunteering(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:    func(_ context.Context, _, _ string) error { return nil },
			DeleteVolunteeringFunc: func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteVolunteering(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Custom Section ---

func TestCreateCustomSection(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	req := &model.CreateCustomSectionRequest{
		Title: "Hobbies", Content: "Reading, hiking", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var saved *model.CustomSection
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			CreateCustomSectionFunc: func(_ context.Context, cs *model.CustomSection) error {
				cs.ID = "cs-1"
				saved = cs
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.CreateCustomSection(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "cs-1", result.ID)
		assert.Equal(t, "Hobbies", result.Title)
		assert.Equal(t, "Reading, hiking", result.Content)
		assert.Equal(t, 0, result.SortOrder)
		assert.Equal(t, rbID, saved.ResumeBuilderID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.CreateCustomSection(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:     func(_ context.Context, _, _ string) error { return nil },
			CreateCustomSectionFunc: func(_ context.Context, _ *model.CustomSection) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.CreateCustomSection(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestUpdateCustomSection(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "cs-1"

	existing := &model.CustomSection{
		ID: entryID, ResumeBuilderID: rbID,
		Title: "Old Title", Content: "Old content", SortOrder: 0,
	}

	t.Run("success", func(t *testing.T) {
		var updated *model.CustomSection
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetCustomSectionByIDFunc: func(_ context.Context, _, _ string) (*model.CustomSection, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateCustomSectionFunc: func(_ context.Context, cs *model.CustomSection) error {
				updated = cs
				return nil
			},
		}
		svc := newService(repo)

		req := &model.UpdateCustomSectionRequest{
			Title: strPtr("New Title"), Content: strPtr("New content"), SortOrder: intPtr(2),
		}
		result, err := svc.UpdateCustomSection(ctx, userID, rbID, entryID, req)
		require.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
		assert.Equal(t, "New content", result.Content)
		assert.Equal(t, 2, result.SortOrder)
		assert.Equal(t, "New Title", updated.Title)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		result, err := svc.UpdateCustomSection(ctx, userID, rbID, entryID, &model.UpdateCustomSectionRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("get by id error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetCustomSectionByIDFunc: func(_ context.Context, _, _ string) (*model.CustomSection, error) {
				return nil, model.ErrSectionEntryNotFound
			},
		}
		svc := newService(repo)

		result, err := svc.UpdateCustomSection(ctx, userID, rbID, entryID, &model.UpdateCustomSectionRequest{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
	})

	t.Run("repo update error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetCustomSectionByIDFunc: func(_ context.Context, _, _ string) (*model.CustomSection, error) {
				copy := *existing
				return &copy, nil
			},
			UpdateCustomSectionFunc: func(_ context.Context, _ *model.CustomSection) error { return errDB },
		}
		svc := newService(repo)

		result, err := svc.UpdateCustomSection(ctx, userID, rbID, entryID, &model.UpdateCustomSectionRequest{Title: strPtr("X")})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})
}

func TestDeleteCustomSection(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"
	entryID := "cs-1"

	t.Run("success", func(t *testing.T) {
		var deletedRBID, deletedEntryID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteCustomSectionFunc: func(_ context.Context, rid, eid string) error {
				deletedRBID = rid
				deletedEntryID = eid
				return nil
			},
		}
		svc := newService(repo)

		err := svc.DeleteCustomSection(ctx, userID, rbID, entryID)
		require.NoError(t, err)
		assert.Equal(t, rbID, deletedRBID)
		assert.Equal(t, entryID, deletedEntryID)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		err := svc.DeleteCustomSection(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:     func(_ context.Context, _, _ string) error { return nil },
			DeleteCustomSectionFunc: func(_ context.Context, _, _ string) error { return errDB },
		}
		svc := newService(repo)

		err := svc.DeleteCustomSection(ctx, userID, rbID, entryID)
		assert.ErrorIs(t, err, errDB)
	})
}

// --- Section Order ---

func TestUpdateSectionOrder(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"

	t.Run("success", func(t *testing.T) {
		var savedOrders []*model.SectionOrder
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertSectionOrderFunc: func(_ context.Context, _ string, orders []*model.SectionOrder) error {
				savedOrders = orders
				return nil
			},
		}
		svc := newService(repo)

		req := &model.BatchUpdateSectionOrderRequest{
			Sections: []model.UpdateSectionOrderRequest{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true, Column: "main"},
				{SectionKey: "education", SortOrder: 1, IsVisible: false, Column: "sidebar"},
			},
		}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "experience", result[0].SectionKey)
		assert.Equal(t, 0, result[0].SortOrder)
		assert.True(t, result[0].IsVisible)
		assert.Equal(t, "main", result[0].Column)
		assert.Equal(t, "education", result[1].SectionKey)
		assert.Equal(t, 1, result[1].SortOrder)
		assert.False(t, result[1].IsVisible)
		assert.Equal(t, "sidebar", result[1].Column)
		assert.Len(t, savedOrders, 2)
		assert.Equal(t, rbID, savedOrders[0].ResumeBuilderID)
	})

	t.Run("defaults empty column to main", func(t *testing.T) {
		var savedOrders []*model.SectionOrder
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			UpsertSectionOrderFunc: func(_ context.Context, _ string, orders []*model.SectionOrder) error {
				savedOrders = orders
				return nil
			},
		}
		svc := newService(repo)

		req := &model.BatchUpdateSectionOrderRequest{
			Sections: []model.UpdateSectionOrderRequest{
				{SectionKey: "skills", SortOrder: 0, IsVisible: true, Column: ""},
			},
		}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Equal(t, "main", result[0].Column)
		assert.Equal(t, "main", savedOrders[0].Column)
	})

	t.Run("invalid section key", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
		}
		svc := newService(repo)

		req := &model.BatchUpdateSectionOrderRequest{
			Sections: []model.UpdateSectionOrderRequest{
				{SectionKey: "invalid_key", SortOrder: 0, IsVisible: true},
			},
		}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrInvalidSectionKey)
	})

	t.Run("invalid column value", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
		}
		svc := newService(repo)

		req := &model.BatchUpdateSectionOrderRequest{
			Sections: []model.UpdateSectionOrderRequest{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true, Column: "footer"},
			},
		}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrInvalidColumnValue)
	})

	t.Run("ownership error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return model.ErrNotOwner },
		}
		svc := newService(repo)

		req := &model.BatchUpdateSectionOrderRequest{
			Sections: []model.UpdateSectionOrderRequest{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true},
			},
		}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:    func(_ context.Context, _, _ string) error { return nil },
			UpsertSectionOrderFunc: func(_ context.Context, _ string, _ []*model.SectionOrder) error { return errDB },
		}
		svc := newService(repo)

		req := &model.BatchUpdateSectionOrderRequest{
			Sections: []model.UpdateSectionOrderRequest{
				{SectionKey: "experience", SortOrder: 0, IsVisible: true, Column: "main"},
			},
		}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, errDB)
	})

	t.Run("all valid section keys accepted", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc:    func(_ context.Context, _, _ string) error { return nil },
			UpsertSectionOrderFunc: func(_ context.Context, _ string, _ []*model.SectionOrder) error { return nil },
		}
		svc := newService(repo)

		sections := make([]model.UpdateSectionOrderRequest, 0, len(ValidSectionKeys))
		i := 0
		for key := range ValidSectionKeys {
			sections = append(sections, model.UpdateSectionOrderRequest{
				SectionKey: key, SortOrder: i, IsVisible: true, Column: "main",
			})
			i++
		}

		req := &model.BatchUpdateSectionOrderRequest{Sections: sections}
		result, err := svc.UpdateSectionOrder(ctx, userID, rbID, req)
		require.NoError(t, err)
		assert.Len(t, result, len(ValidSectionKeys))
	})
}
