package mcp

import (
	"context"
	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

func (f *fakeResumeStore) UpsertContact(_ context.Context, userID, resumeID string, request *rb.UpsertContactRequest) (*rb.ContactDTO, error) {
	return fakeResumeWrite[rb.ContactDTO](f, userID, resumeID, "", "contact", request)
}

func (f *fakeResumeStore) UpsertSummary(_ context.Context, userID, resumeID string, request *rb.UpsertSummaryRequest) (*rb.SummaryDTO, error) {
	return fakeResumeWrite[rb.SummaryDTO](f, userID, resumeID, "", "summary", request)
}

func (f *fakeResumeStore) CreateExperience(_ context.Context, userID, resumeID string, request *rb.CreateExperienceRequest) (*rb.ExperienceDTO, error) {
	return fakeResumeWrite[rb.ExperienceDTO](f, userID, resumeID, "", "experiences", request)
}

func (f *fakeResumeStore) UpdateExperience(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateExperienceRequest) (*rb.ExperienceDTO, error) {
	return fakeResumeWrite[rb.ExperienceDTO](f, userID, resumeID, entryID, "experiences", request)
}

func (f *fakeResumeStore) CreateEducation(_ context.Context, userID, resumeID string, request *rb.CreateEducationRequest) (*rb.EducationDTO, error) {
	return fakeResumeWrite[rb.EducationDTO](f, userID, resumeID, "", "educations", request)
}

func (f *fakeResumeStore) UpdateEducation(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateEducationRequest) (*rb.EducationDTO, error) {
	return fakeResumeWrite[rb.EducationDTO](f, userID, resumeID, entryID, "educations", request)
}

func (f *fakeResumeStore) CreateSkill(_ context.Context, userID, resumeID string, request *rb.CreateSkillRequest) (*rb.SkillDTO, error) {
	return fakeResumeWrite[rb.SkillDTO](f, userID, resumeID, "", "skills", request)
}

func (f *fakeResumeStore) UpdateSkill(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateSkillRequest) (*rb.SkillDTO, error) {
	return fakeResumeWrite[rb.SkillDTO](f, userID, resumeID, entryID, "skills", request)
}

func (f *fakeResumeStore) CreateLanguage(_ context.Context, userID, resumeID string, request *rb.CreateLanguageRequest) (*rb.LanguageDTO, error) {
	return fakeResumeWrite[rb.LanguageDTO](f, userID, resumeID, "", "languages", request)
}

func (f *fakeResumeStore) UpdateLanguage(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateLanguageRequest) (*rb.LanguageDTO, error) {
	return fakeResumeWrite[rb.LanguageDTO](f, userID, resumeID, entryID, "languages", request)
}

func (f *fakeResumeStore) CreateCertification(_ context.Context, userID, resumeID string, request *rb.CreateCertificationRequest) (*rb.CertificationDTO, error) {
	return fakeResumeWrite[rb.CertificationDTO](f, userID, resumeID, "", "certifications", request)
}

func (f *fakeResumeStore) UpdateCertification(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateCertificationRequest) (*rb.CertificationDTO, error) {
	return fakeResumeWrite[rb.CertificationDTO](f, userID, resumeID, entryID, "certifications", request)
}

func (f *fakeResumeStore) CreateProject(_ context.Context, userID, resumeID string, request *rb.CreateProjectRequest) (*rb.ProjectDTO, error) {
	return fakeResumeWrite[rb.ProjectDTO](f, userID, resumeID, "", "projects", request)
}

func (f *fakeResumeStore) UpdateProject(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateProjectRequest) (*rb.ProjectDTO, error) {
	return fakeResumeWrite[rb.ProjectDTO](f, userID, resumeID, entryID, "projects", request)
}

func (f *fakeResumeStore) CreateVolunteering(_ context.Context, userID, resumeID string, request *rb.CreateVolunteeringRequest) (*rb.VolunteeringDTO, error) {
	return fakeResumeWrite[rb.VolunteeringDTO](f, userID, resumeID, "", "volunteering", request)
}

func (f *fakeResumeStore) UpdateVolunteering(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateVolunteeringRequest) (*rb.VolunteeringDTO, error) {
	return fakeResumeWrite[rb.VolunteeringDTO](f, userID, resumeID, entryID, "volunteering", request)
}

func (f *fakeResumeStore) CreateCustomSection(_ context.Context, userID, resumeID string, request *rb.CreateCustomSectionRequest) (*rb.CustomSectionDTO, error) {
	return fakeResumeWrite[rb.CustomSectionDTO](f, userID, resumeID, "", "custom_sections", request)
}

func (f *fakeResumeStore) UpdateCustomSection(_ context.Context, userID, resumeID, entryID string, request *rb.UpdateCustomSectionRequest) (*rb.CustomSectionDTO, error) {
	return fakeResumeWrite[rb.CustomSectionDTO](f, userID, resumeID, entryID, "custom_sections", request)
}
