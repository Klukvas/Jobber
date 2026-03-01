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
	GetAllCounts(ctx context.Context, userID string) (jobs, resumes, apps, aiReqs, jobParses int, err error)
}
