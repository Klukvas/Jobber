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

// GetOverview returns high-level pipeline statistics.
//
// In the single-axis model there is no status, so the old status-derived
// buckets are re-expressed in stage terms:
//   - total    = every non-archived card
//   - active   = cards that currently sit in a column (current_stage_template_id set)
//   - closed   = archived cards (the only "closed" concept left)
//   - rejected = 0 (rejection was a status; it no longer exists)
//   - response_rate = share of cards that reached beyond the FIRST column
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
		cards AS (
			SELECT j.id, j.created_at, j.current_stage_template_id
			FROM jobs j
			WHERE j.user_id = $1 AND j.is_archived = false
		),
		card_stats AS (
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE current_stage_template_id IS NOT NULL) AS active
			FROM cards
		),
		closed_stats AS (
			SELECT COUNT(*) AS closed
			FROM jobs j
			WHERE j.user_id = $1 AND j.is_archived = true
		),
		responded AS (
			-- Cards that reached a column beyond the first — either currently
			-- sitting there or having a recorded job_stages row there.
			SELECT DISTINCT c.id
			FROM cards c
			CROSS JOIN first_template ft
			WHERE
				EXISTS (
					SELECT 1
					FROM job_stages js
					JOIN stage_templates st ON st.id = js.stage_template_id
					WHERE js.job_id = c.id AND st."order" > ft."order"
				)
				OR EXISTS (
					SELECT 1
					FROM stage_templates st
					WHERE st.id = c.current_stage_template_id AND st."order" > ft."order"
				)
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
				CROSS JOIN first_template ft
				WHERE st."order" > ft."order"
				ORDER BY js.job_id, st."order" ASC, js.started_at ASC
			) fr ON fr.job_id = c.id
		)
		SELECT
			COALESCE(card_stats.total, 0) AS total_applications,
			COALESCE(card_stats.active, 0) AS active_applications,
			COALESCE(closed_stats.closed, 0) AS closed_applications,
			0 AS rejected_applications,
			CASE
				WHEN card_stats.total > 0 THEN
					ROUND((response_stats.apps_with_response::numeric / card_stats.total) * 100, 2)
				ELSE 0
			END AS response_rate,
			COALESCE(ROUND(first_response_time.avg_days::numeric, 2), 0) AS avg_days_to_first_response
		FROM card_stats
		CROSS JOIN closed_stats
		CROSS JOIN response_stats
		CROSS JOIN first_response_time
	`

	analytics := &model.OverviewAnalytics{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&analytics.TotalApplications,
		&analytics.ActiveApplications,
		&analytics.ClosedApplications,
		&analytics.RejectedApplications,
		&analytics.ResponseRate,
		&analytics.AvgDaysToFirstResponse,
	)
	if err != nil {
		return nil, err
	}

	return analytics, nil
}

// GetFunnel returns the positional STAGE funnel.
//
// Buckets are the user's own stage_templates in "order". A card "reaches" a
// template when it currently sits in that column or recorded a job_stages row
// for it — counted with COUNT(DISTINCT job) so a card is never double-counted.
// Archived cards are excluded. Conversion/drop-off are relative to the previous
// column (buildStageFunnel).
func (r *AnalyticsRepository) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	const stageQuery = `
		SELECT
			st.name AS stage_name,
			st."order" AS stage_order,
			COUNT(DISTINCT j.id) AS reached_count
		FROM stage_templates st
		LEFT JOIN jobs j
			ON j.user_id = $1
			AND j.is_archived = false
			AND (
				j.current_stage_template_id = st.id
				OR EXISTS (
					SELECT 1 FROM job_stages js
					WHERE js.job_id = j.id AND js.stage_template_id = st.id
				)
			)
		WHERE st.user_id = $1
		GROUP BY st.id, st.name, st."order"
		ORDER BY st."order" ASC, st.name ASC
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
		if i == 0 || prev == 0 {
			s.ConversionRate = 100
			s.DropOffRate = 0
		} else {
			s.ConversionRate = roundRate(float64(c.count) / float64(prev) * 100)
			s.DropOffRate = roundRate(float64(prev-c.count) / float64(prev) * 100)
		}
		stages = append(stages, s)
		prev = c.count
	}
	return stages
}

func roundRate(v float64) float64 {
	return math.Round(v*100) / 100
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
// A "response" is a card that reached a column beyond the first (by first
// "order"). "Interviews" counts cards that entered a column whose name contains
// "interview". Archived cards are excluded.
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
				COUNT(DISTINCT j.id) FILTER (
					WHERE EXISTS (
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
				) AS responses_count,
				COUNT(DISTINCT j.id) FILTER (
					WHERE EXISTS (
						SELECT 1 FROM job_stages js
						JOIN stage_templates st ON st.id = js.stage_template_id
						WHERE js.job_id = j.id
						AND LOWER(st.name) LIKE '%interview%'
					)
				) AS interviews_count
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
// A "response" is a card that reached a column beyond the first. Archived cards
// are excluded.
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
				COUNT(DISTINCT j.id) FILTER (
					WHERE EXISTS (
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
				) AS responses_count
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
