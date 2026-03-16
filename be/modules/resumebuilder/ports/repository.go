package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// ResumeBuilderRepository defines the data access interface for resume builders.
type ResumeBuilderRepository interface {
	Create(ctx context.Context, rb *model.ResumeBuilder) error
	GetByID(ctx context.Context, id string) (*model.ResumeBuilder, error)
	List(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error)
	Update(ctx context.Context, rb *model.ResumeBuilder) error
	Delete(ctx context.Context, id string) error
	GetFullResume(ctx context.Context, id string) (*model.FullResumeDTO, error)
	VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error

	// Transaction support
	RunInTransaction(ctx context.Context, fn func(txRepo ResumeBuilderRepository) error) error

	// Contact (1:1)
	UpsertContact(ctx context.Context, contact *model.Contact) error
	GetContact(ctx context.Context, resumeBuilderID string) (*model.Contact, error)

	// Summary (1:1)
	UpsertSummary(ctx context.Context, summary *model.Summary) error
	GetSummary(ctx context.Context, resumeBuilderID string) (*model.Summary, error)

	// Experiences
	CreateExperience(ctx context.Context, exp *model.Experience) error
	UpdateExperience(ctx context.Context, exp *model.Experience) error
	DeleteExperience(ctx context.Context, resumeBuilderID, id string) error
	ListExperiences(ctx context.Context, resumeBuilderID string) ([]*model.Experience, error)
	GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*model.Experience, error)

	// Educations
	CreateEducation(ctx context.Context, edu *model.Education) error
	UpdateEducation(ctx context.Context, edu *model.Education) error
	DeleteEducation(ctx context.Context, resumeBuilderID, id string) error
	ListEducations(ctx context.Context, resumeBuilderID string) ([]*model.Education, error)
	GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*model.Education, error)

	// Skills
	CreateSkill(ctx context.Context, skill *model.Skill) error
	UpdateSkill(ctx context.Context, skill *model.Skill) error
	DeleteSkill(ctx context.Context, resumeBuilderID, id string) error
	ListSkills(ctx context.Context, resumeBuilderID string) ([]*model.Skill, error)
	GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*model.Skill, error)

	// Languages
	CreateLanguage(ctx context.Context, lang *model.Language) error
	UpdateLanguage(ctx context.Context, lang *model.Language) error
	DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error
	ListLanguages(ctx context.Context, resumeBuilderID string) ([]*model.Language, error)
	GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*model.Language, error)

	// Certifications
	CreateCertification(ctx context.Context, cert *model.Certification) error
	UpdateCertification(ctx context.Context, cert *model.Certification) error
	DeleteCertification(ctx context.Context, resumeBuilderID, id string) error
	ListCertifications(ctx context.Context, resumeBuilderID string) ([]*model.Certification, error)
	GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*model.Certification, error)

	// Projects
	CreateProject(ctx context.Context, proj *model.Project) error
	UpdateProject(ctx context.Context, proj *model.Project) error
	DeleteProject(ctx context.Context, resumeBuilderID, id string) error
	ListProjects(ctx context.Context, resumeBuilderID string) ([]*model.Project, error)
	GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*model.Project, error)

	// Volunteering
	CreateVolunteering(ctx context.Context, vol *model.Volunteering) error
	UpdateVolunteering(ctx context.Context, vol *model.Volunteering) error
	DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error
	ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*model.Volunteering, error)
	GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*model.Volunteering, error)

	// Custom Sections
	CreateCustomSection(ctx context.Context, cs *model.CustomSection) error
	UpdateCustomSection(ctx context.Context, cs *model.CustomSection) error
	DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error
	ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*model.CustomSection, error)
	GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*model.CustomSection, error)

	// Section Order
	UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*model.SectionOrder) error
	ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*model.SectionOrder, error)
}
