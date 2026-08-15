// Package backfill populates jobs.current_stage_template_id for the
// single-pipeline (Status/Stage) refactor introduced by migration 000040.
//
// Migration 000040 added the nullable columns jobs.current_stage_template_id
// and jobs.is_archived, and raised is_archived for old status='archived' cards.
// It deliberately KEPT the old jobs.status and jobs.current_stage_id columns so
// a rolled-back OLD image still runs. This package puts every existing card into
// a pipeline column by resolving current_stage_template_id, creating the four
// canonical columns (Wishlist / Applied / Offer / Rejected) per user on demand.
//
// It is idempotent: it only fills rows where current_stage_template_id IS NULL,
// and column creation uses INSERT ... ON CONFLICT (user_id, name) DO NOTHING.
// Re-running after a partial run (or after new cards appear) is safe.
//
// The same core is shared by two callers:
//   - cmd/backfill: a standalone CLI (apply / --dry-run) for local + manual use.
//   - cmd/api startup: a guarded call right after migrations and before serving,
//     so a fresh single-pipeline image places every card BEFORE it accepts any
//     traffic. Pending() gates that call, so it self-disables once every card is
//     placed — and after migration 041 makes the column NOT NULL, Pending is
//     always false, so the startup call becomes a permanent no-op.
package backfill

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// canonicalColumns are the pipeline columns every user must have after the
// backfill. They are appended (in this order) after the user's current max
// "order", so their existing in-progress stages keep their positions.
var canonicalColumns = []string{"Wishlist", "Applied", "Offer", "Rejected"}

// stagePhase is the phase value used for any column this package creates. The
// stage_templates.phase column still carries a NOT NULL default (until migration
// 041 drops it), so we set it explicitly. 'in_progress' is a valid value per the
// phase CHECK constraint (wishlist|applied|in_progress|offer).
const stagePhase = "in_progress"

// newStageStatus is the job_stages.status assigned to a synthesized "current"
// stage row for cards that had no history. 'active' is valid per the
// job_stages status CHECK (pending|active|completed|skipped|cancelled).
const newStageStatus = "active"

// querier is the subset of pgx behavior this package needs. Both *pgxpool.Pool
// and pgx.Tx satisfy it, which lets the same helpers run inside the dry-run
// transaction or a real per-user transaction.
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Stats aggregates counters across users for a final summary log.
type Stats struct {
	Users          int
	ColumnsCreated int
	JobsUpdated    int
	StagesAppended int
}

func (s *Stats) merge(o Stats) {
	s.Users += o.Users
	s.ColumnsCreated += o.ColumnsCreated
	s.JobsUpdated += o.JobsUpdated
	s.StagesAppended += o.StagesAppended
}

