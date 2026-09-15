package repository

import (
	"context"
	"math"

	"github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBPool defines the interface for database operations used by the repository
type DBPool interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type AnalyticsRepository struct {
	pool DBPool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

// NewAnalyticsRepositoryWithPool creates a repository with a custom pool (for testing)
func NewAnalyticsRepositoryWithPool(pool DBPool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

// Analytics work over the single-axis STAGE pipeline: the user's ordered
// stage_templates ARE the funnel, a card sits in exactly one column
// (jobs.current_stage_template_id) and job_stages records every column it has
// entered. There is no status and no phase.
//
// A card "reached" a stage template X when it currently sits in X
// (current_stage_template_id = X) OR it recorded a job_stages row for X — so
// progress still counts even after the card later moved on. Archived cards
// (is_archived = true) are hidden from every metric.

// respondedPredicate is the single definition of "this card got a response":
// it reached a pipeline column ordered after the user's first one, either by
// sitting there now or by having recorded a job_stages row for it.
//
// Shared verbatim by the overview, resume-effectiveness and source queries so
// the three can never drift apart and report different response rates for the
// same cards. Requires the jobs row to be aliased `j` and the `first_template`
// CTE to be in scope. It is a compile-time constant — no user input is ever
// interpolated into it.
const respondedPredicate = `(
			EXISTS (
				SELECT 1 FROM job_stages js
				JOIN stage_templates st ON st.id = js.stage_template_id
				WHERE js.job_id = j.id
				  AND st."order" > (SELECT "order" FROM first_template)
			)
			OR EXISTS (
				SELECT 1 FROM stage_templates st
				WHERE st.id = j.current_stage_template_id
				  AND st."order" > (SELECT "order" FROM first_template)
			)
		)`

// interviewPredicate is the same reached-a-column rule narrowed to interview
// columns. It deliberately mirrors respondedPredicate's two branches: it used
// to look at job_stages only, so a card sitting in "Interview" without a
// recorded stage row counted as a response but not as an interview.
const interviewPredicate = `(
			EXISTS (
				SELECT 1 FROM job_stages js
				JOIN stage_templates st ON st.id = js.stage_template_id
				WHERE js.job_id = j.id AND LOWER(st.name) LIKE '%interview%'
			)
			OR EXISTS (
				SELECT 1 FROM stage_templates st
				WHERE st.id = j.current_stage_template_id
				  AND LOWER(st.name) LIKE '%interview%'
			)
		)`

// GetOverview returns high-level pipeline statistics.
//
// In the single-axis model there is no status, so the old status-derived
// buckets are re-expressed in stage terms. The three headline counts are one
// partition of the user's cards, so they always add up:
//
//	total  = every card the user has        (= active + closed)
//	active = cards that are not archived
//	closed = archived cards
//
// Counting "active" as "sits in a column" is what used to break the identity:
// a card with no column at all belonged to neither bucket, so Total could read
// 23 next to Active 23 and Closed 3.
//
//   - rejected = 0 (rejection was a status; it no longer exists)
//   - response_rate = share of ACTIVE cards that reached beyond the FIRST
//     column. Archived cards are excluded from every rate, as everywhere else.
//   - avg_days_to_first_response = avg days from card creation to the first
//     job_stages row past the first column
func (r *AnalyticsRepository) GetOverview(ctx context.Context, userID string) (*model.OverviewAnalytics, error) {
	query := `
		WITH first_template AS (
			-- The first pipeline column (lowest "order"). "Beyond first" means
			-- any stage template ordered after it.
			SELECT id, "order"
			FROM stage_templates
			WHERE user_id = $1
			ORDER BY "order" ASC, name ASC
			LIMIT 1
		),
		all_cards AS (
			SELECT j.id, j.created_at, j.current_stage_template_id, j.is_archived
			FROM jobs j
			WHERE j.user_id = $1
		),
		cards AS (
			SELECT id, created_at, current_stage_template_id
			FROM all_cards
			WHERE is_archived = false
		),
		card_stats AS (
			SELECT
				(SELECT COUNT(*) FROM all_cards WHERE is_archived = false) AS active,
				(SELECT COUNT(*) FROM all_cards WHERE is_archived = true) AS closed
		),
		responded AS (
			SELECT j.id
			FROM cards j
			WHERE ` + respondedPredicate + `
		),
		response_stats AS (
			SELECT COUNT(*) AS apps_with_response FROM responded
		),
		first_response_time AS (
			-- Days from card creation to the first stage entered beyond the
			-- first column.
			SELECT AVG(EXTRACT(EPOCH FROM (fr.started_at - c.created_at)) / 86400) AS avg_days
			FROM cards c
			JOIN (
				SELECT DISTINCT ON (js.job_id)
					js.job_id, js.started_at
				FROM job_stages js
				JOIN stage_templates st ON st.id = js.stage_template_id
				WHERE st."order" > (SELECT "order" FROM first_template)
				ORDER BY js.job_id, st."order" ASC, js.started_at ASC
			) fr ON fr.job_id = c.id
		)
		SELECT
			COALESCE(card_stats.active, 0) AS active_applications,
			COALESCE(card_stats.closed, 0) AS closed_applications,
			0 AS rejected_applications,
			CASE
				WHEN card_stats.active > 0 THEN
					ROUND((response_stats.apps_with_response::numeric / card_stats.active) * 100, 2)
				ELSE 0
			END AS response_rate,
			COALESCE(ROUND(first_response_time.avg_days::numeric, 2), 0) AS avg_days_to_first_response
		FROM card_stats
		CROSS JOIN response_stats
		CROSS JOIN first_response_time
	`

	analytics := &model.OverviewAnalytics{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&analytics.ActiveApplications,
		&analytics.ClosedApplications,
		&analytics.RejectedApplications,
		&analytics.ResponseRate,
		&analytics.AvgDaysToFirstResponse,
	)
	if err != nil {
		return nil, err
	}

	// Derived, not queried: active and closed partition the user's cards, so
	// the headline total is their sum by construction. Computing it separately
	// in SQL is what let Total 23 sit next to Active 23 and Closed 3.
	analytics.TotalApplications = analytics.ActiveApplications + analytics.ClosedApplications

	return analytics, nil
}

// GetFunnel returns the positional STAGE funnel.
//
// Buckets are the user's own stage_templates in "order". A bucket counts every
// non-archived card that got AT LEAST that far — i.e. whose furthest reached
// column is that one or a later one. A card "reaches" a template when it
// currently sits in that column or recorded a job_stages row for it; cards are
// counted once, by their furthest column.
//
// "At least this far" is what makes the funnel a funnel: the sets are nested,
// so the counts decrease monotonically and conversion stays within 0–100%.
// Counting only the cards that touched each exact column does not nest — a
// card created straight into "Applied" never touched "Wishlist" — which is how
// the funnel came to report "17 applications, 340% converted, -240% drop-off".
//
// Archived cards are excluded. Conversion/drop-off are relative to the previous
// column (buildStageFunnel).
func (r *AnalyticsRepository) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	const stageQuery = `
		WITH templates AS (
			SELECT id, name, "order"
			FROM stage_templates
			WHERE user_id = $1
		),
		reached AS (
			-- One row per (card, column the card has reached).
			SELECT DISTINCT j.id AS job_id, t."order" AS stage_order
			FROM jobs j
			JOIN templates t
				ON j.current_stage_template_id = t.id
				OR EXISTS (
					SELECT 1 FROM job_stages js
					WHERE js.job_id = j.id AND js.stage_template_id = t.id
				)
			WHERE j.user_id = $1 AND j.is_archived = false
		),
		furthest AS (
			SELECT job_id, MAX(stage_order) AS max_order
			FROM reached
			GROUP BY job_id
		)
		SELECT
			t.name AS stage_name,
			t."order" AS stage_order,
			(SELECT COUNT(*) FROM furthest f WHERE f.max_order >= t."order")::int AS reached_count
		FROM templates t
		ORDER BY t."order" ASC, t.name ASC
	`

	rows, err := r.pool.Query(ctx, stageQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []stageCount
	for rows.Next() {
		var sc stageCount
		if err := rows.Scan(&sc.name, &sc.order, &sc.count); err != nil {
			return nil, err
		}
		counts = append(counts, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// No stage templates → empty funnel (the frontend shows an empty state).
	if len(counts) == 0 {
		return &model.FunnelAnalytics{}, nil
	}

	return &model.FunnelAnalytics{
		Stages: buildStageFunnel(counts),
	}, nil
}

// stageCount is one column's reached-count, read from the funnel query.
type stageCount struct {
	name  string
	order int
	count int
}

// buildStageFunnel turns the ordered per-stage counts into funnel buckets,
// computing conversion/drop-off relative to the previous column. Pure —
// unit-tested.
func buildStageFunnel(counts []stageCount) []model.FunnelStage {
	stages := make([]model.FunnelStage, 0, len(counts))
	prev := 0
	for i, c := range counts {
		s := model.FunnelStage{StageName: c.name, StageOrder: c.order, Count: c.count}
		switch {
		case i == 0:
			// The first column is the funnel's own base. With no cards in it
			// there is nothing to convert — reporting 100% would dress an
			// empty pipeline up as a perfect one.
			if c.count > 0 {
				s.ConversionRate = 100
			}
		case prev == 0:
			// No cards reached the previous column, so there is no
			// denominator: both rates stay at zero rather than divide by zero.
		default:
			rate := float64(c.count) / float64(prev) * 100
			// The counts are cumulative-downstream and therefore
			// non-increasing, so this cannot exceed 100 — clamped anyway so a
			// future change to the query can never print a 340% conversion or
			// a negative drop-off again.
			s.ConversionRate = roundRate(clampPercent(rate))
			s.DropOffRate = roundRate(clampPercent(100 - rate))
		}
		stages = append(stages, s)
		prev = c.count
	}
	return stages
}

func roundRate(v float64) float64 {
	return math.Round(v*100) / 100
}

// clampPercent keeps a rate inside the 0–100 range a percentage can occupy.
func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// GetStageTime returns timing metrics per stage template. Time-in-stage is the
// span from job_stages.started_at to completed_at (or now, if still in the
// column). Archived cards are excluded.
func (r *AnalyticsRepository) GetStageTime(ctx context.Context, userID string) (*model.StageTimeAnalytics, error) {
	query := `
		WITH stage_durations AS (
			SELECT
				st.name AS stage_name,
				st."order" AS stage_order,
				js.job_id,
				CASE
					WHEN js.completed_at IS NOT NULL
					THEN EXTRACT(EPOCH FROM (js.completed_at - js.started_at)) / 86400
					ELSE EXTRACT(EPOCH FROM (NOW() - js.started_at)) / 86400
				END AS duration_days
			FROM job_stages js
			JOIN stage_templates st ON st.id = js.stage_template_id
			JOIN jobs j ON j.id = js.job_id
			WHERE j.user_id = $1 AND j.is_archived = false
		)
		SELECT
			stage_name,
			stage_order,
			ROUND(AVG(duration_days)::numeric, 2) AS avg_days,
			ROUND(MIN(duration_days)::numeric, 2) AS min_days,
			ROUND(MAX(duration_days)::numeric, 2) AS max_days,
			COUNT(DISTINCT job_id) AS applications_count
		FROM stage_durations
		GROUP BY stage_name, stage_order
		ORDER BY stage_order
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []model.StageTimeMetrics
	for rows.Next() {
		var stage model.StageTimeMetrics
		if err := rows.Scan(
			&stage.StageName,
			&stage.StageOrder,
			&stage.AvgDays,
			&stage.MinDays,
			&stage.MaxDays,
			&stage.ApplicationsCount,
		); err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.StageTimeAnalytics{Stages: stages}, nil
}

// GetResumeEffectiveness returns effectiveness metrics per resume.
//
// A "response" is respondedPredicate — the same rule the overview response
// rate uses, so the per-resume rates and the headline rate describe the same
// event. "Interviews" is interviewPredicate. Archived cards are excluded.
func (r *AnalyticsRepository) GetResumeEffectiveness(ctx context.Context, userID string) (*model.ResumeAnalytics, error) {
	query := `
		WITH first_template AS (
			SELECT "order"
			FROM stage_templates
			WHERE user_id = $1
			ORDER BY "order" ASC, name ASC
			LIMIT 1
		),
		resume_stats AS (
			SELECT
				r.id AS resume_id,
				r.title AS resume_title,
				COUNT(DISTINCT j.id) AS applications_count,
				COUNT(DISTINCT j.id) FILTER (WHERE ` + respondedPredicate + `) AS responses_count,
				COUNT(DISTINCT j.id) FILTER (WHERE ` + interviewPredicate + `) AS interviews_count
			FROM resumes r
			LEFT JOIN jobs j ON j.resume_id = r.id AND j.user_id = $1 AND j.is_archived = false
			WHERE r.user_id = $1
			GROUP BY r.id, r.title
		)
		SELECT
			resume_id,
			resume_title,
			applications_count,
			responses_count,
			interviews_count,
			CASE
				WHEN applications_count > 0
				THEN ROUND((responses_count::numeric / applications_count) * 100, 2)
				ELSE 0
			END AS response_rate
		FROM resume_stats
		ORDER BY applications_count DESC, resume_title
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resumes []model.ResumeEffectiveness
	for rows.Next() {
		var resume model.ResumeEffectiveness
		if err := rows.Scan(
			&resume.ResumeID,
			&resume.ResumeTitle,
			&resume.ApplicationsCount,
			&resume.ResponsesCount,
			&resume.InterviewsCount,
			&resume.ResponseRate,
		); err != nil {
			return nil, err
		}
		resumes = append(resumes, resume)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.ResumeAnalytics{Resumes: resumes}, nil
}

// GetSourceAnalytics returns metrics grouped by job source.
//
// A "response" is respondedPredicate — the same rule as the overview and
// resume-effectiveness rates. Archived cards are excluded.
func (r *AnalyticsRepository) GetSourceAnalytics(ctx context.Context, userID string) (*model.SourceAnalytics, error) {
	query := `
		WITH first_template AS (
			SELECT "order"
			FROM stage_templates
			WHERE user_id = $1
			ORDER BY "order" ASC, name ASC
			LIMIT 1
		),
		source_stats AS (
			SELECT
				COALESCE(NULLIF(j.source, ''), 'Unknown') AS source_name,
				COUNT(DISTINCT j.id) AS applications_count,
				COUNT(DISTINCT j.id) FILTER (WHERE ` + respondedPredicate + `) AS responses_count
			FROM jobs j
			WHERE j.user_id = $1 AND j.is_archived = false
			GROUP BY COALESCE(NULLIF(j.source, ''), 'Unknown')
		)
		SELECT
			source_name,
			applications_count,
			responses_count,
			CASE
				WHEN applications_count > 0
				THEN ROUND((responses_count::numeric / applications_count) * 100, 2)
				ELSE 0
			END AS conversion_rate
		FROM source_stats
		ORDER BY applications_count DESC, source_name
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []model.SourceMetrics
	for rows.Next() {
		var source model.SourceMetrics
		if err := rows.Scan(
			&source.SourceName,
			&source.ApplicationsCount,
			&source.ResponsesCount,
			&source.ConversionRate,
		); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.SourceAnalytics{Sources: sources}, nil
}
