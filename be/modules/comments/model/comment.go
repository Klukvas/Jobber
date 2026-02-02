package model

import (
	"errors"
	"time"
)

type Comment struct {
	ID            string
	UserID        string
	ApplicationID string
	StageID       *string
	Content       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CommentDTO struct {
	ID            string     `json:"id"`
	ApplicationID string     `json:"application_id"`
	StageID       *string    `json:"stage_id,omitempty"`
	Content       string     `json:"content"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (c *Comment) ToDTO() *CommentDTO {
	return &CommentDTO{
		ID:            c.ID,
		ApplicationID: c.ApplicationID,
		StageID:       c.StageID,
		Content:       c.Content,
		CreatedAt:     c.CreatedAt,
	}
}

type CreateCommentRequest struct {
	ApplicationID string  `json:"application_id" binding:"required"`
	StageID       *string `json:"stage_id,omitempty"`
	Content       string  `json:"content" binding:"required,min=1"`
}

var (
	ErrCommentNotFound      = errors.New("comment not found")
	ErrContentRequired      = errors.New("content is required")
)

type ErrorCode string

const (
	CodeCommentNotFound ErrorCode = "COMMENT_NOT_FOUND"
	CodeContentRequired ErrorCode = "CONTENT_REQUIRED"
	CodeInternalError   ErrorCode = "INTERNAL_ERROR"
)
