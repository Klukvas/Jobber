package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type MockResumeBuilderRepository struct {
	CreateFunc           func(ctx context.Context, rb *model.ResumeBuilder) error
	GetByIDFunc          func(ctx context.Context, id string) (*model.ResumeBuilder, error)
	ListFunc             func(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error)
	UpdateFunc           func(ctx context.Context, rb *model.ResumeBuilder) error
	DeleteFunc           func(ctx context.Context, id string) error
	GetFullResumeFunc    func(ctx context.Context, id string) (*model.FullResumeDTO, error)
	VerifyOwnershipFunc  func(ctx context.Context, userID, resumeBuilderID string) error
	RunInTransactionFunc func(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error

	UpsertContactFunc func(ctx context.Context, contact *model.Contact) error
	GetContactFunc    func(ctx context.Context, resumeBuilderID string) (*model.Contact, error)
	UpsertSummaryFunc func(ctx context.Context, summary *model.Summary) error
	GetSummaryFunc    func(ctx context.Context, resumeBuilderID string) (*model.Summary, error)

	CreateExperienceFunc    func(ctx context.Context, exp *model.Experience) error
	UpdateExperienceFunc    func(ctx context.Context, exp *model.Experience) error
	DeleteExperienceFunc    func(ctx context.Context, resumeBuilderID, id string) error
	ListExperiencesFunc     func(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error)
	GetExperienceByIDFunc   func(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error)

	CreateEducationFunc     func(ctx context.Context, edu *model.Education) error
	UpdateEducationFunc     func(ctx context.Context, edu *model.Education) error
	DeleteEducationFunc     func(ctx context.Context, resumeBuilderID, id string) error
	ListEducationsFunc      func(ctx context.Context, resumeBuilderID string) ([]*model.Education, error)
	GetEducationByIDFunc    func(ctx context.Context, resumeBuilderID, id string) (*model.Education, error)

	CreateSkillFunc         func(ctx context.Context, skill *model.Skill) error
	UpdateSkillFunc         func(ctx context.Context, skill *model.Skill) error
	DeleteSkillFunc         func(ctx context.Context, resumeBuilderID, id string) error
	ListSkillsFunc          func(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error)
	GetSkillByIDFunc        func(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error)

	CreateLanguageFunc      func(ctx context.Context, lang *model.Language) error
	UpdateLanguageFunc      func(ctx context.Context, lang *model.Language) error
	DeleteLanguageFunc      func(ctx context.Context, resumeBuilderID, id string) error
	ListLanguagesFunc       func(ctx context.Context, resumeBuilderID string) ([]*model.Language, error)
	GetLanguageByIDFunc     func(ctx context.Context, resumeBuilderID, id string) (*model.Language, error)

	CreateCertificationFunc func(ctx context.Context, cert *model.Certification) error
	UpdateCertificationFunc func(ctx context.Context, cert *model.Certification) error
	DeleteCertificationFunc func(ctx context.Context, resumeBuilderID, id string) error
	ListCertificationsFunc  func(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error)
	GetCertificationByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error)

	CreateProjectFunc       func(ctx context.Context, proj *model.Project) error
	UpdateProjectFunc       func(ctx context.Context, proj *model.Project) error
	DeleteProjectFunc       func(ctx context.Context, resumeBuilderID, id string) error
	ListProjectsFunc        func(ctx context.Context, resumeBuilderID string) ([]*model.Project, error)
	GetProjectByIDFunc      func(ctx context.Context, resumeBuilderID, id string) (*model.Project, error)

	CreateVolunteeringFunc  func(ctx context.Context, vol *model.Volunteering) error
	UpdateVolunteeringFunc  func(ctx context.Context, vol *model.Volunteering) error
	DeleteVolunteeringFunc  func(ctx context.Context, resumeBuilderID, id string) error
	ListVolunteeringFunc    func(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error)
	GetVolunteeringByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error)

	CreateCustomSectionFunc  func(ctx context.Context, cs *model.CustomSection) error
	UpdateCustomSectionFunc  func(ctx context.Context, cs *model.CustomSection) error
	DeleteCustomSectionFunc  func(ctx context.Context, resumeBuilderID, id string) error
	ListCustomSectionsFunc   func(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error)
	GetCustomSectionByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error)

	UpsertSectionOrderFunc func(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error
	ListSectionOrdersFunc  func(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error)
}

