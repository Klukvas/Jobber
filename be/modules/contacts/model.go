// Package contacts manages private contacts belonging to one owned Job.
package contacts

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("contact or job not found")
var ErrInvalid = errors.New("invalid contact")

type Fields struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	Telegram string `json:"telegram"`
	LinkedIn string `json:"linkedin"`
	Note     string `json:"note"`
}

type Contact struct {
	ID    string `json:"id"`
	JobID string `json:"job_id"`
	Fields
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
