package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/internal/platform/email"
	sentryPlatform "github.com/andreypavlenko/jobber/internal/platform/sentry"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	authPorts "github.com/andreypavlenko/jobber/modules/auth/ports"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	userPorts "github.com/andreypavlenko/jobber/modules/users/ports"
	"go.uber.org/zap"
)

const maxCodeAttempts = 5

// SubscriptionCreator creates a free subscription for new users.
type SubscriptionCreator interface {
	EnsureFreeSubscription(ctx context.Context, userID string) error
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo            userPorts.UserRepository
	tokenRepo           authPorts.RefreshTokenRepository
	verificationRepo    authPorts.EmailVerificationRepository
	passwordResetRepo   authPorts.PasswordResetRepository
	emailSender         email.Sender
	jwtManager          *auth.JWTManager
	accessExpiry        time.Duration
	refreshExpiry       time.Duration
	subscriptionCreator SubscriptionCreator
	logger              *zap.Logger
}

// AuthServiceConfig holds all dependencies for AuthService.
type AuthServiceConfig struct {
	UserRepo            userPorts.UserRepository
	TokenRepo           authPorts.RefreshTokenRepository
	VerificationRepo    authPorts.EmailVerificationRepository
	PasswordResetRepo   authPorts.PasswordResetRepository
	EmailSender         email.Sender
	JWTManager          *auth.JWTManager
	AccessExpiry        time.Duration
	RefreshExpiry       time.Duration
	SubscriptionCreator SubscriptionCreator
	Logger              *zap.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(cfg AuthServiceConfig) *AuthService {
	l := cfg.Logger
	if l == nil {
		l = zap.NewNop()
	}
	return &AuthService{
		userRepo:            cfg.UserRepo,
		tokenRepo:           cfg.TokenRepo,
		verificationRepo:    cfg.VerificationRepo,
		passwordResetRepo:   cfg.PasswordResetRepo,
		emailSender:         cfg.EmailSender,
		jwtManager:          cfg.JWTManager,
		accessExpiry:        cfg.AccessExpiry,
		refreshExpiry:       cfg.RefreshExpiry,
		subscriptionCreator: cfg.SubscriptionCreator,
		logger:              l,
	}
}

// RegisterResponse is the response returned after registration.
type RegisterResponse struct {
	Message string `json:"message"`
}

// Register registers a new user and sends a verification email.
func (s *AuthService) Register(ctx context.Context, req *authModel.RegisterRequest) (*RegisterResponse, error) {
	// Validate email
	if !isValidEmail(req.Email) {
		return nil, userModel.ErrInvalidEmail
	}

	// Validate password (min 8, max 72 — bcrypt silently truncates beyond 72 bytes)
	if len(req.Password) < 8 || len(req.Password) > 72 {
		return nil, userModel.ErrInvalidPassword
	}

	// Normalize email
	emailAddr := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err == nil && existingUser != nil {
		return nil, userModel.ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Set default locale
	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	// Default name to email prefix
	name := strings.Split(emailAddr, "@")[0]

	// Create user with email_verified = false
	user := userModel.NewUser(emailAddr, name, passwordHash, locale)
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Create free subscription for new user
	if s.subscriptionCreator != nil {
		if err := s.subscriptionCreator.EnsureFreeSubscription(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("failed to create free subscription for user %s: %w", user.ID, err)
		}
	}

	// Generate verification code and send email (non-fatal: user can resend later)
	if err := s.sendVerificationEmail(ctx, user.ID, emailAddr, locale); err != nil {
		s.logger.Error("failed to send verification email during registration",
			zap.String("user_id", user.ID), zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "register_send_verification", "user_id": user.ID})
	}

	return &RegisterResponse{Message: "Please check your email to verify your account"}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *authModel.LoginRequest) (*userModel.UserDTO, *authModel.AuthTokens, error) {
	// Normalize email
	emailAddr := strings.ToLower(strings.TrimSpace(req.Email))

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
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

	// Check email verification
	if !user.EmailVerified {
		return nil, nil, userModel.ErrEmailNotVerified
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user.ToDTO(), tokens, nil
}

// VerifyEmail verifies a user's email using their email and a 6-digit code.
func (s *AuthService) VerifyEmail(ctx context.Context, emailAddr, code string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil {
		return userModel.ErrInvalidVerificationToken
	}

	token, err := s.verificationRepo.GetActiveForUser(ctx, user.ID)
	if err != nil {
		return userModel.ErrInvalidVerificationToken
	}

	// Fast path: already exhausted
	if token.Attempts >= maxCodeAttempts {
		return userModel.ErrTooManyAttempts
	}

	// Atomic increment — fails if attempts already >= maxCodeAttempts (race-safe)
	newAttempts, err := s.verificationRepo.IncrementAttempts(ctx, token.ID, maxCodeAttempts)
	if err != nil {
		if errors.Is(err, userModel.ErrTooManyAttempts) {
			return userModel.ErrTooManyAttempts
		}
		// Fail closed: if DB error, don't allow verification
		s.logger.Error("failed to increment verification attempts", zap.String("token_id", token.ID), zap.Error(err))
		return userModel.ErrInvalidVerificationToken
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token.Code), []byte(code)) != 1 {
		if newAttempts >= maxCodeAttempts {
			return userModel.ErrTooManyAttempts
		}
		return userModel.ErrInvalidVerificationToken
	}

	// Set user email as verified first (primary operation)
	if err := s.userRepo.SetEmailVerified(ctx, token.UserID); err != nil {
		return err
	}

	// Mark token as used last — if this fails, user is still verified
	if err := s.verificationRepo.MarkUsed(ctx, token.ID); err != nil {
		s.logger.Error("failed to mark verification token as used", zap.String("token_id", token.ID), zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "verify_email_mark_used", "token_id": token.ID})
	}

	return nil
}

