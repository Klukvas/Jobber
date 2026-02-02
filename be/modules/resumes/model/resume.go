package model

import "time"

// StorageType represents the type of storage for a resume
type StorageType string

const (
	StorageTypeExternal StorageType = "external"
	StorageTypeS3       StorageType = "s3"
)

// Resume represents a user's resume
type Resume struct {
	ID          string
	UserID      string
	Title       string
	FileURL     *string
	StorageType StorageType
	StorageKey  *string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ResumeDTO represents resume data transfer object
type ResumeDTO struct {
	ID                string      `json:"id"`
	Title             string      `json:"title"`
	FileURL           *string     `json:"file_url"`
	StorageType       StorageType `json:"storage_type"`
	StorageKey        *string     `json:"storage_key,omitempty"`
	IsActive          bool        `json:"is_active"`
	ApplicationsCount int         `json:"applications_count"`
	CanDelete         bool        `json:"can_delete"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// ToDTOWithCounts converts Resume to ResumeDTO with application counts
func (r *Resume) ToDTOWithCounts(applicationsCount int) *ResumeDTO {
	return &ResumeDTO{
		ID:                r.ID,
		Title:             r.Title,
		FileURL:           r.FileURL,
		StorageType:       r.StorageType,
		StorageKey:        r.StorageKey,
		IsActive:          r.IsActive,
		ApplicationsCount: applicationsCount,
		CanDelete:         applicationsCount == 0,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

// ToDTO converts Resume to ResumeDTO (without counts)
func (r *Resume) ToDTO() *ResumeDTO {
	return &ResumeDTO{
		ID:                r.ID,
		Title:             r.Title,
		FileURL:           r.FileURL,
		StorageType:       r.StorageType,
		StorageKey:        r.StorageKey,
		IsActive:          r.IsActive,
		ApplicationsCount: 0,
		CanDelete:         true,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}
