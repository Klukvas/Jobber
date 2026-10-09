package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

// SubscriptionRepository defines the data access interface for subscriptions.
type SubscriptionRepository interface {
	GetByUserID(ctx context.Context, userID string) (*model.Subscription, error)
	// GetByExternalSubscriptionID looks a subscription up by the billing
	// provider's subscription ID (Creem `sub_…`).
	GetByExternalSubscriptionID(ctx context.Context, externalSubID string) (*model.Subscription, error)
	// GetByExternalAccountID looks a subscription up by the billing provider's
	// customer ID (Creem `cust_…`). This is the server-side link
	// between a purchase and a local user.
	GetByExternalAccountID(ctx context.Context, externalAccountID string) (*model.Subscription, error)
	// EnsureFree creates a free row for a user when none exists and leaves an
	// existing row alone, so it can never downgrade a paying subscriber.
	EnsureFree(ctx context.Context, userID string) error
	// LinkExternalAccount records the provider account ID for a user without
	// touching plan or status, so an abandoned checkout grants nothing.
	LinkExternalAccount(ctx context.Context, userID, externalAccountID string) error
	// GetUserContact returns the details needed to pre-fill a provider checkout.
	GetUserContact(ctx context.Context, userID string) (*model.UserContact, error)
	CountUserJobs(ctx context.Context, userID string) (int, error)
	CountUserResumes(ctx context.Context, userID string) (int, error)
	CountUserAIRequestsThisMonth(ctx context.Context, userID string) (int, error)
	CountUserJobParsesThisMonth(ctx context.Context, userID string) (int, error)
	RecordAIUsage(ctx context.Context, userID string) error
	RecordJobParseUsage(ctx context.Context, userID string) error
	RecordResumeAutofillUsage(ctx context.Context, userID string) error
	CountUserResumeBuilders(ctx context.Context, userID string) (int, error)
	CountUserCoverLetters(ctx context.Context, userID string) (int, error)
	GetAllCounts(ctx context.Context, userID string) (jobs, resumes, aiReqs, jobParses, resumeBuilders, coverLetters int, err error)
	// ApplySubscriptionEvent claims a provider webhook event and writes the
	// subscription state it carries in one atomic operation.
	//
	// Claiming and writing must not be separate calls: a crash between them
	// would leave an event marked processed with its state lost, and the
	// provider's retry would then be acknowledged as a duplicate.
	//
	// The same operation owns two guards, because both have to hold against a
	// concurrent delivery rather than against a stale read: lifecycle ordering,
	// so an older event can never leave the row in the older state, and the
	// single-subscription link, so a user's external_subscription_id is never
	// replaced while the subscription it names is still billing.
	ApplySubscriptionEvent(ctx context.Context, eventID, eventType string, sub *model.Subscription) (model.WebhookApplyOutcome, error)
}
