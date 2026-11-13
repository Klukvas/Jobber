package repository

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

const listActivitySQL = `SELECT a.id, a.tool, a.ok, a.error_code, a.duration_ms,
 a.created_at, a.api_token_id, t.name
 FROM mcp_audit_log a LEFT JOIN api_tokens t ON t.id = a.api_token_id AND t.user_id = a.user_id
 WHERE a.user_id = $1 ORDER BY a.created_at DESC, a.id DESC LIMIT $2`

func activityRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"id", "tool", "ok", "error_code", "duration_ms", "created_at", "api_token_id", "name"})
}

func TestListActivity(t *testing.T) {
	mock := newMock(t)
	now := time.Now().UTC()
	code, tokenID, tokenName := "not_found", "token", "laptop"
	mock.ExpectQuery(regexp.QuoteMeta(listActivitySQL)).WithArgs("owner", 20).
		WillReturnRows(activityRows().
			AddRow("newer", "get_job", false, &code, 7, now, &tokenID, &tokenName).
			AddRow("older", "list_jobs", true, nil, 5, now.Add(-time.Hour), nil, nil)).RowsWillBeClosed()
	items, err := NewAPITokenRepository(mock).ListActivity(t.Context(), "owner", 20)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "newer", items[0].ID)
	require.False(t, items[0].OK)
	require.Equal(t, "not_found", *items[0].ErrorCode)
	require.Equal(t, 7, items[0].DurationMS)
	require.Equal(t, now, items[0].CreatedAt)
	require.Equal(t, "token", *items[0].APITokenID)
	require.Equal(t, "laptop", *items[0].TokenName)
	require.True(t, items[1].OK)
	require.Nil(t, items[1].ErrorCode)
	require.Nil(t, items[1].APITokenID)
	require.Nil(t, items[1].TokenName)
}

func TestListActivityEmptyAndFailures(t *testing.T) {
	for _, phase := range []string{"empty", "query", "scan", "iterate"} {
		t.Run(phase, func(t *testing.T) {
			mock := newMock(t)
			query := mock.ExpectQuery(regexp.QuoteMeta(listActivitySQL)).WithArgs("owner", 1)
			switch phase {
			case "query":
				query.WillReturnError(errors.New("read failed"))
			case "scan":
				query.WillReturnRows(activityRows().AddRow("id", "tool", "invalid bool", nil, 0, time.Now(), nil, nil)).RowsWillBeClosed()
			case "iterate":
				query.WillReturnRows(activityRows().AddRow("id", "tool", true, nil, 0, time.Now(), nil, nil).
					RowError(0, errors.New("iteration failed"))).RowsWillBeClosed()
			default:
				query.WillReturnRows(activityRows()).RowsWillBeClosed()
			}
			items, err := NewAPITokenRepository(mock).ListActivity(t.Context(), "owner", 1)
			if phase == "empty" {
				require.NoError(t, err)
				require.NotNil(t, items)
				require.Empty(t, items)
			} else {
				require.Error(t, err)
			}
		})
	}
}
