package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
)

// SubscriptionRepository implements ports.SubscriptionRepository with PostgreSQL.
type SubscriptionRepository struct {
	pool PgxDB
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(pool PgxDB) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

// subscriptionColumns is the shared projection for every single-row read, so
// the column order can only drift in one place.
const subscriptionColumns = `id, user_id, external_subscription_id, external_account_id,
	       status, plan, current_period_start, current_period_end,
	       cancel_at, last_event_at, created_at, updated_at`

// GetByUserID retrieves a subscription by user ID.
func (r *SubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	return r.queryOne(ctx, `SELECT `+subscriptionColumns+` FROM subscriptions WHERE user_id = $1`, userID)
}

// GetByExternalSubscriptionID retrieves a subscription by the billing provider's
// subscription ID.
func (r *SubscriptionRepository) GetByExternalSubscriptionID(ctx context.Context, externalSubID string) (*model.Subscription, error) {
	return r.queryOne(ctx, `SELECT `+subscriptionColumns+` FROM subscriptions WHERE external_subscription_id = $1`, externalSubID)
}

// GetByExternalAccountID retrieves a subscription by the billing provider's
// customer account ID.
func (r *SubscriptionRepository) GetByExternalAccountID(ctx context.Context, externalAccountID string) (*model.Subscription, error) {
	return r.queryOne(ctx, `SELECT `+subscriptionColumns+` FROM subscriptions WHERE external_account_id = $1`, externalAccountID)
}

func (r *SubscriptionRepository) queryOne(ctx context.Context, query string, arg any) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&sub.ID, &sub.UserID, &sub.ExternalSubscriptionID, &sub.ExternalAccountID,
		&sub.Status, &sub.Plan, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.CancelAt, &sub.LastEventAt, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

// LinkExternalAccount stores the provider account ID for a user, creating the
// free row if it is missing. Plan and status are untouched: linking happens when
// a checkout starts, and an abandoned checkout must not grant anything.
func (r *SubscriptionRepository) LinkExternalAccount(ctx context.Context, userID, externalAccountID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, external_account_id, status, plan, updated_at)
		VALUES ($1, $2, 'free', 'free', NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			external_account_id = EXCLUDED.external_account_id,
			updated_at = NOW()`,
		userID, externalAccountID,
	)
	return err
}

// EnsureFree creates a free subscription row for a user if none exists.
// An existing row is left untouched — this must never downgrade a payer.
func (r *SubscriptionRepository) EnsureFree(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, status, plan)
		VALUES ($1, 'free', 'free')
		ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

// GetUserContact returns the buyer details used to pre-fill a provider checkout.
func (r *SubscriptionRepository) GetUserContact(ctx context.Context, userID string) (*model.UserContact, error) {
	var contact model.UserContact
	err := r.pool.QueryRow(ctx,
		`SELECT email, name, locale FROM users WHERE id = $1`, userID,
	).Scan(&contact.Email, &contact.Name, &contact.Locale)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrSubscriptionNotFound
		}
		return nil, err
	}
	return &contact, nil
}

// CountUserJobs counts non-archived jobs for a user.
// Archived jobs are excluded intentionally — only tracked cards consume the limit.
func (r *SubscriptionRepository) CountUserJobs(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM jobs WHERE user_id = $1 AND is_archived = false`, userID,
	).Scan(&count)
	return count, err
}

// CountUserResumes counts all resumes for a user.
// Resumes have no archive concept, so all are counted against the limit.
func (r *SubscriptionRepository) CountUserResumes(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM resumes WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

// CountUserAIRequestsThisMonth counts general AI requests (match score, resume
// autofill extraction) for a user in the current calendar month. Job parses
// have their own limit and are counted separately.
func (r *SubscriptionRepository) CountUserAIRequestsThisMonth(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_usage
		 WHERE user_id = $1
		   AND usage_type IN ('match_score', 'resume_autofill_parse')
		   AND created_at >= date_trunc('month', NOW())`, userID,
	).Scan(&count)
	return count, err
}

// CountUserJobParsesThisMonth counts job parse requests for a user in the current calendar month.
func (r *SubscriptionRepository) CountUserJobParsesThisMonth(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ai_usage
		 WHERE user_id = $1
		   AND usage_type = 'job_parse'
		   AND created_at >= date_trunc('month', NOW())`, userID,
	).Scan(&count)
	return count, err
}

// RecordAIUsage inserts an AI usage record for a user (match_score type).
func (r *SubscriptionRepository) RecordAIUsage(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_usage (user_id, usage_type) VALUES ($1, 'match_score')`, userID,
	)
	return err
}

// RecordJobParseUsage inserts a job parse usage record for a user.
func (r *SubscriptionRepository) RecordJobParseUsage(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_usage (user_id, usage_type) VALUES ($1, 'job_parse')`, userID,
	)
	return err
}