// Pending reports whether any job still has current_stage_template_id IS NULL,
// i.e. whether there is any work to do. It is the guard the API startup uses to
// skip the backfill entirely (a single cheap EXISTS) once all cards are placed.
// It only reads the current_stage_template_id column (added in 040, never
// dropped), so it stays safe to call even after migration 041.
func Pending(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM jobs WHERE current_stage_template_id IS NULL)`,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check pending backfill: %w", err)
	}
	return exists, nil
}

// RunDry executes the whole backfill in ONE transaction and rolls it back, so it
// is safe to point at a restored prod dump. The final assertion runs inside the
// transaction (against the would-be state) before the rollback.
func RunDry(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) (Stats, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("begin dry-run tx: %w", err)
	}
	// Roll back no matter what — this is a dry run.
	defer func() { _ = tx.Rollback(ctx) }()

	stats, err := backfillAll(ctx, logger, tx)
	if err != nil {
		return Stats{}, fmt.Errorf("dry-run backfill: %w", err)
	}

	if err := assertNoNullPointers(ctx, logger, tx); err != nil {
		return Stats{}, err
	}

	logger.Info("DRY RUN complete — rolling back, nothing persisted",
		"users_processed", stats.Users,
		"columns_created", stats.ColumnsCreated,
		"jobs_updated", stats.JobsUpdated,
		"stage_rows_appended", stats.StagesAppended,
	)
	return stats, nil
}

// Run processes each user in its own transaction, committing as it goes. Per-user
// transactions keep the blast radius small on large datasets while still being
// atomic per user. A final read-only assertion (outside any tx) confirms zero
// remaining NULL pointers.
func Run(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) (Stats, error) {
	userIDs, err := usersNeedingBackfillQ(ctx, pool)
	if err != nil {
		return Stats{}, err
	}
	logger.Info("users needing backfill", "count", len(userIDs))

	total := Stats{}
	for _, userID := range userIDs {
		s, err := backfillUserTx(ctx, logger, pool, userID)
		if err != nil {
			return Stats{}, fmt.Errorf("backfill user %s: %w", userID, err)
		}
		total.merge(s)
	}

	if err := assertNoNullPointers(ctx, logger, pool); err != nil {
		return Stats{}, err
	}

	logger.Info("backfill complete",
		"users_processed", total.Users,
		"columns_created", total.ColumnsCreated,
		"jobs_updated", total.JobsUpdated,
		"stage_rows_appended", total.StagesAppended,
	)
	return total, nil
}

// backfillUserTx wraps a single user's backfill in its own transaction.
func backfillUserTx(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, userID string) (Stats, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	s, err := backfillUser(ctx, logger, tx, userID)
	if err != nil {
		return Stats{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Stats{}, fmt.Errorf("commit tx: %w", err)
	}
	return s, nil
}

// backfillAll (dry-run path) processes every user needing a backfill using the
// provided querier, which is the single dry-run transaction.
func backfillAll(ctx context.Context, logger *slog.Logger, q querier) (Stats, error) {
	userIDs, err := usersNeedingBackfillQ(ctx, q)
	if err != nil {
		return Stats{}, err
	}
	logger.Info("users needing backfill (dry-run)", "count", len(userIDs))

	total := Stats{}
	for _, userID := range userIDs {
		s, err := backfillUser(ctx, logger, q, userID)
		if err != nil {
			return Stats{}, fmt.Errorf("backfill user %s: %w", userID, err)
		}
		total.merge(s)
	}
	return total, nil
}

// backfillUser is the core per-user algorithm. It is agnostic to whether q is a
// pool (dry-run tx) or a per-user tx.
//
// Steps:
//  1. Ensure the four canonical columns exist (create missing ones after the
//     user's current max "order"), then build a name->id map.
//  2. For each job with current_stage_template_id IS NULL, resolve the target
//     column id and UPDATE it.
//  3. For jobs that had NO job_stages history, append one active job_stages row
//     so the funnel/timeline has an entry.
func backfillUser(ctx context.Context, logger *slog.Logger, q querier, userID string) (Stats, error) {
	stats := Stats{Users: 1}

	created, err := ensureColumns(ctx, q, userID)
	if err != nil {
		return Stats{}, fmt.Errorf("ensure columns: %w", err)
	}
	stats.ColumnsCreated = created

	columnByName, err := columnMap(ctx, q, userID)
	if err != nil {
		return Stats{}, fmt.Errorf("build column map: %w", err)
	}
	// Guard: the canonical columns must all exist now (either pre-existing or
	// just created). If any is missing something is badly wrong — fail loud.
	for _, name := range canonicalColumns {
		if _, ok := columnByName[name]; !ok {
			return Stats{}, fmt.Errorf("user %s missing canonical column %q after ensureColumns", userID, name)
		}
	}

	jobs, err := jobsToResolve(ctx, q, userID)
	if err != nil {
		return Stats{}, fmt.Errorf("load jobs: %w", err)
	}

	for _, j := range jobs {
		targetID, err := resolveTarget(j, columnByName)
		if err != nil {
			return Stats{}, fmt.Errorf("resolve target for job %s: %w", j.id, err)
		}

		if err := setJobColumn(ctx, q, j.id, targetID); err != nil {
			return Stats{}, fmt.Errorf("update job %s: %w", j.id, err)
		}
		stats.JobsUpdated++

		if !j.hasStageHistory {
			if err := appendActiveStage(ctx, q, j.id, targetID); err != nil {
				return Stats{}, fmt.Errorf("append stage for job %s: %w", j.id, err)
			}
			stats.StagesAppended++
		}
	}

	logger.Info("user backfilled",
		"user_id", userID,
		"columns_created", stats.ColumnsCreated,
		"jobs_updated", stats.JobsUpdated,
		"stage_rows_appended", stats.StagesAppended,
	)
	return stats, nil
}

// ── data access ──────────────────────────────────────────────────────────────

// usersNeedingBackfillQ returns the ids of users that own at least one job with
// current_stage_template_id IS NULL. Only these users need any work.
func usersNeedingBackfillQ(ctx context.Context, q querier) ([]string, error) {
	rows, err := q.Query(ctx,
		`SELECT DISTINCT user_id
		   FROM jobs
		  WHERE current_stage_template_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return ids, nil
}

