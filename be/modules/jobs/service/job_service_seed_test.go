package service

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SeedDefaultStageTemplates writes the starter pipeline in one transaction. The
// tests drive that transaction through the pgxmock pool.

func TestJobService_SeedDefaultStageTemplates(t *testing.T) {
	userID := stagesUserID

	seedSvc := func(pool txBeginner) *JobService {
		return svcWith(pool, &MockJobRepository{}, nil, &MockStageTemplateRepository{}, nil)
	}

	t.Run("inserts every default column in one committed transaction", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		// One INSERT per default column (7 of them).
		for range defaultPipeline {
			mock.ExpectExec(`INSERT INTO stage_templates`).WithArgs(
				pgxmock.AnyArg(), userID, pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		}
		mock.ExpectCommit()

		err := seedSvc(mock).SeedDefaultStageTemplates(context.Background(), userID)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("writes exactly the default pipeline columns in order", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		for _, col := range defaultPipeline {
			mock.ExpectExec(`INSERT INTO stage_templates`).WithArgs(
				pgxmock.AnyArg(), userID, col.Name, col.Order, pgxmock.AnyArg(),
			).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		}
		mock.ExpectCommit()

		err := seedSvc(mock).SeedDefaultStageTemplates(context.Background(), userID)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps a begin error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin().WillReturnError(errors.New("no conn"))

		err := seedSvc(mock).SeedDefaultStageTemplates(context.Background(), userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "begin seed transaction")
	})

	t.Run("wraps an insert error and names the offending column", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		anyFive := func() *pgxmock.ExpectedExec {
			return mock.ExpectExec(`INSERT INTO stage_templates`).WithArgs(
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			)
		}
		// First column inserts fine, the second fails.
		anyFive().WillReturnResult(pgxmock.NewResult("INSERT", 1))
		anyFive().WillReturnError(errors.New("duplicate key"))

		err := seedSvc(mock).SeedDefaultStageTemplates(context.Background(), userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "seed column")
		// The second default column is "Applied".
		assert.Contains(t, err.Error(), defaultPipeline[1].Name)
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		for range defaultPipeline {
			mock.ExpectExec(`INSERT INTO stage_templates`).WithArgs(
				pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		}
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		err := seedSvc(mock).SeedDefaultStageTemplates(context.Background(), userID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "commit seed transaction")
	})
}
