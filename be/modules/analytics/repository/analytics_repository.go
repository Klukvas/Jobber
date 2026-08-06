package repository

import (
	"context"

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

// An "application" is a job card that has been applied to: applied_at IS NOT NULL.
// Saved (wishlist) cards are excluded from all analytics.

// GetOverview returns high-level application statistics
func (r *AnalyticsRepository) GetOverview(ctx context.Context, userID string) (*model.OverviewAnalytics, error) {
	query := `
		WITH app_stats AS (
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status IN ('applied', 'on_hold')) AS active,
				COUNT(*) FILTER (WHERE status IN ('rejected', 'offer', 'archived')) AS closed,
				COUNT(*) FILTER (WHERE status = 'rejected') AS rejected
			FROM jobs
			WHERE user_id = $1 AND applied_at IS NOT NULL
		),
		response_stats AS (
			-- Applications that have at least one stage beyond "Applied"
			SELECT
				COUNT(DISTINCT j.id) AS apps_with_response
			FROM jobs j
			JOIN job_stages js ON js.job_id = j.id
			JOIN stage_templates st ON st.id = js.stage_template_id
			WHERE j.user_id = $1
			AND j.applied_at IS NOT NULL
			AND st."order" > 1
		),
		first_response_time AS (
			-- Average days to first response (first stage after "Applied")
			SELECT
				AVG(EXTRACT(EPOCH FROM (js.started_at - j.applied_at)) / 86400) AS avg_days
			FROM jobs j
			JOIN (
				SELECT DISTINCT ON (job_id)
					job_id, started_at
				FROM job_stages js
				JOIN stage_templates st ON st.id = js.stage_template_id
				WHERE st."order" > 1
				ORDER BY job_id, js."order" ASC
			) js ON js.job_id = j.id
			WHERE j.user_id = $1 AND j.applied_at IS NOT NULL
		)
		SELECT
			COALESCE(app_stats.total, 0) AS total_applications,
			COALESCE(app_stats.active, 0) AS active_applications,
			COALESCE(app_stats.closed, 0) AS closed_applications,
			COALESCE(app_stats.rejected, 0) AS rejected_applications,
			CASE
				WHEN app_stats.total > 0 THEN
					ROUND((response_stats.apps_with_response::numeric / app_stats.total) * 100, 2)
				ELSE 0
			END AS response_rate,
			COALESCE(ROUND(first_response_time.avg_days::numeric, 2), 0) AS avg_days_to_first_response
		FROM app_stats
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

// GetFunnel returns stage-based funnel metrics.
//
// The first "Applied" bucket is derived directly from job cards
// (applied_at IS NOT NULL): every application counts even when the user has
// not tracked any pipeline stages for it, and it does not depend on stage
// templates existing. Later buckets come from stage templates with order > 1
// (the same "got a response" convention used by response_rate and sources).
func (r *AnalyticsRepository) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	query := `
		WITH applied_total AS (
			SELECT COUNT(*) AS app_count
			FROM jobs
			WHERE user_id = $1 AND applied_at IS NOT NULL
		),
		template_counts AS (
			SELECT
				st.name AS stage_name,
				st."order" AS stage_order,
				COUNT(DISTINCT j.id) AS app_count
			FROM stage_templates st
			LEFT JOIN job_stages js ON js.stage_template_id = st.id
			LEFT JOIN jobs j ON j.id = js.job_id AND j.user_id = $1 AND j.applied_at IS NOT NULL
			WHERE st.user_id = $1 AND st."order" > 1
			GROUP BY st.id, st.name, st."order"
		),
		combined AS (
			SELECT 'Applied' AS stage_name, 1 AS stage_order, app_count
			FROM applied_total
			WHERE app_count > 0
			UNION ALL
			SELECT stage_name, stage_order, app_count
			FROM template_counts
		),
		ordered_stages AS (
			SELECT
				stage_name,
				stage_order,
				app_count,
				LAG(app_count) OVER (ORDER BY stage_order) AS prev_count
			FROM combined
		)
		SELECT
			stage_name,
			stage_order,
			app_count,
			CASE
				WHEN prev_count IS NULL OR prev_count = 0 THEN 100.0
				ELSE ROUND((app_count::numeric / prev_count) * 100, 2)
			END AS conversion_rate,
			CASE
				WHEN prev_count IS NULL THEN 0.0
				WHEN prev_count = 0 THEN 0.0
				ELSE ROUND(((prev_count - app_count)::numeric / prev_count) * 100, 2)
			END AS drop_off_rate
		FROM ordered_stages
		ORDER BY stage_order
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []model.FunnelStage
	for rows.Next() {
		var stage model.FunnelStage
		if err := rows.Scan(
			&stage.StageName,
			&stage.StageOrder,
			&stage.Count,
			&stage.ConversionRate,
			&stage.DropOffRate,
		); err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.FunnelAnalytics{Stages: stages}, nil
}

// GetStageTime returns timing metrics per stage
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
			WHERE j.user_id = $1 AND j.applied_at IS NOT NULL
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

// GetResumeEffectiveness returns effectiveness metrics per resume
func (r *AnalyticsRepository) GetResumeEffectiveness(ctx context.Context, userID string) (*model.ResumeAnalytics, error) {
	query := `
		WITH resume_stats AS (
			SELECT
				r.id AS resume_id,
				r.title AS resume_title,
				COUNT(DISTINCT j.id) AS applications_count,
				COUNT(DISTINCT j.id) FILTER (
					WHERE EXISTS (
						SELECT 1 FROM job_stages js
						JOIN stage_templates st ON st.id = js.stage_template_id
						WHERE js.job_id = j.id AND st."order" > 1
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
			LEFT JOIN jobs j ON j.resume_id = r.id AND j.user_id = $1 AND j.applied_at IS NOT NULL
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

// GetSourceAnalytics returns metrics grouped by job source
func (r *AnalyticsRepository) GetSourceAnalytics(ctx context.Context, userID string) (*model.SourceAnalytics, error) {
	query := `
		WITH source_stats AS (
			SELECT
				COALESCE(NULLIF(j.source, ''), 'Unknown') AS source_name,
				COUNT(DISTINCT j.id) AS applications_count,
				COUNT(DISTINCT j.id) FILTER (
					WHERE EXISTS (
						SELECT 1 FROM job_stages js
						JOIN stage_templates st ON st.id = js.stage_template_id
						WHERE js.job_id = j.id AND st."order" > 1
					)
				) AS responses_count
			FROM jobs j
			WHERE j.user_id = $1 AND j.applied_at IS NOT NULL
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
