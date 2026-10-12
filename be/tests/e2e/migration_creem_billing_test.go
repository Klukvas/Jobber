//go:build integration

package e2e

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Migration 000046 (the FastSpring → Creem cut-over) used to reset every
// `subscriptions` row whose identifiers did not look like Creem's. It is a no-op
// now, because production holds such a row and clearing it is irreversible.
//
// These tests run the real migration files against a real Postgres, on a
// throwaway database of their own, so the guarantee is checked where it is
// actually made — in SQL — instead of in a comment:
//
//   - the forward migration 45 → 46 → 47 changes no `subscriptions` row;
//   - rolling 46 back changes none either;
//   - 000047 still does its own work on top (that is what makes 000046 a no-op
//     and not a broken file);
//   - and the historical, destructive 000046 is run as well, to prove the
//     assertions above can actually fail. The tracked migration is never
//     rewritten for that: the old statement is replayed from the constant below.

// destructive46SQL is the body of 000046_creem_billing.up.sql as it stood before
// this release, verbatim except for its comments. It exists only so the negative
// probe below has something to fail against; nothing in the repository runs it.
const destructive46SQL = `
UPDATE subscriptions
SET external_subscription_id = NULL,
    external_account_id      = NULL,
    status                   = 'free',
    plan                     = 'free',
    current_period_start     = NULL,
    current_period_end       = NULL,
    cancel_at                = NULL,
    last_event_at            = NULL,
    updated_at               = NOW()
WHERE (external_subscription_id IS NOT NULL AND external_subscription_id NOT LIKE 'sub\_%')
   OR (external_account_id IS NOT NULL AND external_account_id NOT LIKE 'cust\_%');
`

// subscriptionRow is every column of `subscriptions` at schema 45–47, read as
// text. Timestamps are rendered by Postgres rather than scanned into time.Time
// so that a comparison cannot turn on a monotonic reading or a location
// pointer, and so a failure prints the value a person can recognise.
type subscriptionRow struct {
	ID                string
	UserID            string
	ExternalSubID     *string
	ExternalAccountID *string
	Status            string
	Plan              string
	PeriodStart       *string
	PeriodEnd         *string
	CancelAt          *string
	LastEventAt       *string
	CreatedAt         string
	UpdatedAt         string
}

// billingFixture describes one seeded subscriptions row.
//
// Every timestamp column is deliberately populated on every row, including the
// free ones where a period or a cancellation date would not occur in practice:
// the fixture is here to make any nulling visible, not to be realistic.
type billingFixture struct {
	label  string
	subID  *string
	acctID *string
	status string
	plan   string
	// legacy records whether the original 000046 WHERE clause matches this row,
	// i.e. whether the destructive version would have reset it.
	legacy bool
}

func billingFixtures() []billingFixture {
	s := func(v string) *string { return &v }
	return []billingFixture{
		{
			label:  "legacy paying subscriber",
			subID:  s("FSP-SUB-00000000001"),
			acctID: s("FSP-ACCT-00000000001"),
			status: "active",
			plan:   "pro",
			legacy: true,
		},
		{
			label:  "legacy identifiers on a free row",
			subID:  s("legacy-order-77"),
			acctID: s("legacy-account-77"),
			status: "free",
			plan:   "free",
			legacy: true,
		},
		{
			label:  "legacy customer, no subscription id",
			subID:  nil,
			acctID: s("legacy-account-88"),
			status: "past_due",
			plan:   "pro",
			legacy: true,
		},
		{
			label:  "Creem subscriber",
			subID:  s("sub_creem_00000001"),
			acctID: s("cust_creem_00000001"),
			status: "active",
			plan:   "enterprise",
			legacy: false,
		},
		{
			label:  "no identifiers at all",
			subID:  nil,
			acctID: nil,
			status: "free",
			plan:   "free",
			legacy: false,
		},
	}
}

