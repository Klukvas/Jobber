package model

import (
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
)

// JobStatus represents valid job pipeline status values.
// A job starts as a saved (wishlist) card; leaving "saved" makes it an application.
type JobStatus string

const (
	StatusSaved    JobStatus = "saved"
	StatusApplied  JobStatus = "applied"
	StatusOnHold   JobStatus = "on_hold"
	StatusRejected JobStatus = "rejected"
	StatusOffer    JobStatus = "offer"
	StatusArchived JobStatus = "archived"
)

// ValidStatuses is the set of allowed job pipeline statuses.
var ValidStatuses = map[string]bool{
	string(StatusSaved):    true,
	string(StatusApplied):  true,
	string(StatusOnHold):   true,
	string(StatusRejected): true,
	string(StatusOffer):    true,
	string(StatusArchived): true,
}

// Job represents a tracked job posting with its application pipeline state (CORE AGGREGATE).
// AppliedAt is NULL for saved (wishlist) cards; non-NULL means the job is an application.
type Job struct {
	ID              string
	UserID          string
	CompanyID       *string
	Title           string
	Source          *string
	URL             *string
	Notes           *string
	Description     *string
	Status          string
	IsFavorite      bool
	AppliedAt       *time.Time
	ResumeID        *string
	ResumeBuilderID *string
	CurrentStageID  *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ResumeNestedDTO represents resume information attached to a job
type ResumeNestedDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "uploaded" or "builder"
}

// JobDTO represents job data transfer object
type JobDTO struct {
	ID               string                     `json:"id"`
	CompanyID        *string                    `json:"company_id,omitempty"`
	CompanyName      *string                    `json:"company_name,omitempty"`
	Title            string                     `json:"title"`
	Source           *string                    `json:"source,omitempty"`
	URL              *string                    `json:"url,omitempty"`
	Notes            *string                    `json:"notes,omitempty"`
	Description      *string                    `json:"description,omitempty"`
	Status           string                     `json:"status"`
	IsFavorite       bool                       `json:"is_favorite"`
	AppliedAt        *time.Time                 `json:"applied_at,omitempty"`
	CurrentStageID   *string                    `json:"current_stage_id,omitempty"`
	CurrentStageName *string                    `json:"current_stage_name,omitempty"`
	LastActivityAt   time.Time                  `json:"last_activity_at"`
	Resume           *ResumeNestedDTO           `json:"resume,omitempty"`
	JobComments      []*commentModel.CommentDTO `json:"job_comments,omitempty"`
	StageComments    []*commentModel.CommentDTO `json:"stage_comments,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

// ToDTO converts Job to JobDTO
// Note: CompanyName, Resume, CurrentStageName, LastActivityAt are set separately
// by the repository or service.
func (j *Job) ToDTO() *JobDTO {
	return &JobDTO{
		ID:             j.ID,
		CompanyID:      j.CompanyID,
		Title:          j.Title,
		Source:         j.Source,
		URL:            j.URL,
		Notes:          j.Notes,
		Description:    j.Description,
		Status:         j.Status,
		IsFavorite:     j.IsFavorite,
		AppliedAt:      j.AppliedAt,
		CurrentStageID: j.CurrentStageID,
		LastActivityAt: j.UpdatedAt,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
}
