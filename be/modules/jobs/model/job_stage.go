package model

import "time"

// JobStage represents a stage in a job's application lifecycle (append-only)
// Status values: pending, active, completed, skipped, cancelled
type JobStage struct {
	ID              string
	JobID           string
	StageTemplateID string
	Status          string // pending, active, completed, skipped, cancelled
	Order           int
	StartedAt       time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
}

// JobStageDTO represents job stage data transfer object
type JobStageDTO struct {
	ID              string     `json:"id"`
	JobID           string     `json:"job_id"`
	StageTemplateID string     `json:"stage_template_id"`
	StageName       string     `json:"stage_name"`
	Status          string     `json:"status"`
	Order           int        `json:"order"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ToDTO converts JobStage to JobStageDTO
func (s *JobStage) ToDTO(stageName string) *JobStageDTO {
	return &JobStageDTO{
		ID:              s.ID,
		JobID:           s.JobID,
		StageTemplateID: s.StageTemplateID,
		StageName:       stageName,
		Status:          s.Status,
		Order:           s.Order,
		StartedAt:       s.StartedAt,
		CompletedAt:     s.CompletedAt,
		CreatedAt:       s.CreatedAt,
	}
}
