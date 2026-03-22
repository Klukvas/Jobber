package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/jobimport/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock limit checker
// ---------------------------------------------------------------------------

type MockLimitChecker struct {
	CheckLimitFunc        func(ctx context.Context, userID, resource string) error
	RecordJobParseUsageFunc func(ctx context.Context, userID string) error
}

func (m *MockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

func (m *MockLimitChecker) RecordJobParseUsage(ctx context.Context, userID string) error {
	if m.RecordJobParseUsageFunc != nil {
		return m.RecordJobParseUsageFunc(ctx, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// ParseJobPage tests
// ---------------------------------------------------------------------------

func TestImportService_ParseJobPage(t *testing.T) {
	userID := "user-123"

	tests := []struct {
		name         string
		aiClient     *ai.AnthropicClient
		limitChecker *MockLimitChecker
		req          *model.ParseJobRequest
		wantErr      bool
		wantErrIs    error
		errContains  string
	}{
		{
			name:     "returns ErrAINotConfigured when aiClient is nil",
			aiClient: nil,
			req: &model.ParseJobRequest{
				PageText: "Job posting text...",
				PageURL:  "https://example.com/job/123",
			},
			wantErr:   true,
			wantErrIs: model.ErrAINotConfigured,
		},
		{
			name:     "returns limit error when plan limit reached",
			aiClient: &ai.AnthropicClient{}, // non-nil placeholder
			limitChecker: &MockLimitChecker{
				CheckLimitFunc: func(_ context.Context, _ string, resource string) error {
					assert.Equal(t, "job_parses", resource)
					return subModel.ErrLimitReached
				},
			},
			req: &model.ParseJobRequest{
				PageText: "Job posting text...",
				PageURL:  "https://example.com/job/123",
			},
			wantErr:   true,
			wantErrIs: subModel.ErrLimitReached,
		},
		{
			name:         "passes limit check with nil limitChecker",
			aiClient:     nil, // will fail on AI call but limit check should pass
			limitChecker: nil,
			req: &model.ParseJobRequest{
				PageText: "Job posting text...",
				PageURL:  "https://example.com/job/123",
			},
			wantErr:   true,
			wantErrIs: model.ErrAINotConfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lc LimitChecker
			if tt.limitChecker != nil {
				lc = tt.limitChecker
			}

			svc := NewImportService(tt.aiClient, lc)
			result, err := svc.ParseJobPage(context.Background(), userID, tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

func TestImportService_ParseJobPage_RecordUsageError(t *testing.T) {
	// When limitChecker.RecordJobParseUsage fails, it should be logged
	// but should not cause ParseJobPage to fail (if AI call succeeds).
	// Since we can't mock the actual AI client (it's a concrete struct),
	// we verify the nil-client path instead.
	svc := NewImportService(nil, nil)
	_, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "text",
		PageURL:  "https://example.com",
	})
	assert.ErrorIs(t, err, model.ErrAINotConfigured)
}

func TestNewImportService(t *testing.T) {
	tests := []struct {
		name         string
		aiClient     *ai.AnthropicClient
		limitChecker LimitChecker
	}{
		{
			name:         "creates with nil aiClient and nil limitChecker",
			aiClient:     nil,
			limitChecker: nil,
		},
		{
			name:         "creates with nil aiClient and non-nil limitChecker",
			aiClient:     nil,
			limitChecker: &MockLimitChecker{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewImportService(tt.aiClient, tt.limitChecker)
			require.NotNil(t, svc)
		})
	}
}

func TestImportService_ParseJobPage_LimitCheckError(t *testing.T) {
	lc := &MockLimitChecker{
		CheckLimitFunc: func(_ context.Context, _, _ string) error {
			return errors.New("subscription service unavailable")
		},
	}

	// Need a non-nil aiClient to get past the nil check
	svc := &ImportService{
		aiClient:     &ai.AnthropicClient{},
		limitChecker: lc,
	}

	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Job posting text...",
		PageURL:  "https://example.com/job/123",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscription service unavailable")
}

func TestImportService_ParseJobPage_AICallSuccess(t *testing.T) {
	companyName := "Acme Corp"
	source := "LinkedIn"
	pageURL := "https://example.com/job/123"
	description := "Build amazing things"

	parsedJSON, err := json.Marshal(ai.ParsedJob{
		Title:       "Senior Go Developer",
		CompanyName: &companyName,
		Source:      &source,
		URL:         &pageURL,
		Description: &description,
	})
	require.NoError(t, err)

	// Create an AI client with mocked callAPIFunc
	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(parsedJSON)},
			},
		}, nil
	})

	var recordedUserID string
	lc := &MockLimitChecker{
		RecordJobParseUsageFunc: func(_ context.Context, userID string) error {
			recordedUserID = userID
			return nil
		},
	}

	svc := NewImportService(aiClient, lc)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Senior Go Developer at Acme Corp...",
		PageURL:  pageURL,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Senior Go Developer", result.Title)
	assert.Equal(t, &companyName, result.CompanyName)
	assert.Equal(t, &source, result.Source)
	assert.Equal(t, &pageURL, result.URL)
	assert.Equal(t, &description, result.Description)
	assert.Equal(t, "user-123", recordedUserID)
}

func TestImportService_ParseJobPage_AICallError(t *testing.T) {
	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return nil, errors.New("API rate limit exceeded")
	})

	svc := NewImportService(aiClient, nil)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Some job posting text...",
		PageURL:  "https://example.com/job/123",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrParsingFailed)
}