func TestIntegrationCreemBillingMigrationPreservesSubscriptions(t *testing.T) {
	t.Run("45 to 47 preserves every subscriptions row", func(t *testing.T) {
		dsn, dbName := newThrowawayDatabase(t)
		m := newMigrator(t, repoMigrationsDir(t), dsn)
		db := connectTo(t, dsn)

		require.NoError(t, m.Migrate(45), "chain up to schema 45 (%s)", dbName)
		requireSchema(t, m, 45)

		seedBillingFixtures(t, db)
		stageUser := seedStageTemplates(t, db)

		before := readSubscriptions(t, db)
		require.Len(t, before, len(billingFixtures()))
		stagesBefore := readStageUpdatedAt(t, db, stageUser)

		require.NoError(t, m.Migrate(46))
		requireSchema(t, m, 46)
		require.Equal(t, before, readSubscriptions(t, db),
			"migration 000046 must not change any subscriptions row")

		require.NoError(t, m.Migrate(47))
		requireSchema(t, m, 47)
		require.Equal(t, before, readSubscriptions(t, db),
			"migration 000047 must not touch subscriptions either")

		// 000047 is the migration that has to keep working on top of the no-op:
		// its own backfill, and its promise not to move updated_at while doing it.
		require.Equal(t, map[string]string{
			"Wishlist":            "saved",
			"Technical Interview": "interview",
			"Follow-up reminder":  "other",
		}, readStageKinds(t, db, stageUser), "000047 must still classify seeded columns")
		require.Equal(t, stagesBefore, readStageUpdatedAt(t, db, stageUser),
			"000047 backfills data, so it must not move stage_templates.updated_at")

		// Down again: 000046's own down must be as harmless as its up.
		require.NoError(t, m.Migrate(46))
		requireSchema(t, m, 46)
		require.NoError(t, m.Migrate(45))
		requireSchema(t, m, 45)
		require.Equal(t, before, readSubscriptions(t, db),
			"rolling migration 000046 back must not change any subscriptions row")

		// Guard against a vacuous test: the rows must still be the legacy ones
		// the destructive version targeted.
		require.Equal(t, 3, countLegacyRows(t, db),
			"the fixture must still hold rows the original 000046 WHERE clause matches, or it proves nothing")
	})

	// The negative probe. Same fixture, same chain, but 000046 is replaced — in a
	// copy of the migrations directory under the test's own temp dir, never in
	// the repository — by the statement that shipped before this release. If this
	// subtest stopped failing the rows it should fail, the assertions above would
	// be proving nothing.
	t.Run("the historical destructive 000046 is detected", func(t *testing.T) {
		dir := copyMigrations(t, repoMigrationsDir(t))
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "000046_creem_billing.up.sql"), []byte(destructive46SQL), 0o600))

		dsn, dbName := newThrowawayDatabase(t)
		m := newMigrator(t, dir, dsn)
		db := connectTo(t, dsn)

		require.NoError(t, m.Migrate(45), "chain up to schema 45 (%s)", dbName)
		ids := seedBillingFixtures(t, db)
		before := byID(readSubscriptions(t, db))

		require.NoError(t, m.Migrate(46))
		requireSchema(t, m, 46)
		after := byID(readSubscriptions(t, db))

		for _, f := range billingFixtures() {
			id := ids[f.label]
			was, is := before[id], after[id]
			if !f.legacy {
				require.Equal(t, was, is, "%s must survive even the destructive version", f.label)
				continue
			}
			require.NotEqual(t, was, is, "%s: the destructive 000046 must be seen to change this row", f.label)
			require.Nil(t, is.ExternalSubID, "%s: destructive 000046 nulls the subscription id", f.label)
			require.Nil(t, is.ExternalAccountID, "%s: destructive 000046 nulls the customer id", f.label)
			require.Nil(t, is.PeriodStart, "%s: destructive 000046 nulls the period start", f.label)
			require.Nil(t, is.PeriodEnd, "%s: destructive 000046 nulls the period end", f.label)
			require.Nil(t, is.CancelAt, "%s: destructive 000046 nulls the cancellation date", f.label)
			require.Nil(t, is.LastEventAt, "%s: destructive 000046 nulls the ordering guard", f.label)
			require.Equal(t, "free", is.Status, "%s: destructive 000046 forces status free", f.label)
			require.Equal(t, "free", is.Plan, "%s: destructive 000046 forces plan free", f.label)
			require.Equal(t, was.CreatedAt, is.CreatedAt, "%s: created_at is not part of the reset", f.label)
		}
		require.Equal(t, 0, countLegacyRows(t, db),
			"after the destructive version no legacy row is left to match its clause")
	})
}