// ensureColumns inserts any missing canonical columns for the user, appended
// after their current max "order". Uses ON CONFLICT (user_id, name) DO NOTHING
// for idempotency and returns the number of columns actually created.
func ensureColumns(ctx context.Context, q querier, userID string) (int, error) {
	// Current max "order" for this user's columns; -1 if the user has none, so
	// the first appended column lands at order 0.
	var maxOrder int
	err := q.QueryRow(ctx,
		`SELECT COALESCE(MAX("order"), -1) FROM stage_templates WHERE user_id = $1`,
		userID,
	).Scan(&maxOrder)
	if err != nil {
		return 0, fmt.Errorf("read max order: %w", err)
	}

	created := 0
	nextOrder := maxOrder + 1
	now := time.Now().UTC()

	for _, name := range canonicalColumns {
		tag, err := q.Exec(ctx,
			`INSERT INTO stage_templates (id, user_id, name, "order", phase, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)
			 ON CONFLICT (user_id, name) DO NOTHING`,
			uuid.New().String(), userID, name, nextOrder, stagePhase, now,
		)
		if err != nil {
			return 0, fmt.Errorf("insert column %q: %w", name, err)
		}
		// Only advance the order counter when a row was actually inserted, so
		// existing columns don't leave gaps and repeated names stay stable.
		if tag.RowsAffected() > 0 {
			created++
			nextOrder++
		}
	}
	return created, nil
}