func (m *MockResumeBuilderRepository) Create(ctx context.Context, rb *model.ResumeBuilder) error {
	if m.CreateFunc != nil { return m.CreateFunc(ctx, rb) }
	return nil
}
func (m *MockResumeBuilderRepository) GetByID(ctx context.Context, id string) (*model.ResumeBuilder, error) {
	if m.GetByIDFunc != nil { return m.GetByIDFunc(ctx, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) List(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error) {
	if m.ListFunc != nil { return m.ListFunc(ctx, userID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) Update(ctx context.Context, rb *model.ResumeBuilder) error {
	if m.UpdateFunc != nil { return m.UpdateFunc(ctx, rb) }
	return nil
}
func (m *MockResumeBuilderRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil { return m.DeleteFunc(ctx, id) }
	return nil
}
func (m *MockResumeBuilderRepository) GetFullResume(ctx context.Context, id string) (*model.FullResumeDTO, error) {
	if m.GetFullResumeFunc != nil { return m.GetFullResumeFunc(ctx, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	if m.VerifyOwnershipFunc != nil { return m.VerifyOwnershipFunc(ctx, userID, resumeBuilderID) }
	return nil
}
func (m *MockResumeBuilderRepository) RunInTransaction(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error {
	if m.RunInTransactionFunc != nil { return m.RunInTransactionFunc(ctx, fn) }
	return fn(m)
}
func (m *MockResumeBuilderRepository) UpsertContact(ctx context.Context, contact *model.Contact) error {
	if m.UpsertContactFunc != nil { return m.UpsertContactFunc(ctx, contact) }
	return nil
}
func (m *MockResumeBuilderRepository) GetContact(ctx context.Context, resumeBuilderID string) (*model.Contact, error) {
	if m.GetContactFunc != nil { return m.GetContactFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) UpsertSummary(ctx context.Context, summary *model.Summary) error {
	if m.UpsertSummaryFunc != nil { return m.UpsertSummaryFunc(ctx, summary) }
	return nil
}
func (m *MockResumeBuilderRepository) GetSummary(ctx context.Context, resumeBuilderID string) (*model.Summary, error) {
	if m.GetSummaryFunc != nil { return m.GetSummaryFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateExperience(ctx context.Context, exp *model.Experience) error {
	if m.CreateExperienceFunc != nil { return m.CreateExperienceFunc(ctx, exp) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateExperience(ctx context.Context, exp *model.Experience) error {
	if m.UpdateExperienceFunc != nil { return m.UpdateExperienceFunc(ctx, exp) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteExperience(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteExperienceFunc != nil { return m.DeleteExperienceFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListExperiences(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error) {
	if m.ListExperiencesFunc != nil { return m.ListExperiencesFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error) {
	if m.GetExperienceByIDFunc != nil { return m.GetExperienceByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateEducation(ctx context.Context, edu *model.Education) error {
	if m.CreateEducationFunc != nil { return m.CreateEducationFunc(ctx, edu) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateEducation(ctx context.Context, edu *model.Education) error {
	if m.UpdateEducationFunc != nil { return m.UpdateEducationFunc(ctx, edu) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteEducation(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteEducationFunc != nil { return m.DeleteEducationFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListEducations(ctx context.Context, resumeBuilderID string) ([]*model.Education, error) {
	if m.ListEducationsFunc != nil { return m.ListEducationsFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*model.Education, error) {
	if m.GetEducationByIDFunc != nil { return m.GetEducationByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateSkill(ctx context.Context, skill *model.Skill) error {
	if m.CreateSkillFunc != nil { return m.CreateSkillFunc(ctx, skill) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateSkill(ctx context.Context, skill *model.Skill) error {
	if m.UpdateSkillFunc != nil { return m.UpdateSkillFunc(ctx, skill) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteSkill(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteSkillFunc != nil { return m.DeleteSkillFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListSkills(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error) {
	if m.ListSkillsFunc != nil { return m.ListSkillsFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error) {
	if m.GetSkillByIDFunc != nil { return m.GetSkillByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateLanguage(ctx context.Context, lang *model.Language) error {
	if m.CreateLanguageFunc != nil { return m.CreateLanguageFunc(ctx, lang) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateLanguage(ctx context.Context, lang *model.Language) error {
	if m.UpdateLanguageFunc != nil { return m.UpdateLanguageFunc(ctx, lang) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteLanguageFunc != nil { return m.DeleteLanguageFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListLanguages(ctx context.Context, resumeBuilderID string) ([]*model.Language, error) {
	if m.ListLanguagesFunc != nil { return m.ListLanguagesFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*model.Language, error) {
	if m.GetLanguageByIDFunc != nil { return m.GetLanguageByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateCertification(ctx context.Context, cert *model.Certification) error {
	if m.CreateCertificationFunc != nil { return m.CreateCertificationFunc(ctx, cert) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateCertification(ctx context.Context, cert *model.Certification) error {
	if m.UpdateCertificationFunc != nil { return m.UpdateCertificationFunc(ctx, cert) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteCertification(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCertificationFunc != nil { return m.DeleteCertificationFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListCertifications(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error) {
	if m.ListCertificationsFunc != nil { return m.ListCertificationsFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error) {
	if m.GetCertificationByIDFunc != nil { return m.GetCertificationByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateProject(ctx context.Context, proj *model.Project) error {
	if m.CreateProjectFunc != nil { return m.CreateProjectFunc(ctx, proj) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateProject(ctx context.Context, proj *model.Project) error {
	if m.UpdateProjectFunc != nil { return m.UpdateProjectFunc(ctx, proj) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteProject(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteProjectFunc != nil { return m.DeleteProjectFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListProjects(ctx context.Context, resumeBuilderID string) ([]*model.Project, error) {
	if m.ListProjectsFunc != nil { return m.ListProjectsFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*model.Project, error) {
	if m.GetProjectByIDFunc != nil { return m.GetProjectByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	if m.CreateVolunteeringFunc != nil { return m.CreateVolunteeringFunc(ctx, vol) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	if m.UpdateVolunteeringFunc != nil { return m.UpdateVolunteeringFunc(ctx, vol) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteVolunteeringFunc != nil { return m.DeleteVolunteeringFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error) {
	if m.ListVolunteeringFunc != nil { return m.ListVolunteeringFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error) {
	if m.GetVolunteeringByIDFunc != nil { return m.GetVolunteeringByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	if m.CreateCustomSectionFunc != nil { return m.CreateCustomSectionFunc(ctx, cs) }
	return nil
}
func (m *MockResumeBuilderRepository) UpdateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	if m.UpdateCustomSectionFunc != nil { return m.UpdateCustomSectionFunc(ctx, cs) }
	return nil
}
func (m *MockResumeBuilderRepository) DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCustomSectionFunc != nil { return m.DeleteCustomSectionFunc(ctx, resumeBuilderID, id) }
	return nil
}
func (m *MockResumeBuilderRepository) ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error) {
	if m.ListCustomSectionsFunc != nil { return m.ListCustomSectionsFunc(ctx, resumeBuilderID) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error) {
	if m.GetCustomSectionByIDFunc != nil { return m.GetCustomSectionByIDFunc(ctx, resumeBuilderID, id) }
	return nil, nil
}
func (m *MockResumeBuilderRepository) UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error {
	if m.UpsertSectionOrderFunc != nil { return m.UpsertSectionOrderFunc(ctx, resumeBuilderID, orders) }
	return nil
}
func (m *MockResumeBuilderRepository) ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error) {
	if m.ListSectionOrdersFunc != nil { return m.ListSectionOrdersFunc(ctx, resumeBuilderID) }
	return nil, nil
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

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func newTestResume() *model.ResumeBuilder {
	return &model.ResumeBuilder{
		ID:           "rb-1",
		UserID:       "user-1",
		Title:        "Test Resume",
		TemplateID:   "00000000-0000-0000-0000-000000000001",
		FontFamily:   "Inter",
		PrimaryColor: "#2563eb",
		Spacing:      100,
		MarginTop:    40,
		MarginBottom: 40,
		MarginLeft:   40,
		MarginRight:  40,
		LayoutMode:   "single",
		SidebarWidth: 35,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func newService(repo *MockResumeBuilderRepository) *ResumeBuilderService {
	return NewResumeBuilderService(repo, &MockLimitChecker{})
}

// --- Tests ---

func TestValidTemplateIDs(t *testing.T) {
	t.Run("contains all 12 template UUIDs", func(t *testing.T) {
		expectedIDs := []string{
			"00000000-0000-0000-0000-000000000001",
			"00000000-0000-0000-0000-000000000002",
			"00000000-0000-0000-0000-000000000003",
			"00000000-0000-0000-0000-000000000004",
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006",
			"00000000-0000-0000-0000-000000000007",
			"00000000-0000-0000-0000-000000000008",
			"00000000-0000-0000-0000-000000000009",
			"00000000-0000-0000-0000-00000000000a",
			"00000000-0000-0000-0000-00000000000b",
			"00000000-0000-0000-0000-00000000000c",
		}

		assert.Len(t, ValidTemplateIDs, 12)
		for _, id := range expectedIDs {
			assert.True(t, ValidTemplateIDs[id], "expected %s to be valid", id)
		}
	})

	t.Run("rejects unknown template IDs", func(t *testing.T) {
		invalidIDs := []string{
			"00000000-0000-0000-0000-000000000000",
			"00000000-0000-0000-0000-00000000000d",
			"not-a-uuid",
			"",
		}
		for _, id := range invalidIDs {
			assert.False(t, ValidTemplateIDs[id], "expected %s to be invalid", id)
		}
	})
}

func TestUpdate_TemplateValidation(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"

	t.Run("accepts all valid template IDs", func(t *testing.T) {
		for templateID := range ValidTemplateIDs {
			repo := &MockResumeBuilderRepository{
				VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
				GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
					return newTestResume(), nil
				},
				UpdateFunc: func(_ context.Context, _ *model.ResumeBuilder) error { return nil },
			}
			svc := newService(repo)

			result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
				TemplateID: strPtr(templateID),
			})

			require.NoError(t, err, "template %s should be valid", templateID)
			assert.Equal(t, templateID, result.TemplateID)
		}
	})

	t.Run("accepts new template IDs (bold, accent, timeline, vivid)", func(t *testing.T) {
		newTemplates := map[string]string{
			"bold":     "00000000-0000-0000-0000-000000000009",
			"accent":   "00000000-0000-0000-0000-00000000000a",
			"timeline": "00000000-0000-0000-0000-00000000000b",
			"vivid":    "00000000-0000-0000-0000-00000000000c",
		}

		for name, templateID := range newTemplates {
			var updatedRB *model.ResumeBuilder
			repo := &MockResumeBuilderRepository{
				VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
				GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
					return newTestResume(), nil
				},
				UpdateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
					updatedRB = rb
					return nil
				},
			}
			svc := newService(repo)

			result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
				TemplateID: strPtr(templateID),
			})

			require.NoError(t, err, "template %s (%s) should be valid", name, templateID)
			assert.Equal(t, templateID, result.TemplateID)
			assert.Equal(t, templateID, updatedRB.TemplateID, "repo should receive updated template ID")
		}
	})

	t.Run("rejects invalid template ID", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
				return newTestResume(), nil
			},
		}
		svc := newService(repo)

		result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			TemplateID: strPtr("00000000-0000-0000-0000-ffffffffffff"),
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrInvalidTemplate)
	})

	t.Run("rejects empty template ID", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
				return newTestResume(), nil
			},
		}
		svc := newService(repo)

		result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			TemplateID: strPtr(""),
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrInvalidTemplate)
	})

	t.Run("nil template ID does not change existing template", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
				return newTestResume(), nil
			},
			UpdateFunc: func(_ context.Context, _ *model.ResumeBuilder) error { return nil },
		}
		svc := newService(repo)

		result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			// TemplateID is nil
		})

		require.NoError(t, err)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", result.TemplateID)
	})
}

func TestUpdate_OwnershipCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when not owner", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}
		svc := newService(repo)

		result, err := svc.Update(ctx, "user-2", "rb-1", &model.UpdateResumeBuilderRequest{
			TemplateID: strPtr("00000000-0000-0000-0000-000000000009"),
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})
}

func TestUpdate_AllFieldValidation(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"

	defaultRepo := func() *MockResumeBuilderRepository {
		return &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
				return newTestResume(), nil
			},
			UpdateFunc: func(_ context.Context, _ *model.ResumeBuilder) error { return nil },
		}
	}

	t.Run("updates all fields at once", func(t *testing.T) {
		var saved *model.ResumeBuilder
		repo := defaultRepo()
		repo.UpdateFunc = func(_ context.Context, rb *model.ResumeBuilder) error {
			saved = rb
			return nil
		}
		svc := newService(repo)

		result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			Title:        strPtr("New Title"),
			TemplateID:   strPtr("00000000-0000-0000-0000-00000000000c"),
			FontFamily:   strPtr("Georgia"),
			PrimaryColor: strPtr("#FF5733"),
			Spacing:      intPtr(80),
			MarginTop:    intPtr(50),
			MarginBottom: intPtr(50),
			MarginLeft:   intPtr(30),
			MarginRight:  intPtr(30),
			LayoutMode:   strPtr("double-left"),
			SidebarWidth: intPtr(40),
		})

		require.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
		assert.Equal(t, "00000000-0000-0000-0000-00000000000c", result.TemplateID)
		assert.Equal(t, "Georgia", result.FontFamily)
		assert.Equal(t, "#FF5733", result.PrimaryColor)
		assert.Equal(t, 80, result.Spacing)
		assert.Equal(t, 50, result.MarginTop)
		assert.Equal(t, 50, result.MarginBottom)
		assert.Equal(t, 30, result.MarginLeft)
		assert.Equal(t, 30, result.MarginRight)
		assert.Equal(t, "double-left", result.LayoutMode)
		assert.Equal(t, 40, result.SidebarWidth)
		assert.Equal(t, saved.TemplateID, "00000000-0000-0000-0000-00000000000c")
	})

	t.Run("rejects invalid font", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			FontFamily: strPtr("ComicSansXYZ"),
		})
		assert.ErrorIs(t, err, model.ErrInvalidFont)
	})

	t.Run("rejects invalid color", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			PrimaryColor: strPtr("not-a-color"),
		})
		assert.ErrorIs(t, err, model.ErrInvalidColor)
	})

	t.Run("rejects spacing below 50", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			Spacing: intPtr(10),
		})
		assert.ErrorIs(t, err, model.ErrInvalidSpacing)
	})

	t.Run("rejects spacing above 150", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			Spacing: intPtr(200),
		})
		assert.ErrorIs(t, err, model.ErrInvalidSpacing)
	})

	t.Run("rejects margin below 0", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			MarginTop: intPtr(-1),
		})
		assert.ErrorIs(t, err, model.ErrInvalidMargin)
	})

	t.Run("rejects margin above 200", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			MarginBottom: intPtr(201),
		})
		assert.ErrorIs(t, err, model.ErrInvalidMargin)
	})

	t.Run("rejects invalid layout mode", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			LayoutMode: strPtr("triple"),
		})
		assert.ErrorIs(t, err, model.ErrInvalidLayoutMode)
	})

	t.Run("accepts all valid layout modes", func(t *testing.T) {
		for _, mode := range []string{"single", "double-left", "double-right", "custom"} {
			svc := newService(defaultRepo())
			result, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
				LayoutMode: strPtr(mode),
			})
			require.NoError(t, err, "layout mode %s should be valid", mode)
			assert.Equal(t, mode, result.LayoutMode)
		}
	})

	t.Run("rejects sidebar width below 25", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			SidebarWidth: intPtr(20),
		})
		assert.ErrorIs(t, err, model.ErrInvalidSidebarWidth)
	})

	t.Run("rejects sidebar width above 50", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			SidebarWidth: intPtr(60),
		})
		assert.ErrorIs(t, err, model.ErrInvalidSidebarWidth)
	})

	t.Run("returns error when repo GetByID fails", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetByIDFunc: func(_ context.Context, _ string) (*model.ResumeBuilder, error) {
				return nil, model.ErrResumeBuilderNotFound
			},
		}
		svc := newService(repo)
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{})
		assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	})

	t.Run("returns error when repo Update fails", func(t *testing.T) {
		repo := defaultRepo()
		repo.UpdateFunc = func(_ context.Context, _ *model.ResumeBuilder) error {
			return errors.New("db error")
		}
		svc := newService(repo)
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			Title: strPtr("New Title"),
		})
		assert.Error(t, err)
	})

	t.Run("validates left and right margins", func(t *testing.T) {
		svc := newService(defaultRepo())
		_, err := svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			MarginLeft: intPtr(-5),
		})
		assert.ErrorIs(t, err, model.ErrInvalidMargin)

		_, err = svc.Update(ctx, userID, rbID, &model.UpdateResumeBuilderRequest{
			MarginRight: intPtr(250),
		})
		assert.ErrorIs(t, err, model.ErrInvalidMargin)
	})
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	t.Run("creates with default values when empty request", func(t *testing.T) {
		var createdRB *model.ResumeBuilder
		repo := &MockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "new-rb-1"
				createdRB = rb
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.Create(ctx, userID, &model.CreateResumeBuilderRequest{})
		require.NoError(t, err)
		assert.Equal(t, "Untitled Resume", result.Title)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", result.TemplateID)
		assert.Equal(t, "Georgia", createdRB.FontFamily)
		assert.Equal(t, "#2563eb", createdRB.PrimaryColor)
		assert.Equal(t, 100, createdRB.Spacing)
		assert.Equal(t, "single", createdRB.LayoutMode)
	})

	t.Run("creates with new template IDs", func(t *testing.T) {
		for _, tid := range []string{
			"00000000-0000-0000-0000-000000000009",
			"00000000-0000-0000-0000-00000000000a",
			"00000000-0000-0000-0000-00000000000b",
			"00000000-0000-0000-0000-00000000000c",
		} {
			repo := &MockResumeBuilderRepository{
				CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
					rb.ID = "new-rb"
					return nil
				},
			}
			svc := newService(repo)

			result, err := svc.Create(ctx, userID, &model.CreateResumeBuilderRequest{
				Title:      "Bold Resume",
				TemplateID: tid,
			})
			require.NoError(t, err)
			assert.Equal(t, tid, result.TemplateID)
		}
	})

	t.Run("trims title whitespace", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "new-rb"
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.Create(ctx, userID, &model.CreateResumeBuilderRequest{
			Title: "  My Resume  ",
		})
		require.NoError(t, err)
		assert.Equal(t, "My Resume", result.Title)
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		limitErr := errors.New("limit exceeded")
		svc := NewResumeBuilderService(
			&MockResumeBuilderRepository{},
			&MockLimitChecker{
				CheckLimitFunc: func(_ context.Context, _, _ string) error { return limitErr },
			},
		)

		_, err := svc.Create(ctx, userID, &model.CreateResumeBuilderRequest{})
		assert.ErrorIs(t, err, limitErr)
	})

	t.Run("returns error when repo Create fails", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, _ *model.ResumeBuilder) error {
				return errors.New("db error")
			},
		}
		svc := newService(repo)

		_, err := svc.Create(ctx, userID, &model.CreateResumeBuilderRequest{})
		assert.Error(t, err)
	})

	t.Run("seeds default section order", func(t *testing.T) {
		var seededOrders []*model.SectionOrder
		repo := &MockResumeBuilderRepository{
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "new-rb"
				return nil
			},
			UpsertSectionOrderFunc: func(_ context.Context, _ string, orders []*model.SectionOrder) error {
				seededOrders = orders
				return nil
			},
		}
		svc := newService(repo)

		_, err := svc.Create(ctx, userID, &model.CreateResumeBuilderRequest{})
		require.NoError(t, err)
		assert.Len(t, seededOrders, 10) // 10 default sections
	})
}

