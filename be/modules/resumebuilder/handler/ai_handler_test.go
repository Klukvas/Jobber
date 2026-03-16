package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockResumeBuilderRepository struct {
	VerifyOwnershipFunc  func(ctx context.Context, userID, resumeBuilderID string) error
	GetFullResumeFunc    func(ctx context.Context, id string) (*model.FullResumeDTO, error)
	CreateFunc           func(ctx context.Context, rb *model.ResumeBuilder) error
	GetByIDFunc          func(ctx context.Context, id string) (*model.ResumeBuilder, error)
	ListFunc             func(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error)
	UpdateFunc           func(ctx context.Context, rb *model.ResumeBuilder) error
	DeleteFunc           func(ctx context.Context, id string) error
	RunInTransactionFunc func(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error

	UpsertContactFunc func(ctx context.Context, contact *model.Contact) error
	GetContactFunc    func(ctx context.Context, resumeBuilderID string) (*model.Contact, error)
	UpsertSummaryFunc func(ctx context.Context, summary *model.Summary) error
	GetSummaryFunc    func(ctx context.Context, resumeBuilderID string) (*model.Summary, error)

	CreateExperienceFunc     func(ctx context.Context, exp *model.Experience) error
	UpdateExperienceFunc     func(ctx context.Context, exp *model.Experience) error
	DeleteExperienceFunc     func(ctx context.Context, resumeBuilderID, id string) error
	ListExperiencesFunc      func(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error)
	GetExperienceByIDFunc    func(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error)
	CreateEducationFunc      func(ctx context.Context, edu *model.Education) error
	UpdateEducationFunc      func(ctx context.Context, edu *model.Education) error
	DeleteEducationFunc      func(ctx context.Context, resumeBuilderID, id string) error
	ListEducationsFunc       func(ctx context.Context, resumeBuilderID string) ([]*model.Education, error)
	GetEducationByIDFunc     func(ctx context.Context, resumeBuilderID, id string) (*model.Education, error)
	CreateSkillFunc          func(ctx context.Context, skill *model.Skill) error
	UpdateSkillFunc          func(ctx context.Context, skill *model.Skill) error
	DeleteSkillFunc          func(ctx context.Context, resumeBuilderID, id string) error
	ListSkillsFunc           func(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error)
	GetSkillByIDFunc         func(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error)
	CreateLanguageFunc       func(ctx context.Context, lang *model.Language) error
	UpdateLanguageFunc       func(ctx context.Context, lang *model.Language) error
	DeleteLanguageFunc       func(ctx context.Context, resumeBuilderID, id string) error
	ListLanguagesFunc        func(ctx context.Context, resumeBuilderID string) ([]*model.Language, error)
	GetLanguageByIDFunc      func(ctx context.Context, resumeBuilderID, id string) (*model.Language, error)
	CreateCertificationFunc  func(ctx context.Context, cert *model.Certification) error
	UpdateCertificationFunc  func(ctx context.Context, cert *model.Certification) error
	DeleteCertificationFunc  func(ctx context.Context, resumeBuilderID, id string) error
	ListCertificationsFunc   func(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error)
	GetCertificationByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error)
	CreateProjectFunc        func(ctx context.Context, proj *model.Project) error
	UpdateProjectFunc        func(ctx context.Context, proj *model.Project) error
	DeleteProjectFunc        func(ctx context.Context, resumeBuilderID, id string) error
	ListProjectsFunc         func(ctx context.Context, resumeBuilderID string) ([]*model.Project, error)
	GetProjectByIDFunc       func(ctx context.Context, resumeBuilderID, id string) (*model.Project, error)
	CreateVolunteeringFunc   func(ctx context.Context, vol *model.Volunteering) error
	UpdateVolunteeringFunc   func(ctx context.Context, vol *model.Volunteering) error
	DeleteVolunteeringFunc   func(ctx context.Context, resumeBuilderID, id string) error
	ListVolunteeringFunc     func(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error)
	GetVolunteeringByIDFunc  func(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error)
	CreateCustomSectionFunc  func(ctx context.Context, cs *model.CustomSection) error
	UpdateCustomSectionFunc  func(ctx context.Context, cs *model.CustomSection) error
	DeleteCustomSectionFunc  func(ctx context.Context, resumeBuilderID, id string) error
	ListCustomSectionsFunc   func(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error)
	GetCustomSectionByIDFunc func(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error)
	UpsertSectionOrderFunc   func(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error
	ListSectionOrdersFunc    func(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error)
}

func (m *mockResumeBuilderRepository) Create(ctx context.Context, rb *model.ResumeBuilder) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, rb)
	}
	return nil
}
func (m *mockResumeBuilderRepository) GetByID(ctx context.Context, id string) (*model.ResumeBuilder, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) List(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) Update(ctx context.Context, rb *model.ResumeBuilder) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, rb)
	}
	return nil
}
func (m *mockResumeBuilderRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) GetFullResume(ctx context.Context, id string) (*model.FullResumeDTO, error) {
	if m.GetFullResumeFunc != nil {
		return m.GetFullResumeFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	if m.VerifyOwnershipFunc != nil {
		return m.VerifyOwnershipFunc(ctx, userID, resumeBuilderID)
	}
	return nil
}
func (m *mockResumeBuilderRepository) RunInTransaction(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error {
	if m.RunInTransactionFunc != nil {
		return m.RunInTransactionFunc(ctx, fn)
	}
	return fn(m)
}
func (m *mockResumeBuilderRepository) UpsertContact(ctx context.Context, contact *model.Contact) error {
	if m.UpsertContactFunc != nil {
		return m.UpsertContactFunc(ctx, contact)
	}
	return nil
}
func (m *mockResumeBuilderRepository) GetContact(ctx context.Context, resumeBuilderID string) (*model.Contact, error) {
	if m.GetContactFunc != nil {
		return m.GetContactFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) UpsertSummary(ctx context.Context, summary *model.Summary) error {
	if m.UpsertSummaryFunc != nil {
		return m.UpsertSummaryFunc(ctx, summary)
	}
	return nil
}
func (m *mockResumeBuilderRepository) GetSummary(ctx context.Context, resumeBuilderID string) (*model.Summary, error) {
	if m.GetSummaryFunc != nil {
		return m.GetSummaryFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateExperience(ctx context.Context, exp *model.Experience) error {
	if m.CreateExperienceFunc != nil {
		return m.CreateExperienceFunc(ctx, exp)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateExperience(ctx context.Context, exp *model.Experience) error {
	if m.UpdateExperienceFunc != nil {
		return m.UpdateExperienceFunc(ctx, exp)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteExperience(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteExperienceFunc != nil {
		return m.DeleteExperienceFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListExperiences(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error) {
	if m.ListExperiencesFunc != nil {
		return m.ListExperiencesFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error) {
	if m.GetExperienceByIDFunc != nil {
		return m.GetExperienceByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateEducation(ctx context.Context, edu *model.Education) error {
	if m.CreateEducationFunc != nil {
		return m.CreateEducationFunc(ctx, edu)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateEducation(ctx context.Context, edu *model.Education) error {
	if m.UpdateEducationFunc != nil {
		return m.UpdateEducationFunc(ctx, edu)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteEducation(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteEducationFunc != nil {
		return m.DeleteEducationFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListEducations(ctx context.Context, resumeBuilderID string) ([]*model.Education, error) {
	if m.ListEducationsFunc != nil {
		return m.ListEducationsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*model.Education, error) {
	if m.GetEducationByIDFunc != nil {
		return m.GetEducationByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateSkill(ctx context.Context, skill *model.Skill) error {
	if m.CreateSkillFunc != nil {
		return m.CreateSkillFunc(ctx, skill)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateSkill(ctx context.Context, skill *model.Skill) error {
	if m.UpdateSkillFunc != nil {
		return m.UpdateSkillFunc(ctx, skill)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteSkill(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteSkillFunc != nil {
		return m.DeleteSkillFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListSkills(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error) {
	if m.ListSkillsFunc != nil {
		return m.ListSkillsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error) {
	if m.GetSkillByIDFunc != nil {
		return m.GetSkillByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateLanguage(ctx context.Context, lang *model.Language) error {
	if m.CreateLanguageFunc != nil {
		return m.CreateLanguageFunc(ctx, lang)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateLanguage(ctx context.Context, lang *model.Language) error {
	if m.UpdateLanguageFunc != nil {
		return m.UpdateLanguageFunc(ctx, lang)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteLanguageFunc != nil {
		return m.DeleteLanguageFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListLanguages(ctx context.Context, resumeBuilderID string) ([]*model.Language, error) {
	if m.ListLanguagesFunc != nil {
		return m.ListLanguagesFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*model.Language, error) {
	if m.GetLanguageByIDFunc != nil {
		return m.GetLanguageByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateCertification(ctx context.Context, cert *model.Certification) error {
	if m.CreateCertificationFunc != nil {
		return m.CreateCertificationFunc(ctx, cert)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateCertification(ctx context.Context, cert *model.Certification) error {
	if m.UpdateCertificationFunc != nil {
		return m.UpdateCertificationFunc(ctx, cert)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteCertification(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCertificationFunc != nil {
		return m.DeleteCertificationFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListCertifications(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error) {
	if m.ListCertificationsFunc != nil {
		return m.ListCertificationsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error) {
	if m.GetCertificationByIDFunc != nil {
		return m.GetCertificationByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateProject(ctx context.Context, proj *model.Project) error {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, proj)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateProject(ctx context.Context, proj *model.Project) error {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, proj)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteProject(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListProjects(ctx context.Context, resumeBuilderID string) ([]*model.Project, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*model.Project, error) {
	if m.GetProjectByIDFunc != nil {
		return m.GetProjectByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	if m.CreateVolunteeringFunc != nil {
		return m.CreateVolunteeringFunc(ctx, vol)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	if m.UpdateVolunteeringFunc != nil {
		return m.UpdateVolunteeringFunc(ctx, vol)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteVolunteeringFunc != nil {
		return m.DeleteVolunteeringFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error) {
	if m.ListVolunteeringFunc != nil {
		return m.ListVolunteeringFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error) {
	if m.GetVolunteeringByIDFunc != nil {
		return m.GetVolunteeringByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) CreateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	if m.CreateCustomSectionFunc != nil {
		return m.CreateCustomSectionFunc(ctx, cs)
	}
	return nil
}
func (m *mockResumeBuilderRepository) UpdateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	if m.UpdateCustomSectionFunc != nil {
		return m.UpdateCustomSectionFunc(ctx, cs)
	}
	return nil
}
func (m *mockResumeBuilderRepository) DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCustomSectionFunc != nil {
		return m.DeleteCustomSectionFunc(ctx, resumeBuilderID, id)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error) {
	if m.ListCustomSectionsFunc != nil {
		return m.ListCustomSectionsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error) {
	if m.GetCustomSectionByIDFunc != nil {
		return m.GetCustomSectionByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}
func (m *mockResumeBuilderRepository) UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error {
	if m.UpsertSectionOrderFunc != nil {
		return m.UpsertSectionOrderFunc(ctx, resumeBuilderID, orders)
	}
	return nil
}
func (m *mockResumeBuilderRepository) ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error) {
	if m.ListSectionOrdersFunc != nil {
		return m.ListSectionOrdersFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

type mockResumeAIClient struct {
	SuggestBulletPointsFunc func(ctx context.Context, jobTitle, company, currentDescription string) (*ai.BulletSuggestions, error)
	SuggestSummaryFunc      func(ctx context.Context, name, jobTitle, experienceContext string) (string, error)
	ImproveTextFunc         func(ctx context.Context, text, instruction string) (string, error)
	AnalyzeATSFunc          func(ctx context.Context, resumeContent, locale string) (*ai.ATSCheckResult, error)
}

func (m *mockResumeAIClient) SuggestBulletPoints(ctx context.Context, jobTitle, company, currentDescription string) (*ai.BulletSuggestions, error) {
	if m.SuggestBulletPointsFunc != nil {
		return m.SuggestBulletPointsFunc(ctx, jobTitle, company, currentDescription)
	}
	return &ai.BulletSuggestions{}, nil
}
func (m *mockResumeAIClient) SuggestSummary(ctx context.Context, name, jobTitle, experienceContext string) (string, error) {
	if m.SuggestSummaryFunc != nil {
		return m.SuggestSummaryFunc(ctx, name, jobTitle, experienceContext)
	}
	return "", nil
}
func (m *mockResumeAIClient) ImproveText(ctx context.Context, text, instruction string) (string, error) {
	if m.ImproveTextFunc != nil {
		return m.ImproveTextFunc(ctx, text, instruction)
	}
	return "", nil
}
func (m *mockResumeAIClient) AnalyzeATS(ctx context.Context, resumeContent, locale string) (*ai.ATSCheckResult, error) {
	if m.AnalyzeATSFunc != nil {
		return m.AnalyzeATSFunc(ctx, resumeContent, locale)
	}
	return &ai.ATSCheckResult{}, nil
}

// --- Helpers ---

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func authMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func noopMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func newTestAIHandler(repo *mockResumeBuilderRepository, aiClient *mockResumeAIClient, limiter *mockLimitChecker) *AIHandler {
	aiSvc := service.NewAIService(repo, aiClient, limiter)
	return NewAIHandler(aiSvc)
}

type errorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// --- SuggestBullets Handler Tests ---

func TestAIHandler_SuggestBullets(t *testing.T) {
	userID := "user-123"

	t.Run("returns 200 with bullet suggestions", func(t *testing.T) {
		aiClient := &mockResumeAIClient{
			SuggestBulletPointsFunc: func(_ context.Context, jobTitle, company, desc string) (*ai.BulletSuggestions, error) {
				return &ai.BulletSuggestions{
					Bullets: []string{"Led team of 5", "Improved latency by 40%"},
				}, nil
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-bullets", authMiddleware(userID), handler.SuggestBullets)

		body := `{"job_title":"Senior Engineer","company":"Acme Corp","current_description":"Led a team"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-bullets", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result ai.BulletSuggestions
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Len(t, result.Bullets, 2)
		assert.Equal(t, "Led team of 5", result.Bullets[0])
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-bullets", handler.SuggestBullets) // no auth middleware

		body := `{"job_title":"Engineer","company":"Corp"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-bullets", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UNAUTHORIZED", resp.ErrorCode)
	})

	t.Run("returns 400 for missing required fields", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-bullets", authMiddleware(userID), handler.SuggestBullets)

		// Missing company field
		body := `{"job_title":"Engineer"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-bullets", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", resp.ErrorCode)
	})

	t.Run("returns 400 for empty body", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-bullets", authMiddleware(userID), handler.SuggestBullets)

		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-bullets", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, limiter)
		router := setupRouter()
		router.POST("/ai/suggest-bullets", authMiddleware(userID), handler.SuggestBullets)

		body := `{"job_title":"Engineer","company":"Corp"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-bullets", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PLAN_LIMIT_REACHED", resp.ErrorCode)
	})

	t.Run("returns 500 when AI service fails", func(t *testing.T) {
		aiClient := &mockResumeAIClient{
			SuggestBulletPointsFunc: func(_ context.Context, _, _, _ string) (*ai.BulletSuggestions, error) {
				return nil, assert.AnError
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-bullets", authMiddleware(userID), handler.SuggestBullets)

		body := `{"job_title":"Engineer","company":"Corp"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-bullets", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- SuggestSummary Handler Tests ---

func TestAIHandler_SuggestSummary(t *testing.T) {
	userID := "user-123"
	validUUID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 with summary", func(t *testing.T) {
		aiClient := &mockResumeAIClient{
			SuggestSummaryFunc: func(_ context.Context, _, _, _ string) (string, error) {
				return "A seasoned engineer with deep expertise.", nil
			},
		}

		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: validUUID},
					Contact:          &model.ContactDTO{FullName: "Jane"},
					Experiences: []*model.ExperienceDTO{
						{Position: "Dev", Company: "Co"},
					},
				}, nil
			},
		}

		handler := newTestAIHandler(repo, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-summary", authMiddleware(userID), handler.SuggestSummary)

		body := `{"resume_id":"` + validUUID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "A seasoned engineer with deep expertise.", result["summary"])
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-summary", handler.SuggestSummary)

		body := `{"resume_id":"` + validUUID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for missing resume_id", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-summary", authMiddleware(userID), handler.SuggestSummary)

		body := `{}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", resp.ErrorCode)
	})

	t.Run("returns 400 for invalid resume_id format", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-summary", authMiddleware(userID), handler.SuggestSummary)

		body := `{"resume_id":"not-a-uuid"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", resp.ErrorCode)
		assert.Contains(t, resp.ErrorMessage, "Invalid resume_id format")
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, limiter)
		router := setupRouter()
		router.POST("/ai/suggest-summary", authMiddleware(userID), handler.SuggestSummary)

		body := `{"resume_id":"` + validUUID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PLAN_LIMIT_REACHED", resp.ErrorCode)
	})

	t.Run("returns 403 when not owner", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}

		handler := newTestAIHandler(repo, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/suggest-summary", authMiddleware(userID), handler.SuggestSummary)

		body := `{"resume_id":"` + validUUID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NOT_OWNER", resp.ErrorCode)
	})
}

// --- ImproveText Handler Tests ---

func TestAIHandler_ImproveText(t *testing.T) {
	userID := "user-123"

	t.Run("returns 200 with improved text", func(t *testing.T) {
		aiClient := &mockResumeAIClient{
			ImproveTextFunc: func(_ context.Context, text, instruction string) (string, error) {
				return "Delivered key outcomes through strategic leadership.", nil
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/improve-text", authMiddleware(userID), handler.ImproveText)

		body := `{"text":"I did stuff","instruction":"Make professional"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/improve-text", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "Delivered key outcomes through strategic leadership.", result["improved"])
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/improve-text", handler.ImproveText)

		body := `{"text":"text","instruction":"improve"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/improve-text", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for missing text field", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/improve-text", authMiddleware(userID), handler.ImproveText)

		body := `{"instruction":"improve"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/improve-text", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing instruction field", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/improve-text", authMiddleware(userID), handler.ImproveText)

		body := `{"text":"some text"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/improve-text", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, limiter)
		router := setupRouter()
		router.POST("/ai/improve-text", authMiddleware(userID), handler.ImproveText)

		body := `{"text":"text","instruction":"improve"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/improve-text", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("returns 500 when AI service fails", func(t *testing.T) {
		aiClient := &mockResumeAIClient{
			ImproveTextFunc: func(_ context.Context, _, _ string) (string, error) {
				return "", assert.AnError
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/ai/improve-text", authMiddleware(userID), handler.ImproveText)

		body := `{"text":"text","instruction":"improve"}`
		req, _ := http.NewRequest(http.MethodPost, "/ai/improve-text", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- ATSCheck Handler Tests ---

func TestAIHandler_ATSCheck(t *testing.T) {
	userID := "user-123"
	validUUID := "00000000-0000-0000-0000-000000000001"

	t.Run("returns 200 with ATS result", func(t *testing.T) {
		expected := &ai.ATSCheckResult{
			Score:       85,
			Issues:      []ai.ATSIssue{{Severity: "warning", Description: "Missing keywords"}},
			Suggestions: []string{"Add more metrics"},
			Keywords:    []string{"Go", "microservices"},
		}

		aiClient := &mockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, _, _ string) (*ai.ATSCheckResult, error) {
				return expected, nil
			},
		}

		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: validUUID},
				}, nil
			},
		}

		handler := newTestAIHandler(repo, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+validUUID+"/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var result ai.ATSCheckResult
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, 85, result.Score)
		assert.Len(t, result.Issues, 1)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+validUUID+"/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid resume ID format", func(t *testing.T) {
		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/not-a-uuid/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "VALIDATION_ERROR", resp.ErrorCode)
		assert.Contains(t, resp.ErrorMessage, "Invalid resume ID format")
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		limiter := &mockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		handler := newTestAIHandler(&mockResumeBuilderRepository{}, &mockResumeAIClient{}, limiter)
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+validUUID+"/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PLAN_LIMIT_REACHED", resp.ErrorCode)
	})

	t.Run("returns 403 when not owner", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}

		handler := newTestAIHandler(repo, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+validUUID+"/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "NOT_OWNER", resp.ErrorCode)
	})

	t.Run("returns 404 when resume not found", func(t *testing.T) {
		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrResumeBuilderNotFound
			},
		}

		handler := newTestAIHandler(repo, &mockResumeAIClient{}, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+validUUID+"/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp errorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "RESUME_BUILDER_NOT_FOUND", resp.ErrorCode)
	})

	t.Run("returns 500 when AI service fails", func(t *testing.T) {
		aiClient := &mockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, _, _ string) (*ai.ATSCheckResult, error) {
				return nil, assert.AnError
			},
		}

		repo := &mockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return &model.FullResumeDTO{
					ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: validUUID},
				}, nil
			},
		}

		handler := newTestAIHandler(repo, aiClient, &mockLimitChecker{})
		router := setupRouter()
		router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

		req, _ := http.NewRequest(http.MethodPost, "/resume-builder/"+validUUID+"/ats-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- handleError Tests ---

func TestAIHandler_handleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "limit reached returns 403 with PLAN_LIMIT_REACHED",
			err:            subModel.ErrLimitReached,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "PLAN_LIMIT_REACHED",
		},
		{
			name:           "not owner returns 403 with NOT_OWNER",
			err:            model.ErrNotOwner,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "NOT_OWNER",
		},
		{
			name:           "not found returns 404 with RESUME_BUILDER_NOT_FOUND",
			err:            model.ErrResumeBuilderNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "RESUME_BUILDER_NOT_FOUND",
		},
		{
			name:           "generic error returns 500 with INTERNAL_ERROR",
			err:            assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			handler := &AIHandler{}
			handler.handleError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var resp errorResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.ErrorCode)
		})
	}
}
