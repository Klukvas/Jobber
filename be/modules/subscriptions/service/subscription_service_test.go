package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type MockSubscriptionRepository struct {
	GetByUserIDFunc               func(ctx context.Context, userID string) (*model.Subscription, error)
	GetByPaddleSubscriptionIDFunc func(ctx context.Context, paddleSubID string) (*model.Subscription, error)
	UpsertFunc                    func(ctx context.Context, sub *model.Subscription) error
	CountUserJobsFunc             func(ctx context.Context, userID string) (int, error)
	CountUserResumesFunc          func(ctx context.Context, userID string) (int, error)
	CountUserApplicationsFunc     func(ctx context.Context, userID string) (int, error)
	CountUserAIRequestsFunc       func(ctx context.Context, userID string) (int, error)
	CountUserJobParsesFunc        func(ctx context.Context, userID string) (int, error)
	RecordAIUsageFunc             func(ctx context.Context, userID string) error
	RecordJobParseUsageFunc       func(ctx context.Context, userID string) error
	CountUserResumeBuildersFunc   func(ctx context.Context, userID string) (int, error)
	CountUserCoverLettersFunc     func(ctx context.Context, userID string) (int, error)
	GetAllCountsFunc              func(ctx context.Context, userID string) (int, int, int, int, int, int, int, error)
	WebhookEventExistsFunc        func(ctx context.Context, eventID string) (bool, error)
	RecordWebhookEventFunc        func(ctx context.Context, eventID, eventType string) error
	TryClaimWebhookEventFunc      func(ctx context.Context, eventID, eventType string) (bool, error)
}

func (m *MockSubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) GetByPaddleSubscriptionID(ctx context.Context, paddleSubID string) (*model.Subscription, error) {
	if m.GetByPaddleSubscriptionIDFunc != nil {
		return m.GetByPaddleSubscriptionIDFunc(ctx, paddleSubID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) Upsert(ctx context.Context, sub *model.Subscription) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, sub)
	}
	return nil
}

