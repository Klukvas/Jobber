package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockSubscriptionRepository struct {
	GetByUserIDFunc                 func(ctx context.Context, userID string) (*model.Subscription, error)
	GetByExternalSubscriptionIDFunc func(ctx context.Context, externalSubID string) (*model.Subscription, error)
	GetByExternalAccountIDFunc      func(ctx context.Context, externalAccountID string) (*model.Subscription, error)
	EnsureFreeFunc                  func(ctx context.Context, userID string) error
	LinkExternalAccountFunc         func(ctx context.Context, userID, externalAccountID string) error
	GetUserContactFunc              func(ctx context.Context, userID string) (*model.UserContact, error)
	CountUserJobsFunc               func(ctx context.Context, userID string) (int, error)
	CountUserResumesFunc            func(ctx context.Context, userID string) (int, error)
	CountUserAIRequestsFunc         func(ctx context.Context, userID string) (int, error)
	CountUserJobParsesFunc          func(ctx context.Context, userID string) (int, error)
	RecordAIUsageFunc               func(ctx context.Context, userID string) error
	RecordJobParseUsageFunc         func(ctx context.Context, userID string) error
	RecordResumeAutofillUsageFunc   func(ctx context.Context, userID string) error
	CountUserResumeBuildersFunc     func(ctx context.Context, userID string) (int, error)
	CountUserCoverLettersFunc       func(ctx context.Context, userID string) (int, error)
	GetAllCountsFunc                func(ctx context.Context, userID string) (int, int, int, int, int, int, error)
	ApplySubscriptionEventFunc      func(ctx context.Context, eventID, eventType string, sub *model.Subscription) (model.WebhookApplyOutcome, error)
}