// RecordResumeAutofillUsage inserts a resume autofill extraction usage record.
func (r *SubscriptionRepository) RecordResumeAutofillUsage(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ai_usage (user_id, usage_type) VALUES ($1, 'resume_autofill_parse')`, userID,
	)
	return err
}

// CountUserResumeBuilders counts resume builders for a user.
func (r *SubscriptionRepository) CountUserResumeBuilders(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM resume_builders WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

// CountUserCoverLetters counts cover letters for a user.
func (r *SubscriptionRepository) CountUserCoverLetters(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM cover_letters WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

// GetAllCounts returns all resource counts in a single query (6 sub-selects, 1 round-trip).
func (r *SubscriptionRepository) GetAllCounts(ctx context.Context, userID string) (jobs, resumes, aiReqs, jobParses, resumeBuilders, coverLetters int, err error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM jobs WHERE user_id = $1 AND is_archived = false),
			(SELECT COUNT(*) FROM resumes WHERE user_id = $1),
			(SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND usage_type IN ('match_score', 'resume_autofill_parse') AND created_at >= date_trunc('month', NOW())),
			(SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND usage_type = 'job_parse' AND created_at >= date_trunc('month', NOW())),
			(SELECT COUNT(*) FROM resume_builders WHERE user_id = $1),
			(SELECT COUNT(*) FROM cover_letters WHERE user_id = $1)
	`
	err = r.pool.QueryRow(ctx, query, userID).Scan(&jobs, &resumes, &aiReqs, &jobParses, &resumeBuilders, &coverLetters)
	return
}

// ApplySubscriptionEvent claims the webhook event and writes the subscription
// state it carries in a single statement.
//
// The claim (`webhook_events`) and the entitlement write (`subscriptions`) sit
// in one CTE so PostgreSQL rolls both back together: a failure can never leave
// an event claimed with its state unapplied, which would make the provider's
// retry look like a duplicate and strand the subscriber on the old plan.
//
// The lifecycle-ordering guard lives in the UPDATE's WHERE clause rather than
// in a prior read, so two deliveries racing on the same row cannot interleave
// into the older state: ON CONFLICT DO UPDATE re-evaluates the condition
// against the row version the other writer just committed.
//
// The comparison is strictly greater-than, so an event bearing the *same*
// `data.changed` as the applied state is superseded rather than replayed. Two
// events describing one change (a charge and the subscription update it
// triggers) normalise to the same state, so re-applying the second can only
// undo a correct write when it arrives out of order — first writer wins, and
// the loser is still recorded as processed.
//
// The outcome distinguishes the three cases the caller must treat differently:
// applied, an already-seen duplicate, and a claimed-but-superseded event (which
// is still recorded as processed so the provider stops redelivering it).
func (r *SubscriptionRepository) ApplySubscriptionEvent(
	ctx context.Context, eventID, eventType string, sub *model.Subscription,
) (model.WebhookApplyOutcome, error) {
	const query = `
		WITH claim AS (
			INSERT INTO webhook_events (event_id, event_type)
			VALUES ($1, $2)
			ON CONFLICT (event_id) DO NOTHING
			RETURNING event_id
		), applied AS (
			INSERT INTO subscriptions (user_id, external_subscription_id, external_account_id,
			                           status, plan, current_period_start, current_period_end,
			                           cancel_at, last_event_at, updated_at)
			-- Types are pinned rather than inferred: in an INSERT ... SELECT the
			-- parameters are analysed as a subquery first, where an untyped
			-- parameter can resolve to text and then fail to encode a UUID or a
			-- timestamp.
			SELECT $3::uuid, $4::text, $5::text, $6::text, $7::text,
			       $8::timestamptz, $9::timestamptz, $10::timestamptz, $11::timestamptz, NOW()
			FROM claim
			ON CONFLICT (user_id) DO UPDATE SET
				external_subscription_id = EXCLUDED.external_subscription_id,
				external_account_id = COALESCE(EXCLUDED.external_account_id, subscriptions.external_account_id),
				status = EXCLUDED.status,
				plan = EXCLUDED.plan,
				current_period_start = EXCLUDED.current_period_start,
				current_period_end = EXCLUDED.current_period_end,
				cancel_at = EXCLUDED.cancel_at,
				last_event_at = EXCLUDED.last_event_at,
				updated_at = NOW()
			WHERE subscriptions.last_event_at IS NULL
			   OR EXCLUDED.last_event_at > subscriptions.last_event_at
			RETURNING user_id
		)
		SELECT EXISTS (SELECT 1 FROM claim), EXISTS (SELECT 1 FROM applied)`

	var claimed, applied bool
	err := r.pool.QueryRow(ctx, query,
		eventID, eventType,
		sub.UserID, sub.ExternalSubscriptionID, sub.ExternalAccountID,
		sub.Status, sub.Plan, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAt, sub.LastEventAt,
	).Scan(&claimed, &applied)
	if err != nil {
		return "", fmt.Errorf("failed to apply subscription event %q: %w", eventID, err)
	}

	switch {
	case !claimed:
		return model.WebhookDuplicate, nil
	case !applied:
		return model.WebhookSuperseded, nil
	default:
		return model.WebhookApplied, nil
	}
}
