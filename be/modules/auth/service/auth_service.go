package service

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	authPorts "github.com/andreypavlenko/jobber/modules/auth/ports"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	userPorts "github.com/andreypavlenko/jobber/modules/users/ports"
)

// SubscriptionCreator creates a free subscription for new users.
type SubscriptionCreator interface {
	EnsureFreeSubscription(ctx context.Context, userID string) error
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo            userPorts.UserRepository
	tokenRepo           authPorts.RefreshTokenRepository
	jwtManager          *auth.JWTManager
	accessExpiry        time.Duration
	refreshExpiry       time.Duration
	subscriptionCreator SubscriptionCreator
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo userPorts.UserRepository,
	tokenRepo authPorts.RefreshTokenRepository,
	jwtManager *auth.JWTManager,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
	subscriptionCreator ...SubscriptionCreator,
) *AuthService {
	svc := &AuthService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtManager:    jwtManager,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
	if len(subscriptionCreator) > 0 {
		svc.subscriptionCreator = subscriptionCreator[0]
	}
	return svc
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, req *authModel.RegisterRequest) (*userModel.UserDTO, *authModel.AuthTokens, error) {
	// Validate email
	if !isValidEmail(req.Email) {
		return nil, nil, userModel.ErrInvalidEmail
	}

	// Validate password (min 8, max 72 — bcrypt silently truncates beyond 72 bytes)
	if len(req.Password) < 8 || len(req.Password) > 72 {
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

	// Default name to email prefix
	name := strings.Split(email, "@")[0]

	// Create user
	user := userModel.NewUser(email, name, passwordHash, locale)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	// Create free subscription for new user
	if s.subscriptionCreator != nil {
		if err := s.subscriptionCreator.EnsureFreeSubscription(ctx, user.ID); err != nil {
			log.Printf("[ERROR] Failed to create free subscription for user %s: %v", user.ID, err)
		}
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

// RefreshTokens refreshes access token using refresh token.
// Uses atomic revocation to prevent race conditions: the old token is revoked
// first, and only the request that successfully revokes it gets new tokens.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshTokenString string) (*authModel.AuthTokens, error) {
	// Validate refresh token JWT signature and expiry
	claims, err := s.jwtManager.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Atomically revoke the old token — only one concurrent request wins
	tokenHash := auth.HashToken(refreshTokenString)
	revoked, err := s.tokenRepo.RevokeIfValid(ctx, tokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}
	if !revoked {
		// Token was already revoked (by another request) or expired
		return nil, errors.New("refresh token expired or revoked")
	}

	// Only the winning request reaches here — generate new tokens
	tokens, err := s.generateTokens(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

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

// emailRegex is compiled once at package level to avoid recompilation on every call.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidEmail validates email format
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}
