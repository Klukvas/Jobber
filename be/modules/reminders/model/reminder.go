package model

import (
	"errors"
	"time"
)

type Reminder struct {
	ID        string
	UserID    string
	JobID     string
	StageID   *string
	RemindAt  time.Time
	Message   string
	IsDone    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReminderDTO struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	StageID   *string   `json:"stage_id,omitempty"`
	RemindAt  time.Time `json:"remind_at"`
	Message   string    `json:"message"`
	IsDone    bool      `json:"is_done"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Reminder) ToDTO() *ReminderDTO {
	return &ReminderDTO{
		ID:        r.ID,
		JobID:     r.JobID,
		StageID:   r.StageID,
		RemindAt:  r.RemindAt,
		Message:   r.Message,
		IsDone:    r.IsDone,
		CreatedAt: r.CreatedAt,
	}
}

type CreateReminderRequest struct {
	JobID    string    `json:"job_id" binding:"required"`
	StageID  *string   `json:"stage_id,omitempty"`
	RemindAt time.Time `json:"remind_at" binding:"required"`
	Message  string    `json:"message" binding:"required,min=1,max=2000"`
}

// UpdateReminderRequest is a partial update — every field is optional; only the
// provided fields are changed.
type UpdateReminderRequest struct {
	Message  *string    `json:"message,omitempty"`
	RemindAt *time.Time `json:"remind_at,omitempty"`
	IsDone   *bool      `json:"is_done,omitempty"`
}

var (
	ErrReminderNotFound = errors.New("reminder not found")
	ErrJobNotFound      = errors.New("job not found")
	ErrMessageRequired  = errors.New("message is required")
)

type ErrorCode string

const (
	CodeReminderNotFound ErrorCode = "REMINDER_NOT_FOUND"
	CodeJobNotFound      ErrorCode = "JOB_NOT_FOUND"
	CodeMessageRequired  ErrorCode = "MESSAGE_REQUIRED"
	CodeValidationError  ErrorCode = "VALIDATION_ERROR"
	CodeInternalError    ErrorCode = "INTERNAL_ERROR"
)
