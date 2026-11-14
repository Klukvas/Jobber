package oauth

import (
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestCleanupMetadataClients(t *testing.T) {
	for _, mode := range []string{"delete", "select error", "scan error", "grant error", "client error"} {
		t.Run(mode, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			failure := errors.New("database unavailable")
			mock.ExpectBegin()
			query := mock.ExpectQuery(`(?s)SELECT c.client_id.*c.is_metadata_document.*fetched_at.*24 hours.*NOT EXISTS.*g.revoked_at IS NULL.*LIMIT 100 FOR UPDATE OF c SKIP LOCKED`)
			if mode == "select error" {
				query.WillReturnError(failure)
			} else if mode == "scan error" {
				query.WillReturnRows(pgxmock.NewRows([]string{"client_id"}).AddRow(nil)).RowsWillBeClosed()
			} else {
				query.WillReturnRows(pgxmock.NewRows([]string{"client_id"}).AddRow("https://client.test/meta")).RowsWillBeClosed()
				expectClientCleanupDeletes(mock, mode, failure)
			}
			if mode == "delete" {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			err = NewRepository(mock).cleanupClients(t.Context())
			if mode == "delete" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "clean oauth metadata clients")
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func expectClientCleanupDeletes(mock pgxmock.PgxPoolIface, mode string, failure error) {
	ids := []string{"https://client.test/meta"}
	grants := mock.ExpectExec(`(?s)DELETE FROM oauth_grants.*client_id=ANY\(\$1\) AND revoked_at IS NOT NULL.*LIMIT 100 FOR UPDATE SKIP LOCKED`).WithArgs(ids)
	if mode == "grant error" {
		grants.WillReturnError(failure)
		return
	}
	grants.WillReturnResult(pgxmock.NewResult("DELETE", 1))
	clients := mock.ExpectExec(`(?s)DELETE FROM oauth_clients.*NOT EXISTS.*oauth_grants`).WithArgs(ids)
	if mode == "client error" {
		clients.WillReturnError(failure)
	} else {
		clients.WillReturnResult(pgxmock.NewResult("DELETE", 1))
	}
}
