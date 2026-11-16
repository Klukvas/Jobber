package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchScoreCacheRepository_Get(t *testing.T) {
	t.Run("returns unmarshalled result", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		raw := []byte(`{"overall_score":88,"summary":"good fit","from_cache":false}`)
		rows := pgxmock.NewRows([]string{"result"}).AddRow(raw)
		mock.ExpectQuery(`SELECT result FROM match_score_cache WHERE user_id = \$1 AND job_id = \$2 AND resume_id = \$3`).
			WithArgs("user-1", "job-1", "resume-1").
			WillReturnRows(rows)

		repo := NewMatchScoreCacheRepository(mock)
		got, err := repo.Get(context.Background(), "user-1", "job-1", "resume-1")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, 88, got.OverallScore)
		assert.Equal(t, "good fit", got.Summary)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil,nil on cache miss", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT result FROM match_score_cache WHERE user_id = \$1 AND job_id = \$2 AND resume_id = \$3`).
			WithArgs("user-1", "job-1", "resume-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewMatchScoreCacheRepository(mock)
		got, err := repo.Get(context.Background(), "user-1", "job-1", "resume-1")
		require.NoError(t, err)
		assert.Nil(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery(`SELECT result FROM match_score_cache WHERE user_id = \$1 AND job_id = \$2 AND resume_id = \$3`).
			WithArgs("user-1", "job-1", "resume-1").
			WillReturnError(errors.New("boom"))

		repo := NewMatchScoreCacheRepository(mock)
		got, err := repo.Get(context.Background(), "user-1", "job-1", "resume-1")
		assert.Nil(t, got)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"result"}).AddRow([]byte(`{not json`))
		mock.ExpectQuery(`SELECT result FROM match_score_cache WHERE user_id = \$1 AND job_id = \$2 AND resume_id = \$3`).
			WithArgs("user-1", "job-1", "resume-1").
			WillReturnRows(rows)

		repo := NewMatchScoreCacheRepository(mock)
		got, err := repo.Get(context.Background(), "user-1", "job-1", "resume-1")
		assert.Nil(t, got)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMatchScoreCacheRepository_Upsert(t *testing.T) {
	t.Run("marshals result and upserts", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// Marshalled JSON is opaque; AnyArg keeps the assertion resilient.
		mock.ExpectExec("INSERT INTO match_score_cache").
			WithArgs("user-1", "job-1", "resume-1", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewMatchScoreCacheRepository(mock)
		result := &model.MatchScoreResponse{OverallScore: 75, Summary: "ok"}
		require.NoError(t, repo.Upsert(context.Background(), "user-1", "job-1", "resume-1", result))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO match_score_cache").
			WithArgs("user-1", "job-1", "resume-1", pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewMatchScoreCacheRepository(mock)
		err = repo.Upsert(context.Background(), "user-1", "job-1", "resume-1", &model.MatchScoreResponse{})
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMatchScoreCacheRepository_InvalidateByJob(t *testing.T) {
	t.Run("deletes rows for job", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM match_score_cache WHERE job_id").
			WithArgs("job-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 3))

		repo := NewMatchScoreCacheRepository(mock)
		require.NoError(t, repo.InvalidateByJob(context.Background(), "job-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM match_score_cache WHERE job_id").
			WithArgs("job-1").
			WillReturnError(errors.New("boom"))

		repo := NewMatchScoreCacheRepository(mock)
		assert.Error(t, repo.InvalidateByJob(context.Background(), "job-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMatchScoreCacheRepository_InvalidateByResume(t *testing.T) {
	t.Run("deletes rows for resume", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM match_score_cache WHERE resume_id").
			WithArgs("resume-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 2))

		repo := NewMatchScoreCacheRepository(mock)
		require.NoError(t, repo.InvalidateByResume(context.Background(), "resume-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM match_score_cache WHERE resume_id").
			WithArgs("resume-1").
			WillReturnError(errors.New("boom"))

		repo := NewMatchScoreCacheRepository(mock)
		assert.Error(t, repo.InvalidateByResume(context.Background(), "resume-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMatchScoreCacheRepository_AtomicCreditResult(t *testing.T) {
	for _, scenario := range []string{"success", "save failure", "charge failure", "commit failure"} {
		t.Run(scenario, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			ctx := ai.CreditContext(context.Background(), func() string { return "user-1" }, "operation-1", "match")
			ai.MarkCreditAdmission(ctx)
			mock.ExpectBegin()
			save := mock.ExpectExec("INSERT INTO match_score_cache").WithArgs("user-1", "job-1", "resume-1", pgxmock.AnyArg())
			if scenario == "save failure" {
				save.WillReturnError(assert.AnError)
				mock.ExpectRollback()
			} else {
				save.WillReturnResult(pgxmock.NewResult("INSERT", 1))
				mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("ai-quota:user-1").WillReturnResult(pgxmock.NewResult("SELECT", 1))
				charge := mock.ExpectExec("UPDATE ai_credit_operations SET charge_state='charged'").WithArgs("user-1", "operation-1")
				if scenario == "charge failure" {
					charge.WillReturnResult(pgxmock.NewResult("UPDATE", 0))
					mock.ExpectRollback()
				} else {
					charge.WillReturnResult(pgxmock.NewResult("UPDATE", 1))
					if scenario == "commit failure" {
						mock.ExpectCommit().WillReturnError(assert.AnError)
						mock.ExpectRollback()
					} else {
						mock.ExpectCommit()
					}
				}
			}
			err = NewMatchScoreCacheRepository(mock).Upsert(ctx, "user-1", "job-1", "resume-1", &model.MatchScoreResponse{OverallScore: 75})
			_, durable, _ := ai.CurrentCreditOperation(ctx)
			if scenario == "success" {
				require.NoError(t, err)
				assert.True(t, durable)
				assert.True(t, ai.CreditResultReady(ctx))
			} else {
				require.Error(t, err)
				assert.False(t, durable)
				assert.False(t, ai.CreditResultReady(ctx))
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
