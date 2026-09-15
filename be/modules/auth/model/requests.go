package model

// RegisterRequest represents a registration request.
//
// Password length is deliberately NOT a binding tag: the validator counts
// runes and answers with a generic "Invalid request payload", which used to
// surface an over-long password as an unexplained error on the email field.
// The service checks the real byte length and returns a specific error.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Locale   string `json:"locale"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents a refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
