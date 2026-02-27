package model

// ValidBoardColumns defines the allowed board column values
var ValidBoardColumns = map[string]bool{
	"wishlist":  true,
	"applied":   true,
	"interview": true,
	"offer":     true,
	"rejected":  true,
}

// CreateJobRequest represents a create job request
type CreateJobRequest struct {
	CompanyID   *string `json:"company_id,omitempty"`
	Title       string  `json:"title" binding:"required,min=1,max=255"`
	Source      *string `json:"source,omitempty"`
	URL         *string `json:"url,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	BoardColumn *string `json:"board_column,omitempty"`
}

// ImportParseRequest represents a request to parse a job URL
type ImportParseRequest struct {
	URL string `json:"url" binding:"required"`
}

// ImportParseResponse represents the parsed job data
type ImportParseResponse struct {
	Title       string `json:"title"`
	CompanyName string `json:"company_name,omitempty"`
	Location    string `json:"location,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
	URL         string `json:"url"`
}

// UpdateJobRequest represents an update job request
type UpdateJobRequest struct {
	CompanyID   *string `json:"company_id,omitempty"`
	Title       *string `json:"title,omitempty"`
	Source      *string `json:"source,omitempty"`
	URL         *string `json:"url,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	Status      *string `json:"status,omitempty"`
	BoardColumn *string `json:"board_column,omitempty"`
}
