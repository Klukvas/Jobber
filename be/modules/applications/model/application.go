package model

import (
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
)

// ApplicationStatus represents valid application status values
type ApplicationStatus string

const (
	StatusActive   ApplicationStatus = "active"
	StatusOnHold   ApplicationStatus = "on_hold"
	StatusRejected ApplicationStatus = "rejected"
	StatusOffer    ApplicationStatus = "offer"
	StatusArchived ApplicationStatus = "archived"
)

// Application represents a job application (CORE AGGREGATE)
type Application struct {
	ID             string
	UserID         string
	JobID          string
	ResumeID       string
	Name           string
	CurrentStageID *string
	Status         string // active, on_hold, rejected, offer, archived
	AppliedAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// JobNestedDTO represents a job with company information for application list
type JobNestedDTO struct {
	ID      string                    `json:"id"`
	Title   string                    `json:"title"`
	Company *companyModel.CompanyDTO  `json:"company,omitempty"`
}

// ResumeNestedDTO represents resume information for application list
type ResumeNestedDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ApplicationDTO represents application data transfer object
type ApplicationDTO struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Status             string                    `json:"status"`
	AppliedAt          time.Time                 `json:"applied_at"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
	LastActivityAt     time.Time                 `json:"last_activity_at"`
	CurrentStageID     *string                   `json:"current_stage_id,omitempty"`
	Job                *JobNestedDTO             `json:"job"`
	Resume             *ResumeNestedDTO          `json:"resume"`
	ApplicationComments []*commentModel.CommentDTO `json:"application_comments,omitempty"`
	StageComments      []*commentModel.CommentDTO `json:"stage_comments,omitempty"`
}

// NewApplicationDTO creates a new ApplicationDTO with nested entities
func NewApplicationDTO(
	app *Application,
	job *jobModel.Job,
	company *companyModel.Company,
	resume *resumeModel.Resume,
	lastActivityAt time.Time,
) *ApplicationDTO {
	dto := &ApplicationDTO{
		ID:             app.ID,
		Name:           app.Name,
		Status:         app.Status,
		AppliedAt:      app.AppliedAt,
		CreatedAt:      app.CreatedAt,
		UpdatedAt:      app.UpdatedAt,
		LastActivityAt: lastActivityAt,
		CurrentStageID: app.CurrentStageID,
	}

	// Add job with optional company
	if job != nil {
		dto.Job = &JobNestedDTO{
			ID:    job.ID,
			Title: job.Title,
		}
		if company != nil {
			dto.Job.Company = company.ToDTO()
		}
	}

	// Add resume
	if resume != nil {
		dto.Resume = &ResumeNestedDTO{
			ID:   resume.ID,
			Name: resume.Title,
		}
	}

	return dto
}
