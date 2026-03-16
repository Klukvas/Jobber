package service

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// --- Contact ---

func (s *ResumeBuilderService) UpsertContact(ctx context.Context, userID, resumeBuilderID string, req *model.UpsertContactRequest) (*model.ContactDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	contact := &model.Contact{
		ResumeBuilderID: resumeBuilderID,
		FullName:        req.FullName,
		Email:           req.Email,
		Phone:           req.Phone,
		Location:        req.Location,
		Website:         req.Website,
		LinkedIn:        req.LinkedIn,
		GitHub:          req.GitHub,
	}

	if err := s.repo.UpsertContact(ctx, contact); err != nil {
		return nil, err
	}

	return contact.ToDTO(), nil
}

// --- Summary ---

func (s *ResumeBuilderService) UpsertSummary(ctx context.Context, userID, resumeBuilderID string, req *model.UpsertSummaryRequest) (*model.SummaryDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	summary := &model.Summary{
		ResumeBuilderID: resumeBuilderID,
		Content:         req.Content,
	}

	if err := s.repo.UpsertSummary(ctx, summary); err != nil {
		return nil, err
	}

	return &model.SummaryDTO{Content: summary.Content}, nil
}

// --- Experiences ---

func (s *ResumeBuilderService) CreateExperience(ctx context.Context, userID, resumeBuilderID string, req *model.CreateExperienceRequest) (*model.ExperienceDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	exp := &model.Experience{
		ResumeBuilderID: resumeBuilderID,
		Company:         req.Company,
		Position:        req.Position,
		Location:        req.Location,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		IsCurrent:       req.IsCurrent,
		Description:     req.Description,
		SortOrder:       req.SortOrder,
	}

	if err := s.repo.CreateExperience(ctx, exp); err != nil {
		return nil, err
	}

	return exp.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateExperience(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateExperienceRequest) (*model.ExperienceDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	exp, err := s.repo.GetExperienceByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Company != nil {
		exp.Company = *req.Company
	}
	if req.Position != nil {
		exp.Position = *req.Position
	}
	if req.Location != nil {
		exp.Location = *req.Location
	}
	if req.StartDate != nil {
		exp.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		exp.EndDate = *req.EndDate
	}
	if req.IsCurrent != nil {
		exp.IsCurrent = *req.IsCurrent
	}
	if req.Description != nil {
		exp.Description = *req.Description
	}
	if req.SortOrder != nil {
		exp.SortOrder = *req.SortOrder
	}

	if err := s.repo.UpdateExperience(ctx, exp); err != nil {
		return nil, err
	}

	return exp.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteExperience(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteExperience(ctx, resumeBuilderID, entryID)
}

// --- Educations ---

func (s *ResumeBuilderService) CreateEducation(ctx context.Context, userID, resumeBuilderID string, req *model.CreateEducationRequest) (*model.EducationDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	edu := &model.Education{
		ResumeBuilderID: resumeBuilderID,
		Institution:     req.Institution,
		Degree:          req.Degree,
		FieldOfStudy:    req.FieldOfStudy,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		IsCurrent:       req.IsCurrent,
		GPA:             req.GPA,
		Description:     req.Description,
		SortOrder:       req.SortOrder,
	}

	if err := s.repo.CreateEducation(ctx, edu); err != nil {
		return nil, err
	}

	return edu.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateEducation(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateEducationRequest) (*model.EducationDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	edu, err := s.repo.GetEducationByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Institution != nil {
		edu.Institution = *req.Institution
	}
	if req.Degree != nil {
		edu.Degree = *req.Degree
	}
	if req.FieldOfStudy != nil {
		edu.FieldOfStudy = *req.FieldOfStudy
	}
	if req.StartDate != nil {
		edu.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		edu.EndDate = *req.EndDate
	}
	if req.IsCurrent != nil {
		edu.IsCurrent = *req.IsCurrent
	}
	if req.GPA != nil {
		edu.GPA = *req.GPA
	}
	if req.Description != nil {
		edu.Description = *req.Description
	}
	if req.SortOrder != nil {
		edu.SortOrder = *req.SortOrder
	}

	if err := s.repo.UpdateEducation(ctx, edu); err != nil {
		return nil, err
	}

	return edu.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteEducation(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteEducation(ctx, resumeBuilderID, entryID)
}

// --- Skills ---

func (s *ResumeBuilderService) CreateSkill(ctx context.Context, userID, resumeBuilderID string, req *model.CreateSkillRequest) (*model.SkillDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}
	skill := &model.Skill{ResumeBuilderID: resumeBuilderID, Name: req.Name, Level: req.Level, SortOrder: req.SortOrder}
	if err := s.repo.CreateSkill(ctx, skill); err != nil {
		return nil, err
	}
	return skill.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateSkill(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateSkillRequest) (*model.SkillDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	skill, err := s.repo.GetSkillByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		skill.Name = *req.Name
	}
	if req.Level != nil {
		skill.Level = *req.Level
	}
	if req.SortOrder != nil {
		skill.SortOrder = *req.SortOrder
	}
	if err := s.repo.UpdateSkill(ctx, skill); err != nil {
		return nil, err
	}
	return skill.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteSkill(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteSkill(ctx, resumeBuilderID, entryID)
}

// --- Languages ---

func (s *ResumeBuilderService) CreateLanguage(ctx context.Context, userID, resumeBuilderID string, req *model.CreateLanguageRequest) (*model.LanguageDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}
	lang := &model.Language{ResumeBuilderID: resumeBuilderID, Name: req.Name, Proficiency: req.Proficiency, SortOrder: req.SortOrder}
	if err := s.repo.CreateLanguage(ctx, lang); err != nil {
		return nil, err
	}
	return lang.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateLanguage(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateLanguageRequest) (*model.LanguageDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	lang, err := s.repo.GetLanguageByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		lang.Name = *req.Name
	}
	if req.Proficiency != nil {
		lang.Proficiency = *req.Proficiency
	}
	if req.SortOrder != nil {
		lang.SortOrder = *req.SortOrder
	}
	if err := s.repo.UpdateLanguage(ctx, lang); err != nil {
		return nil, err
	}
	return lang.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteLanguage(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteLanguage(ctx, resumeBuilderID, entryID)
}

// --- Certifications ---

func (s *ResumeBuilderService) CreateCertification(ctx context.Context, userID, resumeBuilderID string, req *model.CreateCertificationRequest) (*model.CertificationDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}
	cert := &model.Certification{
		ResumeBuilderID: resumeBuilderID, Name: req.Name, Issuer: req.Issuer,
		IssueDate: req.IssueDate, ExpiryDate: req.ExpiryDate, URL: req.URL, SortOrder: req.SortOrder,
	}
	if err := s.repo.CreateCertification(ctx, cert); err != nil {
		return nil, err
	}
	return cert.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateCertification(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateCertificationRequest) (*model.CertificationDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	cert, err := s.repo.GetCertificationByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		cert.Name = *req.Name
	}
	if req.Issuer != nil {
		cert.Issuer = *req.Issuer
	}
	if req.IssueDate != nil {
		cert.IssueDate = *req.IssueDate
	}
	if req.ExpiryDate != nil {
		cert.ExpiryDate = *req.ExpiryDate
	}
	if req.URL != nil {
		cert.URL = *req.URL
	}
	if req.SortOrder != nil {
		cert.SortOrder = *req.SortOrder
	}
	if err := s.repo.UpdateCertification(ctx, cert); err != nil {
		return nil, err
	}
	return cert.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteCertification(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteCertification(ctx, resumeBuilderID, entryID)
}

// --- Projects ---

func (s *ResumeBuilderService) CreateProject(ctx context.Context, userID, resumeBuilderID string, req *model.CreateProjectRequest) (*model.ProjectDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}
	proj := &model.Project{
		ResumeBuilderID: resumeBuilderID, Name: req.Name, URL: req.URL,
		StartDate: req.StartDate, EndDate: req.EndDate, Description: req.Description, SortOrder: req.SortOrder,
	}
	if err := s.repo.CreateProject(ctx, proj); err != nil {
		return nil, err
	}
	return proj.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateProject(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateProjectRequest) (*model.ProjectDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	proj, err := s.repo.GetProjectByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		proj.Name = *req.Name
	}
	if req.URL != nil {
		proj.URL = *req.URL
	}
	if req.StartDate != nil {
		proj.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		proj.EndDate = *req.EndDate
	}
	if req.Description != nil {
		proj.Description = *req.Description
	}
	if req.SortOrder != nil {
		proj.SortOrder = *req.SortOrder
	}
	if err := s.repo.UpdateProject(ctx, proj); err != nil {
		return nil, err
	}
	return proj.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteProject(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteProject(ctx, resumeBuilderID, entryID)
}

// --- Volunteering ---

func (s *ResumeBuilderService) CreateVolunteering(ctx context.Context, userID, resumeBuilderID string, req *model.CreateVolunteeringRequest) (*model.VolunteeringDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}
	vol := &model.Volunteering{
		ResumeBuilderID: resumeBuilderID, Organization: req.Organization, Role: req.Role,
		StartDate: req.StartDate, EndDate: req.EndDate, Description: req.Description, SortOrder: req.SortOrder,
	}
	if err := s.repo.CreateVolunteering(ctx, vol); err != nil {
		return nil, err
	}
	return vol.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateVolunteering(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateVolunteeringRequest) (*model.VolunteeringDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	vol, err := s.repo.GetVolunteeringByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Organization != nil {
		vol.Organization = *req.Organization
	}
	if req.Role != nil {
		vol.Role = *req.Role
	}
	if req.StartDate != nil {
		vol.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		vol.EndDate = *req.EndDate
	}
	if req.Description != nil {
		vol.Description = *req.Description
	}
	if req.SortOrder != nil {
		vol.SortOrder = *req.SortOrder
	}
	if err := s.repo.UpdateVolunteering(ctx, vol); err != nil {
		return nil, err
	}
	return vol.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteVolunteering(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteVolunteering(ctx, resumeBuilderID, entryID)
}

// --- Custom Sections ---

func (s *ResumeBuilderService) CreateCustomSection(ctx context.Context, userID, resumeBuilderID string, req *model.CreateCustomSectionRequest) (*model.CustomSectionDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}
	cs := &model.CustomSection{ResumeBuilderID: resumeBuilderID, Title: req.Title, Content: req.Content, SortOrder: req.SortOrder}
	if err := s.repo.CreateCustomSection(ctx, cs); err != nil {
		return nil, err
	}
	return cs.ToDTO(), nil
}

func (s *ResumeBuilderService) UpdateCustomSection(ctx context.Context, userID, resumeBuilderID, entryID string, req *model.UpdateCustomSectionRequest) (*model.CustomSectionDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	cs, err := s.repo.GetCustomSectionByID(ctx, resumeBuilderID, entryID)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		cs.Title = *req.Title
	}
	if req.Content != nil {
		cs.Content = *req.Content
	}
	if req.SortOrder != nil {
		cs.SortOrder = *req.SortOrder
	}
	if err := s.repo.UpdateCustomSection(ctx, cs); err != nil {
		return nil, err
	}
	return cs.ToDTO(), nil
}

func (s *ResumeBuilderService) DeleteCustomSection(ctx context.Context, userID, resumeBuilderID, entryID string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return err
	}
	return s.repo.DeleteCustomSection(ctx, resumeBuilderID, entryID)
}

// --- Section Order ---

func (s *ResumeBuilderService) UpdateSectionOrder(ctx context.Context, userID, resumeBuilderID string, req *model.BatchUpdateSectionOrderRequest) ([]*model.SectionOrderDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, resumeBuilderID); err != nil {
		return nil, err
	}

	// Validate all section keys and column values against allowlists
	for _, sec := range req.Sections {
		if !ValidSectionKeys[sec.SectionKey] {
			return nil, model.ErrInvalidSectionKey
		}
		if sec.Column != "" && !ValidColumnValues[sec.Column] {
			return nil, model.ErrInvalidColumnValue
		}
	}

	orders := make([]*model.SectionOrder, len(req.Sections))
	for i, sec := range req.Sections {
		col := sec.Column
		if col == "" {
			col = "main"
		}
		orders[i] = &model.SectionOrder{
			ResumeBuilderID: resumeBuilderID,
			SectionKey:      sec.SectionKey,
			SortOrder:       sec.SortOrder,
			IsVisible:       sec.IsVisible,
			Column:          col,
		}
	}

	if err := s.repo.UpsertSectionOrder(ctx, resumeBuilderID, orders); err != nil {
		return nil, err
	}

	result := make([]*model.SectionOrderDTO, len(orders))
	for i, o := range orders {
		result[i] = o.ToDTO()
	}
	return result, nil
}
