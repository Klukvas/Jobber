package repository

import (
	"context"
	"errors"
	"fmt"

	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// LinkExternalAccount stores the provider customer ID for a user, creating the
// free row if it is missing. Plan and status are untouched: linking happens when
// a checkout completes, and it grants nothing by itself.
//
// A customer already on the row is kept. This runs off checkout.completed, which
// can be replayed or arrive late; letting it overwrite would point the customer
// portal at a stale customer. The subscription events carry the authoritative
// customer and replace it when they land.
//
// external_account_id carries a partial UNIQUE index, so one provider account
// resolves to exactly one user. Hitting it is not a generic database failure:
// it means this account is already somebody else's, which the checkout must
// refuse rather than risk mis-granting — and which a person has to untangle.
// Naming it is what separates that from an outage.
func (r *SubscriptionRepository) LinkExternalAccount(ctx context.Context, userID, externalAccountID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, external_account_id, status, plan, updated_at)
		VALUES ($1, $2, 'free', 'free', NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			external_account_id = COALESCE(subscriptions.external_account_id, EXCLUDED.external_account_id),
			updated_at = NOW()`,
		userID, externalAccountID,
	)
	switch {
	case isAccountUniqueViolation(err):
		return fmt.Errorf("%w: account %q cannot also be linked to user %q",
			model.ErrBillingAccountTaken, externalAccountID, userID)
	case isMissingUser(err):
		return fmt.Errorf("%w: cannot link account %q to user %q", model.ErrUserNotFound, externalAccountID, userID)
	}
	return err
}

// PostgreSQL SQLSTATEs the repository maps to domain errors.
const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
)

// externalAccountUniqueIndex is the partial UNIQUE index that makes one provider
// customer resolve to exactly one user (migration 000045).
const externalAccountUniqueIndex = "idx_subscriptions_external_account_id"

// isAccountUniqueViolation reports a breach of the customer index specifically.
// Any other unique breach (the subscription ID, say) is a different problem and
// must not be reported as two users sharing a customer.
func isAccountUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode &&
		pgErr.ConstraintName == externalAccountUniqueIndex
}

// isMissingUser reports a foreign-key breach: the row names a user that does not
// exist. Subscription rows cascade with their user, so this means the account
// was deleted.
func isMissingUser(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode
}

// EnsureFree creates a free subscription row for a user if none exists.
// An existing row is left untouched — this must never downgrade a payer. A user
// that does not exist comes back as model.ErrUserNotFound.
func (r *SubscriptionRepository) EnsureFree(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO subscriptions (user_id, status, plan)
		VALUES ($1, 'free', 'free')
		ON CONFLICT (user_id) DO NOTHING`, userID)
	if isMissingUser(err) {
		return fmt.Errorf("%w: %q", model.ErrUserNotFound, userID)
	}
	return err
}

