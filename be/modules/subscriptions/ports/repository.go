package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

// SubscriptionRepository defines the data access interface for subscriptions.
type SubscriptionRepository interface {
	GetByUserID(ctx context.Context, userID string) (*model.Subscription, error)
	GetByPaddleSubscriptionID(ctx context.Context, paddleSubID string) (*model.Subscription, error)
	Upsert(ctx context.Context, sub *model.Subscription) error
	CountUserJobs(ctx context.Context, userID string) (int, error)
	CountUserResumes(ctx context.Context, userID string) (int, error)
	CountUserApplications(ctx context.Context, userID string) (int, error)
	CountUserAIRequestsThisMonth(ctx context.Context, userID string) (int, error)
	CountUserJobParsesThisMonth(ctx context.Context, userID string) (int, error)
	RecordAIUsage(ctx context.Context, userID string) error
	RecordJobParseUsage(ctx context.Context, userID string) error
	CountUserResumeBuilders(ctx context.Context, userID string) (int, error)
	CountUserCoverLetters(ctx context.Context, userID string) (int, error)
	GetAllCounts(ctx context.Context, userID string) (jobs, resumes, apps, aiReqs, jobParses, resumeBuilders, coverLetters int, err error)
	// WebhookEventExists returns true if the event ID has already been processed.
	WebhookEventExists(ctx context.Context, eventID string) (bool, error)
	// RecordWebhookEvent stores a processed event ID to prevent duplicate processing.
	RecordWebhookEvent(ctx context.Context, eventID, eventType string) error
}
