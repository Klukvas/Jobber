package model

import (
	"errors"
	"time"
)

const (
	DefaultActivityLimit = 20
	MaxActivityLimit     = 100
)

var ErrInvalidActivityLimit = errors.New("limit must be an integer between 1 and 100")

type ActivityDTO struct {
	ID         string    `json:"id"`
	Tool       string    `json:"tool"`
	OK         bool      `json:"ok"`
	ErrorCode  *string   `json:"error_code"`
	DurationMS int       `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
	APITokenID *string   `json:"api_token_id"`
	TokenName  *string   `json:"token_name"`
}
