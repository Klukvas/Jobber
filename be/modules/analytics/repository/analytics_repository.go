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

// GetOverview returns high-level application statistics
func (r *AnalyticsRepository) GetOverview(ctx context.Context, userID string) (*model.OverviewAnalytics, error) {
	query := `
		WITH app_stats AS (
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status IN ('active', 'on_hold')) AS active,
				COUNT(*) FILTER (WHERE status IN ('rejected', 'offer', 'archived')) AS closed
			FROM applications
			WHERE user_id = $1
		),
		response_stats AS (
			-- Applications that have at least one stage beyond "Applied"
			SELECT 
				COUNT(DISTINCT a.id) AS apps_with_response
			FROM applications a
			JOIN application_stages ast ON ast.application_id = a.id
			JOIN stage_templates st ON st.id = ast.stage_template_id
			WHERE a.user_id = $1
			AND st."order" > 1
		),
		first_response_time AS (
			-- Average days to first response (first stage after "Applied")
			SELECT
				AVG(EXTRACT(EPOCH FROM (ast.started_at - a.applied_at)) / 86400) AS avg_days
			FROM applications a
			JOIN (
				SELECT DISTINCT ON (application_id) 
					application_id, started_at
				FROM application_stages ast
				JOIN stage_templates st ON st.id = ast.stage_template_id
				WHERE st."order" > 1
				ORDER BY application_id, ast."order" ASC
			) ast ON ast.application_id = a.id
			WHERE a.user_id = $1
		)
		SELECT
			COALESCE(app_stats.total, 0) AS total_applications,
			COALESCE(app_stats.active, 0) AS active_applications,
			COALESCE(app_stats.closed, 0) AS closed_applications,
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
		&analytics.ResponseRate,
		&analytics.AvgDaysToFirstResponse,
	)
	if err != nil {
		return nil, err
	}

	return analytics, nil
}

// GetFunnel returns stage-based funnel metrics
func (r *AnalyticsRepository) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	query := `
		WITH total_apps AS (
			SELECT COUNT(*) AS total FROM applications WHERE user_id = $1
		),
		stage_counts AS (
			SELECT
				st.name AS stage_name,
				st."order" AS stage_order,
				COUNT(DISTINCT ast.application_id) AS app_count
			FROM stage_templates st
			LEFT JOIN application_stages ast ON ast.stage_template_id = st.id
			LEFT JOIN applications a ON a.id = ast.application_id AND a.user_id = $1
			WHERE st.user_id = $1
			GROUP BY st.id, st.name, st."order"
			ORDER BY st."order"
		),
		ordered_stages AS (
			SELECT 
				stage_name,
				stage_order,
				app_count,
				LAG(app_count) OVER (ORDER BY stage_order) AS prev_count,
				FIRST_VALUE(app_count) OVER (ORDER BY stage_order) AS first_count
			FROM stage_counts
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
				ast.application_id,
				CASE 
					WHEN ast.completed_at IS NOT NULL 
					THEN EXTRACT(EPOCH FROM (ast.completed_at - ast.started_at)) / 86400
					ELSE EXTRACT(EPOCH FROM (NOW() - ast.started_at)) / 86400
				END AS duration_days
			FROM application_stages ast
			JOIN stage_templates st ON st.id = ast.stage_template_id
			JOIN applications a ON a.id = ast.application_id
			WHERE a.user_id = $1
		)
		SELECT
			stage_name,
			stage_order,
			ROUND(AVG(duration_days)::numeric, 2) AS avg_days,
			ROUND(MIN(duration_days)::numeric, 2) AS min_days,
			ROUND(MAX(duration_days)::numeric, 2) AS max_days,
			COUNT(DISTINCT application_id) AS applications_count
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
				COUNT(DISTINCT a.id) AS applications_count,
				COUNT(DISTINCT a.id) FILTER (
					WHERE EXISTS (
						SELECT 1 FROM application_stages ast
						JOIN stage_templates st ON st.id = ast.stage_template_id
						WHERE ast.application_id = a.id AND st."order" > 1
					)
				) AS responses_count,
				COUNT(DISTINCT a.id) FILTER (
					WHERE EXISTS (
						SELECT 1 FROM application_stages ast
						JOIN stage_templates st ON st.id = ast.stage_template_id
						WHERE ast.application_id = a.id 
						AND LOWER(st.name) LIKE '%interview%'
					)
				) AS interviews_count
			FROM resumes r
			LEFT JOIN applications a ON a.resume_id = r.id AND a.user_id = $1
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
				COUNT(DISTINCT a.id) AS applications_count,
				COUNT(DISTINCT a.id) FILTER (
					WHERE EXISTS (
						SELECT 1 FROM application_stages ast
						JOIN stage_templates st ON st.id = ast.stage_template_id
						WHERE ast.application_id = a.id AND st."order" > 1
					)
				) AS responses_count
			FROM applications a
			JOIN jobs j ON j.id = a.job_id
			WHERE a.user_id = $1
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
