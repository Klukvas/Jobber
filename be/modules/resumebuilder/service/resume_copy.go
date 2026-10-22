package service

import (
	"context"
	"fmt"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

// copyResumeSections copies all content and layout ordering without sharing entry IDs.
func copyResumeSections(ctx context.Context, txRepo ports.ResumeBuilderRepository, resumeID string, full *model.FullResumeDTO) error {
	// Copy contact
	if full.Contact != nil {
		if err := txRepo.UpsertContact(ctx, &model.Contact{
			ResumeBuilderID: resumeID,
			FullName:        full.Contact.FullName,
			Email:           full.Contact.Email,
			Phone:           full.Contact.Phone,
			Location:        full.Contact.Location,
			Website:         full.Contact.Website,
			LinkedIn:        full.Contact.LinkedIn,
			GitHub:          full.Contact.GitHub,
		}); err != nil {
			return fmt.Errorf("failed to copy contact: %w", err)
		}
	}

	// Copy summary
	if full.Summary != nil {
		if err := txRepo.UpsertSummary(ctx, &model.Summary{
			ResumeBuilderID: resumeID,
			Content:         full.Summary.Content,
		}); err != nil {
			return fmt.Errorf("failed to copy summary: %w", err)
		}
	}

	// Copy experiences
	for _, e := range full.Experiences {
		if err := txRepo.CreateExperience(ctx, &model.Experience{
			ResumeBuilderID: resumeID,
			Company:         e.Company,
			Position:        e.Position,
			Location:        e.Location,
			StartDate:       e.StartDate,
			EndDate:         e.EndDate,
			IsCurrent:       e.IsCurrent,
			Description:     e.Description,
			SortOrder:       e.SortOrder,
		}); err != nil {
			return fmt.Errorf("failed to copy experience: %w", err)
		}
	}

	// Copy educations
	for _, e := range full.Educations {
		if err := txRepo.CreateEducation(ctx, &model.Education{
			ResumeBuilderID: resumeID,
			Institution:     e.Institution,
			Degree:          e.Degree,
			FieldOfStudy:    e.FieldOfStudy,
			StartDate:       e.StartDate,
			EndDate:         e.EndDate,
			IsCurrent:       e.IsCurrent,
			GPA:             e.GPA,
			Description:     e.Description,
			SortOrder:       e.SortOrder,
		}); err != nil {
			return fmt.Errorf("failed to copy education: %w", err)
		}
	}

	// Copy skills
	for _, sk := range full.Skills {
		if err := txRepo.CreateSkill(ctx, &model.Skill{ResumeBuilderID: resumeID, Name: sk.Name, Level: sk.Level, SortOrder: sk.SortOrder}); err != nil {
			return fmt.Errorf("failed to copy skill: %w", err)
		}
	}

	// Copy languages
	for _, l := range full.Languages {
		if err := txRepo.CreateLanguage(ctx, &model.Language{ResumeBuilderID: resumeID, Name: l.Name, Proficiency: l.Proficiency, SortOrder: l.SortOrder}); err != nil {
			return fmt.Errorf("failed to copy language: %w", err)
		}
	}

	// Copy certifications
	for _, cert := range full.Certifications {
		if err := txRepo.CreateCertification(ctx, &model.Certification{
			ResumeBuilderID: resumeID, Name: cert.Name, Issuer: cert.Issuer,
			IssueDate: cert.IssueDate, ExpiryDate: cert.ExpiryDate, URL: cert.URL, SortOrder: cert.SortOrder,
		}); err != nil {
			return fmt.Errorf("failed to copy certification: %w", err)
		}
	}

	// Copy projects
	for _, p := range full.Projects {
		if err := txRepo.CreateProject(ctx, &model.Project{
			ResumeBuilderID: resumeID, Name: p.Name, URL: p.URL,
			StartDate: p.StartDate, EndDate: p.EndDate, Description: p.Description, SortOrder: p.SortOrder,
		}); err != nil {
			return fmt.Errorf("failed to copy project: %w", err)
		}
	}

	// Copy volunteering
	for _, v := range full.Volunteering {
		if err := txRepo.CreateVolunteering(ctx, &model.Volunteering{
			ResumeBuilderID: resumeID, Organization: v.Organization, Role: v.Role,
			StartDate: v.StartDate, EndDate: v.EndDate, Description: v.Description, SortOrder: v.SortOrder,
		}); err != nil {
			return fmt.Errorf("failed to copy volunteering: %w", err)
		}
	}

	// Copy custom sections
	for _, cs := range full.CustomSections {
		if err := txRepo.CreateCustomSection(ctx, &model.CustomSection{
			ResumeBuilderID: resumeID, Title: cs.Title, Content: cs.Content, SortOrder: cs.SortOrder,
		}); err != nil {
			return fmt.Errorf("failed to copy custom section: %w", err)
		}
	}

	// Copy section order
	if len(full.SectionOrder) > 0 {
		orders := make([]*model.SectionOrder, len(full.SectionOrder))
		for i, o := range full.SectionOrder {
			orders[i] = &model.SectionOrder{
				ResumeBuilderID: resumeID,
				SectionKey:      o.SectionKey,
				SortOrder:       o.SortOrder,
				IsVisible:       o.IsVisible,
				Column:          o.Column,
			}
		}
		if err := txRepo.UpsertSectionOrder(ctx, resumeID, orders); err != nil {
			return fmt.Errorf("failed to copy section order: %w", err)
		}
	}
	return nil
}
