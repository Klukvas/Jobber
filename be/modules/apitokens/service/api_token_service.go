package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/apitokens/ports"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PaidPlanChecker interface {
	RequirePaidPlan(ctx context.Context, userID string) error
}

type UsageDebouncer interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
}

type APITokenService struct {
	repo     ports.APITokenRepository
	paid     PaidPlanChecker
	debounce UsageDebouncer
	logger   *zap.Logger
}

func NewAPITokenService(repo ports.APITokenRepository, paid PaidPlanChecker, debounce UsageDebouncer, logger *zap.Logger) *APITokenService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &APITokenService{repo: repo, paid: paid, debounce: debounce, logger: logger}
}

// Create returns the plaintext exactly once. nil expiry means no expiration;
// the HTTP handler applies the 90-day default only when the field is omitted.
func (s *APITokenService) Create(ctx context.Context, userID, name string, expiresInDays *int) (string, *model.APITokenDTO, error) {
	if strings.ContainsFunc(name, unicode.IsControl) {
		return "", nil, model.ErrInvalidName
	}
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return "", nil, model.ErrInvalidName
	}
	expiresAt, err := expiration(expiresInDays)
	if err != nil {
		return "", nil, err
	}
	if s.paid == nil {
		return "", nil, errors.New("api token paid plan checker unavailable")
	}
	if err := s.paid.RequirePaidPlan(ctx, userID); err != nil {
		return "", nil, fmt.Errorf("require paid api tokens: %w", err)
	}
	plaintext, err := generateToken()
	if err != nil {
		return "", nil, err
	}
	token, err := s.repo.Create(ctx, &model.APIToken{
		ID: uuid.NewString(), UserID: userID, Name: name, TokenPrefix: plaintext[:12],
		TokenHash: hashToken(plaintext), Scopes: []string{"mcp"}, ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", nil, fmt.Errorf("create api token: %w", err)
	}
	return plaintext, token.ToDTO(), nil
}

func expiration(days *int) (*time.Time, error) {
	if days == nil {
		return nil, nil
	}
	if *days != 30 && *days != 90 && *days != 365 {
		return nil, model.ErrInvalidExpiry
	}
	expires := time.Now().UTC().Add(time.Duration(*days) * 24 * time.Hour)
	return &expires, nil
}

func (s *APITokenService) List(ctx context.Context, userID string) ([]*model.APITokenDTO, error) {
	tokens, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	dtos := make([]*model.APITokenDTO, 0, len(tokens))
	for _, token := range tokens {
		dtos = append(dtos, token.ToDTO())
	}
	return dtos, nil
}

func (s *APITokenService) Revoke(ctx context.Context, userID, tokenID string) error {
	id, err := uuid.Parse(tokenID)
	if err != nil {
		return model.ErrInvalidID
	}
	if err := s.repo.Revoke(ctx, userID, id.String()); err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	return nil
}

func (s *APITokenService) Authenticate(ctx context.Context, plaintext string) (string, string, error) {
	info, err := s.AuthenticateWithInfo(ctx, plaintext)
	if err != nil {
		return "", "", err
	}
	return info.UserID, info.TokenID, nil
}

// AuthenticateWithInfo performs the same single lookup as Authenticate and also
// returns the verified expiration for transports that enforce token lifetimes.
func (s *APITokenService) AuthenticateWithInfo(ctx context.Context, plaintext string) (*model.AuthenticationInfo, error) {
	if !validToken(plaintext) {
		return nil, model.ErrInvalidToken
	}
	token, err := s.repo.GetByHash(ctx, hashToken(plaintext))
	if errors.Is(err, model.ErrTokenNotFound) {
		return nil, model.ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("authenticate api token: %w", err)
	}
	if token.RevokedAt != nil || token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now()) {
		return nil, model.ErrInvalidToken
	}
	s.recordUsage(ctx, token.ID)
	info := &model.AuthenticationInfo{UserID: token.UserID, TokenID: token.ID}
	if token.ExpiresAt != nil {
		info.ExpiresAt = *token.ExpiresAt
	}
	return info, nil
}

func (s *APITokenService) recordUsage(ctx context.Context, tokenID string) {
	if s.debounce == nil {
		s.logger.Warn("API token usage debounce unavailable", zap.String("api_token_id", tokenID))
		return
	}
	update, err := s.debounce.SetNX(ctx, "api_tokens:last_used:"+tokenID, "1", time.Minute).Result()
	if err != nil {
		s.logger.Warn("API token usage debounce failed", zap.String("api_token_id", tokenID))
		return
	}
	if update {
		if err := s.repo.TouchLastUsed(ctx, tokenID); err != nil {
			s.logger.Warn("API token usage update failed", zap.String("api_token_id", tokenID))
		}
	}
}