func TestDuplicate(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	rbID := "rb-1"

	fullResume := &model.FullResumeDTO{
		ResumeBuilderDTO: &model.ResumeBuilderDTO{
			ID:           rbID,
			Title:        "Original",
			TemplateID:   "00000000-0000-0000-0000-00000000000c",
			FontFamily:   "Georgia",
			PrimaryColor: "#e11d48",
			Spacing:      100,
			MarginTop:    40,
			MarginBottom: 40,
			MarginLeft:   40,
			MarginRight:  40,
			LayoutMode:   "single",
			SidebarWidth: 35,
		},
		Contact: &model.ContactDTO{
			FullName: "Jane Doe",
			Email:    "jane@test.com",
		},
		Summary: &model.SummaryDTO{
			Content: "Experienced dev",
		},
		Experiences:    []*model.ExperienceDTO{{Company: "Acme"}},
		Educations:     []*model.EducationDTO{},
		Skills:         []*model.SkillDTO{{Name: "Go"}},
		Languages:      []*model.LanguageDTO{},
		Certifications: []*model.CertificationDTO{},
		Projects:       []*model.ProjectDTO{},
		Volunteering:   []*model.VolunteeringDTO{},
		CustomSections: []*model.CustomSectionDTO{},
		SectionOrder:   []*model.SectionOrderDTO{{SectionKey: "experience", SortOrder: 0, IsVisible: true}},
	}

	t.Run("duplicates with new template ID preserved", func(t *testing.T) {
		var createdRB *model.ResumeBuilder
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "new-rb"
				createdRB = rb
				return nil
			},
		}
		svc := newService(repo)

		result, err := svc.Duplicate(ctx, userID, rbID)
		require.NoError(t, err)
		assert.Contains(t, result.Title, "(Copy)")
		assert.Equal(t, "00000000-0000-0000-0000-00000000000c", result.TemplateID)
		assert.Equal(t, "#e11d48", createdRB.PrimaryColor)
	})

	t.Run("copies contact and summary", func(t *testing.T) {
		var contactCopied, summaryCopied bool
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "new-rb"
				return nil
			},
			UpsertContactFunc: func(_ context.Context, c *model.Contact) error {
				contactCopied = c.FullName == "Jane Doe"
				return nil
			},
			UpsertSummaryFunc: func(_ context.Context, s *model.Summary) error {
				summaryCopied = s.Content == "Experienced dev"
				return nil
			},
		}
		svc := newService(repo)

		_, err := svc.Duplicate(ctx, userID, rbID)
		require.NoError(t, err)
		assert.True(t, contactCopied)
		assert.True(t, summaryCopied)
	})

	t.Run("copies section order", func(t *testing.T) {
		var ordersCopied bool
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
			CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
				rb.ID = "new-rb"
				return nil
			},
			UpsertSectionOrderFunc: func(_ context.Context, _ string, orders []*model.SectionOrder) error {
				ordersCopied = len(orders) == 1 && orders[0].SectionKey == "experience"
				return nil
			},
		}
		svc := newService(repo)

		_, err := svc.Duplicate(ctx, userID, rbID)
		require.NoError(t, err)
		assert.True(t, ordersCopied)
	})

	t.Run("returns error when not owner", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}
		svc := newService(repo)
		_, err := svc.Duplicate(ctx, userID, rbID)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		limitErr := errors.New("limit exceeded")
		svc := NewResumeBuilderService(
			&MockResumeBuilderRepository{},
			&MockLimitChecker{
				CheckLimitFunc: func(_ context.Context, _, _ string) error { return limitErr },
			},
		)
		_, err := svc.Duplicate(ctx, userID, rbID)
		assert.ErrorIs(t, err, limitErr)
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes successfully", func(t *testing.T) {
		var deletedID string
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			DeleteFunc: func(_ context.Context, id string) error {
				deletedID = id
				return nil
			},
		}
		svc := newService(repo)
		err := svc.Delete(ctx, "user-1", "rb-1")
		require.NoError(t, err)
		assert.Equal(t, "rb-1", deletedID)
	})

	t.Run("returns error when not owner", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}
		svc := newService(repo)
		err := svc.Delete(ctx, "user-2", "rb-1")
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})
}

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("returns list of resumes", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			ListFunc: func(_ context.Context, _ string) ([]*model.ResumeBuilderDTO, error) {
				return []*model.ResumeBuilderDTO{
					{ID: "rb-1", Title: "Resume 1"},
					{ID: "rb-2", Title: "Resume 2"},
				}, nil
			},
		}
		svc := newService(repo)
		results, err := svc.List(ctx, "user-1")
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	t.Run("returns full resume", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "rb-1", TemplateID: "00000000-0000-0000-0000-000000000009"},
				}, nil
			},
		}
		svc := newService(repo)
		result, err := svc.Get(ctx, "user-1", "rb-1")
		require.NoError(t, err)
		assert.Equal(t, "00000000-0000-0000-0000-000000000009", result.TemplateID)
	})

	t.Run("returns error when not owner", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}
		svc := newService(repo)
		_, err := svc.Get(ctx, "user-2", "rb-1")
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})
}
