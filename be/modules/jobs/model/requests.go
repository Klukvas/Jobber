package model

import "time"

// CreateJobRequest represents a create job request.
// Status/applied_at/resume fields are optional: omitting them creates a "saved"
// (wishlist) card — this is what the Chrome extension sends.
type CreateJobRequest struct {
	CompanyID       *string    `json:"company_id,omitempty"`
	Title           string     `json:"title" binding:"required,min=1,max=255"`
	Source          *string    `json:"source,omitempty"`
	URL             *string    `json:"url,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Status          *string    `json:"status,omitempty"`
	AppliedAt       *time.Time `json:"applied_at,omitempty"`
	ResumeID        *string    `json:"resume_id,omitempty"`
	ResumeBuilderID *string    `json:"resume_builder_id,omitempty"`
}

// UpdateJobRequest represents an update job request.
// Moving status out of "saved" sets applied_at server-side (unless provided);
// moving back to "saved" clears applied_at.
type UpdateJobRequest struct {
	CompanyID       *string    `json:"company_id,omitempty"`
	Title           *string    `json:"title,omitempty"`
	Source          *string    `json:"source,omitempty"`
	URL             *string    `json:"url,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Status          *string    `json:"status,omitempty"`
	AppliedAt       *time.Time `json:"applied_at,omitempty"`
	ResumeID        *string    `json:"resume_id,omitempty"`
	ResumeBuilderID *string    `json:"resume_builder_id,omitempty"`
}

// CreateStageTemplateRequest represents a create stage template request
type CreateStageTemplateRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=255"`
	Order int    `json:"order" binding:"min=0"`
}

// UpdateStageTemplateRequest represents an update stage template request
type UpdateStageTemplateRequest struct {
	Name  *string `json:"name,omitempty"`
	Order *int    `json:"order,omitempty"`
}

// AddStageRequest represents adding a stage to a job
type AddStageRequest struct {
	StageTemplateID string  `json:"stage_template_id" binding:"required"`
	Comment         *string `json:"comment,omitempty"` // Optional comment when adding a stage
}

// UpdateStageRequest represents updating a stage's status
type UpdateStageRequest struct {
	Status      *string    `json:"status,omitempty" binding:"omitempty,oneof=pending active completed skipped cancelled"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
