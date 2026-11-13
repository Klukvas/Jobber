//go:build integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func runSnapshotPostgresContenders(t *testing.T, f *tailoringIntegrationFixture, actions []func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	blocker, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer blocker.Rollback(context.Background()) //nolint:errcheck // no-op after commit
	var resumeID string
	err = blocker.QueryRow(ctx, `SELECT id FROM resume_builders WHERE id=$1 FOR UPDATE`, f.sourceID).Scan(&resumeID)
	require.NoError(t, err)
	var blockerPID int
	require.NoError(t, blocker.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID))
	start := make(chan struct{})
	ready := make(chan struct{}, len(actions))
	results := make(chan error, len(actions))
	for _, action := range actions {
		go func() {
			ready <- struct{}{}
			select {
			case <-start:
				results <- action(ctx)
			case <-ctx.Done():
				results <- ctx.Err()
			}
		}()
	}
	for range actions {
		<-ready
	}
	close(start)
	// Assert actual production parent locks, not a mutex in a fake. Removing the
	// lock or taking section locks first makes this bounded database barrier fail.
	require.NoError(t, awaitSnapshotParentLocks(ctx, f.pool, blockerPID, len(actions)))
	require.NoError(t, blocker.Commit(ctx))
	for range actions {
		select {
		case err := <-results:
			require.NoError(t, err, "concurrent snapshot/reorder must finish without a deadlock")
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

func awaitSnapshotParentLocks(ctx context.Context, pool *pgxpool.Pool, blockerPID, want int) error {
	const query = `WITH RECURSIVE waiters AS (
		SELECT pid, query FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid))
		UNION
		SELECT a.pid, a.query FROM pg_stat_activity a JOIN waiters w ON w.pid = ANY(pg_blocking_pids(a.pid))
	) SELECT count(*) FROM waiters
		WHERE query LIKE 'SELECT id FROM resume_builders WHERE id =%'
		AND query LIKE '%FOR UPDATE'`
	for {
		var count int
		if err := pool.QueryRow(ctx, query, blockerPID).Scan(&count); err != nil {
			return fmt.Errorf("wait for %d real parent locks: %w", want, err)
		}
		if count == want {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("only %d of %d transactions reached the parent-lock barrier: %w", count, want, err)
		}
	}
}
