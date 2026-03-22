package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	clModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/service"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Mocks for AI handler tests ---

type mockAICoverLetterRepo struct {
	CreateFunc  func(ctx context.Context, cl *clModel.CoverLetter) (*clModel.CoverLetter, error)
	GetByIDFunc func(ctx context.Context, id string) (*clModel.CoverLetter, error)
	ListFunc    func(ctx context.Context, userID string) ([]*clModel.CoverLetter, error)
	UpdateFunc  func(ctx context.Context, cl *clModel.CoverLetter) (*clModel.CoverLetter, error)
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *mockAICoverLetterRepo) Create(ctx context.Context, cl *clModel.CoverLetter) (*clModel.CoverLetter, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, cl)
	}
	return cl, nil
}

func (m *mockAICoverLetterRepo) GetByID(ctx context.Context, id string) (*clModel.CoverLetter, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAICoverLetterRepo) List(ctx context.Context, userID string) ([]*clModel.CoverLetter, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockAICoverLetterRepo) Update(ctx context.Context, cl *clModel.CoverLetter) (*clModel.CoverLetter, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, cl)
	}
	return cl, nil
}

func (m *mockAICoverLetterRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

type mockAIResumeBuilderRepo struct {
	VerifyOwnershipFunc  func(ctx context.Context, userID, resumeBuilderID string) error
	GetFullResumeFunc    func(ctx context.Context, id string) (*rbModel.FullResumeDTO, error)
	CreateFunc           func(ctx context.Context, rb *rbModel.ResumeBuilder) error
	GetByIDFunc          func(ctx context.Context, id string) (*rbModel.ResumeBuilder, error)
	ListFunc             func(ctx context.Context, userID string) ([]*rbModel.ResumeBuilderDTO, error)
	UpdateFunc           func(ctx context.Context, rb *rbModel.ResumeBuilder) error
	DeleteFunc           func(ctx context.Context, id string) error
	RunInTransactionFunc func(ctx context.Context, fn func(txRepo rbPorts.ResumeBuilderRepository) error) error

	UpsertContactFunc func(ctx context.Context, contact *rbModel.Contact) error
	GetContactFunc    func(ctx context.Context, resumeBuilderID string) (*rbModel.Contact, error)
	UpsertSummaryFunc func(ctx context.Context, summary *rbModel.Summary) error
	GetSummaryFunc    func(ctx context.Context, resumeBuilderID string) (*rbModel.Summary, error)

	CreateExperienceFunc     func(ctx context.Context, exp *rbModel.Experience) error
	UpdateExperienceFunc     func(ctx context.Context, exp *rbModel.Experience) error
	DeleteExperienceFunc     func(ctx context.Context, resumeBuilderID, id string) error
	ListExperiencesFunc      func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Experience, error)
	GetExperienceByIDFunc    func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Experience, error)
	CreateEducationFunc      func(ctx context.Context, edu *rbModel.Education) error
	UpdateEducationFunc      func(ctx context.Context, edu *rbModel.Education) error
	DeleteEducationFunc      func(ctx context.Context, resumeBuilderID, id string) error
	ListEducationsFunc       func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Education, error)
	GetEducationByIDFunc     func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Education, error)
	CreateSkillFunc          func(ctx context.Context, skill *rbModel.Skill) error
	UpdateSkillFunc          func(ctx context.Context, skill *rbModel.Skill) error
	DeleteSkillFunc          func(ctx context.Context, resumeBuilderID, id string) error
	ListSkillsFunc           func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Skill, error)
	GetSkillByIDFunc         func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Skill, error)
	CreateLanguageFunc       func(ctx context.Context, lang *rbModel.Language) error
	UpdateLanguageFunc       func(ctx context.Context, lang *rbModel.Language) error
	DeleteLanguageFunc       func(ctx context.Context, resumeBuilderID, id string) error
	ListLanguagesFunc        func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Language, error)
	GetLanguageByIDFunc      func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Language, error)
	CreateCertificationFunc  func(ctx context.Context, cert *rbModel.Certification) error
	UpdateCertificationFunc  func(ctx context.Context, cert *rbModel.Certification) error
	DeleteCertificationFunc  func(ctx context.Context, resumeBuilderID, id string) error
	ListCertificationsFunc   func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Certification, error)
	GetCertificationByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Certification, error)
	CreateProjectFunc        func(ctx context.Context, proj *rbModel.Project) error
	UpdateProjectFunc        func(ctx context.Context, proj *rbModel.Project) error
	DeleteProjectFunc        func(ctx context.Context, resumeBuilderID, id string) error
	ListProjectsFunc         func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Project, error)
	GetProjectByIDFunc       func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Project, error)
	CreateVolunteeringFunc   func(ctx context.Context, vol *rbModel.Volunteering) error
	UpdateVolunteeringFunc   func(ctx context.Context, vol *rbModel.Volunteering) error
	DeleteVolunteeringFunc   func(ctx context.Context, resumeBuilderID, id string) error
	ListVolunteeringFunc     func(ctx context.Context, resumeBuilderID string) ([]*rbModel.Volunteering, error)
	GetVolunteeringByIDFunc  func(ctx context.Context, resumeBuilderID, id string) (*rbModel.Volunteering, error)
	CreateCustomSectionFunc  func(ctx context.Context, cs *rbModel.CustomSection) error
	UpdateCustomSectionFunc  func(ctx context.Context, cs *rbModel.CustomSection) error
	DeleteCustomSectionFunc  func(ctx context.Context, resumeBuilderID, id string) error
	ListCustomSectionsFunc   func(ctx context.Context, resumeBuilderID string) ([]*rbModel.CustomSection, error)
	GetCustomSectionByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*rbModel.CustomSection, error)
	UpsertSectionOrderFunc   func(ctx context.Context, resumeBuilderID string, orders []*rbModel.SectionOrder) error
	ListSectionOrdersFunc    func(ctx context.Context, resumeBuilderID string) ([]*rbModel.SectionOrder, error)
}

