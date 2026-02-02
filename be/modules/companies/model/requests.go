package model

// CreateCompanyRequest represents a create company request
type CreateCompanyRequest struct {
	Name     string  `json:"name" binding:"required,min=1,max=255"`
	Location *string `json:"location,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

// UpdateCompanyRequest represents an update company request
type UpdateCompanyRequest struct {
	Name     *string `json:"name,omitempty"`
	Location *string `json:"location,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}
