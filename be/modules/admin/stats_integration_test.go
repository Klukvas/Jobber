package admin

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRegistrationStatsKyivCalendarAndTestAccounts(t *testing.T) {
	for _, instant := range []string{"2026-03-29T09:00:00Z", "2026-10-25T09:00:00Z"} {
		t.Run(instant, func(t *testing.T) {
			pool := isolatedPool(t)
			repository := NewRepository(pool)
			now, err := time.Parse(time.RFC3339, instant)
			require.NoError(t, err)
			zone, err := time.LoadLocation(statsTimezone)
			require.NoError(t, err)
			local := now.In(zone)
			today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
			start7, start30 := today.AddDate(0, 0, -6), today.AddDate(0, 0, -29)
			const layout = "2006-01-02 15:04:05.999999999"
			// UTC-naive values match the production schema, including both sides of
			// Kyiv midnight and a future registration in the current calendar day.
			for _, at := range []time.Time{start30.Add(-time.Microsecond), start30, start7.Add(-time.Microsecond), start7, today.Add(-time.Microsecond), today, now, now.Add(time.Second)} {
				_, err := pool.Exec(t.Context(), `INSERT INTO users(email,name,created_at) VALUES($1,'Fixture',$2::timestamp)`, uuid.NewString()+"@example.com", at.UTC().Format(layout))
				require.NoError(t, err)
			}
			_, err = pool.Exec(t.Context(), `INSERT INTO users(id,email,name,is_test,created_at) VALUES($1,'test@example.com','Test',true,$2::timestamp)`, testUser, now.UTC().Format(layout))
			require.NoError(t, err)
			// Force a non-UTC DB session on every connection, so implicit casts
			// cannot accidentally pass just because the developer's DB uses UTC.
			conn, err := pool.Acquire(t.Context())
			require.NoError(t, err)
			defer conn.Release()
			_, err = conn.Exec(t.Context(), `SET TIME ZONE 'Pacific/Honolulu'; SET DateStyle TO 'German'`)
			require.NoError(t, err)
			repository = NewRepository(conn)
			for _, days := range []int{7, 30} {
				for _, include := range []bool{false, true} {
					stats, err := repository.statsAt(t.Context(), days, include, now)
					require.NoError(t, err)
					extra := int64(0)
					if include {
						extra = 1
					}
					require.EqualValues(t, 7+extra, stats.TotalUsers)
					require.EqualValues(t, 4+extra, stats.Registrations7)
					require.EqualValues(t, 6+extra, stats.Registrations30)
					require.Len(t, stats.Daily, days)
					require.Equal(t, today.AddDate(0, 0, 1-days).Format(time.DateOnly), stats.Daily[0].Date)
					require.Equal(t, today.Format(time.DateOnly), stats.Daily[days-1].Date)
					require.EqualValues(t, 2+extra, stats.Daily[days-1].Count)
					var sum int64
					for index, day := range stats.Daily {
						require.Equal(t, today.AddDate(0, 0, 1-days+index).Format(time.DateOnly), day.Date)
						sum += day.Count
					}
					if days == 7 {
						require.Equal(t, stats.Registrations7, sum)
					} else {
						require.Equal(t, stats.Registrations30, sum)
					}
					require.Zero(t, stats.Daily[2].Count)
				}
			}
		})
	}
}

func TestIntegrationTestAccountMetadataAndMigrationReversibility(t *testing.T) {
	pool := isolatedPool(t)
	fixtureUsers(t, pool)
	repository := NewRepository(pool)
	before, err := repository.Detail(t.Context(), testGrant)
	require.NoError(t, err)
	require.False(t, before.User.IsTest)
	require.NoError(t, repository.Grant(t.Context(), testGrant, testUser, GrantRequest{ID: uuid.NewString(), Days: 7, Reason: "Fixture allowance"}))
	withGrant, err := repository.Detail(t.Context(), testGrant)
	require.NoError(t, err)
	for _, flag := range []bool{true, true, false, false} {
		require.NoError(t, repository.SetTest(t.Context(), testGrant, flag))
		after, err := repository.Detail(t.Context(), testGrant)
		require.NoError(t, err)
		require.Equal(t, flag, after.User.IsTest)
		after.User.IsTest = withGrant.User.IsTest
		require.Equal(t, withGrant, after, "flag must not change profile, identity, billing, access or grant ledger")
		list, err := repository.List(t.Context(), "user@example.com", 1, 1)
		require.NoError(t, err)
		require.Equal(t, 1, list.Total)
		require.Len(t, list.Users, 1)
		require.Equal(t, flag, list.Users[0].IsTest, "test accounts remain searchable and paginated")
	}
	require.ErrorIs(t, repository.SetTest(t.Context(), uuid.NewString(), true), ErrNotFound)
	applyMigration(t, pool, "000054_test_accounts.down.sql")
	var count int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM users`).Scan(&count))
	require.Equal(t, 2, count)
	applyMigration(t, pool, "000054_test_accounts.up.sql")
	after, err := repository.Detail(t.Context(), testGrant)
	require.NoError(t, err)
	require.Equal(t, withGrant, after)
}
