package model

import (
	"errors"
	"time"
)

type Reminder struct {
	ID            string
	UserID        string
	ApplicationID string
	StageID       *string
	RemindAt      time.Time
	Message       string
	IsDone        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ReminderDTO struct {
	ID            string     `json:"id"`
	ApplicationID string     `json:"application_id"`
	StageID       *string    `json:"stage_id,omitempty"`
	RemindAt      time.Time  `json:"remind_at"`
	Message       string     `json:"message"`
	IsDone        bool       `json:"is_done"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (r *Reminder) ToDTO() *ReminderDTO {
	return &ReminderDTO{
		ID:            r.ID,
		ApplicationID: r.ApplicationID,
		StageID:       r.StageID,
		RemindAt:      r.RemindAt,
		Message:       r.Message,
		IsDone:        r.IsDone,
		CreatedAt:     r.CreatedAt,
	}
}

type CreateReminderRequest struct {
	ApplicationID string    `json:"application_id" binding:"required"`
	StageID       *string   `json:"stage_id,omitempty"`
	RemindAt      time.Time `json:"remind_at" binding:"required"`
	Message       string    `json:"message" binding:"required,min=1"`
}

type UpdateReminderRequest struct {
	IsDone *bool `json:"is_done,omitempty"`
}

var (
	ErrReminderNotFound = errors.New("reminder not found")
)

type ErrorCode string

const (
	CodeReminderNotFound ErrorCode = "REMINDER_NOT_FOUND"
	CodeInternalError    ErrorCode = "INTERNAL_ERROR"
)
