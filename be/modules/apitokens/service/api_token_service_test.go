package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeRepo struct {
	tokens      []*model.APIToken
	createCalls int
	lookupHash  string
	touches     int
	lookupErr   error
	touchErr    error
}

func (r *fakeRepo) Create(_ context.Context, token *model.APIToken) (*model.APIToken, error) {
	r.createCalls++
	count := 0
	for _, stored := range r.tokens {
		if stored.UserID == token.UserID && stored.RevokedAt == nil && (stored.ExpiresAt == nil || stored.ExpiresAt.After(time.Now())) {
			count++
		}
	}
	if count >= model.MaxActiveTokens {
		return nil, model.ErrTokenLimit
	}
	stored := *token
	stored.CreatedAt = time.Now()
	r.tokens = append(r.tokens, &stored)
	return &stored, nil
}
func (r *fakeRepo) List(_ context.Context, userID string) ([]*model.APIToken, error) {
	var tokens []*model.APIToken
	for index := len(r.tokens) - 1; index >= 0; index-- {
		if r.tokens[index].UserID == userID {
			tokens = append(tokens, r.tokens[index])
		}
	}
	return tokens, nil
}
func (r *fakeRepo) GetByHash(_ context.Context, hash string) (*model.APIToken, error) {
	r.lookupHash = hash
	if r.lookupErr != nil {
		return nil, r.lookupErr
	}
	for _, token := range r.tokens {
		if token.TokenHash == hash {
			return token, nil
		}
	}
	return nil, model.ErrTokenNotFound
}
func (r *fakeRepo) Revoke(_ context.Context, userID, tokenID string) error {
	for index, token := range r.tokens {
		if token.ID == tokenID && token.UserID == userID {
			revoked := *token
			now := time.Now()
			revoked.RevokedAt = &now
			r.tokens[index] = &revoked
			return nil
		}
	}
	return model.ErrTokenNotFound
}
func (r *fakeRepo) TouchLastUsed(_ context.Context, _ string) error { r.touches++; return r.touchErr }

type paidChecker struct{ err error }

func (p paidChecker) RequirePaidPlan(context.Context, string) error { return p.err }

func TestAPITokenService_Create(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
	plaintext, dto, err := svc.Create(t.Context(), "user", "  laptop  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "laptop", dto.Name)
	assert.Equal(t, plaintext[:12], dto.TokenPrefix)
	assert.Equal(t, []string{"mcp"}, dto.Scopes)
	assert.Nil(t, dto.ExpiresAt)
	require.Len(t, repo.tokens, 1)
	assert.Equal(t, hashToken(plaintext), repo.tokens[0].TokenHash)
	assert.NotEqual(t, plaintext, repo.tokens[0].TokenHash)
	encoded, err := json.Marshal(dto)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), plaintext)
	assert.NotContains(t, string(encoded), repo.tokens[0].TokenHash)
	dto.Scopes[0] = "changed"
	assert.Equal(t, []string{"mcp"}, repo.tokens[0].Scopes)
}

func TestAPITokenService_CreateValidation(t *testing.T) {
	tests := []struct {
		name, tokenName string
		days            *int
		want            error
	}{
		{name: "blank", tokenName: " \t", want: model.ErrInvalidName},
		{name: "long", tokenName: strings.Repeat("é", 101), want: model.ErrInvalidName},
		{name: "NUL", tokenName: "laptop\x00", want: model.ErrInvalidName},
		{name: "newline", tokenName: "lap\ntop", want: model.ErrInvalidName},
		{name: "tab", tokenName: "lap\ttop", want: model.ErrInvalidName},
		{name: "escape", tokenName: "laptop\x1b", want: model.ErrInvalidName},
		{name: "delete", tokenName: "laptop\x7f", want: model.ErrInvalidName},
		{name: "C1 control", tokenName: "laptop\u0085", want: model.ErrInvalidName},
		{name: "invalid expiry", tokenName: "ok", days: intPointer(31), want: model.ErrInvalidExpiry},
		{name: "zero expiry", tokenName: "ok", days: intPointer(0), want: model.ErrInvalidExpiry},
		{name: "negative expiry", tokenName: "ok", days: intPointer(-1), want: model.ErrInvalidExpiry},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
			token, dto, err := svc.Create(t.Context(), "user", test.tokenName, test.days)
			require.ErrorIs(t, err, test.want)
			assert.Empty(t, token)
			assert.Nil(t, dto)
			assert.Empty(t, repo.tokens)
			assert.Zero(t, repo.createCalls)
		})
	}
}

func intPointer(value int) *int { return &value }

func TestAPITokenService_CreateExpiry(t *testing.T) {
	for _, days := range []int{30, 90, 365} {
		t.Run(time.Duration(days).String(), func(t *testing.T) {
			svc := NewAPITokenService(&fakeRepo{}, paidChecker{}, nil, nil)
			before := time.Now().Add(time.Duration(days) * 24 * time.Hour)
			_, dto, err := svc.Create(t.Context(), "user", strings.Repeat("é", 100), &days)
			require.NoError(t, err)
			require.NotNil(t, dto.ExpiresAt)
			assert.WithinDuration(t, before, *dto.ExpiresAt, time.Second)
		})
	}
}

func TestAPITokenService_CreatePaidPlan(t *testing.T) {
	for _, planErr := range []error{subModel.ErrPaidFeature, errors.New("subscription unavailable")} {
		t.Run(planErr.Error(), func(t *testing.T) {
			repo := &fakeRepo{}
			svc := NewAPITokenService(repo, paidChecker{err: planErr}, nil, nil)
			token, dto, err := svc.Create(t.Context(), "free", "name", nil)
			require.ErrorIs(t, err, planErr)
			assert.Empty(t, token)
			assert.Nil(t, dto)
			assert.Empty(t, repo.tokens)
		})
	}
}

