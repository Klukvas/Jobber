package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMatchScoreCacheRepo implements ports.MatchScoreCacheRepository for testing.
type MockMatchScoreCacheRepo struct {
	GetFunc              func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error)
	UpsertFunc           func(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error
	InvalidateByJobFunc  func(ctx context.Context, jobID string) error
	InvalidateByResumeFunc func(ctx context.Context, resumeID string) error
}

func (m *MockMatchScoreCacheRepo) Get(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, userID, jobID, resumeID)
	}
	return nil, nil
}

func (m *MockMatchScoreCacheRepo) Upsert(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, userID, jobID, resumeID, result)
	}
	return nil
}

func (m *MockMatchScoreCacheRepo) InvalidateByJob(ctx context.Context, jobID string) error {
	if m.InvalidateByJobFunc != nil {
		return m.InvalidateByJobFunc(ctx, jobID)
	}
	return nil
}

func (m *MockMatchScoreCacheRepo) InvalidateByResume(ctx context.Context, resumeID string) error {
	if m.InvalidateByResumeFunc != nil {
		return m.InvalidateByResumeFunc(ctx, resumeID)
	}
	return nil
}

func TestCheckMatch_CacheHit(t *testing.T) {
	cached := &model.MatchScoreResponse{
		OverallScore:    85,
		Categories:      []model.MatchScoreCategory{{Name: "Skills", Score: 90, Details: "Good match"}},
		MissingKeywords: []string{"Docker"},
		Strengths:       []string{"Go"},
		Summary:         "Strong match",
	}

	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "job-1", jobID)
			assert.Equal(t, "resume-1", resumeID)
			return cached, nil
		},
	}

	// AI client, s3, jobRepo, resumeRepo are all nil — cache hit must return before touching them
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	result, err := svc.CheckMatch(context.Background(), "user-1", req)

	require.NoError(t, err)
	assert.Equal(t, 85, result.OverallScore)
	assert.Equal(t, "Strong match", result.Summary)
	assert.True(t, result.FromCache)
}

func TestCheckMatch_CacheReadError_FallsThrough(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			return nil, errors.New("connection refused")
		},
	}

	// limitChecker returns an error so we can verify the code fell through past the cache
	mockLimitChecker := &MockLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewMatchScoreService(nil, nil, nil, nil, mockLimitChecker, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	// Should have fallen through cache and hit the limit checker
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

func TestCheckMatch_CacheMiss_FallsThrough(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			return nil, nil // cache miss
		},
	}

	mockLimitChecker := &MockLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewMatchScoreService(nil, nil, nil, nil, mockLimitChecker, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	// Should have fallen through cache miss and hit the limit checker
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

func TestCheckMatch_NilCacheRepo_DoesNotPanic(t *testing.T) {
	mockLimitChecker := &MockLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewMatchScoreService(nil, nil, nil, nil, mockLimitChecker, nil)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	// Should skip cache and hit limit checker
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

// MockLimitChecker implements LimitChecker for testing.
type MockLimitChecker struct {
	CheckLimitFunc   func(ctx context.Context, userID, resource string) error
	RecordAIUsageFunc func(ctx context.Context, userID string) error
}

func (m *MockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

func (m *MockLimitChecker) RecordAIUsage(ctx context.Context, userID string) error {
	if m.RecordAIUsageFunc != nil {
		return m.RecordAIUsageFunc(ctx, userID)
	}
	return nil
}