func (m *MockSubscriptionRepository) CountUserJobs(ctx context.Context, userID string) (int, error) {
	if m.CountUserJobsFunc != nil {
		return m.CountUserJobsFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserResumes(ctx context.Context, userID string) (int, error) {
	if m.CountUserResumesFunc != nil {
		return m.CountUserResumesFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserApplications(ctx context.Context, userID string) (int, error) {
	if m.CountUserApplicationsFunc != nil {
		return m.CountUserApplicationsFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserAIRequestsThisMonth(ctx context.Context, userID string) (int, error) {
	if m.CountUserAIRequestsFunc != nil {
		return m.CountUserAIRequestsFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserJobParsesThisMonth(ctx context.Context, userID string) (int, error) {
	if m.CountUserJobParsesFunc != nil {
		return m.CountUserJobParsesFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) RecordAIUsage(ctx context.Context, userID string) error {
	if m.RecordAIUsageFunc != nil {
		return m.RecordAIUsageFunc(ctx, userID)
	}
	return nil
}

func (m *MockSubscriptionRepository) RecordJobParseUsage(ctx context.Context, userID string) error {
	if m.RecordJobParseUsageFunc != nil {
		return m.RecordJobParseUsageFunc(ctx, userID)
	}
	return nil
}

func (m *MockSubscriptionRepository) CountUserResumeBuilders(ctx context.Context, userID string) (int, error) {
	if m.CountUserResumeBuildersFunc != nil {
		return m.CountUserResumeBuildersFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserCoverLetters(ctx context.Context, userID string) (int, error) {
	if m.CountUserCoverLettersFunc != nil {
		return m.CountUserCoverLettersFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) GetAllCounts(ctx context.Context, userID string) (int, int, int, int, int, int, int, error) {
	if m.GetAllCountsFunc != nil {
		return m.GetAllCountsFunc(ctx, userID)
	}
	return 0, 0, 0, 0, 0, 0, 0, nil
}

func (m *MockSubscriptionRepository) WebhookEventExists(ctx context.Context, eventID string) (bool, error) {
	if m.WebhookEventExistsFunc != nil {
		return m.WebhookEventExistsFunc(ctx, eventID)
	}
	return false, nil
}

func (m *MockSubscriptionRepository) RecordWebhookEvent(ctx context.Context, eventID, eventType string) error {
	if m.RecordWebhookEventFunc != nil {
		return m.RecordWebhookEventFunc(ctx, eventID, eventType)
	}
	return nil
}

func (m *MockSubscriptionRepository) TryClaimWebhookEvent(ctx context.Context, eventID, eventType string) (bool, error) {
	if m.TryClaimWebhookEventFunc != nil {
		return m.TryClaimWebhookEventFunc(ctx, eventID, eventType)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const (
	testWebhookSecret     = "test-webhook-secret"
	testProPriceID        = "pri_pro_123"
	testEnterprisePriceID = "pri_ent_456"
	testUserID            = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
)

func newTestService(repo *MockSubscriptionRepository) *SubscriptionService {
	return NewSubscriptionService(
		repo,
		testWebhookSecret,
		"paddle-api-key",
		testProPriceID,
		testEnterprisePriceID,
		"client-token",
		"sandbox",
	)
}

// signPayload produces a valid Paddle-style "ts=<unix>;h1=<hmac>" signature.
func signPayload(payload []byte, secret string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + ":" + string(payload)))
	h := hex.EncodeToString(mac.Sum(nil))
	return "ts=" + ts + ";h1=" + h
}

// buildWebhookBody creates a JSON-encoded paddleEvent for testing.
func buildWebhookBody(eventID, eventType string, data interface{}) []byte {
	raw, _ := json.Marshal(data)
	evt := map[string]interface{}{
		"event_id":   eventID,
		"event_type": eventType,
		"data":       json.RawMessage(raw),
	}
	b, _ := json.Marshal(evt)
	return b
}

// ---------------------------------------------------------------------------
// HandleWebhook tests
// ---------------------------------------------------------------------------

func TestHandleWebhook(t *testing.T) {
	validSubData := map[string]interface{}{
		"id":          "sub_123",
		"status":      "active",
		"customer_id": "ctm_456",
		"custom_data": map[string]string{"user_id": testUserID},
		"items":       []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
		"current_billing_period": map[string]string{
			"starts_at": "2025-01-01T00:00:00Z",
			"ends_at":   "2025-02-01T00:00:00Z",
		},
	}

	tests := []struct {
		name        string
		eventID     string
		eventType   string
		data        interface{}
		setupRepo   func(repo *MockSubscriptionRepository)
		signature   func(body []byte) string
		wantErr     bool
		errContains string
	}{
		{
			name:      "rejects invalid signature",
			eventID:   "evt_1",
			eventType: "subscription.created",
			data:      validSubData,
			signature: func(_ []byte) string {
				return "ts=9999999999;h1=badhash"
			},
			wantErr:     true,
			errContains: "invalid webhook signature",
		},
		{
			name:      "skips already-claimed event (idempotency)",
			eventID:   "evt_dup",
			eventType: "subscription.created",
			data:      validSubData,
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.TryClaimWebhookEventFunc = func(_ context.Context, _, _ string) (bool, error) {
					return false, nil // already claimed
				}
			},
			signature: func(body []byte) string { return signPayload(body, testWebhookSecret) },
			wantErr:   false,
		},
		{
			name:      "processes subscription.created correctly",
			eventID:   "evt_create",
			eventType: "subscription.created",
			data:      validSubData,
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.UpsertFunc = func(_ context.Context, sub *model.Subscription) error {
					assert.Equal(t, testUserID, sub.UserID)
					assert.Equal(t, "pro", sub.Plan)
					assert.Equal(t, "active", sub.Status)
					assert.NotNil(t, sub.PaddleSubscriptionID)
					assert.Equal(t, "sub_123", *sub.PaddleSubscriptionID)
					return nil
				}
			},
			signature: func(body []byte) string { return signPayload(body, testWebhookSecret) },
			wantErr:   false,
		},
		{
			name:      "processes subscription.canceled correctly",
			eventID:   "evt_cancel",
			eventType: "subscription.canceled",
			data: map[string]interface{}{
				"id":          "sub_123",
				"status":      "canceled",
				"customer_id": "ctm_456",
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, paddleSubID string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Status:               "active",
						Plan:                 "pro",
					}, nil
				}
				repo.UpsertFunc = func(_ context.Context, sub *model.Subscription) error {
					assert.Equal(t, "cancelled", sub.Status)
					assert.Equal(t, "free", sub.Plan)
					assert.Nil(t, sub.CancelAt)
					return nil
				}
			},
			signature: func(body []byte) string { return signPayload(body, testWebhookSecret) },
			wantErr:   false,
		},
		{
			name:      "ignores unknown event types",
			eventID:   "evt_unknown",
			eventType: "transaction.completed",
			data:      map[string]interface{}{},
			signature: func(body []byte) string { return signPayload(body, testWebhookSecret) },
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newTestService(repo)

			body := buildWebhookBody(tt.eventID, tt.eventType, tt.data)
			sig := tt.signature(body)

			err := svc.HandleWebhook(context.Background(), body, sig)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// determinePlanFromEvent tests
// ---------------------------------------------------------------------------

func TestDeterminePlanFromEvent(t *testing.T) {
	svc := newTestService(&MockSubscriptionRepository{})

	tests := []struct {
		name     string
		items    []struct{ PriceID string }
		wantPlan string
		wantErr  bool
	}{
		{
			name:     "returns pro for pro price ID",
			items:    []struct{ PriceID string }{{testProPriceID}},
			wantPlan: "pro",
		},
		{
			name:     "returns enterprise for enterprise price ID",
			items:    []struct{ PriceID string }{{testEnterprisePriceID}},
			wantPlan: "enterprise",
		},
		{
			name:    "errors on unknown price ID",
			items:   []struct{ PriceID string }{{"pri_unknown_999"}},
			wantErr: true,
		},
		{
			name:    "errors when items are empty",
			items:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &paddleSubscriptionData{}
			for _, item := range tt.items {
				data.Items = append(data.Items, struct {
					Price struct {
						ID string `json:"id"`
					} `json:"price"`
				}{
					Price: struct {
						ID string `json:"id"`
					}{ID: item.PriceID},
				})
			}

			plan, err := svc.determinePlanFromEvent(data)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPlan, plan)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// mapPaddleStatus tests
// ---------------------------------------------------------------------------

func TestMapPaddleStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"active", "active"},
		{"past_due", "past_due"},
		{"canceled", "cancelled"},
		{"paused", "paused"},
		{"trialing", "trialing"}, // unknown passes through
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, mapPaddleStatus(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// parseUUID tests
// ---------------------------------------------------------------------------

func TestParseUUID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase UUID", "a1b2c3d4-e5f6-7890-abcd-ef1234567890", false},
		{"valid uppercase UUID", "A1B2C3D4-E5F6-7890-ABCD-EF1234567890", false},
		{"too short", "abc-def", true},
		{"wrong length", "a1b2c3d4-e5f6-7890-abcd-ef12345678", true},
		{"missing dashes", "a1b2c3d4e5f67890abcdef1234567890xxxx", true},
		{"invalid character", "g1b2c3d4-e5f6-7890-abcd-ef1234567890", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseUUID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.input, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// verifyWebhookSignature tests
// ---------------------------------------------------------------------------

func TestVerifyWebhookSignature(t *testing.T) {
	svc := newTestService(&MockSubscriptionRepository{})
	payload := []byte(`{"event_type":"test"}`)

	tests := []struct {
		name        string
		signature   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid signature",
			signature: signPayload(payload, testWebhookSecret),
			wantErr:   false,
		},
		{
			name: "rejects expired timestamp",
			signature: func() string {
				oldTs := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
				mac := hmac.New(sha256.New, []byte(testWebhookSecret))
				mac.Write([]byte(oldTs + ":" + string(payload)))
				h := hex.EncodeToString(mac.Sum(nil))
				return "ts=" + oldTs + ";h1=" + h
			}(),
			wantErr:     true,
			errContains: "timestamp too old",
		},
		{
			name:        "rejects wrong hash",
			signature:   fmt.Sprintf("ts=%d;h1=deadbeef", time.Now().Unix()),
			wantErr:     true,
			errContains: "signature mismatch",
		},
		{
			name:        "rejects invalid format",
			signature:   "garbage",
			wantErr:     true,
			errContains: "invalid signature format",
		},
		{
			name:        "rejects missing hash",
			signature:   fmt.Sprintf("ts=%d;xx=abc", time.Now().Unix()),
			wantErr:     true,
			errContains: "missing timestamp or hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.verifyWebhookSignature(payload, tt.signature)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CheckLimit tests
// ---------------------------------------------------------------------------

func TestCheckLimit(t *testing.T) {
	tests := []struct {
		name      string
		resource  string
		plan      string
		current   int
		max       int
		setupRepo func(repo *MockSubscriptionRepository)
		wantErr   error
	}{
		{
			name:     "returns nil when under job limit (free plan)",
			resource: "jobs",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserJobsFunc = func(_ context.Context, _ string) (int, error) {
					return 2, nil // under free limit of 5
				}
			},
			wantErr: nil,
		},
		{
			name:     "returns ErrLimitReached when at job limit (free plan)",
			resource: "jobs",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserJobsFunc = func(_ context.Context, _ string) (int, error) {
					return 5, nil // at free limit of 5
				}
			},
			wantErr: model.ErrLimitReached,
		},
		{
			name:     "returns nil for unlimited resource (pro plan, AI requests)",
			resource: "ai",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "pro", Status: "active"}, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "returns nil for resumes under limit (free plan)",
			resource: "resumes",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserResumesFunc = func(_ context.Context, _ string) (int, error) {
					return 0, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "returns ErrLimitReached for resumes at limit (free plan)",
			resource: "resumes",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserResumesFunc = func(_ context.Context, _ string) (int, error) {
					return 1, nil // free limit is 1
				}
			},
			wantErr: model.ErrLimitReached,
		},
		{
			name:     "falls back to free plan when no subscription found",
			resource: "jobs",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return nil, model.ErrSubscriptionNotFound
				}
				repo.CountUserJobsFunc = func(_ context.Context, _ string) (int, error) {
					return 0, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "returns nil for unknown resource type",
			resource: "widgets",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "returns nil for enterprise plan (all unlimited)",
			resource: "jobs",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "enterprise", Status: "active"}, nil
				}
			},
			wantErr: nil,
		},
		{
			name:     "returns ErrLimitReached for applications at limit (free plan)",
			resource: "applications",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserApplicationsFunc = func(_ context.Context, _ string) (int, error) {
					return 5, nil
				}
			},
			wantErr: model.ErrLimitReached,
		},
		{
			name:     "returns ErrLimitReached for job_parses at limit (free plan)",
			resource: "job_parses",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserJobParsesFunc = func(_ context.Context, _ string) (int, error) {
					return 5, nil
				}
			},
			wantErr: model.ErrLimitReached,
		},
		{
			name:     "returns ErrLimitReached for resume_builders at zero-max plan",
			resource: "resume_builders",
			setupRepo: func(repo *MockSubscriptionRepository) {
				// Create a plan that has 0 max for resume builders
				// Free plan has MaxResumeBuilders = 1, so at limit with 1
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserResumeBuildersFunc = func(_ context.Context, _ string) (int, error) {
					return 1, nil
				}
			},
			wantErr: model.ErrLimitReached,
		},
		{
			name:     "returns ErrLimitReached for cover_letters at limit (free plan)",
			resource: "cover_letters",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserCoverLettersFunc = func(_ context.Context, _ string) (int, error) {
					return 1, nil
				}
			},
			wantErr: model.ErrLimitReached,
		},
		{
			name:     "returns error when repo returns non-subscription error",
			resource: "jobs",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return nil, errors.New("database connection failed")
				}
			},
			wantErr: errors.New("failed to get subscription"),
		},
		{
			name:     "returns error when count fails",
			resource: "jobs",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.CountUserJobsFunc = func(_ context.Context, _ string) (int, error) {
					return 0, errors.New("count error")
				}
			},
			wantErr: errors.New("failed to count jobs"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newTestService(repo)

			err := svc.CheckLimit(context.Background(), testUserID, tt.resource)
			if tt.wantErr != nil {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetSubscription tests
// ---------------------------------------------------------------------------

func TestGetSubscription(t *testing.T) {
	tests := []struct {
		name      string
		setupRepo func(repo *MockSubscriptionRepository)
		wantErr   bool
		validate  func(t *testing.T, dto *model.SubscriptionDTO)
	}{
		{
			name: "returns subscription with usage",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, uid string) (*model.Subscription, error) {
					assert.Equal(t, testUserID, uid)
					return &model.Subscription{
						UserID: testUserID,
						Plan:   "pro",
						Status: "active",
					}, nil
				}
				repo.GetAllCountsFunc = func(_ context.Context, uid string) (int, int, int, int, int, int, int, error) {
					return 3, 2, 1, 10, 5, 1, 0, nil
				}
			},
			validate: func(t *testing.T, dto *model.SubscriptionDTO) {
				assert.Equal(t, "pro", dto.Plan)
				assert.Equal(t, "active", dto.Status)
				assert.Equal(t, 3, dto.Usage.Jobs)
				assert.Equal(t, 2, dto.Usage.Resumes)
				assert.Equal(t, 1, dto.Usage.Applications)
				assert.Equal(t, 10, dto.Usage.AIRequests)
				assert.Equal(t, 5, dto.Usage.JobParses)
				assert.Equal(t, 1, dto.Usage.ResumeBuilders)
				assert.Equal(t, 0, dto.Usage.CoverLetters)
			},
		},
		{
			name: "returns error when subscription not found",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return nil, model.ErrSubscriptionNotFound
				}
			},
			wantErr: true,
		},
		{
			name: "returns error when usage query fails",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: "free", Status: "free"}, nil
				}
				repo.GetAllCountsFunc = func(_ context.Context, _ string) (int, int, int, int, int, int, int, error) {
					return 0, 0, 0, 0, 0, 0, 0, errors.New("count error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newTestService(repo)

			result, err := svc.GetSubscription(context.Background(), testUserID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetCheckoutConfig tests
// ---------------------------------------------------------------------------

func TestGetCheckoutConfig(t *testing.T) {
	t.Run("returns config with both prices", func(t *testing.T) {
		svc := newTestService(&MockSubscriptionRepository{})
		config := svc.GetCheckoutConfig()

		assert.Equal(t, "client-token", config.ClientToken)
		assert.Equal(t, "sandbox", config.Environment)
		assert.Equal(t, testProPriceID, config.Prices["pro"])
		assert.Equal(t, testEnterprisePriceID, config.Prices["enterprise"])
	})

	t.Run("returns config without enterprise when not configured", func(t *testing.T) {
		svc := NewSubscriptionService(
			&MockSubscriptionRepository{},
			testWebhookSecret,
			"api-key",
			testProPriceID,
			"", // no enterprise price
			"client-token",
			"sandbox",
		)
		config := svc.GetCheckoutConfig()

		assert.Equal(t, testProPriceID, config.Prices["pro"])
		_, hasEnterprise := config.Prices["enterprise"]
		assert.False(t, hasEnterprise)
	})
}

// ---------------------------------------------------------------------------
// EnsureFreeSubscription tests
// ---------------------------------------------------------------------------

func TestEnsureFreeSubscription(t *testing.T) {
	t.Run("upserts free subscription", func(t *testing.T) {
		var upsertedSub *model.Subscription
		repo := &MockSubscriptionRepository{
			UpsertFunc: func(_ context.Context, sub *model.Subscription) error {
				upsertedSub = sub
				return nil
			},
		}
		svc := newTestService(repo)

		err := svc.EnsureFreeSubscription(context.Background(), testUserID)

		require.NoError(t, err)
		require.NotNil(t, upsertedSub)
		assert.Equal(t, testUserID, upsertedSub.UserID)
		assert.Equal(t, "free", upsertedSub.Plan)
		assert.Equal(t, "free", upsertedSub.Status)
	})

	t.Run("returns error when upsert fails", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			UpsertFunc: func(_ context.Context, _ *model.Subscription) error {
				return errors.New("upsert failed")
			},
		}
		svc := newTestService(repo)

		err := svc.EnsureFreeSubscription(context.Background(), testUserID)

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RecordAIUsage tests
// ---------------------------------------------------------------------------

func TestRecordAIUsage(t *testing.T) {
	t.Run("delegates to repository", func(t *testing.T) {
		var recordedUserID string
		repo := &MockSubscriptionRepository{
			RecordAIUsageFunc: func(_ context.Context, uid string) error {
				recordedUserID = uid
				return nil
			},
		}
		svc := newTestService(repo)

		err := svc.RecordAIUsage(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, testUserID, recordedUserID)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			RecordAIUsageFunc: func(_ context.Context, _ string) error {
				return errors.New("record failed")
			},
		}
		svc := newTestService(repo)

		err := svc.RecordAIUsage(context.Background(), testUserID)

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RecordJobParseUsage tests
// ---------------------------------------------------------------------------

func TestRecordJobParseUsage(t *testing.T) {
	t.Run("delegates to repository", func(t *testing.T) {
		var recordedUserID string
		repo := &MockSubscriptionRepository{
			RecordJobParseUsageFunc: func(_ context.Context, uid string) error {
				recordedUserID = uid
				return nil
			},
		}
		svc := newTestService(repo)

		err := svc.RecordJobParseUsage(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, testUserID, recordedUserID)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			RecordJobParseUsageFunc: func(_ context.Context, _ string) error {
				return errors.New("record failed")
			},
		}
		svc := newTestService(repo)

		err := svc.RecordJobParseUsage(context.Background(), testUserID)

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Helper to create service pointing at a test HTTP server
// ---------------------------------------------------------------------------

func newTestServiceWithHTTP(repo *MockSubscriptionRepository, serverURL string) *SubscriptionService {
	svc := newTestService(repo)
	svc.httpClient = &http.Client{Timeout: 5 * time.Second}
	// Override the environment so paddleBaseURL() returns our test server URL.
	// We override the method by setting the breaker to a no-fail breaker and
	// directly manipulating the httpClient to point to the test server.
	// Actually, we need to override paddleBaseURL. Since it checks environment,
	// and we use "sandbox", we need to set httpClient to redirect to test server.
	// The cleaner approach: override the httpClient with a custom transport.
	svc.httpClient = &http.Client{
		Transport: &testTransport{baseURL: serverURL},
		Timeout:   5 * time.Second,
	}
	return svc
}

// testTransport rewrites requests to point at the test HTTP server.
type testTransport struct {
	baseURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the Paddle API URL with the test server URL
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[len("http://"):]
	return http.DefaultTransport.RoundTrip(req)
}

// ---------------------------------------------------------------------------
// ChangePlan tests
// ---------------------------------------------------------------------------

func TestChangePlan(t *testing.T) {
	paddleSubID := "sub_123"

	tests := []struct {
		name           string
		newPlan        string
		setupRepo      func(repo *MockSubscriptionRepository)
		serverHandler  http.HandlerFunc
		wantErr        bool
		errContains    string
	}{
		{
			name:    "changes plan successfully",
			newPlan: "pro",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Plan:                 "enterprise",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Contains(t, r.URL.Path, "/subscriptions/sub_123")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{}}`))
			},
		},
		{
			name:    "returns error for no paddle subscription",
			newPlan: "pro",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID: testUserID,
						Plan:   "free",
						Status: "free",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "no active paddle subscription found",
		},
		{
			name:    "returns error for invalid plan",
			newPlan: "invalid",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "invalid plan",
		},
		{
			name:    "returns error on paddle API error",
			newPlan: "pro",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Plan:                 "enterprise",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad request"}`))
			},
			wantErr:     true,
			errContains: "paddle API error",
		},
		{
			name:    "returns error when subscription lookup fails",
			newPlan: "pro",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return nil, model.ErrSubscriptionNotFound
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			var svc *SubscriptionService
			if tt.serverHandler != nil {
				server := httptest.NewServer(tt.serverHandler)
				defer server.Close()
				svc = newTestServiceWithHTTP(repo, server.URL)
			} else {
				svc = newTestService(repo)
			}

			err := svc.ChangePlan(context.Background(), testUserID, tt.newPlan)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CancelSubscription tests
// ---------------------------------------------------------------------------

func TestCancelSubscription(t *testing.T) {
	paddleSubID := "sub_123"

	tests := []struct {
		name          string
		setupRepo     func(repo *MockSubscriptionRepository)
		serverHandler http.HandlerFunc
		wantErr       bool
		errContains   string
	}{
		{
			name: "cancels subscription successfully",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/subscriptions/sub_123/cancel")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{}}`))
			},
		},
		{
			name: "returns error for no paddle subscription",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID: testUserID,
						Plan:   "free",
						Status: "free",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "no active paddle subscription found",
		},
		{
			name: "returns error on paddle API error",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"server error"}`))
			},
			wantErr:     true,
			errContains: "failed to call Paddle API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			var svc *SubscriptionService
			if tt.serverHandler != nil {
				server := httptest.NewServer(tt.serverHandler)
				defer server.Close()
				svc = newTestServiceWithHTTP(repo, server.URL)
			} else {
				svc = newTestService(repo)
			}

			err := svc.CancelSubscription(context.Background(), testUserID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CreatePortalSession tests
// ---------------------------------------------------------------------------

func TestCreatePortalSession(t *testing.T) {
	paddleSubID := "sub_123"
	paddleCustID := "ctm_456"

	tests := []struct {
		name          string
		setupRepo     func(repo *MockSubscriptionRepository)
		serverHandler http.HandlerFunc
		wantErr       bool
		errContains   string
		wantURL       string
	}{
		{
			name: "creates portal session successfully",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						PaddleCustomerID:     &paddleCustID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/customers/ctm_456/portal-sessions")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"urls":{"general":{"overview":"https://portal.paddle.com/session/123"}}}}`))
			},
			wantURL: "https://portal.paddle.com/session/123",
		},
		{
			name: "returns error for no paddle subscription",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID: testUserID,
						Plan:   "free",
						Status: "free",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "no paddle subscription found",
		},
		{
			name: "returns error for no customer ID",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			wantErr:     true,
			errContains: "no paddle customer ID found",
		},
		{
			name: "returns error when portal URL is empty",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						PaddleCustomerID:     &paddleCustID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":{"urls":{"general":{"overview":""}}}}`))
			},
			wantErr:     true,
			errContains: "no portal URL returned",
		},
		{
			name: "returns error on paddle API error",
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleSubID,
						PaddleCustomerID:     &paddleCustID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
			},
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad request"}`))
			},
			wantErr:     true,
			errContains: "paddle API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			var svc *SubscriptionService
			if tt.serverHandler != nil {
				server := httptest.NewServer(tt.serverHandler)
				defer server.Close()
				svc = newTestServiceWithHTTP(repo, server.URL)
			} else {
				svc = newTestService(repo)
			}

			url, err := svc.CreatePortalSession(context.Background(), testUserID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantURL, url)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handleSubscriptionUpdated tests
// ---------------------------------------------------------------------------

func TestHandleSubscriptionUpdated(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		setupRepo   func(repo *MockSubscriptionRepository)
		wantErr     bool
		errContains string
		validate    func(t *testing.T, sub *model.Subscription)
	}{
		{
			name: "updates subscription with billing period",
			data: map[string]interface{}{
				"id":          "sub_123",
				"status":      "active",
				"customer_id": "ctm_456",
				"items":       []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
				"current_billing_period": map[string]string{
					"starts_at": "2025-03-01T00:00:00Z",
					"ends_at":   "2025-04-01T00:00:00Z",
				},
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, paddleID string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleID,
						Plan:                 "free",
						Status:               "free",
					}, nil
				}
				repo.UpsertFunc = func(_ context.Context, sub *model.Subscription) error {
					assert.Equal(t, "active", sub.Status)
					assert.Equal(t, "pro", sub.Plan)
					assert.NotNil(t, sub.CurrentPeriodStart)
					assert.NotNil(t, sub.CurrentPeriodEnd)
					assert.Nil(t, sub.CancelAt)
					return nil
				}
			},
		},
		{
			name: "updates subscription with scheduled cancellation",
			data: map[string]interface{}{
				"id":          "sub_123",
				"status":      "active",
				"customer_id": "ctm_456",
				"items":       []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
				"scheduled_change": map[string]interface{}{
					"action":       "cancel",
					"effective_at": "2025-05-01T00:00:00Z",
				},
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, paddleID string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
				repo.UpsertFunc = func(_ context.Context, sub *model.Subscription) error {
					assert.NotNil(t, sub.CancelAt)
					return nil
				}
			},
		},
		{
			name: "clears cancel_at when no scheduled change",
			data: map[string]interface{}{
				"id":          "sub_123",
				"status":      "active",
				"customer_id": "ctm_456",
				"items":       []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				cancelAt := time.Now().Add(30 * 24 * time.Hour)
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, paddleID string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleID,
						Plan:                 "pro",
						Status:               "active",
						CancelAt:             &cancelAt,
					}, nil
				}
				repo.UpsertFunc = func(_ context.Context, sub *model.Subscription) error {
					assert.Nil(t, sub.CancelAt, "CancelAt should be cleared")
					return nil
				}
			},
		},
		{
			name: "returns error when subscription not found",
			data: map[string]interface{}{
				"id":     "sub_unknown",
				"status": "active",
				"items":  []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return nil, model.ErrSubscriptionNotFound
				}
			},
			wantErr:     true,
			errContains: "subscription not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newTestService(repo)

			body := buildWebhookBody("evt_upd", "subscription.updated", tt.data)
			sig := signPayload(body, testWebhookSecret)

			err := svc.HandleWebhook(context.Background(), body, sig)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// handleSubscriptionPastDue tests
// ---------------------------------------------------------------------------

func TestHandleSubscriptionPastDue(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		setupRepo   func(repo *MockSubscriptionRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "sets status to past_due",
			data: map[string]interface{}{
				"id":          "sub_123",
				"status":      "past_due",
				"customer_id": "ctm_456",
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, paddleID string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
				repo.UpsertFunc = func(_ context.Context, sub *model.Subscription) error {
					assert.Equal(t, "past_due", sub.Status)
					return nil
				}
			},
		},
		{
			name: "returns error when subscription not found",
			data: map[string]interface{}{
				"id":     "sub_unknown",
				"status": "past_due",
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, _ string) (*model.Subscription, error) {
					return nil, model.ErrSubscriptionNotFound
				}
			},
			wantErr:     true,
			errContains: "subscription not found",
		},
		{
			name: "returns error when upsert fails",
			data: map[string]interface{}{
				"id":          "sub_123",
				"status":      "past_due",
				"customer_id": "ctm_456",
			},
			setupRepo: func(repo *MockSubscriptionRepository) {
				repo.GetByPaddleSubscriptionIDFunc = func(_ context.Context, paddleID string) (*model.Subscription, error) {
					return &model.Subscription{
						UserID:               testUserID,
						PaddleSubscriptionID: &paddleID,
						Plan:                 "pro",
						Status:               "active",
					}, nil
				}
				repo.UpsertFunc = func(_ context.Context, _ *model.Subscription) error {
					return errors.New("upsert failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{}
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newTestService(repo)

			body := buildWebhookBody("evt_pd", "subscription.past_due", tt.data)
			sig := signPayload(body, testWebhookSecret)

			err := svc.HandleWebhook(context.Background(), body, sig)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HandleWebhook edge cases
// ---------------------------------------------------------------------------

func TestHandleWebhook_ActivatedMissingUserID(t *testing.T) {
	repo := &MockSubscriptionRepository{}
	svc := newTestService(repo)

	data := map[string]interface{}{
		"id":          "sub_123",
		"status":      "active",
		"customer_id": "ctm_456",
		"items":       []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
		// missing custom_data
	}

	body := buildWebhookBody("evt_no_user", "subscription.activated", data)
	sig := signPayload(body, testWebhookSecret)

	err := svc.HandleWebhook(context.Background(), body, sig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing user_id")
}

func TestHandleWebhook_ActivatedInvalidUserID(t *testing.T) {
	repo := &MockSubscriptionRepository{}
	svc := newTestService(repo)

	data := map[string]interface{}{
		"id":          "sub_123",
		"status":      "active",
		"customer_id": "ctm_456",
		"custom_data": map[string]string{"user_id": "not-a-uuid"},
		"items":       []map[string]interface{}{{"price": map[string]string{"id": testProPriceID}}},
	}

	body := buildWebhookBody("evt_bad_uid", "subscription.activated", data)
	sig := signPayload(body, testWebhookSecret)

	err := svc.HandleWebhook(context.Background(), body, sig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user_id")
}

func TestHandleWebhook_ClaimEventError(t *testing.T) {
	repo := &MockSubscriptionRepository{
		TryClaimWebhookEventFunc: func(_ context.Context, _, _ string) (bool, error) {
			return false, errors.New("redis error")
		},
	}
	svc := newTestService(repo)

	data := map[string]interface{}{"id": "sub_123", "status": "active"}
	body := buildWebhookBody("evt_claim_err", "subscription.created", data)
	sig := signPayload(body, testWebhookSecret)

	err := svc.HandleWebhook(context.Background(), body, sig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to claim webhook event")
}

func TestHandleWebhook_EmptyWebhookSecret(t *testing.T) {
	svc := NewSubscriptionService(
		&MockSubscriptionRepository{},
		"", // empty webhook secret
		"api-key",
		testProPriceID,
		testEnterprisePriceID,
		"client-token",
		"sandbox",
	)

	body := []byte(`{"event_type":"test"}`)
	err := svc.HandleWebhook(context.Background(), body, "ts=123;h1=abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook secret is not configured")
}

// ---------------------------------------------------------------------------
// paddleBaseURL tests
// ---------------------------------------------------------------------------

func TestPaddleBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		wantURL     string
	}{
		{"sandbox returns sandbox URL", "sandbox", "https://sandbox-api.paddle.com"},
		{"production returns production URL", "production", "https://api.paddle.com"},
		{"empty returns production URL", "", "https://api.paddle.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				&MockSubscriptionRepository{},
				testWebhookSecret,
				"api-key",
				testProPriceID,
				testEnterprisePriceID,
				"client-token",
				tt.environment,
			)
			assert.Equal(t, tt.wantURL, svc.paddleBaseURL())
		})
	}
}
