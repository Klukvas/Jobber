package mcp

import (
	"errors"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const auditInsertSQL = `INSERT INTO mcp_audit_log
 (id, user_id, api_token_id, tool, ok, error_code, duration_ms)
 VALUES ($1, $2, $3, $4, $5, $6, $7)`

const auditPruneSQL = `DELETE FROM mcp_audit_log WHERE user_id = $1 AND id IN
 (SELECT id FROM mcp_audit_log WHERE user_id = $1 ORDER BY created_at DESC, id DESC OFFSET $2)`

func auditMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, mock.ExpectationsWereMet()); mock.Close() })
	return mock
}

func TestAuditRepositoryRecord(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success with nullable token"
		entry := AuditEntry{ID: jobA, UserID: userA, Tool: "get_job", OK: true, DurationMS: 23}
		if failed {
			name = "tool error with token"
			entry.OK, entry.APITokenID, entry.ErrorCode = false, optionalText(tokenA), optionalText("not_found")
		}
		t.Run(name, func(t *testing.T) {
			mock := auditMock(t)
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 FOR NO KEY UPDATE`).WithArgs(userA).
				WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(userA))
			// Exact columns and arguments also guard against storing input/output text.
			mock.ExpectExec(regexp.QuoteMeta(auditInsertSQL)).WithArgs(entry.ID, userA, entry.APITokenID, entry.Tool, entry.OK, entry.ErrorCode, 23).
				WillReturnResult(pgxmock.NewResult("INSERT", 1))
			mock.ExpectExec(regexp.QuoteMeta(auditPruneSQL)).WithArgs(userA, MaxAuditRowsPerUser).
				WillReturnResult(pgxmock.NewResult("DELETE", 1))
			mock.ExpectCommit()
			require.NoError(t, NewAuditRepository(mock).Record(t.Context(), entry))
		})
	}
}

func TestAuditRepositoryFailuresRollback(t *testing.T) {
	for _, phase := range []string{"begin", "lock", "insert", "prune", "commit"} {
		t.Run(phase, func(t *testing.T) {
			mock := auditMock(t)
			failure := errors.New("database unavailable")
			begin := mock.ExpectBegin()
			if phase == "begin" {
				begin.WillReturnError(failure)
			} else {
				lock := mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 FOR NO KEY UPDATE`).WithArgs(userA)
				if phase == "lock" {
					lock.WillReturnError(failure)
				} else {
					lock.WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(userA))
					insert := mock.ExpectExec(regexp.QuoteMeta(auditInsertSQL)).WithArgs("", userA, (*string)(nil), "", false, (*string)(nil), 0)
					if phase == "insert" {
						insert.WillReturnError(failure)
					} else {
						insert.WillReturnResult(pgxmock.NewResult("INSERT", 1))
						prune := mock.ExpectExec(regexp.QuoteMeta(auditPruneSQL)).WithArgs(userA, MaxAuditRowsPerUser)
						if phase == "prune" {
							prune.WillReturnError(failure)
						} else {
							prune.WillReturnResult(pgxmock.NewResult("DELETE", 0))
							mock.ExpectCommit().WillReturnError(failure)
						}
					}
				}
				mock.ExpectRollback()
			}
			err := NewAuditRepository(mock).Record(t.Context(), AuditEntry{UserID: userA})
			require.ErrorIs(t, err, failure)
		})
	}
}