func (m *mockAIResumeBuilderRepo) Create(ctx context.Context, rb *rbModel.ResumeBuilder) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, rb)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) GetByID(ctx context.Context, id string) (*rbModel.ResumeBuilder, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) List(ctx context.Context, userID string) ([]*rbModel.ResumeBuilderDTO, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) Update(ctx context.Context, rb *rbModel.ResumeBuilder) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, rb)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) GetFullResume(ctx context.Context, id string) (*rbModel.FullResumeDTO, error) {
	if m.GetFullResumeFunc != nil {
		return m.GetFullResumeFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	if m.VerifyOwnershipFunc != nil {
		return m.VerifyOwnershipFunc(ctx, userID, resumeBuilderID)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) RunInTransaction(ctx context.Context, fn func(txRepo rbPorts.ResumeBuilderRepository) error) error {
	if m.RunInTransactionFunc != nil {
		return m.RunInTransactionFunc(ctx, fn)
	}
	return fn(m)
}
func (m *mockAIResumeBuilderRepo) UpsertContact(ctx context.Context, contact *rbModel.Contact) error {
	if m.UpsertContactFunc != nil {
		return m.UpsertContactFunc(ctx, contact)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) GetContact(ctx context.Context, resumeBuilderID string) (*rbModel.Contact, error) {
	if m.GetContactFunc != nil {
		return m.GetContactFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) UpsertSummary(ctx context.Context, summary *rbModel.Summary) error {
	if m.UpsertSummaryFunc != nil {
		return m.UpsertSummaryFunc(ctx, summary)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) GetSummary(ctx context.Context, resumeBuilderID string) (*rbModel.Summary, error) {
	if m.GetSummaryFunc != nil {
		return m.GetSummaryFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateExperience(ctx context.Context, exp *rbModel.Experience) error {
	if m.CreateExperienceFunc != nil {
		return m.CreateExperienceFunc(ctx, exp)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateExperience(ctx context.Context, exp *rbModel.Experience) error {
	if m.UpdateExperienceFunc != nil {
		return m.UpdateExperienceFunc(ctx, exp)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteExperience(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteExperienceFunc != nil {
		return m.DeleteExperienceFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListExperiences(ctx context.Context, resumeBuilderID string) ([]*rbModel.Experience, error) {
	if m.ListExperiencesFunc != nil {
		return m.ListExperiencesFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Experience, error) {
	if m.GetExperienceByIDFunc != nil {
		return m.GetExperienceByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateEducation(ctx context.Context, edu *rbModel.Education) error {
	if m.CreateEducationFunc != nil {
		return m.CreateEducationFunc(ctx, edu)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateEducation(ctx context.Context, edu *rbModel.Education) error {
	if m.UpdateEducationFunc != nil {
		return m.UpdateEducationFunc(ctx, edu)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteEducation(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteEducationFunc != nil {
		return m.DeleteEducationFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListEducations(ctx context.Context, resumeBuilderID string) ([]*rbModel.Education, error) {
	if m.ListEducationsFunc != nil {
		return m.ListEducationsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Education, error) {
	if m.GetEducationByIDFunc != nil {
		return m.GetEducationByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateSkill(ctx context.Context, skill *rbModel.Skill) error {
	if m.CreateSkillFunc != nil {
		return m.CreateSkillFunc(ctx, skill)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateSkill(ctx context.Context, skill *rbModel.Skill) error {
	if m.UpdateSkillFunc != nil {
		return m.UpdateSkillFunc(ctx, skill)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteSkill(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteSkillFunc != nil {
		return m.DeleteSkillFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListSkills(ctx context.Context, resumeBuilderID string) ([]*rbModel.Skill, error) {
	if m.ListSkillsFunc != nil {
		return m.ListSkillsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Skill, error) {
	if m.GetSkillByIDFunc != nil {
		return m.GetSkillByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateLanguage(ctx context.Context, lang *rbModel.Language) error {
	if m.CreateLanguageFunc != nil {
		return m.CreateLanguageFunc(ctx, lang)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateLanguage(ctx context.Context, lang *rbModel.Language) error {
	if m.UpdateLanguageFunc != nil {
		return m.UpdateLanguageFunc(ctx, lang)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteLanguageFunc != nil {
		return m.DeleteLanguageFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListLanguages(ctx context.Context, resumeBuilderID string) ([]*rbModel.Language, error) {
	if m.ListLanguagesFunc != nil {
		return m.ListLanguagesFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Language, error) {
	if m.GetLanguageByIDFunc != nil {
		return m.GetLanguageByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateCertification(ctx context.Context, cert *rbModel.Certification) error {
	if m.CreateCertificationFunc != nil {
		return m.CreateCertificationFunc(ctx, cert)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateCertification(ctx context.Context, cert *rbModel.Certification) error {
	if m.UpdateCertificationFunc != nil {
		return m.UpdateCertificationFunc(ctx, cert)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteCertification(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCertificationFunc != nil {
		return m.DeleteCertificationFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListCertifications(ctx context.Context, resumeBuilderID string) ([]*rbModel.Certification, error) {
	if m.ListCertificationsFunc != nil {
		return m.ListCertificationsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Certification, error) {
	if m.GetCertificationByIDFunc != nil {
		return m.GetCertificationByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateProject(ctx context.Context, proj *rbModel.Project) error {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, proj)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateProject(ctx context.Context, proj *rbModel.Project) error {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, proj)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteProject(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListProjects(ctx context.Context, resumeBuilderID string) ([]*rbModel.Project, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Project, error) {
	if m.GetProjectByIDFunc != nil {
		return m.GetProjectByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateVolunteering(ctx context.Context, vol *rbModel.Volunteering) error {
	if m.CreateVolunteeringFunc != nil {
		return m.CreateVolunteeringFunc(ctx, vol)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateVolunteering(ctx context.Context, vol *rbModel.Volunteering) error {
	if m.UpdateVolunteeringFunc != nil {
		return m.UpdateVolunteeringFunc(ctx, vol)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteVolunteeringFunc != nil {
		return m.DeleteVolunteeringFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*rbModel.Volunteering, error) {
	if m.ListVolunteeringFunc != nil {
		return m.ListVolunteeringFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Volunteering, error) {
	if m.GetVolunteeringByIDFunc != nil {
		return m.GetVolunteeringByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) CreateCustomSection(ctx context.Context, cs *rbModel.CustomSection) error {
	if m.CreateCustomSectionFunc != nil {
		return m.CreateCustomSectionFunc(ctx, cs)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) UpdateCustomSection(ctx context.Context, cs *rbModel.CustomSection) error {
	if m.UpdateCustomSectionFunc != nil {
		return m.UpdateCustomSectionFunc(ctx, cs)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCustomSectionFunc != nil {
		return m.DeleteCustomSectionFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*rbModel.CustomSection, error) {
	if m.ListCustomSectionsFunc != nil {
		return m.ListCustomSectionsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.CustomSection, error) {
	if m.GetCustomSectionByIDFunc != nil {
		return m.GetCustomSectionByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockAIResumeBuilderRepo) UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*rbModel.SectionOrder) error {
	if m.UpsertSectionOrderFunc != nil {
		return m.UpsertSectionOrderFunc(ctx, resumeBuilderID, orders)
	}
	return nil
}
func (m *mockAIResumeBuilderRepo) ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*rbModel.SectionOrder, error) {
	if m.ListSectionOrdersFunc != nil {
		return m.ListSectionOrdersFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

type mockCoverLetterAIClient struct {
	GenerateCoverLetterFunc func(ctx context.Context, companyName, recipientName, recipientTitle, jobDescription, resumeContext string) (*ai.CoverLetterContent, error)
}

func (m *mockCoverLetterAIClient) GenerateCoverLetter(ctx context.Context, companyName, recipientName, recipientTitle, jobDescription, resumeContext string) (*ai.CoverLetterContent, error) {
	if m.GenerateCoverLetterFunc != nil {
		return m.GenerateCoverLetterFunc(ctx, companyName, recipientName, recipientTitle, jobDescription, resumeContext)
	}
	return &ai.CoverLetterContent{}, nil
}

type mockAILimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *mockAILimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

// --- Helpers ---

type aiErrorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

func newTestCoverLetterAIHandler(
	clRepo *mockAICoverLetterRepo,
	rbRepo *mockAIResumeBuilderRepo,
	aiClient *mockCoverLetterAIClient,
	limiter *mockAILimitChecker,
) *AIHandler {
	aiSvc := service.NewAIService(clRepo, rbRepo, aiClient, limiter)
	return NewAIHandler(aiSvc, zap.NewNop())
}

// --- Generate Handler Tests ---

func TestAIHandler_Generate(t *testing.T) {
	userID := "user-123"
	validCoverLetterID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 with generated content", func(t *testing.T) {
		clRepo := &mockAICoverLetterRepo{
			GetByIDFunc: func(_ context.Context, id string) (*clModel.CoverLetter, error) {
				return &clModel.CoverLetter{
					ID:            id,
					UserID:        userID,
					CompanyName:   "Acme Corp",
					RecipientName: "Jane Doe",
				}, nil
			},
		}

		aiClient := &mockCoverLetterAIClient{
			GenerateCoverLetterFunc: func(_ context.Context, _, _, _, _, _ string) (*ai.CoverLetterContent, error) {
				return &ai.CoverLetterContent{
					Greeting:   "Dear Jane Doe,",
					Paragraphs: []string{"I am writing to express my interest.", "Thank you for considering."},
					Closing:    "Sincerely,",
				}, nil
			},
		}

		handler := newTestCoverLetterAIHandler(clRepo, &mockAIResumeBuilderRepo{}, aiClient, &mockAILimitChecker{})
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		body := `{"cover_letter_id":"` + validCoverLetterID + `","job_description":"Software Engineer"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result ai.CoverLetterContent
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "Dear Jane Doe,", result.Greeting)
		assert.Len(t, result.Paragraphs, 2)
		assert.Equal(t, "Sincerely,", result.Closing)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestCoverLetterAIHandler(
			&mockAICoverLetterRepo{},
			&mockAIResumeBuilderRepo{},
			&mockCoverLetterAIClient{},
			&mockAILimitChecker{},
		)
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", handler.Generate) // no auth middleware

		body := `{"cover_letter_id":"` + validCoverLetterID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UNAUTHORIZED", resp.ErrorCode)
	})

	t.Run("returns 400 for invalid cover_letter_id UUID format", func(t *testing.T) {
		handler := newTestCoverLetterAIHandler(
			&mockAICoverLetterRepo{},
			&mockAIResumeBuilderRepo{},
			&mockCoverLetterAIClient{},
			&mockAILimitChecker{},
		)
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		body := `{"cover_letter_id":"not-a-uuid","job_description":"Engineer"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", resp.ErrorCode)
		assert.Contains(t, resp.ErrorMessage, "Invalid cover_letter_id format")
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockAILimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestCoverLetterAIHandler(
			&mockAICoverLetterRepo{},
			&mockAIResumeBuilderRepo{},
			&mockCoverLetterAIClient{},
			limiter,
		)
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		body := `{"cover_letter_id":"` + validCoverLetterID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PLAN_LIMIT_REACHED", resp.ErrorCode)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler := newTestCoverLetterAIHandler(
			&mockAICoverLetterRepo{},
			&mockAIResumeBuilderRepo{},
			&mockCoverLetterAIClient{},
			&mockAILimitChecker{},
		)
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", resp.ErrorCode)
	})

	t.Run("returns 403 when not authorized", func(t *testing.T) {
		clRepo := &mockAICoverLetterRepo{
			GetByIDFunc: func(_ context.Context, id string) (*clModel.CoverLetter, error) {
				return &clModel.CoverLetter{
					ID:     id,
					UserID: "different-user", // different user
				}, nil
			},
		}

		handler := newTestCoverLetterAIHandler(
			clRepo,
			&mockAIResumeBuilderRepo{},
			&mockCoverLetterAIClient{},
			&mockAILimitChecker{},
		)
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		body := `{"cover_letter_id":"` + validCoverLetterID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NOT_AUTHORIZED", resp.ErrorCode)
	})

	t.Run("returns 500 on internal AI error", func(t *testing.T) {
		clRepo := &mockAICoverLetterRepo{
			GetByIDFunc: func(_ context.Context, id string) (*clModel.CoverLetter, error) {
				return &clModel.CoverLetter{
					ID:     id,
					UserID: userID,
				}, nil
			},
		}

		aiClient := &mockCoverLetterAIClient{
			GenerateCoverLetterFunc: func(_ context.Context, _, _, _, _, _ string) (*ai.CoverLetterContent, error) {
				return nil, errors.New("AI service unavailable")
			},
		}

		handler := newTestCoverLetterAIHandler(clRepo, &mockAIResumeBuilderRepo{}, aiClient, &mockAILimitChecker{})
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		body := `{"cover_letter_id":"` + validCoverLetterID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INTERNAL_ERROR", resp.ErrorCode)
	})

	t.Run("returns 404 when cover letter not found", func(t *testing.T) {
		clRepo := &mockAICoverLetterRepo{
			GetByIDFunc: func(_ context.Context, _ string) (*clModel.CoverLetter, error) {
				return nil, clModel.ErrCoverLetterNotFound
			},
		}

		handler := newTestCoverLetterAIHandler(
			clRepo,
			&mockAIResumeBuilderRepo{},
			&mockCoverLetterAIClient{},
			&mockAILimitChecker{},
		)
		router := setupTestRouter()
		router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

		body := `{"cover_letter_id":"` + validCoverLetterID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp aiErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "COVER_LETTER_NOT_FOUND", resp.ErrorCode)
	})
}
