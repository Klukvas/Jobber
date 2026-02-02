package model

import (
	"time"
)

// User represents a platform user
type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Locale       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser creates a new user
func NewUser(email, name, passwordHash, locale string) *User {
	now := time.Now().UTC()
	return &User{
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		Locale:       locale,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// UserDTO represents user data transfer object (without sensitive data)
type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Locale    string    `json:"locale"`
	CreatedAt time.Time `json:"created_at"`
}

// ToDTO converts User to UserDTO
func (u *User) ToDTO() *UserDTO {
	return &UserDTO{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Locale:    u.Locale,
		CreatedAt: u.CreatedAt,
	}
}