// columnMap returns a name->id map of all the user's stage_templates.
func columnMap(ctx context.Context, q querier, userID string) (map[string]string, error) {
	rows, err := q.Query(ctx,
		`SELECT name, id FROM stage_templates WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var name, id string
		if err := rows.Scan(&name, &id); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}
		m[name] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate columns: %w", err)
	}
	return m, nil
}

// jobRow carries everything resolveTarget needs, resolved in SQL to avoid N+1s.
type jobRow struct {
	id string
	// oldStatus is the legacy jobs.status value (saved|applied|on_hold|offer|
	// rejected|archived).
	oldStatus string
	// appliedAtIsNull distinguishes never-applied cards (used for the archived
	// fallback: no applied_at -> Wishlist, else Applied).
	appliedAtIsNull bool
	// currentStageTemplateID is the stage_template_id of the job_stages row
	// referenced by jobs.current_stage_id, if any (resolved via LEFT JOIN).
	currentStageTemplateID *string
	// hasStageHistory is true when the job has at least one job_stages row.
	hasStageHistory bool
}

// jobsToResolve loads all of the user's jobs that still need a column pointer,
// pre-joining the current_stage_id -> stage_template_id lookup and a
// has-any-history flag so resolveTarget needs no further queries.
func jobsToResolve(ctx context.Context, q querier, userID string) ([]jobRow, error) {
	rows, err := q.Query(ctx,
		`SELECT
		     j.id,
		     j.status,
		     (j.applied_at IS NULL)                              AS applied_at_is_null,
		     cs.stage_template_id                                AS current_stage_template_id,
		     EXISTS (SELECT 1 FROM job_stages s WHERE s.job_id = j.id) AS has_history
		   FROM jobs j
		   LEFT JOIN job_stages cs ON cs.id = j.current_stage_id
		  WHERE j.user_id = $1
		    AND j.current_stage_template_id IS NULL`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var out []jobRow
	for rows.Next() {
		var j jobRow
		if err := rows.Scan(
			&j.id, &j.oldStatus, &j.appliedAtIsNull,
			&j.currentStageTemplateID, &j.hasStageHistory,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}
	return out, nil
}

// resolveTarget maps one job to a target stage_template id.
//
// Precedence:
//  1. If the job points at a current_stage (current_stage_id IS NOT NULL), use
//     that stage's template — this preserves exactly where the card sat.
//  2. Otherwise map by the old status:
//     saved -> Wishlist; applied|on_hold -> Applied; offer -> Offer;
//     rejected -> Rejected;
//     archived -> current stage's template if any (already covered by (1)),
//     else Wishlist when never applied, else Applied.
func resolveTarget(j jobRow, columnByName map[string]string) (string, error) {
	// (1) Honor an existing current_stage pointer regardless of status.
	if j.currentStageTemplateID != nil && *j.currentStageTemplateID != "" {
		return *j.currentStageTemplateID, nil
	}

	// (2) Map by legacy status.
	switch j.oldStatus {
	case "saved":
		return columnByName["Wishlist"], nil
	case "applied", "on_hold":
		return columnByName["Applied"], nil
	case "offer":
		return columnByName["Offer"], nil
	case "rejected":
		return columnByName["Rejected"], nil
	case "archived":
		// No current stage (else handled above): fall back by applied state.
		if j.appliedAtIsNull {
			return columnByName["Wishlist"], nil
		}
		return columnByName["Applied"], nil
	default:
		return "", fmt.Errorf("unexpected old status %q", j.oldStatus)
	}
}

// setJobColumn writes the resolved column pointer. The WHERE clause re-checks
// current_stage_template_id IS NULL so a concurrent/re-run write never clobbers
// an already-filled pointer (belt-and-suspenders idempotency).
func setJobColumn(ctx context.Context, q querier, jobID, templateID string) error {
	_, err := q.Exec(ctx,
		`UPDATE jobs
		    SET current_stage_template_id = $1
		  WHERE id = $2
		    AND current_stage_template_id IS NULL`,
		templateID, jobID,
	)
	if err != nil {
		return fmt.Errorf("update job column: %w", err)
	}
	return nil
}

// appendActiveStage adds one active job_stages row for a history-less card so
// the funnel/timeline has an entry. "order" is COALESCE(MAX(order),-1)+1 for the
// job (0 for the first). started_at/created_at default to now.
func appendActiveStage(ctx context.Context, q querier, jobID, templateID string) error {
	var nextOrder int
	err := q.QueryRow(ctx,
		`SELECT COALESCE(MAX("order"), -1) + 1 FROM job_stages WHERE job_id = $1`,
		jobID,
	).Scan(&nextOrder)
	if err != nil {
		return fmt.Errorf("read stage order: %w", err)
	}

	now := time.Now().UTC()
	_, err = q.Exec(ctx,
		`INSERT INTO job_stages
		     (id, job_id, stage_template_id, status, "order", started_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		uuid.New().String(), jobID, templateID, newStageStatus, nextOrder, now,
	)
	if err != nil {
		return fmt.Errorf("insert job_stage: %w", err)
	}
	return nil
}

// assertNoNullPointers is the post-condition: every card must now sit in a
// column. It logs the count and returns an error if any remain NULL.
func assertNoNullPointers(ctx context.Context, logger *slog.Logger, q querier) error {
	var remaining int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM jobs WHERE current_stage_template_id IS NULL`,
	).Scan(&remaining)
	if err != nil {
		return fmt.Errorf("assert null pointers: %w", err)
	}
	if remaining != 0 {
		return fmt.Errorf("assertion failed: %d jobs still have current_stage_template_id IS NULL", remaining)
	}
	logger.Info("assertion passed: no jobs with NULL current_stage_template_id", "remaining", remaining)
	return nil
}