// --- fixtures ---------------------------------------------------------------

// seedBillingFixtures inserts one user and one subscriptions row per fixture and
// returns the subscription id for each label.
func seedBillingFixtures(t *testing.T, db *pgxpool.Pool) map[string]string {
	t.Helper()
	ctx := context.Background()
	ids := make(map[string]string, len(billingFixtures()))

	for i, f := range billingFixtures() {
		userID := uuid.New().String()
		_, err := db.Exec(ctx,
			`INSERT INTO users (id, email, name, password_hash, locale)
			 VALUES ($1, $2, '', 'x', 'en')`,
			userID, fmt.Sprintf("billing-fixture-%d@jobber.test", i),
		)
		require.NoError(t, err, f.label)

		subID := uuid.New().String()
		// Distinct, explicit timestamps: a reset to NOW() or NULL is then
		// unambiguous in the diff.
		_, err = db.Exec(ctx,
			`INSERT INTO subscriptions (id, user_id, external_subscription_id, external_account_id,
			                            status, plan, current_period_start, current_period_end,
			                            cancel_at, last_event_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6,
			         TIMESTAMPTZ '2026-01-02 03:04:05.123456+00',
			         TIMESTAMPTZ '2026-02-02 03:04:05.123456+00',
			         TIMESTAMPTZ '2026-02-01 00:00:00+00',
			         TIMESTAMPTZ '2026-01-15 10:11:12.5+00',
			         TIMESTAMPTZ '2025-12-01 08:00:00+00',
			         TIMESTAMPTZ '2026-01-15 10:11:12.5+00')`,
			subID, userID, f.subID, f.acctID, f.status, f.plan,
		)
		require.NoError(t, err, f.label)
		ids[f.label] = subID
	}
	return ids
}

