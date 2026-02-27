package model

import "time"

// CalendarToken represents an encrypted Google OAuth token
type CalendarToken struct {
	ID         string
	UserID     string
	TokenBlob  string // AES-256-GCM encrypted JSON
	TokenNonce string // AES-GCM nonce (base64)
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CalendarStatusDTO represents the calendar connection status
type CalendarStatusDTO struct {
	Connected bool   `json:"connected"`
	Email     string `json:"email,omitempty"`
}
