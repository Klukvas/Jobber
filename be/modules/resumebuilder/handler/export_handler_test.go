package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/docx"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- Test helpers ---

func setupExportTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func exportMockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// --- Mock repository (reuses the same pattern as service tests) ---

type mockResumeBuilderRepo struct {
	VerifyOwnershipFunc func(ctx context.Context, userID, resumeBuilderID string) error
	GetFullResumeFunc   func(ctx context.Context, id string) (*model.FullResumeDTO, error)
	CreateFunc          func(ctx context.Context, rb *model.ResumeBuilder) error
	GetByIDFunc         func(ctx context.Context, id string) (*model.ResumeBuilder, error)
	ListFunc            func(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error)
	UpdateFunc          func(ctx context.Context, rb *model.ResumeBuilder) error
	DeleteFunc          func(ctx context.Context, id string) error
	RunInTransactionFunc func(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error

	UpsertContactFunc        func(ctx context.Context, contact *model.Contact) error
	GetContactFunc           func(ctx context.Context, resumeBuilderID string) (*model.Contact, error)
	UpsertSummaryFunc        func(ctx context.Context, summary *model.Summary) error
	GetSummaryFunc           func(ctx context.Context, resumeBuilderID string) (*model.Summary, error)
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

func (m *mockResumeBuilderRepo) Create(ctx context.Context, rb *model.ResumeBuilder) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, rb)
	}
	return nil
}

func (m *mockResumeBuilderRepo) GetByID(ctx context.Context, id string) (*model.ResumeBuilder, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) List(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) Update(ctx context.Context, rb *model.ResumeBuilder) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, rb)
	}
	return nil
}