// GetUserContact returns the buyer details used to pre-fill a provider checkout.
func (r *SubscriptionRepository) GetUserContact(ctx context.Context, userID string) (*model.UserContact, error) {
	var contact model.UserContact
	err := r.pool.QueryRow(ctx,
		`SELECT email, name FROM users WHERE id = $1`, userID,
	).Scan(&contact.Email, &contact.Name)
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

// countableResumes is the WHERE clause both resume counts share.
//
// Resumes have no archive concept, so every real one is counted against the
// limit. What is left out is the placeholder row the pre-finalize upload flow
// wrote before there was a file: those were never resumes, and counting them
// let abandoned uploads fill a plan permanently. See
// resumeModel.CountableResumeCondition — the resume repository re-counts with
// the same condition inside the transaction that inserts, and the two must
// agree or an upload the count allowed would be refused by the write.
const countableResumes = `SELECT COUNT(*) FROM resumes WHERE user_id = $1 AND ` +
	resumeModel.CountableResumeCondition

// CountUserResumes counts the resumes that consume a plan slot.
func (r *SubscriptionRepository) CountUserResumes(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, countableResumes, userID).Scan(&count)
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
			(` + countableResumes + `),
			(SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND usage_type IN ('match_score', 'resume_autofill_parse') AND created_at >= date_trunc('month', NOW())),
			(SELECT COUNT(*) FROM ai_usage WHERE user_id = $1 AND usage_type = 'job_parse' AND created_at >= date_trunc('month', NOW())),
			(SELECT COUNT(*) FROM resume_builders WHERE user_id = $1),
			(SELECT COUNT(*) FROM cover_letters WHERE user_id = $1)
	`
	err = r.pool.QueryRow(ctx, query, userID).Scan(&jobs, &resumes, &aiReqs, &jobParses, &resumeBuilders, &coverLetters)
	return
}

// statusCancelled is the one stored status that means the provider has ended a
// subscription, so its identifier is safe to replace. It mirrors
// service.StatusCancelled, which this package cannot import; the two are pinned
// together by TestStoredCancelledStatusMatchesTheService.
const statusCancelled = "cancelled"

// WebhookEventRetentionDays is how long an event claim is kept.
//
// The claim exists to recognise a redelivery. Creem stops retrying after 24
// hours, so a claim older than that has nothing left to catch but a manual
// resend from the dashboard, and the table would otherwise grow for the life of
// the product. The window is far wider than the retry schedule because
// forgetting a claim early is not free: a manual resend of a very old event
// would be applied again.
// Even then the lifecycle ordering guard refuses anything not strictly newer
// than the state on the row, so the cost is a no-op rather than a wrong plan.
const WebhookEventRetentionDays = 90

// deleteExpiredWebhookEventsSQL drops claims no redelivery can still reference.
// The window is a parameter rather than string-built SQL, so the interval is
// data like every other bound value.
const deleteExpiredWebhookEventsSQL = `
		DELETE FROM webhook_events
		WHERE processed_at < NOW() - make_interval(days => $1)`

// DeleteExpiredWebhookEvents removes event claims older than the retention
// window and reports how many were dropped.
func (r *SubscriptionRepository) DeleteExpiredWebhookEvents(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := r.pool.Exec(ctx, deleteExpiredWebhookEventsSQL, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to delete webhook event claims older than %d days: %w", retentionDays, err)
	}
	return tag.RowsAffected(), nil
}

// applySubscriptionEventSQL claims the webhook event and writes the state it
// carries.
//
// The claim (`webhook_events`) and the entitlement write (`subscriptions`) sit
// in one CTE so PostgreSQL rolls both back together: a failure can never leave
// an event claimed with its state unapplied, which would make the provider's
// retry look like a duplicate and strand the subscriber on the old plan.
//
// Both guards live in the UPDATE's WHERE clause rather than in a prior read, so
// two deliveries racing on the same row cannot interleave: ON CONFLICT DO UPDATE
// re-evaluates the condition against the row version the other writer just
// committed.
//
//   - Lifecycle ordering is strictly greater-than, so an event bearing the
//     *same* `updated_at` as the applied state is superseded rather than
//     replayed. Two events describing one change (a payment and the subscription
//     update it triggers) normalise to the same state, so re-applying the second
//     can only undo a correct write when it arrives out of order — first writer
//     wins, and the loser is still recorded as processed. The one exception is
//     an ending: a cancellation that ties with the applied state still lands,
//     because Creem is not documented to bump `updated_at` between the last
//     payment and the cancel, and dropping it would keep a non-paying user on a
//     paid plan. Ending access is also the only transition that is safe to apply
//     twice.
//   - The link guard refuses to overwrite a *different*, still-live
//     external_subscription_id. The row holds exactly one, so replacing it would
//     leave a subscription billing at Creem with nothing in Jobber pointing
//     at it. A first link (nothing stored), the same subscription moving through
//     its lifecycle, and a replacement after the provider ended the old one are
//     all allowed.
const applySubscriptionEventSQL = `
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
			WHERE (subscriptions.last_event_at IS NULL
			       OR EXCLUDED.last_event_at > subscriptions.last_event_at
			       OR (EXCLUDED.last_event_at = subscriptions.last_event_at
			           AND EXCLUDED.status = '` + statusCancelled + `'
			           AND subscriptions.status <> '` + statusCancelled + `'))
			  AND (COALESCE(subscriptions.external_subscription_id, '') = ''
			       OR subscriptions.external_subscription_id = EXCLUDED.external_subscription_id
			       OR subscriptions.status = '` + statusCancelled + `')
			RETURNING user_id
		)
		SELECT EXISTS (SELECT 1 FROM claim), EXISTS (SELECT 1 FROM applied)`

// lockLinkedSubscriptionSQL reads the provider subscription a user's row is
// currently linked to and locks that row for the rest of the transaction, so the
// link cannot change between the decision and the write.
const lockLinkedSubscriptionSQL = `
		SELECT external_subscription_id, status
		FROM subscriptions
		WHERE user_id = $1::uuid
		FOR UPDATE`

// ApplySubscriptionEvent claims the webhook event and writes the subscription
// state it carries, in one transaction.
//
// Two invariants have to hold together, which is why this is a transaction and
// not a bare statement:
//
//  1. an event is claimed if and only if its state was written. Claiming
//     separately would let a crash record an event as processed with its state
//     lost, and the provider's retry would then be dismissed as a duplicate.
//  2. a user's single external_subscription_id is never replaced while the
//     subscription it names is still billing. Two checkouts started while free
//     and both paid would otherwise leave the second activation overwriting the
//     first identifier, stranding a live subscription neither the user nor the
//     app can cancel.
//
// The second invariant is decided here, under the row lock the opening read
// takes, so it holds for *every* way the owning row was resolved — the checkout
// metadata, the subscription ID and the provider customer alike. A conflict
// returns before anything is claimed or written, so the caller can ask the
// provider to redeliver and the event lands once the stale link is genuinely
// ended; the same condition is repeated in the write's WHERE clause as
// the backstop for the one case the lock cannot cover, a row that did not exist
// when the read ran.
//
// The outcome distinguishes the four cases the caller must treat differently:
// applied, an already-seen duplicate, a claimed-but-superseded event (still
// recorded as processed so the provider stops redelivering it), and a link
// conflict.
func (r *SubscriptionRepository) ApplySubscriptionEvent(
	ctx context.Context, eventID, eventType string, sub *model.Subscription,
) (model.WebhookApplyOutcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin apply of subscription event %q: %w", eventID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	conflict, err := linkConflicts(ctx, tx, sub)
	if err != nil {
		return "", fmt.Errorf("failed to read the subscription linked to user %q: %w", sub.UserID, err)
	}
	if conflict {
		return model.WebhookLinkConflict, nil
	}

	var claimed, applied bool
	err = tx.QueryRow(ctx, applySubscriptionEventSQL,
		eventID, eventType,
		sub.UserID, sub.ExternalSubscriptionID, sub.ExternalAccountID,
		sub.Status, sub.Plan, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAt, sub.LastEventAt,
	).Scan(&claimed, &applied)
	if err != nil {
		// The account on this event already belongs to another user. The whole
		// transaction rolls back, so nothing is claimed and nothing is written —
		// but no retry will ever untangle two users behind one provider customer,
		// so it is reported as its own terminal outcome rather than as a failure
		// the provider should keep re-attempting.
		if isAccountUniqueViolation(err) {
			return model.WebhookAccountConflict, nil
		}
		return "", fmt.Errorf("failed to apply subscription event %q: %w", eventID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit subscription event %q: %w", eventID, err)
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

// linkConflicts locks the user's subscription row and reports whether the
// incoming event would replace a provider subscription that is still alive.
//
// A missing row is not a conflict: there is no identifier to strand, and the
// write's own WHERE clause still refuses a replacement if a row appears in the
// meantime.
func linkConflicts(ctx context.Context, tx pgx.Tx, sub *model.Subscription) (bool, error) {
	var linkedID *string
	var status string
	err := tx.QueryRow(ctx, lockLinkedSubscriptionSQL, sub.UserID).Scan(&linkedID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return replacesLiveSubscription(linkedID, status, sub.ExternalSubscriptionID), nil
}

// replacesLiveSubscription mirrors the write's link guard: the stored
// identifier may be replaced only when there is none, when the event carries
// that very subscription, or when the provider has already ended it.
func replacesLiveSubscription(linkedID *string, status string, incomingID *string) bool {
	if linkedID == nil || *linkedID == "" {
		return false
	}
	if incomingID != nil && *incomingID == *linkedID {
		return false
	}
	return status != statusCancelled
}
