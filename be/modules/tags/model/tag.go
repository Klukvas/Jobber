package model

import (
	"errors"
	"time"
)

// EntityType values a tag relation can point at (narrowed to job|company in
// migration 000034).
const (
	EntityTypeJob     = "job"
	EntityTypeCompany = "company"
)

type Tag struct {
	ID        string
	UserID    string
	Name      string
	Color     *string
	CreatedAt time.Time
}

type TagDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (t *Tag) ToDTO() *TagDTO {
	return &TagDTO{
		ID:        t.ID,
		Name:      t.Name,
		Color:     t.Color,
		CreatedAt: t.CreatedAt,
	}
}

type CreateTagRequest struct {
	Name  string  `json:"name" binding:"required,min=1,max=100"`
	Color *string `json:"color,omitempty"`
}

// AttachTagRequest attaches a tag to a job or company.
type AttachTagRequest struct {
	EntityType string `json:"entity_type" binding:"required,oneof=job company"`
	EntityID   string `json:"entity_id" binding:"required"`
}

type TagRelation struct {
	ID         string
	TagID      string
	EntityType string
	EntityID   string
	CreatedAt  time.Time
}

var (
	ErrTagNotFound       = errors.New("tag not found")
	ErrTagNameRequired   = errors.New("tag name is required")
	ErrTagNameExists     = errors.New("tag with this name already exists")
	ErrInvalidColor      = errors.New("color must be a hex value like #2563eb")
	ErrInvalidEntityType = errors.New("entity_type must be 'job' or 'company'")
	ErrEntityNotFound    = errors.New("entity not found")
)

type ErrorCode string

const (
	CodeTagNotFound       ErrorCode = "TAG_NOT_FOUND"
	CodeTagNameRequired   ErrorCode = "TAG_NAME_REQUIRED"
	CodeTagNameExists     ErrorCode = "TAG_NAME_EXISTS"
	CodeInvalidColor      ErrorCode = "INVALID_COLOR"
	CodeInvalidEntityType ErrorCode = "INVALID_ENTITY_TYPE"
	CodeEntityNotFound    ErrorCode = "ENTITY_NOT_FOUND"
	CodeValidationError   ErrorCode = "VALIDATION_ERROR"
	CodeInternalError     ErrorCode = "INTERNAL_ERROR"
)
