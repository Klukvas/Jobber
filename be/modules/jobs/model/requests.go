package model

import "time"

// CreateJobRequest represents a create job request. Omitting stage_template_id
// creates the card in the user's first pipeline column (wishlist) — this is what
// the Chrome extension sends.
type CreateJobRequest struct {
	CompanyID       *string `json:"company_id,omitempty"`
	Title           string  `json:"title" binding:"required,min=1,max=255"`
	Source          *string `json:"source,omitempty"`
	URL             *string `json:"url,omitempty"`
	Description     *string `json:"description,omitempty"`
	StageTemplateID *string `json:"stage_template_id,omitempty"` // column to place into; default = first
	ResumeID        *string `json:"resume_id,omitempty"`
	ResumeBuilderID *string `json:"resume_builder_id,omitempty"`
}

// UpdateJobRequest represents an update job request.
type UpdateJobRequest struct {
	CompanyID       *string `json:"company_id,omitempty"`
	Title           *string `json:"title,omitempty"`
	Source          *string `json:"source,omitempty"`
	URL             *string `json:"url,omitempty"`
	Description     *string `json:"description,omitempty"`
	IsArchived      *bool   `json:"is_archived,omitempty"` // archive / unarchive
	ResumeID        *string `json:"resume_id,omitempty"`
	ResumeBuilderID *string `json:"resume_builder_id,omitempty"`
}

// CreateStageTemplateRequest represents a create pipeline-column request
type CreateStageTemplateRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=255"`
	Order int    `json:"order" binding:"min=0"`
}

// UpdateStageTemplateRequest represents an update pipeline-column request
type UpdateStageTemplateRequest struct {
	Name  *string `json:"name,omitempty"`
	Order *int    `json:"order,omitempty"`
}

// ReorderStageTemplatesRequest sets a new order for the user's pipeline columns.
// StageIDs is the full ordered list of the user's stage template IDs.
type ReorderStageTemplatesRequest struct {
	StageIDs []string `json:"stage_ids" binding:"required,min=1"`
}

// MoveJobRequest moves a card to a pipeline column (the single write path for
// pipeline position — board drag or the detail-page stage selector).
type MoveJobRequest struct {
	StageTemplateID string `json:"stage_template_id" binding:"required"`
}

// AddStageRequest represents adding a stage-history entry to a job
type AddStageRequest struct {
	StageTemplateID string  `json:"stage_template_id" binding:"required"`
	Comment         *string `json:"comment,omitempty"` // Optional comment when adding a stage
}

// UpdateStageRequest represents updating a stage-history entry's status
type UpdateStageRequest struct {
	Status      *string    `json:"status,omitempty" binding:"omitempty,oneof=pending active completed skipped cancelled"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
