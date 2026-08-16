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

func subscriptionColumns() []string {
	return []string{
		"id", "user_id", "paddle_subscription_id", "paddle_customer_id",
		"status", "plan", "current_period_start", "current_period_end",
		"cancel_at", "created_at", "updated_at",
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
		rows := pgxmock.NewRows(subscriptionColumns()).AddRow(
			"sub-1", "user-1", &psid, &pcid, "active", "pro",
			&now, &now, (*time.Time)(nil), now, now,
		)

		mock.ExpectQuery("SELECT id, user_id, paddle_subscription_id, paddle_customer_id").
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

		mock.ExpectQuery("SELECT id, user_id, paddle_subscription_id, paddle_customer_id").
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

		mock.ExpectQuery("SELECT id, user_id, paddle_subscription_id, paddle_customer_id").
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

func TestSubscriptionRepository_GetByPaddleSubscriptionID(t *testing.T) {
	t.Run("returns subscription", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		psid := "psub-1"
		rows := pgxmock.NewRows(subscriptionColumns()).AddRow(
			"sub-1", "user-1", &psid, (*string)(nil), "active", "pro",
			(*time.Time)(nil), (*time.Time)(nil), (*time.Time)(nil), now, now,
		)

		mock.ExpectQuery("WHERE paddle_subscription_id = ").
			WithArgs("psub-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByPaddleSubscriptionID(context.Background(), "psub-1")
		require.NoError(t, err)
		assert.Equal(t, "sub-1", sub.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrSubscriptionNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("WHERE paddle_subscription_id = ").
			WithArgs("psub-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByPaddleSubscriptionID(context.Background(), "psub-1")
		assert.Nil(t, sub)
		assert.ErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("WHERE paddle_subscription_id = ").
			WithArgs("psub-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		sub, err := repo.GetByPaddleSubscriptionID(context.Background(), "psub-1")
		assert.Nil(t, sub)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrSubscriptionNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_Upsert(t *testing.T) {
	t.Run("upserts and populates generated fields", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		psid := "psub-1"
		pcid := "pcust-1"
		sub := &model.Subscription{
			UserID:               "user-1",
			PaddleSubscriptionID: &psid,
			PaddleCustomerID:     &pcid,
			Status:               "active",
			Plan:                 "pro",
			CurrentPeriodStart:   &now,
			CurrentPeriodEnd:     &now,
			CancelAt:             nil,
		}

		rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("sub-1", now, now)
		mock.ExpectQuery("INSERT INTO subscriptions").
			WithArgs(
				sub.UserID, sub.PaddleSubscriptionID, sub.PaddleCustomerID,
				sub.Status, sub.Plan, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CancelAt,
			).
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		require.NoError(t, repo.Upsert(context.Background(), sub))
		assert.Equal(t, "sub-1", sub.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("INSERT INTO subscriptions").
			WithArgs(
				"user-1", (*string)(nil), (*string)(nil),
				"free", "free", (*time.Time)(nil), (*time.Time)(nil), (*time.Time)(nil),
			).
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		err = repo.Upsert(context.Background(), &model.Subscription{
			UserID: "user-1", Status: "free", Plan: "free",
		})
		assert.Error(t, err)
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
		countTest(t, "usage_type = 'match_score'", func(r *SubscriptionRepository) (int, error) {
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

func TestSubscriptionRepository_GetAllCounts(t *testing.T) {
	t.Run("returns all six counts in order", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"jobs", "resumes", "ai", "parses", "builders", "letters"}).
			AddRow(1, 2, 3, 4, 5, 6)
		mock.ExpectQuery("SELECT").
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

func TestSubscriptionRepository_WebhookEventExists(t *testing.T) {
	t.Run("returns true when event exists", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs("evt-1").
			WillReturnRows(rows)

		repo := NewSubscriptionRepository(mock)
		exists, err := repo.WebhookEventExists(context.Background(), "evt-1")
		require.NoError(t, err)
		assert.True(t, exists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT EXISTS").
			WithArgs("evt-1").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		_, err = repo.WebhookEventExists(context.Background(), "evt-1")
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_RecordWebhookEvent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO webhook_events").
		WithArgs("evt-1", "subscription.updated").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewSubscriptionRepository(mock)
	require.NoError(t, repo.RecordWebhookEvent(context.Background(), "evt-1", "subscription.updated"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionRepository_TryClaimWebhookEvent(t *testing.T) {
	t.Run("returns true when row inserted (won the race)", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO webhook_events").
			WithArgs("evt-1", "type").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewSubscriptionRepository(mock)
		claimed, err := repo.TryClaimWebhookEvent(context.Background(), "evt-1", "type")
		require.NoError(t, err)
		assert.True(t, claimed)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when conflict (already processed)", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO webhook_events").
			WithArgs("evt-1", "type").
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		repo := NewSubscriptionRepository(mock)
		claimed, err := repo.TryClaimWebhookEvent(context.Background(), "evt-1", "type")
		require.NoError(t, err)
		assert.False(t, claimed)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO webhook_events").
			WithArgs("evt-1", "type").
			WillReturnError(errors.New("boom"))

		repo := NewSubscriptionRepository(mock)
		claimed, err := repo.TryClaimWebhookEvent(context.Background(), "evt-1", "type")
		assert.Error(t, err)
		assert.False(t, claimed)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSubscriptionRepository_ReleaseWebhookEvent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM webhook_events").
		WithArgs("evt-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewSubscriptionRepository(mock)
	require.NoError(t, repo.ReleaseWebhookEvent(context.Background(), "evt-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}
