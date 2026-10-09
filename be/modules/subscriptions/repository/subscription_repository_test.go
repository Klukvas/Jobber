package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func subscriptionRowColumns() []string {
	return []string{
		"id", "user_id", "external_subscription_id", "external_account_id",
		"status", "plan", "current_period_start", "current_period_end",
		"cancel_at", "last_event_at", "created_at", "updated_at",
	}
}

func TestSubscriptionRepository_GetByUserID(t *testing.T) {
	t.Run("returns subscription", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		psid := "psub-1"
		pcid := "pcust-1"
		rows := pgxmock.NewRows(subscriptionRowColumns()).AddRow(
			"sub-1", "user-1", &psid, &pcid, "active", "pro",
			&now, &now, (*time.Time)(nil), &now, now, now,
		)

		mock.ExpectQuery("SELECT id, user_id, external_subscription_id, external_account_id").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByUserID(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "sub-1", sub.ID)
		assert.Equal(t, "active", sub.Status)
		assert.Equal(t, "pro", sub.Plan)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrSubscriptionNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, external_subscription_id, external_account_id").
			WithArgs("user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByUserID(context.Background(), "user-1")
		assert.Nil(t, sub)
		assert.ErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, external_subscription_id, external_account_id").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByUserID(context.Background(), "user-1")
		assert.Nil(t, sub)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_GetByExternalSubscriptionID(t *testing.T) {
	t.Run("returns subscription", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		psid := "psub-1"
		rows := pgxmock.NewRows(subscriptionRowColumns()).AddRow(
			"sub-1", "user-1", &psid, (*string)(nil), "active", "pro",
			(*time.Time)(nil), (*time.Time)(nil), (*time.Time)(nil), (*time.Time)(nil), now, now,
		)

		mock.ExpectQuery("WHERE external_subscription_id = ").
			WithArgs("psub-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByExternalSubscriptionID(context.Background(), "psub-1")
		require.NoError(t, err)
		assert.Equal(t, "sub-1", sub.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrSubscriptionNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("WHERE external_subscription_id = ").
			WithArgs("psub-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByExternalSubscriptionID(context.Background(), "psub-1")
		assert.Nil(t, sub)
		assert.ErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("WHERE external_subscription_id = ").
			WithArgs("psub-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByExternalSubscriptionID(context.Background(), "psub-1")
		assert.Nil(t, sub)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// countTest exercises the single-arg COUNT(*) helpers.
func countTest(t *testing.T, sqlFragment string, call func(repo *SubscriptionRepository) (int, error)) {
	t.Helper()

	t.Run("returns count", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"count"}).AddRow(7)
		mock.ExpectQuery(sqlFragment).
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		n, err := call(repo)
		require.NoError(t, err)
		assert.Equal(t, 7, n)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(sqlFragment).
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		_, err = call(repo)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_Counts(t *testing.T) {
	ctx := context.Background()

	t.Run("CountUserJobs", func(t *testing.T) {
		countTest(t, "SELECT COUNT.+FROM jobs WHERE user_id", func(r *SubscriptionRepository) (int, error) {
			return r.CountUserJobs(ctx, "user-1")
		})
	})
	t.Run("CountUserResumes", func(t *testing.T) {
		countTest(t, "SELECT COUNT.+FROM resumes WHERE user_id", func(r *SubscriptionRepository) (int, error) {
			return r.CountUserResumes(ctx, "user-1")
		})
	})
	// The placeholder rows the presign-era upload flow left behind are not
	// resumes: counting them let abandoned uploads fill a plan permanently,
	// with no way for the customer to clear them.
	t.Run("CountUserResumes leaves out unfinalized upload placeholders", func(t *testing.T) {
		countTest(t, regexp.QuoteMeta(resumeModel.CountableResumeCondition), func(r *SubscriptionRepository) (int, error) {
			return r.CountUserResumes(ctx, "user-1")
		})
	})
	t.Run("CountUserAIRequestsThisMonth", func(t *testing.T) {
		countTest(t, `usage_type IN \('match_score', 'resume_autofill_parse'\)`, func(r *SubscriptionRepository) (int, error) {
			return r.CountUserAIRequestsThisMonth(ctx, "user-1")
		})
	})
	t.Run("CountUserJobParsesThisMonth", func(t *testing.T) {
		countTest(t, "usage_type = 'job_parse'", func(r *SubscriptionRepository) (int, error) {
			return r.CountUserJobParsesThisMonth(ctx, "user-1")
		})
	})
	t.Run("CountUserResumeBuilders", func(t *testing.T) {
		countTest(t, "SELECT COUNT.+FROM resume_builders WHERE user_id", func(r *SubscriptionRepository) (int, error) {
			return r.CountUserResumeBuilders(ctx, "user-1")
		})
	})
	t.Run("CountUserCoverLetters", func(t *testing.T) {
		countTest(t, "SELECT COUNT.+FROM cover_letters WHERE user_id", func(r *SubscriptionRepository) (int, error) {
			return r.CountUserCoverLetters(ctx, "user-1")
		})
	})
}

func TestSubscriptionRepository_RecordAIUsage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO ai_usage").
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewSubscriptionRepository(mock)
	require.NoError(t, repo.RecordAIUsage(context.Background(), "user-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepository_RecordJobParseUsage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO ai_usage").
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewSubscriptionRepository(mock)
	require.NoError(t, repo.RecordJobParseUsage(context.Background(), "user-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepository_RecordResumeAutofillUsage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// The literal is load-bearing: CountUserAIRequestsThisMonth only counts
	// types in its IN-list, so recording a wrong one would silently bypass
	// the quota.
	mock.ExpectExec(`INSERT INTO ai_usage \(user_id, usage_type\) VALUES \(\$1, 'resume_autofill_parse'\)`).
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewSubscriptionRepository(mock)
	require.NoError(t, repo.RecordResumeAutofillUsage(context.Background(), "user-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepository_GetAllCounts(t *testing.T) {
	t.Run("returns all six counts in order", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"jobs", "resumes", "ai", "parses", "builders", "letters"}).
			AddRow(1, 2, 3, 4, 5, 6)
		// Pin the AI-count subquery: it must count the same usage types as
		// CountUserAIRequestsThisMonth or the usage bar drifts from the limit.
		mock.ExpectQuery(`usage_type IN \('match_score', 'resume_autofill_parse'\)`).
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		jobs, resumes, aiReqs, jobParses, builders, letters, err := repo.GetAllCounts(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, 1, jobs)
		assert.Equal(t, 2, resumes)
		assert.Equal(t, 3, aiReqs)
		assert.Equal(t, 4, jobParses)
		assert.Equal(t, 5, builders)
		assert.Equal(t, 6, letters)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		_, _, _, _, _, _, err = repo.GetAllCounts(context.Background(), "user-1")
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// applyArgs is the argument list the claim-and-write statement sends: the event
// claim followed by the subscription state it carries.
func applyArgs(sub *model.Subscription) []any {
	return []any{
		"evt-1", "subscription.activated",
		sub.UserID, sub.ExternalSubscriptionID, sub.ExternalAccountID,
		sub.Status, sub.Plan, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.CancelAt, sub.LastEventAt,
	}
}

func eventSubscription() *model.Subscription {
	extSubID := "psub-1"
	extAccountID := "acct-1"
	changedAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return &model.Subscription{
		UserID:                 "user-1",
		ExternalSubscriptionID: &extSubID,
		ExternalAccountID:      &extAccountID,
		Status:                 "active",
		Plan:                   "pro",
		LastEventAt:            &changedAt,
	}
}

// linkRow is what the apply transaction locks and reads before it writes: the
// provider subscription this user is linked to, and the status that says
// whether it still bills.
func linkRow(linkedID *string, status string) *pgxmock.Rows {
	return pgxmock.NewRows([]string{"external_subscription_id", "status"}).AddRow(linkedID, status)
}

// expectLinkRead queues the locking read the transaction opens with. Its
// argument is the user ID, never the event, because the row it protects belongs
// to the user rather than to any one delivery.
func expectLinkRead(mock pgxmock.PgxPoolIface, userID string, linkedID *string, status string) {
	mock.ExpectQuery("FOR UPDATE").WithArgs(userID).WillReturnRows(linkRow(linkedID, status))
}

func TestSubscriptionRepository_ApplySubscriptionEvent(t *testing.T) {
	tests := []struct {
		name        string
		claimed     bool
		applied     bool
		wantOutcome model.WebhookApplyOutcome
	}{
		{
			name:        "first delivery claims and writes",
			claimed:     true,
			applied:     true,
			wantOutcome: model.WebhookApplied,
		},
		{
			name:        "redelivery of a claimed event writes nothing",
			claimed:     false,
			applied:     false,
			wantOutcome: model.WebhookDuplicate,
		},
		{
			// The strict guard folds two cases into one outcome: an event older
			// than the applied state, and one carrying the same `data.changed`.
			// Either way the row is left alone and the event is still recorded
			// as processed, so the provider stops redelivering it.
			name:        "claimed but not newer than the applied state",
			claimed:     true,
			applied:     false,
			wantOutcome: model.WebhookSuperseded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			sub := eventSubscription()
			mock.ExpectBegin()
			expectLinkRead(mock, sub.UserID, sub.ExternalSubscriptionID, "active")
			mock.ExpectQuery("WITH claim AS").
				WithArgs(applyArgs(sub)...).
				WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(tc.claimed, tc.applied))
			mock.ExpectCommit()

			repo := NewSubscriptionRepository(mock)
			outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

			require.NoError(t, err)
			assert.Equal(t, tc.wantOutcome, outcome)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("a user with no row yet is written without a link check to make", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		sub := eventSubscription()
		mock.ExpectBegin()
		mock.ExpectQuery("FOR UPDATE").WithArgs(sub.UserID).WillReturnError(pgx.ErrNoRows)
		mock.ExpectQuery("WITH claim AS").
			WithArgs(applyArgs(sub)...).
			WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))
		mock.ExpectCommit()

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

		require.NoError(t, err)
		assert.Equal(t, model.WebhookApplied, outcome)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error without claiming anything", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		sub := eventSubscription()
		mock.ExpectBegin()
		expectLinkRead(mock, sub.UserID, sub.ExternalSubscriptionID, "active")
		mock.ExpectQuery("WITH claim AS").
			WithArgs(applyArgs(sub)...).
			WillReturnError(errors.New("boom"))
		mock.ExpectRollback()

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

		require.Error(t, err)
		assert.Empty(t, string(outcome), "a failed statement has no outcome to report")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("an unreadable link is an error, never a silent overwrite", func(t *testing.T) {
		// The read is the guard. If it cannot be answered the event must be
		// retried, not applied on the assumption that there was nothing to
		// protect.
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		sub := eventSubscription()
		mock.ExpectBegin()
		mock.ExpectQuery("FOR UPDATE").WithArgs(sub.UserID).WillReturnError(errors.New("boom"))
		mock.ExpectRollback()

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

		require.Error(t, err)
		assert.Empty(t, string(outcome))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a failed commit is reported rather than counted as applied", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		sub := eventSubscription()
		mock.ExpectBegin()
		expectLinkRead(mock, sub.UserID, sub.ExternalSubscriptionID, "active")
		mock.ExpectQuery("WITH claim AS").
			WithArgs(applyArgs(sub)...).
			WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))
		mock.ExpectCommit().WillReturnError(errors.New("connection lost"))

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

		require.Error(t, err)
		assert.Empty(t, string(outcome), "an uncommitted transaction claimed nothing")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a transaction that cannot be opened is an error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectBegin().WillReturnError(errors.New("no connection"))

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", eventSubscription())

		require.Error(t, err)
		assert.Empty(t, string(outcome))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestApplySubscriptionEventLinkGuard covers the invariant that a user's single
// external_subscription_id is never replaced while the subscription it names is
// still billing.
//
// Without it, a user who starts two checkouts while free and pays for both ends
// up with the second activation overwriting the first identifier: subscription
// one keeps charging at Creem with nothing in Jobber pointing at it, so
// neither the user nor the app can cancel it.
func TestApplySubscriptionEventLinkGuard(t *testing.T) {
	const incomingID = "psub-1"
	other := "psub-other"
	empty := ""

	tests := []struct {
		name     string
		linkedID *string
		status   string
		refuse   bool
	}{
		{
			name:     "nothing linked yet, so the first activation links",
			linkedID: nil,
			status:   "free",
		},
		{
			name:     "an empty stored id is no link either",
			linkedID: &empty,
			status:   "free",
		},
		{
			name:     "the same subscription moving through its lifecycle",
			linkedID: ptr(incomingID),
			status:   "active",
		},
		{
			name:     "a second subscription while the first is active",
			linkedID: &other,
			status:   "active",
			refuse:   true,
		},
		{
			// past_due is a dunning period: Creem is still trying to charge.
			name:     "a second subscription while the first is past due",
			linkedID: &other,
			status:   "past_due",
			refuse:   true,
		},
		{
			// A paused subscription resumes into billing; its id must survive.
			name:     "a second subscription while the first is paused",
			linkedID: &other,
			status:   "paused",
			refuse:   true,
		},
		{
			// A scheduled cancellation is stored as active with cancel_at, so the
			// status the guard sees is the one that still bills.
			name:     "a second subscription while the first is winding down",
			linkedID: &other,
			status:   "active",
			refuse:   true,
		},
		{
			name:     "a replacement once the provider has ended the old one",
			linkedID: &other,
			status:   "cancelled",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			sub := eventSubscription()
			mock.ExpectBegin()
			expectLinkRead(mock, sub.UserID, tc.linkedID, tc.status)
			if tc.refuse {
				mock.ExpectRollback()
			} else {
				mock.ExpectQuery("WITH claim AS").
					WithArgs(applyArgs(sub)...).
					WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))
				mock.ExpectCommit()
			}

			repo := NewSubscriptionRepository(mock)
			outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

			require.NoError(t, err)
			if tc.refuse {
				assert.Equal(t, model.WebhookLinkConflict, outcome)
			} else {
				assert.Equal(t, model.WebhookApplied, outcome)
			}
			// The expectations are the assertion that a refusal wrote nothing:
			// no claim statement was queued, so running one would fail the mock.
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("an event that carries no subscription id cannot erase a live link", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		sub := eventSubscription()
		sub.ExternalSubscriptionID = nil
		mock.ExpectBegin()
		expectLinkRead(mock, sub.UserID, &other, "active")
		mock.ExpectRollback()

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

		require.NoError(t, err)
		assert.Equal(t, model.WebhookLinkConflict, outcome)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApplySubscriptionEventKeepsItsGuardsInOneStatement(t *testing.T) {
	// The claim gate, the ordering guard and the link guard are what make this
	// write safe. A refactor could drop any of them — or split the statement in
	// two — and still compile, so the SQL itself is asserted.
	var statements []string
	capture := pgxmock.QueryMatcherFunc(func(_, actualSQL string) error {
		statements = append(statements, actualSQL)
		return nil
	})
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(capture))
	require.NoError(t, err)
	defer mock.Close()

	sub := eventSubscription()
	mock.ExpectBegin()
	mock.ExpectQuery("").
		WithArgs(sub.UserID).
		WillReturnRows(linkRow(sub.ExternalSubscriptionID, "active"))
	mock.ExpectQuery("").
		WithArgs(applyArgs(sub)...).
		WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))
	mock.ExpectCommit()

	repo := NewSubscriptionRepository(mock)
	_, err = repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)
	require.NoError(t, err)

	require.Len(t, statements, 2, "expected the locking read and the claim-and-write")
	read, executed := statements[0], statements[1]

	assert.Contains(t, read, "FOR UPDATE",
		"the link must be read under a row lock, or a concurrent activation could pass the check "+
			"and still overwrite it")

	assert.Contains(t, executed, "INSERT INTO webhook_events")
	assert.Contains(t, executed, "INSERT INTO subscriptions")
	assert.Contains(t, executed, "FROM claim",
		"the entitlement write must be gated on winning the claim")
	assert.Contains(t, executed, "EXCLUDED.last_event_at > subscriptions.last_event_at",
		"the lifecycle-ordering guard must live in the SQL WHERE, not in a prior read")
	assert.NotContains(t, executed, "EXCLUDED.last_event_at >= subscriptions.last_event_at",
		"the guard must be strict: an event carrying the same `updated_at` describes a change "+
			"already accounted for, so replaying it could only undo a correct write")
	assert.Contains(t, executed, "EXCLUDED.last_event_at = subscriptions.last_event_at",
		"a tie is allowed for exactly one transition")
	assert.Contains(t, executed, "EXCLUDED.status = 'cancelled'",
		"a cancellation that ties with the applied state must still land, or a non-paying "+
			"user keeps a paid plan")
	assert.Contains(t, executed, "subscriptions.status <> 'cancelled'",
		"an ending is not re-applied over an ending")
	assert.Contains(t, executed, "subscriptions.external_subscription_id = EXCLUDED.external_subscription_id",
		"the link guard must be re-evaluated by the write itself against the row version a "+
			"concurrent writer committed, not only by the read that precedes it")
	assert.Contains(t, executed, "subscriptions.status = 'cancelled'",
		"a replacement is allowed only once the provider has ended the old subscription")
	assert.NotContains(t, executed, ";",
		"one statement only — separate statements would not roll back together")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestStoredCancelledStatusMatchesTheService pins the one string the SQL guard
// and the service's status vocabulary have to agree on. The repository cannot
// import the service package, so nothing but this test would notice a rename.
func TestStoredCancelledStatusMatchesTheService(t *testing.T) {
	assert.Equal(t, service.StatusCancelled, statusCancelled)
}

func TestSubscriptionRepository_GetByExternalAccountID(t *testing.T) {
	t.Run("returns subscription", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		accountID := "acct-1"
		rows := pgxmock.NewRows(subscriptionRowColumns()).AddRow(
			"sub-1", "user-1", (*string)(nil), &accountID, "free", "free",
			(*time.Time)(nil), (*time.Time)(nil), (*time.Time)(nil), (*time.Time)(nil), now, now,
		)

		mock.ExpectQuery("WHERE external_account_id = ").
			WithArgs("acct-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByExternalAccountID(context.Background(), "acct-1")
		require.NoError(t, err)
		assert.Equal(t, "user-1", sub.UserID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrSubscriptionNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("WHERE external_account_id = ").
			WithArgs("acct-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByExternalAccountID(context.Background(), "acct-1")
		assert.Nil(t, sub)
		assert.ErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_LinkExternalAccount(t *testing.T) {
	t.Run("links the account without granting a plan", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// The insert seeds a free row; the conflict branch only refreshes the
		// account so an abandoned checkout cannot upgrade anyone.
		mock.ExpectExec("INSERT INTO subscriptions").
			WithArgs("user-1", "acct-1").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewSubscriptionRepository(mock)
		require.NoError(t, repo.LinkExternalAccount(context.Background(), "user-1", "acct-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("keeps a customer already on the row", func(t *testing.T) {
		// A replayed or late checkout.completed must not point the portal at a
		// stale customer.
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec(`external_account_id = COALESCE\(subscriptions\.external_account_id, EXCLUDED\.external_account_id\)`).
			WithArgs("user-1", "acct-1").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewSubscriptionRepository(mock)
		require.NoError(t, repo.LinkExternalAccount(context.Background(), "user-1", "acct-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a customer owned by another user is reported as such", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO subscriptions").
			WithArgs("user-1", "acct-1").
			WillReturnError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: externalAccountUniqueIndex})

		repo := NewSubscriptionRepository(mock)
		err = repo.LinkExternalAccount(context.Background(), "user-1", "acct-1")

		assert.ErrorIs(t, err, model.ErrBillingAccountTaken)
	})

	t.Run("any other unique violation is not blamed on a shared customer", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO subscriptions").
			WithArgs("user-1", "acct-1").
			WillReturnError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: "subscriptions_external_subscription_id_key"})

		repo := NewSubscriptionRepository(mock)
		err = repo.LinkExternalAccount(context.Background(), "user-1", "acct-1")

		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrBillingAccountTaken)
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO subscriptions").
			WithArgs("user-1", "acct-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		require.Error(t, repo.LinkExternalAccount(context.Background(), "user-1", "acct-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_EnsureFree(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO subscriptions").
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("INSERT", 0))

	repo := NewSubscriptionRepository(mock)
	require.NoError(t, repo.EnsureFree(context.Background(), "user-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepository_GetUserContact(t *testing.T) {
	t.Run("returns the buyer details", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"email", "name"}).
			AddRow("buyer@example.com", "Test Buyer")
		mock.ExpectQuery("SELECT email, name FROM users").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		contact, err := repo.GetUserContact(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "buyer@example.com", contact.Email)
		assert.Equal(t, "Test Buyer", contact.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps a missing user to ErrSubscriptionNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT email, name FROM users").
			WithArgs("user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewSubscriptionRepository(mock)
		_, err = repo.GetUserContact(context.Background(), "user-1")
		assert.ErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT email, name FROM users").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		_, err = repo.GetUserContact(context.Background(), "user-1")
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApplySubscriptionEventPreservesLinkedAccount(t *testing.T) {
	// The conflict branch must keep an already-linked account when an event
	// carries none — the COALESCE is what stops a provider payload without an
	// account from severing the purchase→user link.
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	sub := &model.Subscription{UserID: "user-1", Status: "active", Plan: "pro", LastEventAt: &now}
	mock.ExpectBegin()
	expectLinkRead(mock, sub.UserID, nil, "free")
	mock.ExpectQuery(`external_account_id = COALESCE\(EXCLUDED\.external_account_id, subscriptions\.external_account_id\)`).
		WithArgs(applyArgs(sub)...).
		WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))
	mock.ExpectCommit()

	repo := NewSubscriptionRepository(mock)
	outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

	require.NoError(t, err)
	assert.Equal(t, model.WebhookApplied, outcome)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplySubscriptionEventReportsAnAccountAlreadyLinkedElsewhere(t *testing.T) {
	// external_account_id is unique, so an event carrying an account that is
	// already another user's breaks the index. It must come back as its own
	// terminal outcome rather than as a generic failure: a failure is retried,
	// and no retry can untangle two users behind one provider customer.
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	sub := eventSubscription()
	mock.ExpectBegin()
	expectLinkRead(mock, sub.UserID, nil, "free")
	mock.ExpectQuery("WITH claim AS").
		WithArgs(applyArgs(sub)...).
		WillReturnError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: "idx_subscriptions_external_account_id"})
	mock.ExpectRollback()

	repo := NewSubscriptionRepository(mock)
	outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

	require.NoError(t, err, "a collision is an outcome to report, not an error to retry")
	assert.Equal(t, model.WebhookAccountConflict, outcome)
	assert.NoError(t, mock.ExpectationsWereMet(), "the transaction must roll back, claiming nothing")
}
