package model

import "time"

// EmailVerificationToken represents an email verification token in the database.
type EmailVerificationToken struct {
	ID        string
	UserID    string
	Code      string
	Attempts  int
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// PasswordResetToken represents a password reset token in the database.
type PasswordResetToken struct {
	ID        string
	UserID    string
	Code      string
	Attempts  int
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
