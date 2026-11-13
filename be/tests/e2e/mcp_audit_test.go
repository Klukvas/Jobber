//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	patHandler "github.com/andreypavlenko/jobber/modules/apitokens/handler"
	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	patRepo "github.com/andreypavlenko/jobber/modules/apitokens/repository"
	patService "github.com/andreypavlenko/jobber/modules/apitokens/service"
	mcpModule "github.com/andreypavlenko/jobber/modules/mcp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestMCPAuditMigrationAndActivity(t *testing.T) {
	ctx := t.Context()
	dsn, _ := newThrowawayDatabase(t)
	migrator := newMigrator(t, repoMigrationsDir(t), dsn)
	db := connectTo(t, dsn)
	require.NoError(t, migrator.Migrate(56))
	require.NoError(t, migrator.Migrate(57))
	requireSchema(t, migrator, 57)
	var index string
	require.NoError(t, db.QueryRow(ctx, `SELECT indexdef FROM pg_indexes
 WHERE tablename = 'mcp_audit_log' AND indexname = 'mcp_audit_log_user_created_at_idx'`).Scan(&index))
	require.Contains(t, index, "(user_id, created_at DESC)")
	require.NoError(t, db.QueryRow(ctx, `SELECT indexdef FROM pg_indexes
 WHERE tablename = 'mcp_audit_log' AND indexname = 'mcp_audit_log_api_token_id_idx'`).Scan(&index))
	require.Contains(t, index, "(api_token_id) WHERE (api_token_id IS NOT NULL)")
	var columns int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns
 WHERE table_schema = 'public' AND table_name = 'mcp_audit_log'`).Scan(&columns))
	require.Equal(t, 8, columns, "schema must contain metadata only")
	// Exercise the real down migration and reapply before inserting fixtures.
	require.NoError(t, migrator.Migrate(56))
	var table *string
	require.NoError(t, db.QueryRow(ctx, `SELECT to_regclass('public.mcp_audit_log')::text`).Scan(&table))
	require.Nil(t, table)
	require.NoError(t, migrator.Migrate(57))

	owner, other := seedAuditUser(t, db), seedAuditUser(t, db)
	tokenID := uuid.NewString()
	_, err := db.Exec(ctx, `INSERT INTO api_tokens (id, user_id, name, token_prefix, token_hash)
 VALUES ($1, $2, 'Laptop', 'jbr_pat_test', $3)`, tokenID, owner, uuid.NewString())
	require.NoError(t, err)
	recorder := mcpModule.NewAuditRepository(db)
	otherEntry := mcpModule.AuditEntry{ID: uuid.NewString(), UserID: other, Tool: "list_jobs", OK: true, DurationMS: 2}
	require.NoError(t, recorder.Record(ctx, otherEntry))
	// Fill the cap cheaply, with explicit times so oldest-row pruning is unambiguous.
	_, err = db.Exec(ctx, `INSERT INTO mcp_audit_log (id, user_id, tool, ok, duration_ms, created_at)
 SELECT md5('audit-' || n::text)::uuid, $1, 'list_jobs', true, n,
 now() - interval '1 day' + n * interval '1 second' FROM generate_series(1, $2) n`, owner, mcpModule.MaxAuditRowsPerUser)
	require.NoError(t, err)
	code := "not_found"
	latest := mcpModule.AuditEntry{ID: uuid.NewString(), UserID: owner, APITokenID: &tokenID,
		Tool: "get_job", OK: false, ErrorCode: &code, DurationMS: 17}
	require.NoError(t, recorder.Record(ctx, latest))
	var count, oldest int
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*), min(duration_ms) FROM mcp_audit_log WHERE user_id = $1`, owner).Scan(&count, &oldest))
	require.Equal(t, mcpModule.MaxAuditRowsPerUser, count)
	require.Equal(t, 2, oldest, "oldest fixture must be pruned")
	rows, err := patRepo.NewAPITokenRepository(db).ListActivity(ctx, owner, 100)
	require.NoError(t, err)
	require.Len(t, rows, 100)
	require.Equal(t, latest.ID, rows[0].ID)
	require.Equal(t, "Laptop", *rows[0].TokenName)
	require.Equal(t, tokenID, *rows[0].APITokenID)
	require.Equal(t, code, *rows[0].ErrorCode)
	require.Equal(t, latest.DurationMS, rows[0].DurationMS)
	require.False(t, rows[0].OK)
	for i := 1; i < len(rows); i++ {
		require.False(t, rows[i].CreatedAt.After(rows[i-1].CreatedAt))
		require.NotEqual(t, otherEntry.ID, rows[i].ID)
	}
	otherRows, err := patRepo.NewAPITokenRepository(db).ListActivity(ctx, other, 20)
	require.NoError(t, err)
	require.Len(t, otherRows, 1, "pruning one owner must not touch another owner")
	require.Equal(t, otherEntry.ID, otherRows[0].ID)

	// The JWT endpoint uses the same DB, including after token deletion/downgrade.
	_, err = db.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1`, tokenID)
	require.NoError(t, err)
	rows = auditEndpointRows(t, db, owner)
	require.Len(t, rows, 20)
	require.Equal(t, latest.ID, rows[0].ID)
	require.Nil(t, rows[0].APITokenID, "token FK must become NULL")
	require.Nil(t, rows[0].TokenName)

	// Parallel inserts at the cap must still leave exactly 500 rows.
	var workers sync.WaitGroup
	failures := make(chan error, 12)
	for range 12 {
		workers.Go(func() {
			failures <- recorder.Record(ctx, mcpModule.AuditEntry{ID: uuid.NewString(), UserID: owner, Tool: "list_jobs", OK: true})
		})
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM mcp_audit_log WHERE user_id = $1`, owner).Scan(&count))
	require.Equal(t, mcpModule.MaxAuditRowsPerUser, count)
	_, err = db.Exec(ctx, `DELETE FROM users WHERE id = $1`, owner)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(ctx, `SELECT count(*) FROM mcp_audit_log WHERE user_id = $1`, owner).Scan(&count))
	require.Zero(t, count, "user FK must cascade")
}

