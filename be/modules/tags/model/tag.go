package model

import (
	"errors"
	"time"
)

type Tag struct {
	ID        string
	UserID    string
	Name      string
	Color     *string
	CreatedAt time.Time
}

type TagDTO struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Color     *string    `json:"color,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
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

type TagRelation struct {
	ID         string
	TagID      string
	EntityType string
	EntityID   string
	CreatedAt  time.Time
}

var (
	ErrTagNotFound    = errors.New("tag not found")
	ErrTagNameRequired = errors.New("tag name is required")
)

type ErrorCode string

const (
	CodeTagNotFound    ErrorCode = "TAG_NOT_FOUND"
	CodeTagNameRequired ErrorCode = "TAG_NAME_REQUIRED"
	CodeInternalError  ErrorCode = "INTERNAL_ERROR"
)
