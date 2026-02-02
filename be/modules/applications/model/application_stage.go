package model

import "time"

// ApplicationStage represents a stage in an application lifecycle (append-only)
// Status values: pending, active, completed, skipped, cancelled
type ApplicationStage struct {
	ID              string
	ApplicationID   string
	StageTemplateID string
	Status          string // pending, active, completed, skipped, cancelled
	Order           int
	StartedAt       time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
}

// ApplicationStageDTO represents application stage data transfer object
type ApplicationStageDTO struct {
	ID              string     `json:"id"`
	ApplicationID   string     `json:"application_id"`
	StageTemplateID string     `json:"stage_template_id"`
	StageName       string     `json:"stage_name"`
	Status          string     `json:"status"`
	Order           int        `json:"order"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ToDTO converts ApplicationStage to ApplicationStageDTO
func (a *ApplicationStage) ToDTO(stageName string) *ApplicationStageDTO {
	return &ApplicationStageDTO{
		ID:              a.ID,
		ApplicationID:   a.ApplicationID,
		StageTemplateID: a.StageTemplateID,
		StageName:       stageName,
		Status:          a.Status,
		Order:           a.Order,
		StartedAt:       a.StartedAt,
		CompletedAt:     a.CompletedAt,
		CreatedAt:       a.CreatedAt,
	}
}
