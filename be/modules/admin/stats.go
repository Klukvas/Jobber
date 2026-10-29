package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/gin-gonic/gin"
)

const statsTimezone = "Europe/Kyiv"

type DailyRegistrations struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type Stats struct {
	TotalUsers      int64                `json:"total_users"`
	Registrations7  int64                `json:"registrations_7_days"`
	Registrations30 int64                `json:"registrations_30_days"`
	Daily           []DailyRegistrations `json:"daily_registrations"`
	Days            int                  `json:"days"`
	IncludeTests    bool                 `json:"include_test"`
	Timezone        string               `json:"timezone"`
	AsOf            time.Time            `json:"as_of"`
}

func (h *Handler) stats(c *gin.Context) {
	days := c.DefaultQuery("days", "30")
	if days != "7" && days != "30" {
		fail(c, 400, "INVALID_REQUEST", "Days must be 7 or 30")
		return
	}
	include := c.DefaultQuery("include_test", "false")
	if include != "true" && include != "false" {
		fail(c, 400, "INVALID_REQUEST", "Include tests must be true or false")
		return
	}
	window := 30
	if days == "7" {
		window = 7
	}
	data, err := h.store.Stats(c.Request.Context(), window, include == "true")
	h.respond(c, data, err)
}

func (h *Handler) setTest(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var payload struct {
		IsTest *bool `json:"is_test"`
	}
	if !bind(c, &payload) {
		return
	}
	if payload.IsTest == nil {
		fail(c, 400, "INVALID_REQUEST", "Is test must be a boolean")
		return
	}
	h.respond(c, gin.H{"saved": true, "is_test": *payload.IsTest}, h.store.SetTest(c.Request.Context(), id, *payload.IsTest))
}

func (r *Repository) SetTest(ctx context.Context, id string, isTest bool) error {
	// This flag deliberately has no effect on identity, billing or entitlements.
	tag, err := r.db.Exec(ctx, `UPDATE users SET is_test=$2 WHERE id=$1`, id, isTest)
	if err != nil {
		return fmt.Errorf("set test account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Stats(ctx context.Context, days int, includeTests bool) (Stats, error) {
	return r.statsAt(ctx, days, includeTests, time.Now().UTC())
}

func (r *Repository) statsAt(ctx context.Context, days int, includeTests bool, now time.Time) (Stats, error) {
	result := Stats{Days: days, IncludeTests: includeTests, Timezone: statsTimezone, AsOf: now.UTC(), Daily: []DailyRegistrations{}}
	zone, err := time.LoadLocation(statsTimezone)
	if err != nil {
		return result, fmt.Errorf("load statistics timezone: %w", err)
	}
	local := now.In(zone)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	start7, start30 := today.AddDate(0, 0, -6), today.AddDate(0, 0, -29)
	// users.created_at is a UTC timestamp WITHOUT time zone. Explicit timestamp
	// parameters avoid casts that would reinterpret it in the DB session timezone.
	const utcTimestamp = "2006-01-02 15:04:05.999999999"
	var dailyJSON []byte
	err = r.db.QueryRow(ctx, `WITH eligible AS MATERIALIZED (
 SELECT created_at FROM users WHERE ($4::boolean OR NOT is_test) AND created_at <= $1::timestamp
), daily AS (
 SELECT to_char((created_at AT TIME ZONE 'UTC') AT TIME ZONE 'Europe/Kyiv','YYYY-MM-DD') AS day, count(*) AS registrations
 FROM eligible WHERE created_at >= $3::timestamp GROUP BY 1
)
SELECT count(*), count(*) FILTER (WHERE created_at >= $2::timestamp), count(*) FILTER (WHERE created_at >= $3::timestamp),
 COALESCE((SELECT jsonb_agg(jsonb_build_object('date',day,'count',registrations) ORDER BY day) FROM daily),'[]'::jsonb)
FROM eligible`, now.UTC().Format(utcTimestamp), start7.UTC().Format(utcTimestamp), start30.UTC().Format(utcTimestamp), includeTests).Scan(&result.TotalUsers, &result.Registrations7, &result.Registrations30, &dailyJSON)
	if err != nil {
		return result, fmt.Errorf("read registration statistics: %w", err)
	}
	var daily []DailyRegistrations
	if err := json.Unmarshal(dailyJSON, &daily); err != nil {
		return result, fmt.Errorf("decode registration statistics: %w", err)
	}
	counts := make(map[string]int64, len(daily))
	for _, day := range daily {
		counts[day.Date] = day.Count
	}
	for day := today.AddDate(0, 0, 1-days); !day.After(today); day = day.AddDate(0, 0, 1) {
		date := day.Format(time.DateOnly)
		result.Daily = append(result.Daily, DailyRegistrations{Date: date, Count: counts[date]})
	}
	return result, nil
}
