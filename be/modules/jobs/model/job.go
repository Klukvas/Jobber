package model

import (
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
)

// Job represents a tracked job posting on the single-axis pipeline (CORE AGGREGATE).
// A card lives in exactly one pipeline column: CurrentStageTemplateID. There is no
// separate status — the column IS the state. IsArchived orthogonally hides a card.
// AppliedAt is a display timestamp stamped when a card first leaves the initial column.
type Job struct {
	ID                     string
	UserID                 string
	CompanyID              *string
	Title                  string
	Source                 *string
	URL                    *string
	Description            *string
	IsFavorite             bool
	IsArchived             bool
	AppliedAt              *time.Time
	ResumeID               *string
	ResumeBuilderID        *string
	CurrentStageTemplateID *string // the pipeline column the card is in (source of truth)
	CurrentStageID         *string // the current job_stages history instance
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// ResumeNestedDTO represents resume information attached to a job
type ResumeNestedDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "uploaded" or "builder"
}

// JobDTO represents job data transfer object
type JobDTO struct {
	ID                     string                     `json:"id"`
	CompanyID              *string                    `json:"company_id,omitempty"`
	CompanyName            *string                    `json:"company_name,omitempty"`
	Title                  string                     `json:"title"`
	Source                 *string                    `json:"source,omitempty"`
	URL                    *string                    `json:"url,omitempty"`
	Description            *string                    `json:"description,omitempty"`
	IsFavorite             bool                       `json:"is_favorite"`
	IsArchived             bool                       `json:"is_archived"`
	AppliedAt              *time.Time                 `json:"applied_at,omitempty"`
	CurrentStageTemplateID *string                    `json:"current_stage_template_id,omitempty"`
	CurrentStageID         *string                    `json:"current_stage_id,omitempty"`
	CurrentStageName       *string                    `json:"current_stage_name,omitempty"`
	LastActivityAt         time.Time                  `json:"last_activity_at"`
	Resume                 *ResumeNestedDTO           `json:"resume,omitempty"`
	JobComments            []*commentModel.CommentDTO `json:"job_comments,omitempty"`
	StageComments          []*commentModel.CommentDTO `json:"stage_comments,omitempty"`
	CreatedAt              time.Time                  `json:"created_at"`
	UpdatedAt              time.Time                  `json:"updated_at"`
}

// ToDTO converts Job to JobDTO
// Note: CompanyName, Resume, CurrentStageName, LastActivityAt are set separately
// by the repository or service.
func (j *Job) ToDTO() *JobDTO {
	return &JobDTO{
		ID:                     j.ID,
		CompanyID:              j.CompanyID,
		Title:                  j.Title,
		Source:                 j.Source,
		URL:                    j.URL,
		Description:            j.Description,
		IsFavorite:             j.IsFavorite,
		IsArchived:             j.IsArchived,
		AppliedAt:              j.AppliedAt,
		CurrentStageTemplateID: j.CurrentStageTemplateID,
		CurrentStageID:         j.CurrentStageID,
		LastActivityAt:         j.UpdatedAt,
		CreatedAt:              j.CreatedAt,
		UpdatedAt:              j.UpdatedAt,
	}
}