func (m *mockResumeBuilderRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) GetFullResume(ctx context.Context, id string) (*model.FullResumeDTO, error) {
	if m.GetFullResumeFunc != nil {
		return m.GetFullResumeFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	if m.VerifyOwnershipFunc != nil {
		return m.VerifyOwnershipFunc(ctx, userID, resumeBuilderID)
	}
	return nil
}

func (m *mockResumeBuilderRepo) RunInTransaction(ctx context.Context, fn func(txRepo ports.ResumeBuilderRepository) error) error {
	if m.RunInTransactionFunc != nil {
		return m.RunInTransactionFunc(ctx, fn)
	}
	return fn(m)
}

func (m *mockResumeBuilderRepo) UpsertContact(ctx context.Context, contact *model.Contact) error {
	if m.UpsertContactFunc != nil {
		return m.UpsertContactFunc(ctx, contact)
	}
	return nil
}

func (m *mockResumeBuilderRepo) GetContact(ctx context.Context, resumeBuilderID string) (*model.Contact, error) {
	if m.GetContactFunc != nil {
		return m.GetContactFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) UpsertSummary(ctx context.Context, summary *model.Summary) error {
	if m.UpsertSummaryFunc != nil {
		return m.UpsertSummaryFunc(ctx, summary)
	}
	return nil
}

func (m *mockResumeBuilderRepo) GetSummary(ctx context.Context, resumeBuilderID string) (*model.Summary, error) {
	if m.GetSummaryFunc != nil {
		return m.GetSummaryFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateExperience(ctx context.Context, exp *model.Experience) error {
	if m.CreateExperienceFunc != nil {
		return m.CreateExperienceFunc(ctx, exp)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateExperience(ctx context.Context, exp *model.Experience) error {
	if m.UpdateExperienceFunc != nil {
		return m.UpdateExperienceFunc(ctx, exp)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteExperience(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteExperienceFunc != nil {
		return m.DeleteExperienceFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListExperiences(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error) {
	if m.ListExperiencesFunc != nil {
		return m.ListExperiencesFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error) {
	if m.GetExperienceByIDFunc != nil {
		return m.GetExperienceByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateEducation(ctx context.Context, edu *model.Education) error {
	if m.CreateEducationFunc != nil {
		return m.CreateEducationFunc(ctx, edu)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateEducation(ctx context.Context, edu *model.Education) error {
	if m.UpdateEducationFunc != nil {
		return m.UpdateEducationFunc(ctx, edu)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteEducation(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteEducationFunc != nil {
		return m.DeleteEducationFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListEducations(ctx context.Context, resumeBuilderID string) ([]*model.Education, error) {
	if m.ListEducationsFunc != nil {
		return m.ListEducationsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*model.Education, error) {
	if m.GetEducationByIDFunc != nil {
		return m.GetEducationByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateSkill(ctx context.Context, skill *model.Skill) error {
	if m.CreateSkillFunc != nil {
		return m.CreateSkillFunc(ctx, skill)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateSkill(ctx context.Context, skill *model.Skill) error {
	if m.UpdateSkillFunc != nil {
		return m.UpdateSkillFunc(ctx, skill)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteSkill(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteSkillFunc != nil {
		return m.DeleteSkillFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListSkills(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error) {
	if m.ListSkillsFunc != nil {
		return m.ListSkillsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error) {
	if m.GetSkillByIDFunc != nil {
		return m.GetSkillByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateLanguage(ctx context.Context, lang *model.Language) error {
	if m.CreateLanguageFunc != nil {
		return m.CreateLanguageFunc(ctx, lang)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateLanguage(ctx context.Context, lang *model.Language) error {
	if m.UpdateLanguageFunc != nil {
		return m.UpdateLanguageFunc(ctx, lang)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteLanguageFunc != nil {
		return m.DeleteLanguageFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListLanguages(ctx context.Context, resumeBuilderID string) ([]*model.Language, error) {
	if m.ListLanguagesFunc != nil {
		return m.ListLanguagesFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*model.Language, error) {
	if m.GetLanguageByIDFunc != nil {
		return m.GetLanguageByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateCertification(ctx context.Context, cert *model.Certification) error {
	if m.CreateCertificationFunc != nil {
		return m.CreateCertificationFunc(ctx, cert)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateCertification(ctx context.Context, cert *model.Certification) error {
	if m.UpdateCertificationFunc != nil {
		return m.UpdateCertificationFunc(ctx, cert)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteCertification(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCertificationFunc != nil {
		return m.DeleteCertificationFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListCertifications(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error) {
	if m.ListCertificationsFunc != nil {
		return m.ListCertificationsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error) {
	if m.GetCertificationByIDFunc != nil {
		return m.GetCertificationByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateProject(ctx context.Context, proj *model.Project) error {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, proj)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateProject(ctx context.Context, proj *model.Project) error {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, proj)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteProject(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListProjects(ctx context.Context, resumeBuilderID string) ([]*model.Project, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*model.Project, error) {
	if m.GetProjectByIDFunc != nil {
		return m.GetProjectByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	if m.CreateVolunteeringFunc != nil {
		return m.CreateVolunteeringFunc(ctx, vol)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateVolunteering(ctx context.Context, vol *model.Volunteering) error {
	if m.UpdateVolunteeringFunc != nil {
		return m.UpdateVolunteeringFunc(ctx, vol)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteVolunteeringFunc != nil {
		return m.DeleteVolunteeringFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error) {
	if m.ListVolunteeringFunc != nil {
		return m.ListVolunteeringFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error) {
	if m.GetVolunteeringByIDFunc != nil {
		return m.GetVolunteeringByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) CreateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	if m.CreateCustomSectionFunc != nil {
		return m.CreateCustomSectionFunc(ctx, cs)
	}
	return nil
}

func (m *mockResumeBuilderRepo) UpdateCustomSection(ctx context.Context, cs *model.CustomSection) error {
	if m.UpdateCustomSectionFunc != nil {
		return m.UpdateCustomSectionFunc(ctx, cs)
	}
	return nil
}

func (m *mockResumeBuilderRepo) DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error {
	if m.DeleteCustomSectionFunc != nil {
		return m.DeleteCustomSectionFunc(ctx, resumeBuilderID, id)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error) {
	if m.ListCustomSectionsFunc != nil {
		return m.ListCustomSectionsFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error) {
	if m.GetCustomSectionByIDFunc != nil {
		return m.GetCustomSectionByIDFunc(ctx, resumeBuilderID, id)
	}
	return nil, nil
}

func (m *mockResumeBuilderRepo) UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error {
	if m.UpsertSectionOrderFunc != nil {
		return m.UpsertSectionOrderFunc(ctx, resumeBuilderID, orders)
	}
	return nil
}

func (m *mockResumeBuilderRepo) ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error) {
	if m.ListSectionOrdersFunc != nil {
		return m.ListSectionOrdersFunc(ctx, resumeBuilderID)
	}
	return nil, nil
}

// --- Mock limit checker ---

type mockLimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *mockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

// --- Test data builders ---

func newTestService(repo *mockResumeBuilderRepo) *service.ResumeBuilderService {
	return service.NewResumeBuilderService(repo, &mockLimitChecker{})
}

func newTestFullResumeDTO() *model.FullResumeDTO {
	return &model.FullResumeDTO{
		ResumeBuilderDTO: &model.ResumeBuilderDTO{
			ID:           "rb-1",
			Title:        "Software Engineer Resume",
			TemplateID:   "00000000-0000-0000-0000-000000000001",
			FontFamily:   "Georgia",
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
		},
		Contact: &model.ContactDTO{
			FullName: "John Doe",
			Email:    "john@example.com",
			Phone:    "+1234567890",
		},
		Summary: &model.SummaryDTO{
			Content: "Experienced software engineer",
		},
		Experiences:    []*model.ExperienceDTO{},
		Educations:     []*model.EducationDTO{},
		Skills:         []*model.SkillDTO{},
		Languages:      []*model.LanguageDTO{},
		Certifications: []*model.CertificationDTO{},
		Projects:       []*model.ProjectDTO{},
		Volunteering:   []*model.VolunteeringDTO{},
		CustomSections: []*model.CustomSectionDTO{},
		SectionOrder:   []*model.SectionOrderDTO{},
	}
}

func parseErrorResponse(t *testing.T, w *httptest.ResponseRecorder) httpPlatform.ErrorResponse {
	t.Helper()
	var errResp httpPlatform.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err, "failed to parse error response body")
	return errResp
}

// --- ExportResumePDF tests ---

func TestExportResumePDF_Unauthorized(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-pdf", handler.ExportResumePDF) // no auth middleware

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "UNAUTHORIZED", errResp.ErrorCode)
}

func TestExportResumePDF_NotFound(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return nil, model.ErrResumeBuilderNotFound
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-pdf", exportMockAuthMiddleware("user-1"), handler.ExportResumePDF)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/nonexistent/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "RESUME_BUILDER_NOT_FOUND", errResp.ErrorCode)
}

func TestExportResumePDF_NotOwner(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return model.ErrNotOwner
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-pdf", exportMockAuthMiddleware("user-2"), handler.ExportResumePDF)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "NOT_OWNER", errResp.ErrorCode)
}

func TestExportResumePDF_PlanLimitReached(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return subModel.ErrLimitReached
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-pdf", exportMockAuthMiddleware("user-1"), handler.ExportResumePDF)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "PLAN_LIMIT_REACHED", errResp.ErrorCode)
}

func TestExportResumePDF_ServiceInternalError(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return errors.New("database connection failed")
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-pdf", exportMockAuthMiddleware("user-1"), handler.ExportResumePDF)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "INTERNAL_ERROR", errResp.ErrorCode)
}

func TestExportResumePDF_GetFullResumeError(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return nil, errors.New("unexpected database error")
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-pdf", exportMockAuthMiddleware("user-1"), handler.ExportResumePDF)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- ExportResumeDOCX tests ---

func TestExportResumeDOCX_Unauthorized(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", handler.ExportResumeDOCX) // no auth middleware

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "UNAUTHORIZED", errResp.ErrorCode)
}

func TestExportResumeDOCX_NotFound(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return nil, model.ErrResumeBuilderNotFound
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/nonexistent/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "RESUME_BUILDER_NOT_FOUND", errResp.ErrorCode)
}

func TestExportResumeDOCX_NotOwner(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return model.ErrNotOwner
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-2"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "NOT_OWNER", errResp.ErrorCode)
}

func TestExportResumeDOCX_PlanLimitReached(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return subModel.ErrLimitReached
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "PLAN_LIMIT_REACHED", errResp.ErrorCode)
}

func TestExportResumeDOCX_ServiceInternalError(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return errors.New("database connection failed")
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	errResp := parseErrorResponse(t, w)
	assert.Equal(t, "INTERNAL_ERROR", errResp.ErrorCode)
}

func TestExportResumeDOCX_Success(t *testing.T) {
	fullResume := newTestFullResumeDTO()

	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return fullResume, nil
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "Software Engineer Resume.docx")
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Greater(t, w.Body.Len(), 0, "DOCX body should not be empty")
}

func TestExportResumeDOCX_SuccessWithSpecialCharactersInTitle(t *testing.T) {
	fullResume := newTestFullResumeDTO()
	fullResume.Title = "My Resume / Special: \"Test\""

	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return fullResume, nil
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Special characters should be sanitized in the filename
	disposition := w.Header().Get("Content-Disposition")
	assert.NotContains(t, disposition, "/")
	assert.Contains(t, disposition, ".docx")
}

func TestExportResumeDOCX_GetFullResumeError(t *testing.T) {
	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return nil, errors.New("unexpected database error")
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- handleError tests ---

func TestHandleError_MapsErrorCodesToStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "not found error",
			err:            model.ErrResumeBuilderNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "RESUME_BUILDER_NOT_FOUND",
		},
		{
			name:           "section entry not found",
			err:            model.ErrSectionEntryNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "SECTION_ENTRY_NOT_FOUND",
		},
		{
			name:           "not owner",
			err:            model.ErrNotOwner,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "NOT_OWNER",
		},
		{
			name:           "invalid template",
			err:            model.ErrInvalidTemplate,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_TEMPLATE",
		},
		{
			name:           "invalid spacing",
			err:            model.ErrInvalidSpacing,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_SPACING",
		},
		{
			name:           "invalid color",
			err:            model.ErrInvalidColor,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_COLOR",
		},
		{
			name:           "invalid font",
			err:            model.ErrInvalidFont,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_FONT",
		},
		{
			name:           "invalid section key",
			err:            model.ErrInvalidSectionKey,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_SECTION_KEY",
		},
		{
			name:           "plan limit reached",
			err:            subModel.ErrLimitReached,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "PLAN_LIMIT_REACHED",
		},
		{
			name:           "unknown error maps to 500",
			err:            errors.New("some unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ExportHandler{}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

			handler.handleError(c, tt.err)

			assert.Equal(t, tt.expectedStatus, w.Code)

			errResp := parseErrorResponse(t, w)
			assert.Equal(t, tt.expectedCode, errResp.ErrorCode)
		})
	}
}

// --- sanitizeFilename tests ---

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "My Resume",
			expected: "My Resume",
		},
		{
			name:     "removes forward slash",
			input:    "path/to/file",
			expected: "path_to_file",
		},
		{
			name:     "removes backslash",
			input:    "path\\to\\file",
			expected: "path_to_file",
		},
		{
			name:     "removes colon",
			input:    "file: name",
			expected: "file_ name",
		},
		{
			name:     "removes quotes",
			input:    `"quoted"`,
			expected: "_quoted_",
		},
		{
			name:     "removes control characters",
			input:    "file\x00name\x01test",
			expected: "file_name_test",
		},
		{
			name:     "empty string returns resume",
			input:    "",
			expected: "resume",
		},
		{
			name:     "truncates long names to 200 chars",
			input:    strings.Repeat("a", 250),
			expected: strings.Repeat("a", 200),
		},
		{
			name:     "preserves unicode letters",
			input:    "Resume Lebenslauf",
			expected: "Resume Lebenslauf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename_AllControlCharsProduceDefaultName(t *testing.T) {
	// Input with only null bytes should result in underscores, not empty string
	result := sanitizeFilename("\x00\x00\x00")
	assert.Equal(t, "___", result)
}

func TestSanitizeFilename_LongInputTruncation(t *testing.T) {
	// Create a string longer than 200 characters with valid chars
	longName := ""
	for i := 0; i < 210; i++ {
		longName += "a"
	}
	result := sanitizeFilename(longName)
	assert.Len(t, result, 200)
}

// --- RegisterRoutes tests ---

func TestRegisterRoutes_RegistersPDFRoute(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())

	router := setupExportTestRouter()
	group := router.Group("/resumes")

	noopMiddleware := func(c *gin.Context) { c.Next() }
	handler.RegisterRoutes(group, noopMiddleware, noopMiddleware)

	// Verify PDF route is accessible (will return 401 since no user_id in context)
	req, _ := http.NewRequest(http.MethodPost, "/resumes/resume-builder/rb-1/export-pdf", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 401 (not 404), proving the route was registered
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRegisterRoutes_DOCXRouteNotRegisteredWhenServiceNil(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	// Do NOT set DOCX service

	router := setupExportTestRouter()
	group := router.Group("/resumes")

	noopMiddleware := func(c *gin.Context) { c.Next() }
	handler.RegisterRoutes(group, noopMiddleware, noopMiddleware)

	// DOCX route should not exist, returning 404
	req, _ := http.NewRequest(http.MethodPost, "/resumes/resume-builder/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterRoutes_DOCXRouteRegisteredWhenServiceSet(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	group := router.Group("/resumes")

	noopMiddleware := func(c *gin.Context) { c.Next() }
	handler.RegisterRoutes(group, noopMiddleware, noopMiddleware)

	// DOCX route should exist, returning 401 (not 404)
	req, _ := http.NewRequest(http.MethodPost, "/resumes/resume-builder/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- NewExportHandler tests ---

func TestNewExportHandler(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)

	handler := NewExportHandler(svc, nil, zap.NewNop())
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
	assert.Nil(t, handler.pdfService)
	assert.Nil(t, handler.docxService)
}

func TestSetDOCXService(t *testing.T) {
	repo := &mockResumeBuilderRepo{}
	svc := newTestService(repo)

	handler := NewExportHandler(svc, nil, zap.NewNop())
	assert.Nil(t, handler.docxService)

	docxSvc := docx.NewDOCXService()
	handler.SetDOCXService(docxSvc)
	assert.NotNil(t, handler.docxService)
}

// --- DOCX success with full resume data ---

func TestExportResumeDOCX_SuccessWithCompleteResume(t *testing.T) {
	fullResume := &model.FullResumeDTO{
		ResumeBuilderDTO: &model.ResumeBuilderDTO{
			ID:           "rb-1",
			Title:        "Complete Resume",
			TemplateID:   "00000000-0000-0000-0000-000000000001",
			FontFamily:   "Georgia",
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
		},
		Contact: &model.ContactDTO{
			FullName: "Jane Smith",
			Email:    "jane@example.com",
			Phone:    "+1987654321",
			Location: "New York, NY",
			Website:  "https://jane.dev",
			LinkedIn: "linkedin.com/in/jane",
			GitHub:   "github.com/jane",
		},
		Summary: &model.SummaryDTO{
			Content: "Senior software engineer with 10 years of experience.",
		},
		Experiences: []*model.ExperienceDTO{
			{ID: "exp-1", Company: "Acme Corp", Position: "Senior Dev", StartDate: "2020-01", EndDate: "2023-12"},
		},
		Educations: []*model.EducationDTO{
			{ID: "edu-1", Institution: "MIT", Degree: "BS", FieldOfStudy: "Computer Science"},
		},
		Skills: []*model.SkillDTO{
			{ID: "skill-1", Name: "Go", Level: "Expert"},
			{ID: "skill-2", Name: "TypeScript", Level: "Advanced"},
		},
		Languages: []*model.LanguageDTO{
			{ID: "lang-1", Name: "English", Proficiency: "Native"},
		},
		Certifications: []*model.CertificationDTO{
			{ID: "cert-1", Name: "AWS Solutions Architect", Issuer: "Amazon"},
		},
		Projects: []*model.ProjectDTO{
			{ID: "proj-1", Name: "Open Source Tool", URL: "https://github.com/tool"},
		},
		Volunteering: []*model.VolunteeringDTO{
			{ID: "vol-1", Organization: "Code.org", Role: "Mentor"},
		},
		CustomSections: []*model.CustomSectionDTO{
			{ID: "cs-1", Title: "Publications", Content: "Published paper on distributed systems"},
		},
		SectionOrder: []*model.SectionOrderDTO{
			{SectionKey: "contact", SortOrder: 0, IsVisible: true},
			{SectionKey: "summary", SortOrder: 1, IsVisible: true},
			{SectionKey: "experience", SortOrder: 2, IsVisible: true},
			{SectionKey: "education", SortOrder: 3, IsVisible: true},
			{SectionKey: "skills", SortOrder: 4, IsVisible: true},
		},
	}

	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return fullResume, nil
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), `attachment; filename="Complete Resume.docx"`)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Greater(t, w.Body.Len(), 0)
}

func TestExportResumeDOCX_SuccessWithEmptyTitle(t *testing.T) {
	fullResume := newTestFullResumeDTO()
	fullResume.Title = ""

	repo := &mockResumeBuilderRepo{
		VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return fullResume, nil
		},
	}
	svc := newTestService(repo)
	handler := NewExportHandler(svc, nil, zap.NewNop())
	handler.SetDOCXService(docx.NewDOCXService())

	router := setupExportTestRouter()
	router.POST("/resumes/:id/export-docx", exportMockAuthMiddleware("user-1"), handler.ExportResumeDOCX)

	req, _ := http.NewRequest(http.MethodPost, "/resumes/rb-1/export-docx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Empty title should be sanitized to "resume"
	assert.Contains(t, w.Header().Get("Content-Disposition"), "resume.docx")
}