// seedStageTemplates inserts the three columns 000047's backfill has to tell
// apart — a seeded name, a seeded multi-word name, and a name the user invented —
// and returns their owner.
func seedStageTemplates(t *testing.T, db *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()

	userID := uuid.New().String()
	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, locale)
		 VALUES ($1, 'stage-fixture@jobber.test', '', 'x', 'en')`, userID)
	require.NoError(t, err)

	for order, name := range []string{"Wishlist", "Technical Interview", "Follow-up reminder"} {
		_, err := db.Exec(ctx,
			`INSERT INTO stage_templates (id, user_id, name, "order", created_at, updated_at)
			 VALUES ($1, $2, $3, $4,
			         TIMESTAMP '2025-11-01 09:00:00',
			         TIMESTAMP '2025-11-02 09:00:00')`,
			uuid.New().String(), userID, name, order+1)
		require.NoError(t, err, name)
	}
	return userID
}

// --- reads ------------------------------------------------------------------

func readSubscriptions(t *testing.T, db *pgxpool.Pool) []subscriptionRow {
	t.Helper()
	rows, err := db.Query(context.Background(),
		`SELECT id::text, user_id::text, external_subscription_id, external_account_id,
		        status, plan,
		        to_char(current_period_start AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US'),
		        to_char(current_period_end   AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US'),
		        to_char(cancel_at            AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US'),
		        to_char(last_event_at        AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US'),
		        to_char(created_at           AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US'),
		        to_char(updated_at           AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US')
		 FROM subscriptions
		 ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	var out []subscriptionRow
	for rows.Next() {
		var r subscriptionRow
		require.NoError(t, rows.Scan(&r.ID, &r.UserID, &r.ExternalSubID, &r.ExternalAccountID,
			&r.Status, &r.Plan, &r.PeriodStart, &r.PeriodEnd, &r.CancelAt, &r.LastEventAt,
			&r.CreatedAt, &r.UpdatedAt))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

func byID(rows []subscriptionRow) map[string]subscriptionRow {
	out := make(map[string]subscriptionRow, len(rows))
	for _, r := range rows {
		out[r.ID] = r
	}
	return out
}

// countLegacyRows counts the rows the original 000046 WHERE clause matches. The
// clause is reproduced here exactly as the deploy preflight reproduces it, so
// the test and the release guard agree on what "legacy row" means.
func countLegacyRows(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(context.Background(),
		`SELECT count(*) FROM subscriptions
		 WHERE (external_subscription_id IS NOT NULL AND external_subscription_id NOT LIKE 'sub\_%')
		    OR (external_account_id IS NOT NULL AND external_account_id NOT LIKE 'cust\_%')`,
	).Scan(&n))
	return n
}

func readStageKinds(t *testing.T, db *pgxpool.Pool, userID string) map[string]string {
	t.Helper()
	return readStageColumn(t, db, userID, "kind")
}

func readStageUpdatedAt(t *testing.T, db *pgxpool.Pool, userID string) map[string]string {
	t.Helper()
	return readStageColumn(t, db, userID, `to_char(updated_at, 'YYYY-MM-DD HH24:MI:SS.US')`)
}

// readStageColumn reads one expression per stage template, keyed by name.
// The expression is test-owned (a literal in this file), never user input.
func readStageColumn(t *testing.T, db *pgxpool.Pool, userID, expr string) map[string]string {
	t.Helper()
	rows, err := db.Query(context.Background(),
		fmt.Sprintf(`SELECT name, %s FROM stage_templates WHERE user_id = $1`, expr), userID)
	require.NoError(t, err)
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var name, value string
		require.NoError(t, rows.Scan(&name, &value))
		out[name] = value
	}
	require.NoError(t, rows.Err())
	return out
}

// --- throwaway database and migrator ---------------------------------------

// newThrowawayDatabase creates an empty database on the Postgres the suite
// already runs and returns a DSN for it. The suite's own schema is migrated to
// the latest version by TestMain, so stepping through 45 → 47 needs a database
// of its own; it is dropped when the test ends.
func newThrowawayDatabase(t *testing.T) (dsn, name string) {
	t.Helper()
	ctx := context.Background()

	// Hex only, so quoting the identifier cannot be ambiguous.
	name = "mig_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err := pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, name))
	require.NoError(t, err)

	t.Cleanup(func() {
		// FORCE: the migrator and the fixture pool are closed by their own
		// cleanups, but a lingering backend must not leak a database either.
		if _, err := pool.Exec(context.Background(),
			fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, name)); err != nil {
			t.Logf("could not drop throwaway database %s: %v", name, err)
		}
	})

	cc := pool.Config().ConnConfig
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cc.User, cc.Password),
		Host:     fmt.Sprintf("%s:%d", cc.Host, cc.Port),
		Path:     "/" + name,
		RawQuery: "sslmode=disable",
	}
	return u.String(), name
}

func connectTo(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	db, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	require.NoError(t, db.Ping(context.Background()))
	return db
}

func repoMigrationsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../migrations")
	require.NoError(t, err)
	return dir
}

func newMigrator(t *testing.T, migrationsDir, dsn string) *migrate.Migrate {
	t.Helper()
	m, err := migrate.New("file://"+migrationsDir, dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			t.Logf("closing migrator: source=%v database=%v", srcErr, dbErr)
		}
	})
	return m
}

func requireSchema(t *testing.T, m *migrate.Migrate, want uint) {
	t.Helper()
	version, dirty, err := m.Version()
	require.NoError(t, err)
	require.False(t, dirty, "schema left dirty at version %d", version)
	require.Equal(t, want, version)
}

// copyMigrations copies the migration files into the test's temp directory so a
// single file can be swapped for the negative probe without the repository ever
// being written to.
func copyMigrations(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()

	entries, err := os.ReadDir(src)
	require.NoError(t, err)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(src, e.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dst, e.Name()), body, 0o600))
	}
	return dst
}
