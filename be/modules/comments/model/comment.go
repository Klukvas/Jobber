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
	ID      string  `json:"id"`
	JobID   string  `json:"job_id"`
	StageID *string `json:"stage_id,omitempty"`
	Content string  `json:"content"`
	// UpdatedAt lets the UI mark a comment that has been edited since it was
	// written. Equal to CreatedAt for a comment that was never edited.
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *Comment) ToDTO() *CommentDTO {
	return &CommentDTO{
		ID:        c.ID,
		JobID:     c.JobID,
		StageID:   c.StageID,
		Content:   c.Content,
		UpdatedAt: c.UpdatedAt,
		CreatedAt: c.CreatedAt,
	}
}

// `content` carries no binding tag on either request below, deliberately. The
// rule is the service's: it trims first and answers CONTENT_REQUIRED. A binding
// tag would reject the same input a step earlier as a shapeless
// VALIDATION_ERROR, so an empty comment and an unparseable body came back
// indistinguishable — and the two write endpoints for one resource disagreed
// about which was which.

type CreateCommentRequest struct {
	JobID   string  `json:"job_id" binding:"required"`
	StageID *string `json:"stage_id,omitempty"`
	Content string  `json:"content"`
}

// UpdateCommentRequest carries the new body of an existing comment.
type UpdateCommentRequest struct {
	Content string `json:"content"`
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
