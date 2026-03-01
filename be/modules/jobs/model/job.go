package model

import "time"

// Job represents a job posting
type Job struct {
	ID          string
	UserID      string
	CompanyID   *string
	Title       string
	Source      *string
	URL         *string
	Notes       *string
	Description *string
	Status      string
	IsFavorite  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// JobDTO represents job data transfer object
type JobDTO struct {
	ID                string    `json:"id"`
	CompanyID         *string   `json:"company_id,omitempty"`
	CompanyName       *string   `json:"company_name,omitempty"`
	Title             string    `json:"title"`
	Source            *string   `json:"source,omitempty"`
	URL               *string   `json:"url,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	Description       *string   `json:"description,omitempty"`
	Status            string    `json:"status"`
	IsFavorite        bool      `json:"is_favorite"`
	ApplicationsCount int       `json:"applications_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ToDTO converts Job to JobDTO
// Note: CompanyName and ApplicationsCount must be set separately by the repository
func (j *Job) ToDTO() *JobDTO {
	return &JobDTO{
		ID:                j.ID,
		CompanyID:         j.CompanyID,
		CompanyName:       nil, // Set by repository
		Title:             j.Title,
		Source:            j.Source,
		URL:               j.URL,
		Notes:             j.Notes,
		Description:       j.Description,
		Status:            j.Status,
		IsFavorite:        j.IsFavorite,
		ApplicationsCount: 0, // Set by repository
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}
}
