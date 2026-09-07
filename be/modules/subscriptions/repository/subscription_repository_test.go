package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// applyArgs is the argument list ApplySubscriptionEvent sends: the event claim
// followed by the subscription state it carries.
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
			mock.ExpectQuery("WITH claim AS").
				WithArgs(applyArgs(sub)...).
				WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(tc.claimed, tc.applied))

			repo := NewSubscriptionRepository(mock)
			outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

			require.NoError(t, err)
			assert.Equal(t, tc.wantOutcome, outcome)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("propagates db error without claiming anything", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		sub := eventSubscription()
		mock.ExpectQuery("WITH claim AS").
			WithArgs(applyArgs(sub)...).
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

		require.Error(t, err)
		assert.Empty(t, string(outcome), "a failed statement has no outcome to report")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApplySubscriptionEventKeepsItsGuardsInOneStatement(t *testing.T) {
	// The claim gate and the ordering guard are what make this write safe. A
	// refactor could drop either — or split the statement in two — and still
	// compile, so the SQL itself is asserted.
	var executed string
	capture := pgxmock.QueryMatcherFunc(func(_, actualSQL string) error {
		executed = actualSQL
		return nil
	})
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(capture))
	require.NoError(t, err)
	defer mock.Close()

	sub := eventSubscription()
	mock.ExpectQuery("").
		WithArgs(applyArgs(sub)...).
		WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))

	repo := NewSubscriptionRepository(mock)
	_, err = repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)
	require.NoError(t, err)

	assert.Contains(t, executed, "INSERT INTO webhook_events")
	assert.Contains(t, executed, "INSERT INTO subscriptions")
	assert.Contains(t, executed, "FROM claim",
		"the entitlement write must be gated on winning the claim")
	assert.Contains(t, executed, "EXCLUDED.last_event_at > subscriptions.last_event_at",
		"the lifecycle-ordering guard must live in the SQL WHERE, not in a prior read")
	assert.NotContains(t, executed, "EXCLUDED.last_event_at >= subscriptions.last_event_at",
		"the guard must be strict: an event carrying the same `changed` describes a change "+
			"already accounted for, so replaying it could only undo a correct write")
	assert.NotContains(t, executed, ";",
		"one statement only — separate statements would not roll back together")
	require.NoError(t, mock.ExpectationsWereMet())
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

		rows := pgxmock.NewRows([]string{"email", "name", "locale"}).
			AddRow("buyer@example.com", "Test Buyer", "en")
		mock.ExpectQuery("SELECT email, name, locale FROM users").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		contact, err := repo.GetUserContact(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "buyer@example.com", contact.Email)
		first, last := contact.FirstLast()
		assert.Equal(t, "Test", first)
		assert.Equal(t, "Buyer", last)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps a missing user to ErrSubscriptionNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT email, name, locale FROM users").
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

		mock.ExpectQuery("SELECT email, name, locale FROM users").
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
	mock.ExpectQuery(`external_account_id = COALESCE\(EXCLUDED\.external_account_id, subscriptions\.external_account_id\)`).
		WithArgs(applyArgs(sub)...).
		WillReturnRows(pgxmock.NewRows([]string{"claimed", "applied"}).AddRow(true, true))

	repo := NewSubscriptionRepository(mock)
	outcome, err := repo.ApplySubscriptionEvent(context.Background(), "evt-1", "subscription.activated", sub)

	require.NoError(t, err)
	assert.Equal(t, model.WebhookApplied, outcome)
	require.NoError(t, mock.ExpectationsWereMet())
}