func (m *MockSubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) GetByExternalSubscriptionID(ctx context.Context, externalSubID string) (*model.Subscription, error) {
	if m.GetByExternalSubscriptionIDFunc != nil {
		return m.GetByExternalSubscriptionIDFunc(ctx, externalSubID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) GetByExternalAccountID(ctx context.Context, externalAccountID string) (*model.Subscription, error) {
	if m.GetByExternalAccountIDFunc != nil {
		return m.GetByExternalAccountIDFunc(ctx, externalAccountID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) EnsureFree(ctx context.Context, userID string) error {
	if m.EnsureFreeFunc != nil {
		return m.EnsureFreeFunc(ctx, userID)
	}
	return nil
}

func (m *MockSubscriptionRepository) LinkExternalAccount(ctx context.Context, userID, externalAccountID string) error {
	if m.LinkExternalAccountFunc != nil {
		return m.LinkExternalAccountFunc(ctx, userID, externalAccountID)
	}
	return nil
}

func (m *MockSubscriptionRepository) GetUserContact(ctx context.Context, userID string) (*model.UserContact, error) {
	if m.GetUserContactFunc != nil {
		return m.GetUserContactFunc(ctx, userID)
	}
	return &model.UserContact{Email: "buyer@example.com", Name: "Test Buyer", Locale: "en"}, nil
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

func (m *MockSubscriptionRepository) RecordResumeAutofillUsage(ctx context.Context, userID string) error {
	if m.RecordResumeAutofillUsageFunc != nil {
		return m.RecordResumeAutofillUsageFunc(ctx, userID)
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

func (m *MockSubscriptionRepository) GetAllCounts(ctx context.Context, userID string) (int, int, int, int, int, int, error) {
	if m.GetAllCountsFunc != nil {
		return m.GetAllCountsFunc(ctx, userID)
	}
	return 0, 0, 0, 0, 0, 0, nil
}

func (m *MockSubscriptionRepository) ApplySubscriptionEvent(
	ctx context.Context, eventID, eventType string, sub *model.Subscription,
) (model.WebhookApplyOutcome, error) {
	if m.ApplySubscriptionEventFunc != nil {
		return m.ApplySubscriptionEventFunc(ctx, eventID, eventType, sub)
	}
	return model.WebhookApplied, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Compile-time proof the mock still satisfies the port.
var _ ports.SubscriptionRepository = (*MockSubscriptionRepository)(nil)

const (
	testUserID                = "550e8400-e29b-41d4-a716-446655440000"
	testProProductPath        = "jobber-pro"
	testEnterpriseProductPath = "jobber-enterprise"
	testCheckoutPath          = "fluxlab/popup-jobber"
	testTestStorefront        = "fluxlab.test.onfastspring.com/popup-jobber"
	testLiveStorefront        = "fluxlab.onfastspring.com/popup-jobber"
	testWebhookSecret         = "test-webhook-secret"
)

// testBillingConfig is the default billing setup used by the service tests:
// test mode, both plans purchasable.
func testBillingConfig() BillingConfig {
	return BillingConfig{
		WebhookSecret:         testWebhookSecret,
		CheckoutPath:          testCheckoutPath,
		Environment:           EnvironmentTest,
		ProProductPath:        testProProductPath,
		EnterpriseProductPath: testEnterpriseProductPath,
	}
}

func newTestService(repo *MockSubscriptionRepository) *SubscriptionService {
	return NewSubscriptionService(repo, fastspring.NewClient(fastspring.Config{
		Username: "test-user",
		Password: "test-pass",
	}), testBillingConfig())
}

func TestRequirePaidPlan(t *testing.T) {
	tests := []struct {
		name    string
		sub     *model.Subscription
		subErr  error
		wantErr error
	}{
		{
			name:    "free plan is rejected",
			sub:     &model.Subscription{Plan: "free", Status: "free"},
			wantErr: model.ErrPaidFeature,
		},
		{
			name: "active pro passes",
			sub:  &model.Subscription{Plan: "pro", Status: "active"},
		},
		{
			name: "past_due pro keeps the grace window",
			sub:  &model.Subscription{Plan: "pro", Status: "past_due"},
		},
		{
			name:    "paused pro falls back to free",
			sub:     &model.Subscription{Plan: "pro", Status: "paused"},
			wantErr: model.ErrPaidFeature,
		},
		{
			name:    "cancelled enterprise falls back to free",
			sub:     &model.Subscription{Plan: "enterprise", Status: "cancelled"},
			wantErr: model.ErrPaidFeature,
		},
		{
			name: "active enterprise passes",
			sub:  &model.Subscription{Plan: "enterprise", Status: "active"},
		},
		{
			name:    "missing subscription is treated as free",
			subErr:  model.ErrSubscriptionNotFound,
			wantErr: model.ErrPaidFeature,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(ctx context.Context, userID string) (*model.Subscription, error) {
					if tt.subErr != nil {
						return nil, tt.subErr
					}
					return tt.sub, nil
				},
			}
			svc := newTestService(repo)

			err := svc.RequirePaidPlan(context.Background(), "user-1")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("repo failure propagates as error, not as free", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(ctx context.Context, userID string) (*model.Subscription, error) {
				return nil, errors.New("db down")
			},
		}
		svc := newTestService(repo)

		err := svc.RequirePaidPlan(context.Background(), "user-1")

		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrPaidFeature)
	})
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
					return 10, nil // under free limit of 25
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
					return 25, nil // at free limit of 25
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
				repo.GetAllCountsFunc = func(_ context.Context, uid string) (int, int, int, int, int, int, error) {
					return 3, 2, 10, 5, 1, 0, nil
				}
			},
			validate: func(t *testing.T, dto *model.SubscriptionDTO) {
				assert.Equal(t, "pro", dto.Plan)
				assert.Equal(t, "active", dto.Status)
				assert.Equal(t, 3, dto.Usage.Jobs)
				assert.Equal(t, 2, dto.Usage.Resumes)
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
				repo.GetAllCountsFunc = func(_ context.Context, _ string) (int, int, int, int, int, int, error) {
					return 0, 0, 0, 0, 0, 0, errors.New("count error")
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

func TestEnsureFreeSubscription(t *testing.T) {
	t.Run("creates a free row for the user", func(t *testing.T) {
		var ensuredUserID string
		repo := &MockSubscriptionRepository{
			EnsureFreeFunc: func(_ context.Context, userID string) error {
				ensuredUserID = userID
				return nil
			},
		}

		err := newTestService(repo).EnsureFreeSubscription(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, testUserID, ensuredUserID)
	})

	t.Run("never overwrites an existing subscription", func(t *testing.T) {
		// Ensuring a free row must never take the entitlement write path, which
		// would clobber a paying row.
		repo := &MockSubscriptionRepository{
			ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
				t.Fatal("EnsureFreeSubscription must not write entitlement")
				return "", nil
			},
		}

		require.NoError(t, newTestService(repo).EnsureFreeSubscription(context.Background(), testUserID))
	})

	t.Run("returns error when the write fails", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			EnsureFreeFunc: func(_ context.Context, _ string) error {
				return errors.New("insert failed")
			},
		}

		require.Error(t, newTestService(repo).EnsureFreeSubscription(context.Background(), testUserID))
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
// RecordResumeAutofillUsage tests
// ---------------------------------------------------------------------------

func TestRecordResumeAutofillUsage(t *testing.T) {
	t.Run("delegates to repository", func(t *testing.T) {
		var recordedUserID string
		repo := &MockSubscriptionRepository{
			RecordResumeAutofillUsageFunc: func(_ context.Context, uid string) error {
				recordedUserID = uid
				return nil
			},
		}
		svc := newTestService(repo)

		err := svc.RecordResumeAutofillUsage(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, testUserID, recordedUserID)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			RecordResumeAutofillUsageFunc: func(_ context.Context, _ string) error {
				return errors.New("record failed")
			},
		}
		svc := newTestService(repo)

		err := svc.RecordResumeAutofillUsage(context.Background(), testUserID)

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Helper to create service pointing at a test HTTP server
// ---------------------------------------------------------------------------

// ResourceLimit is what makes the ceiling enforceable at the write. CheckLimit
// answers "is there room right now?", which stops being true the moment it
// returns; a caller that counts and writes in one transaction needs the number.
func TestResourceLimit(t *testing.T) {
	tests := []struct {
		name     string
		plan     string
		status   string
		resource string
		want     int
	}{
		{name: "free plan resumes", plan: "free", status: "free", resource: "resumes", want: model.FreePlanLimits.MaxResumes},
		{name: "pro plan resumes", plan: "pro", status: "active", resource: "resumes", want: model.ProPlanLimits.MaxResumes},
		{name: "enterprise is unlimited", plan: "enterprise", status: "active", resource: "resumes", want: -1},
		{name: "jobs", plan: "free", status: "free", resource: "jobs", want: model.FreePlanLimits.MaxJobs},
		{name: "cover letters", plan: "pro", status: "active", resource: "cover_letters", want: model.ProPlanLimits.MaxCoverLetters},
		{name: "an unknown resource is unmetered", plan: "free", status: "free", resource: "widgets", want: -1},
		// A paid plan that has stopped paying falls back to free, exactly as
		// CheckLimit does — the two must not disagree about the ceiling.
		{name: "a cancelled paid plan is charged free limits", plan: "pro", status: "cancelled", resource: "resumes", want: model.FreePlanLimits.MaxResumes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(_ context.Context, _ string) (*model.Subscription, error) {
					return &model.Subscription{Plan: tt.plan, Status: tt.status}, nil
				},
			}
			svc := NewSubscriptionService(repo, nil, BillingConfig{})

			limit, err := svc.ResourceLimit(context.Background(), "user-1", tt.resource)

			require.NoError(t, err)
			assert.Equal(t, tt.want, limit)
		})
	}

	t.Run("a missing subscription is the free plan", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(_ context.Context, _ string) (*model.Subscription, error) {
				return nil, model.ErrSubscriptionNotFound
			},
		}
		svc := NewSubscriptionService(repo, nil, BillingConfig{})

		limit, err := svc.ResourceLimit(context.Background(), "user-1", "resumes")

		require.NoError(t, err)
		assert.Equal(t, model.FreePlanLimits.MaxResumes, limit)
	})

	t.Run("a storage failure is reported rather than guessed at", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(_ context.Context, _ string) (*model.Subscription, error) {
				return nil, errors.New("boom")
			},
		}
		svc := NewSubscriptionService(repo, nil, BillingConfig{})

		_, err := svc.ResourceLimit(context.Background(), "user-1", "resumes")

		assert.Error(t, err)
	})
}
