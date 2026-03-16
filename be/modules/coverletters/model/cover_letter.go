package model

import (
	"errors"
	"time"
)

var (
	ErrCoverLetterNotFound = errors.New("cover letter not found")
	ErrNotAuthorized       = errors.New("not authorized")
	ErrInvalidFont         = errors.New("invalid font family")
	ErrInvalidColor        = errors.New("invalid color format")
	ErrInvalidFontSize     = errors.New("font size must be between 8 and 18")
)

// CoverLetter represents a cover letter record.
type CoverLetter struct {
	ID              string
	UserID          string
	ResumeBuilderID *string
	JobID           *string
	Title           string
	Template        string
	RecipientName   string
	RecipientTitle  string
	CompanyName     string
	CompanyAddress  string
	Greeting        string
	Paragraphs      []string
	Closing         string
	FontFamily      string
	FontSize        int
	PrimaryColor    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CoverLetterDTO is the JSON response for a cover letter.
type CoverLetterDTO struct {
	ID              string    `json:"id"`
	ResumeBuilderID *string   `json:"resume_builder_id,omitempty"`
	JobID           *string   `json:"job_id,omitempty"`
	Title           string    `json:"title"`
	Template        string    `json:"template"`
	RecipientName   string    `json:"recipient_name"`
	RecipientTitle  string    `json:"recipient_title"`
	CompanyName     string    `json:"company_name"`
	CompanyAddress  string    `json:"company_address"`
	Greeting        string    `json:"greeting"`
	Paragraphs      []string  `json:"paragraphs"`
	Closing         string    `json:"closing"`
	FontFamily      string    `json:"font_family"`
	FontSize        int       `json:"font_size"`
	PrimaryColor    string    `json:"primary_color"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ToDTO converts a CoverLetter to CoverLetterDTO.
func (cl *CoverLetter) ToDTO() *CoverLetterDTO {
	return &CoverLetterDTO{
		ID:              cl.ID,
		ResumeBuilderID: cl.ResumeBuilderID,
		JobID:           cl.JobID,
		Title:           cl.Title,
		Template:        cl.Template,
		RecipientName:   cl.RecipientName,
		RecipientTitle:  cl.RecipientTitle,
		CompanyName:     cl.CompanyName,
		CompanyAddress:  cl.CompanyAddress,
		Greeting:        cl.Greeting,
		Paragraphs:      cl.Paragraphs,
		Closing:         cl.Closing,
		FontFamily:      cl.FontFamily,
		FontSize:        cl.FontSize,
		PrimaryColor:    cl.PrimaryColor,
		CreatedAt:       cl.CreatedAt,
		UpdatedAt:       cl.UpdatedAt,
	}
}

// CreateCoverLetterRequest is the request for creating a cover letter.
type CreateCoverLetterRequest struct {
	Title           string  `json:"title"`
	ResumeBuilderID *string `json:"resume_builder_id"`
	JobID           *string `json:"job_id"`
	Template        string  `json:"template"`
}

// UpdateCoverLetterRequest is the request for updating a cover letter.
type UpdateCoverLetterRequest struct {
	Title           *string   `json:"title"`
	ResumeBuilderID *string   `json:"resume_builder_id"`
	JobID           *string   `json:"job_id"`
	Template        *string   `json:"template"`
	RecipientName   *string   `json:"recipient_name"`
	RecipientTitle  *string   `json:"recipient_title"`
	CompanyName     *string   `json:"company_name"`
	CompanyAddress  *string   `json:"company_address"`
	Greeting        *string   `json:"greeting"`
	Paragraphs      *[]string `json:"paragraphs"`
	Closing         *string   `json:"closing"`
	FontFamily      *string   `json:"font_family"`
	FontSize        *int      `json:"font_size"`
	PrimaryColor    *string   `json:"primary_color"`
}
