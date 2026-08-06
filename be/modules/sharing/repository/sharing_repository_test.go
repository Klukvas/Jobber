package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snapshotFixture() model.StatsSnapshot {
	return model.StatsSnapshot{
		SchemaVersion: model.SnapshotSchemaVersion,
		GeneratedAt:   time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
		Overview:      model.OverviewSnapshot{TotalApplications: 127, ResponseRate: 18.5},
		Funnel: []model.FunnelStageSnapshot{
			{StageName: "Applied", StageOrder: 1, Count: 127, ConversionRate: 100},
		},
	}
}

func TestSharingRepository_Create(t *testing.T) {
	t.Run("inserts share under the cap", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		share := &model.SharedStats{
			UserID:   "user-123",
			Token:    "token-1",
			Snapshot: snapshotFixture(),
		}

		mock.ExpectExec("INSERT INTO shared_stats").
			WithArgs(pgxmock.AnyArg(), share.UserID, share.Token, pgxmock.AnyArg(), pgxmock.AnyArg(), model.MaxActiveShares).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewSharingRepository(mock)
		err = repo.Create(context.Background(), share)

		require.NoError(t, err)
		assert.NotEmpty(t, share.ID)
		assert.False(t, share.CreatedAt.IsZero())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns limit error when cap reached", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO shared_stats").
			WithArgs(pgxmock.AnyArg(), "user-123", "token-1", pgxmock.AnyArg(), pgxmock.AnyArg(), model.MaxActiveShares).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		repo := NewSharingRepository(mock)
		err = repo.Create(context.Background(), &model.SharedStats{
			UserID:   "user-123",
			Token:    "token-1",
			Snapshot: snapshotFixture(),
		})

		assert.ErrorIs(t, err, model.ErrShareLimitReached)
	})

	t.Run("wraps database errors", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		dbErr := errors.New("connection lost")
		mock.ExpectExec("INSERT INTO shared_stats").
			WithArgs(pgxmock.AnyArg(), "user-123", "token-1", pgxmock.AnyArg(), pgxmock.AnyArg(), model.MaxActiveShares).
			WillReturnError(dbErr)

		repo := NewSharingRepository(mock)
		err = repo.Create(context.Background(), &model.SharedStats{
			UserID:   "user-123",
			Token:    "token-1",
			Snapshot: snapshotFixture(),
		})

		assert.ErrorIs(t, err, dbErr)
	})
}

func TestSharingRepository_GetByToken(t *testing.T) {
	columns := []string{"id", "user_id", "token", "snapshot", "created_at"}

	t.Run("returns share with decoded snapshot", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		snapshot := snapshotFixture()
		snapshotJSON, err := json.Marshal(snapshot)
		require.NoError(t, err)

		mock.ExpectQuery("SELECT (.+) FROM shared_stats").
			WithArgs("token-1").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("share-1", "user-123", "token-1", snapshotJSON, time.Now().UTC()))

		repo := NewSharingRepository(mock)
		share, err := repo.GetByToken(context.Background(), "token-1")

		require.NoError(t, err)
		assert.Equal(t, "share-1", share.ID)
		assert.Equal(t, 127, share.Snapshot.Overview.TotalApplications)
		require.Len(t, share.Snapshot.Funnel, 1)
		assert.Equal(t, "Applied", share.Snapshot.Funnel[0].StageName)
	})

	t.Run("returns not found for unknown token", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT (.+) FROM shared_stats").
			WithArgs("missing").
			WillReturnRows(pgxmock.NewRows(columns))

		repo := NewSharingRepository(mock)
		share, err := repo.GetByToken(context.Background(), "missing")

		assert.Nil(t, share)
		assert.ErrorIs(t, err, model.ErrShareNotFound)
	})
}

func TestSharingRepository_ListByUser(t *testing.T) {
	columns := []string{"id", "user_id", "token", "snapshot", "created_at"}

	t.Run("returns user shares", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		snapshotJSON, err := json.Marshal(snapshotFixture())
		require.NoError(t, err)

		mock.ExpectQuery("SELECT (.+) FROM shared_stats").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(columns).
				AddRow("share-1", "user-123", "token-1", snapshotJSON, time.Now().UTC()).
				AddRow("share-2", "user-123", "token-2", snapshotJSON, time.Now().UTC()))

		repo := NewSharingRepository(mock)
		shares, err := repo.ListByUser(context.Background(), "user-123")

		require.NoError(t, err)
		require.Len(t, shares, 2)
		assert.Equal(t, "token-2", shares[1].Token)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT (.+) FROM shared_stats").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(columns))

		repo := NewSharingRepository(mock)
		shares, err := repo.ListByUser(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Empty(t, shares)
	})
}

func TestSharingRepository_Delete(t *testing.T) {
	t.Run("deletes own share", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM shared_stats").
			WithArgs("share-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := NewSharingRepository(mock)
		err = repo.Delete(context.Background(), "user-123", "share-1")

		require.NoError(t, err)
	})

	t.Run("returns not found for foreign share", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM shared_stats").
			WithArgs("share-1", "user-456").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := NewSharingRepository(mock)
		err = repo.Delete(context.Background(), "user-456", "share-1")

		assert.ErrorIs(t, err, model.ErrShareNotFound)
	})
}
