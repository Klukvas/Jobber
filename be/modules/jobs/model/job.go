package model

import "time"

// Job represents a job posting
type Job struct {
	ID        string
	UserID    string
	CompanyID *string
	Title     string
	Source    *string
	URL       *string
	Notes     *string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JobDTO represents job data transfer object
type JobDTO struct {
	ID                string     `json:"id"`
	CompanyID         *string    `json:"company_id,omitempty"`
	CompanyName       *string    `json:"company_name,omitempty"`
	Title             string     `json:"title"`
	Source            *string    `json:"source,omitempty"`
	URL               *string    `json:"url,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	Status            string     `json:"status"`
	ApplicationsCount int        `json:"applications_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
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
		Status:            j.Status,
		ApplicationsCount: 0, // Set by repository
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}
}