func TestAPITokenService_CreateCap(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
	now := time.Now()
	expired := now.Add(-time.Hour)
	repo.tokens = []*model.APIToken{{UserID: "user", RevokedAt: &now}, {UserID: "user", ExpiresAt: &expired}}
	for range model.MaxActiveTokens {
		_, _, err := svc.Create(t.Context(), "user", "name", nil)
		require.NoError(t, err)
	}
	token, dto, err := svc.Create(t.Context(), "user", "name", nil)
	require.ErrorIs(t, err, model.ErrTokenLimit)
	assert.Empty(t, token)
	assert.Nil(t, dto)
	_, _, err = svc.Create(t.Context(), "other", "name", nil)
	require.NoError(t, err)
}

func TestAPITokenService_ListAndRevoke(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
	plaintext, first, err := svc.Create(t.Context(), "user", "first", nil)
	require.NoError(t, err)
	_, second, err := svc.Create(t.Context(), "user", "second", nil)
	require.NoError(t, err)
	require.ErrorIs(t, svc.Revoke(t.Context(), "other", first.ID), model.ErrTokenNotFound)
	require.NoError(t, svc.Revoke(t.Context(), "user", first.ID))
	_, _, err = svc.Authenticate(t.Context(), plaintext)
	require.ErrorIs(t, err, model.ErrInvalidToken)
	require.ErrorIs(t, svc.Revoke(t.Context(), "user", "invalid"), model.ErrInvalidID)
	tokens, err := svc.List(t.Context(), "user")
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, second.ID, tokens[0].ID)
	assert.NotNil(t, tokens[1].RevokedAt)
}

func TestAPITokenService_RevokeCanonicalID(t *testing.T) {
	id := uuid.NewString()
	tests := []struct{ name, input string }{
		{"canonical", id},
		{"URN", "urn:uuid:" + id},
		{"braces", "{" + id + "}"},
		{"uppercase", strings.ToUpper(id)},
		{"compact", strings.ReplaceAll(id, "-", "")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepo{tokens: []*model.APIToken{{ID: id, UserID: "owner"}}}
			svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
			require.NoError(t, svc.Revoke(t.Context(), "owner", test.input))
			assert.NotNil(t, repo.tokens[0].RevokedAt)
		})
	}
}

func TestAPITokenService_Authenticate(t *testing.T) {
	plaintext, err := generateToken()
	require.NoError(t, err)
	past, future := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	tests := []struct {
		name  string
		token *model.APIToken
		input string
		want  error
	}{
		{name: "valid", token: &model.APIToken{ExpiresAt: &future}, input: plaintext},
		{name: "never expires", token: &model.APIToken{}, input: plaintext},
		{name: "revoked", token: &model.APIToken{RevokedAt: &past}, input: plaintext, want: model.ErrInvalidToken},
		{name: "expired", token: &model.APIToken{ExpiresAt: &past}, input: plaintext, want: model.ErrInvalidToken},
		{name: "unknown", input: plaintext, want: model.ErrInvalidToken},
		{name: "malformed", input: "jbr_pat_bad", want: model.ErrInvalidToken},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeRepo{}
			if test.token != nil {
				stored := *test.token
				stored.ID = uuid.NewString()
				stored.UserID = "owner"
				stored.TokenHash = hashToken(plaintext)
				repo.tokens = []*model.APIToken{&stored}
			}
			svc := NewAPITokenService(repo, paidChecker{}, nil, nil)
			userID, tokenID, err := svc.Authenticate(t.Context(), test.input)
			if test.want != nil {
				require.ErrorIs(t, err, test.want)
				assert.Empty(t, userID)
				assert.Empty(t, tokenID)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "owner", userID)
			assert.Equal(t, repo.tokens[0].ID, tokenID)
			assert.Equal(t, hashToken(plaintext), repo.lookupHash)
		})
	}
}

func TestAPITokenService_UsageDebounce(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	repo := &fakeRepo{}
	svc := NewAPITokenService(repo, paidChecker{}, client, nil)
	first, _, err := svc.Create(t.Context(), "user", "first", nil)
	require.NoError(t, err)
	second, _, err := svc.Create(t.Context(), "user", "second", nil)
	require.NoError(t, err)
	for range 5 {
		_, _, err = svc.Authenticate(t.Context(), first)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.touches)
	_, _, err = svc.Authenticate(t.Context(), second)
	require.NoError(t, err)
	assert.Equal(t, 2, repo.touches)
	assert.Equal(t, time.Minute, server.TTL("api_tokens:last_used:"+repo.tokens[0].ID))
	server.FastForward(time.Minute)
	_, _, err = svc.Authenticate(t.Context(), first)
	require.NoError(t, err)
	assert.Equal(t, 3, repo.touches)
}

func TestAPITokenService_UsageFailureStillAuthenticates(t *testing.T) {
	for _, redisFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "database", true: "redis"}[redisFails], func(t *testing.T) {
			server := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			repo := &fakeRepo{touchErr: errors.New("database down")}
			logs, observed := observer.New(zap.WarnLevel)
			svc := NewAPITokenService(repo, paidChecker{}, client, zap.New(logs))
			plaintext, _, err := svc.Create(t.Context(), "user", "name", nil)
			require.NoError(t, err)
			if redisFails {
				server.SetError("redis down")
			}
			userID, _, err := svc.Authenticate(t.Context(), plaintext)
			require.NoError(t, err)
			assert.Equal(t, "user", userID)
			require.Equal(t, 1, observed.Len())
			encoded, err := json.Marshal(observed.AllUntimed())
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), plaintext)
			assert.NotContains(t, string(encoded), hashToken(plaintext))
		})
	}
}
