package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	authPorts "github.com/andreypavlenko/jobber/modules/auth/ports"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	userPorts "github.com/andreypavlenko/jobber/modules/users/ports"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo         userPorts.UserRepository
	tokenRepo        authPorts.RefreshTokenRepository
	jwtManager       *auth.JWTManager
	accessExpiry     time.Duration
	refreshExpiry    time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo userPorts.UserRepository,
	tokenRepo authPorts.RefreshTokenRepository,
	jwtManager *auth.JWTManager,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtManager:    jwtManager,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, req *authModel.RegisterRequest) (*userModel.UserDTO, *authModel.AuthTokens, error) {
	// Validate email
	if !isValidEmail(req.Email) {
		return nil, nil, userModel.ErrInvalidEmail
	}

	// Validate password
	if len(req.Password) < 8 {
		return nil, nil, userModel.ErrInvalidPassword
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, nil, userModel.ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, nil, err
	}

	// Set default locale
	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	// Create user
	user := userModel.NewUser(email, req.Name, passwordHash, locale)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user.ToDTO(), tokens, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *authModel.LoginRequest) (*userModel.UserDTO, *authModel.AuthTokens, error) {
	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, userModel.ErrUserNotFound) {
			return nil, nil, userModel.ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, nil, userModel.ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user.ToDTO(), tokens, nil
}

// RefreshTokens refreshes access token using refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshTokenString string) (*authModel.AuthTokens, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if token exists in database and is valid
	tokenHash := auth.HashToken(refreshTokenString)
	dbToken, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	if !dbToken.IsValid() {
		return nil, errors.New("refresh token expired or revoked")
	}

	// Generate new tokens
	tokens, err := s.generateTokens(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Revoke old refresh token
	_ = s.tokenRepo.Revoke(ctx, tokenHash)

	return tokens, nil
}

// Logout revokes all refresh tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	return s.tokenRepo.RevokeAllForUser(ctx, userID)
}

// generateTokens generates access and refresh tokens
func (s *AuthService) generateTokens(ctx context.Context, userID string) (*authModel.AuthTokens, error) {
	// Generate access token
	accessToken, err := s.jwtManager.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	// Store refresh token in database
	tokenHash := auth.HashToken(refreshToken)
	dbToken := authModel.NewRefreshToken(userID, tokenHash, time.Now().UTC().Add(s.refreshExpiry))
	if err := s.tokenRepo.Create(ctx, dbToken); err != nil {
		return nil, err
	}

	return &authModel.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
