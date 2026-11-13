package mcp

import (
	"context"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	uploaded "github.com/andreypavlenko/jobber/modules/resumes/model"
)

// ResumeTools keeps the resume slice independent of other optional tool groups.
type ResumeTools struct {
	Builder   BuilderResumes
	Uploaded  UploadedResumes
	Snapshots ResumeSnapshots
}

type BuilderResumeReader interface {
	List(context.Context, string) ([]*rb.ResumeBuilderDTO, error)
	Get(context.Context, string, string) (*rb.FullResumeDTO, error)
}

type ResumeContactWriter interface {
	UpsertContact(context.Context, string, string, *rb.UpsertContactRequest) (*rb.ContactDTO, error)
	UpsertSummary(context.Context, string, string, *rb.UpsertSummaryRequest) (*rb.SummaryDTO, error)
}

type ResumeExperienceWriter interface {
	CreateExperience(context.Context, string, string, *rb.CreateExperienceRequest) (*rb.ExperienceDTO, error)
	UpdateExperience(context.Context, string, string, string, *rb.UpdateExperienceRequest) (*rb.ExperienceDTO, error)
}

type ResumeEducationWriter interface {
	CreateEducation(context.Context, string, string, *rb.CreateEducationRequest) (*rb.EducationDTO, error)
	UpdateEducation(context.Context, string, string, string, *rb.UpdateEducationRequest) (*rb.EducationDTO, error)
}

type ResumeSkillWriter interface {
	CreateSkill(context.Context, string, string, *rb.CreateSkillRequest) (*rb.SkillDTO, error)
	UpdateSkill(context.Context, string, string, string, *rb.UpdateSkillRequest) (*rb.SkillDTO, error)
}

type ResumeLanguageWriter interface {
	CreateLanguage(context.Context, string, string, *rb.CreateLanguageRequest) (*rb.LanguageDTO, error)
	UpdateLanguage(context.Context, string, string, string, *rb.UpdateLanguageRequest) (*rb.LanguageDTO, error)
}

type ResumeCertificationWriter interface {
	CreateCertification(context.Context, string, string, *rb.CreateCertificationRequest) (*rb.CertificationDTO, error)
	UpdateCertification(context.Context, string, string, string, *rb.UpdateCertificationRequest) (*rb.CertificationDTO, error)
}

type ResumeProjectWriter interface {
	CreateProject(context.Context, string, string, *rb.CreateProjectRequest) (*rb.ProjectDTO, error)
	UpdateProject(context.Context, string, string, string, *rb.UpdateProjectRequest) (*rb.ProjectDTO, error)
}

type ResumeVolunteeringWriter interface {
	CreateVolunteering(context.Context, string, string, *rb.CreateVolunteeringRequest) (*rb.VolunteeringDTO, error)
	UpdateVolunteering(context.Context, string, string, string, *rb.UpdateVolunteeringRequest) (*rb.VolunteeringDTO, error)
}

type ResumeCustomWriter interface {
	CreateCustomSection(context.Context, string, string, *rb.CreateCustomSectionRequest) (*rb.CustomSectionDTO, error)
	UpdateCustomSection(context.Context, string, string, string, *rb.UpdateCustomSectionRequest) (*rb.CustomSectionDTO, error)
}

type BuilderResumes interface {
	BuilderResumeReader
	ResumeContactWriter
	ResumeExperienceWriter
	ResumeEducationWriter
	ResumeSkillWriter
	ResumeLanguageWriter
	ResumeCertificationWriter
	ResumeProjectWriter
	ResumeVolunteeringWriter
	ResumeCustomWriter
}

type UploadedResumes interface {
	List(context.Context, string, int, int, string, string) ([]*uploaded.ResumeDTO, int, error)
}

type ResumeSnapshots interface {
	SnapshotBeforeWrite(context.Context, string, string, string) error
	ListSnapshots(context.Context, string, string) ([]*rb.ResumeSnapshotDTO, error)
	RestoreSnapshot(context.Context, string, string) (*rb.FullResumeDTO, error)
}