func seedAuditUser(t *testing.T, db *pgxpool.Pool) string {
	t.Helper()
	id := uuid.NewString()
	_, err := db.Exec(t.Context(), `INSERT INTO users (id, email, name, password_hash, locale)
 VALUES ($1, $2, '', 'test', 'en')`, id, id+"@audit.test")
	require.NoError(t, err)
	return id
}

func auditEndpointRows(t *testing.T, db *pgxpool.Pool, owner string) []*patModel.ActivityDTO {
	t.Helper()
	repository := patRepo.NewAPITokenRepository(db)
	service := patService.NewAPITokenService(repository, nil, nil, nil, patService.WithActivityRepository(repository))
	jwt := auth.NewJWTManager("audit-test-access", "audit-test-refresh", time.Hour, time.Hour)
	token, err := jwt.GenerateAccessToken(owner)
	require.NoError(t, err)
	router := gin.New()
	patHandler.NewAPITokenHandler(service).RegisterRoutes(router.Group("/api/v1"), auth.AuthMiddleware(jwt))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens/activity", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 200, response.Code, response.Body.String())
	var items []*patModel.ActivityDTO
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &items))
	return items
}

// Verify that inserting a bad FK never leaves a partial row or prunes history.
func TestMCPAuditInsertRollback(t *testing.T) {
	owner := seedAuditUser(t, pool)
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, owner)
		require.NoError(t, err)
	})
	tokenID := uuid.NewString()
	err := mcpModule.NewAuditRepository(pool).Record(t.Context(), mcpModule.AuditEntry{
		ID: uuid.NewString(), UserID: owner, APITokenID: &tokenID, Tool: "get_job", OK: true,
	})
	require.Error(t, err)
	rows, err := patRepo.NewAPITokenRepository(pool).ListActivity(t.Context(), owner, 20)
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestMCPAuditWriterDrain(t *testing.T) {
	owner := seedAuditUser(t, pool)
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, owner)
		require.NoError(t, err)
	})
	writer := mcpModule.NewAuditWriter(mcpModule.NewAuditRepository(pool), nil)
	writer.Start(t.Context())
	ids := []string{uuid.NewString(), uuid.NewString()}
	for _, id := range ids {
		require.True(t, writer.Enqueue(mcpModule.AuditEntry{ID: id, UserID: owner, Tool: "list_jobs", OK: true}))
	}
	require.NoError(t, writer.Close())
	require.Zero(t, writer.Dropped())
	rows, err := patRepo.NewAPITokenRepository(pool).ListActivity(t.Context(), owner, 20)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	gotIDs := []string{rows[0].ID, rows[1].ID}
	require.ElementsMatch(t, ids, gotIDs, "Close must persist each queued entry exactly once before pool shutdown")
}
