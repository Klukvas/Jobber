package model

// CreateJobRequest represents a create job request
type CreateJobRequest struct {
	CompanyID *string `json:"company_id,omitempty"`
	Title     string  `json:"title" binding:"required,min=1,max=255"`
	Source    *string `json:"source,omitempty"`
	URL       *string `json:"url,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

// UpdateJobRequest represents an update job request
type UpdateJobRequest struct {
	CompanyID *string `json:"company_id,omitempty"`
	Title     *string `json:"title,omitempty"`
	Source    *string `json:"source,omitempty"`
	URL       *string `json:"url,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	Status    *string `json:"status,omitempty"`
}