// ResendVerification resends the verification email. Always returns nil to prevent email enumeration.
func (s *AuthService) ResendVerification(ctx context.Context, emailAddr string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil {
		return nil // Don't reveal if user exists
	}

	if user.EmailVerified {
		return nil // Already verified, no-op
	}

	if err := s.sendVerificationEmail(ctx, user.ID, user.Email, user.Locale); err != nil {
		s.logger.Error("failed to resend verification email", zap.String("user_id", user.ID), zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "resend_verification"})
	}
	return nil
}

// ForgotPassword sends a password reset email. Always returns nil to prevent email enumeration.
func (s *AuthService) ForgotPassword(ctx context.Context, emailAddr string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil {
		return nil // Don't reveal if user exists
	}

	code, err := generateCode()
	if err != nil {
		s.logger.Error("failed to generate password reset code", zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "forgot_password_generate_code"})
		return nil
	}

	// Delete existing unused tokens for this user to prevent accumulation
	if err := s.passwordResetRepo.DeleteForUser(ctx, user.ID); err != nil {
		s.logger.Error("failed to delete old password reset tokens", zap.String("user_id", user.ID), zap.Error(err))
	}

	resetToken := &authModel.PasswordResetToken{
		UserID:    user.ID,
		Code:      code,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.passwordResetRepo.Create(ctx, resetToken); err != nil {
		s.logger.Error("failed to create password reset token", zap.String("user_id", user.ID), zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "forgot_password_create_token"})
		return nil
	}

	if err := s.emailSender.SendPasswordResetEmail(ctx, user.Email, code, user.Locale); err != nil {
		s.logger.Error("failed to send password reset email", zap.String("user_id", user.ID), zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "forgot_password_send_email"})
	}
	return nil
}

// ResetPassword resets a user's password using email, code, and new password.
func (s *AuthService) ResetPassword(ctx context.Context, emailAddr, code, newPassword string) error {
	if len(newPassword) < 8 || len(newPassword) > 72 {
		return userModel.ErrInvalidPassword
	}

	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	user, err := s.userRepo.GetByEmail(ctx, emailAddr)
	if err != nil {
		return userModel.ErrInvalidResetToken
	}

	token, err := s.passwordResetRepo.GetActiveForUser(ctx, user.ID)
	if err != nil {
		return userModel.ErrInvalidResetToken
	}

	// Fast path: already exhausted
	if token.Attempts >= maxCodeAttempts {
		return userModel.ErrTooManyAttempts
	}

	// Atomic increment — fails if attempts already >= maxCodeAttempts (race-safe)
	newAttempts, err := s.passwordResetRepo.IncrementAttempts(ctx, token.ID, maxCodeAttempts)
	if err != nil {
		if errors.Is(err, userModel.ErrTooManyAttempts) {
			return userModel.ErrTooManyAttempts
		}
		s.logger.Error("failed to increment reset attempts", zap.String("token_id", token.ID), zap.Error(err))
		return userModel.ErrInvalidResetToken
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token.Code), []byte(code)) != 1 {
		if newAttempts >= maxCodeAttempts {
			return userModel.ErrTooManyAttempts
		}
		return userModel.ErrInvalidResetToken
	}

	// Hash new password
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Mark token as used first — prevents concurrent use of same token
	if err := s.passwordResetRepo.MarkUsed(ctx, token.ID); err != nil {
		return err
	}

	// Update password (primary operation)
	if err := s.userRepo.UpdatePasswordHash(ctx, token.UserID, passwordHash); err != nil {
		return err
	}

	// Revoke all refresh tokens for security — non-critical, log on failure
	if err := s.tokenRepo.RevokeAllForUser(ctx, token.UserID); err != nil {
		s.logger.Error("failed to revoke refresh tokens after password reset", zap.String("user_id", token.UserID), zap.Error(err))
		sentryPlatform.CaptureError(err, map[string]string{"context": "reset_password_revoke_tokens", "user_id": token.UserID})
	}

	return nil
}

// RefreshTokens refreshes access token using refresh token.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshTokenString string) (*authModel.AuthTokens, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	tokenHash := auth.HashToken(refreshTokenString)
	revoked, err := s.tokenRepo.RevokeIfValid(ctx, tokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}
	if !revoked {
		return nil, errors.New("refresh token expired or revoked")
	}

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
	accessToken, err := s.jwtManager.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

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

func (s *AuthService) sendVerificationEmail(ctx context.Context, userID, emailAddr, locale string) error {
	code, err := generateCode()
	if err != nil {
		return err
	}

	// Delete existing unused tokens for this user to prevent accumulation
	if err := s.verificationRepo.DeleteForUser(ctx, userID); err != nil {
		s.logger.Error("failed to delete old verification tokens", zap.String("user_id", userID), zap.Error(err))
	}

	verificationToken := &authModel.EmailVerificationToken{
		UserID:    userID,
		Code:      code,
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.verificationRepo.Create(ctx, verificationToken); err != nil {
		return err
	}

	return s.emailSender.SendVerificationEmail(ctx, emailAddr, code, locale)
}

// generateCode generates a cryptographically random 6-digit code.
func generateCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// emailRegex is compiled once at package level to avoid recompilation on every call.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// isValidEmail validates email format
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}