func TestImportService_ParseJobPage_AIReturnsEmptyResponse(t *testing.T) {
	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{},
		}, nil
	})

	svc := NewImportService(aiClient, nil)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Some job posting text...",
		PageURL:  "https://example.com/job/123",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrParsingFailed)
}

func TestImportService_ParseJobPage_AIReturnsInvalidJSON(t *testing.T) {
	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: "this is not json"},
			},
		}, nil
	})

	svc := NewImportService(aiClient, nil)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Some job posting text...",
		PageURL:  "https://example.com/job/123",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrParsingFailed)
}

func TestImportService_ParseJobPage_AIReturnsEmptyTitle(t *testing.T) {
	parsedJSON, _ := json.Marshal(ai.ParsedJob{
		Title: "",
	})

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(parsedJSON)},
			},
		}, nil
	})

	svc := NewImportService(aiClient, nil)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Random page with no job...",
		PageURL:  "https://example.com",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrParsingFailed)
}

func TestImportService_ParseJobPage_RecordUsageFailure_NonFatal(t *testing.T) {
	parsedJSON, _ := json.Marshal(ai.ParsedJob{
		Title: "Software Engineer",
	})

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(parsedJSON)},
			},
		}, nil
	})

	lc := &MockLimitChecker{
		RecordJobParseUsageFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("redis unavailable")
		},
	}

	svc := NewImportService(aiClient, lc)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Software Engineer role...",
		PageURL:  "https://example.com/job/456",
	})

	// RecordJobParseUsage failure should NOT cause the overall call to fail
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Software Engineer", result.Title)
}

func TestImportService_ParseJobPage_SuccessWithNilLimitChecker(t *testing.T) {
	pageURL := "https://example.com/job/789"
	parsedJSON, _ := json.Marshal(ai.ParsedJob{
		Title: "Backend Developer",
		URL:   &pageURL,
	})

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(parsedJSON)},
			},
		}, nil
	})

	svc := NewImportService(aiClient, nil)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "Backend Developer role...",
		PageURL:  pageURL,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Backend Developer", result.Title)
}

func TestImportService_ParseJobPage_URLFallback(t *testing.T) {
	// When AI doesn't return a URL, ParseJobPage should use the provided pageURL
	parsedJSON, _ := json.Marshal(ai.ParsedJob{
		Title: "DevOps Engineer",
		// URL is nil
	})

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(parsedJSON)},
			},
		}, nil
	})

	pageURL := "https://example.com/devops"
	svc := NewImportService(aiClient, nil)
	result, err := svc.ParseJobPage(context.Background(), "user-123", &model.ParseJobRequest{
		PageText: "DevOps Engineer needed...",
		PageURL:  pageURL,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "DevOps Engineer", result.Title)
	// The URL from AI client's ParseJobPage will be set to pageURL as fallback
	// The response maps parsed.URL -> result.URL
	require.NotNil(t, result.URL)
	assert.Equal(t, pageURL, *result.URL)
}
