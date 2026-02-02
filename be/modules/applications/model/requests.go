package model

import "time"

// CreateApplicationRequest represents a create application request
type CreateApplicationRequest struct {
	JobID     string    `json:"job_id" binding:"required"`
	ResumeID  string    `json:"resume_id" binding:"required"`
	Name      string    `json:"name" binding:"max=255"` // Optional: auto-generated from job title if empty
	AppliedAt time.Time `json:"applied_at"`
}

// UpdateApplicationRequest represents an update application request
type UpdateApplicationRequest struct {
	Status *string `json:"status,omitempty"`
}

// CreateStageTemplateRequest represents a create stage template request
type CreateStageTemplateRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=255"`
	Order int    `json:"order" binding:"required,min=0"`
}

// UpdateStageTemplateRequest represents an update stage template request
type UpdateStageTemplateRequest struct {
	Name  *string `json:"name,omitempty"`
	Order *int    `json:"order,omitempty"`
}

// AddStageRequest represents adding a stage to an application
type AddStageRequest struct {
	StageTemplateID string  `json:"stage_template_id" binding:"required"`
	Comment         *string `json:"comment,omitempty"` // Optional comment when adding a stage
}

// CompleteStageRequest represents completing a stage
type CompleteStageRequest struct {
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type UpdateStageRequest struct {
	Status      *string    `json:"status,omitempty" binding:"omitempty,oneof=pending active completed skipped cancelled"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
