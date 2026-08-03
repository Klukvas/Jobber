package model

import (
	"errors"
	"time"
)

type Comment struct {
	ID        string
	UserID    string
	JobID     string
	StageID   *string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentDTO struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	StageID   *string   `json:"stage_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *Comment) ToDTO() *CommentDTO {
	return &CommentDTO{
		ID:        c.ID,
		JobID:     c.JobID,
		StageID:   c.StageID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}

type CreateCommentRequest struct {
	JobID   string  `json:"job_id" binding:"required"`
	StageID *string `json:"stage_id,omitempty"`
	Content string  `json:"content" binding:"required,min=1"`
}

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrContentRequired = errors.New("content is required")
	// ErrJobNotFound is returned when the target job does not exist or does not
	// belong to the authenticated user (ownership guard on comment creation).
	ErrJobNotFound = errors.New("job not found")
)

type ErrorCode string

const (
	CodeCommentNotFound ErrorCode = "COMMENT_NOT_FOUND"
	CodeContentRequired ErrorCode = "CONTENT_REQUIRED"
	CodeJobNotFound     ErrorCode = "JOB_NOT_FOUND"
	CodeInternalError   ErrorCode = "INTERNAL_ERROR"
)
