package model

import "time"

// Company represents a company entity
type Company struct {
	ID        string
	UserID    string
	Name      string
	Location  *string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CompanyDTO represents company data transfer object with enriched fields
type CompanyDTO struct {
	ID                      string     `json:"id"`
	Name                    string     `json:"name"`
	Location                *string    `json:"location,omitempty"`
	Notes                   *string    `json:"notes,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	ApplicationsCount       int        `json:"applications_count"`
	ActiveApplicationsCount int        `json:"active_applications_count"`
	DerivedStatus           string     `json:"derived_status"`
	LastActivityAt          *time.Time `json:"last_activity_at,omitempty"`
}

// CompanyStatus represents the derived status of a company
type CompanyStatus string

const (
	CompanyStatusIdle         CompanyStatus = "idle"         // No applications
	CompanyStatusActive       CompanyStatus = "active"       // Has active applications
	CompanyStatusInterviewing CompanyStatus = "interviewing" // Has applications past "Applied" stage
)

// ToDTO converts Company to CompanyDTO
func (c *Company) ToDTO() *CompanyDTO {
	return &CompanyDTO{
		ID:        c.ID,
		Name:      c.Name,
		Location:  c.Location,
		Notes:     c.Notes,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
