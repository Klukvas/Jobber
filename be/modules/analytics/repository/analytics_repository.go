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

// Fixed pipeline phases, in funnel order. Rejection is terminal (reported
// separately by getRejectedSummary), so it is not one of these buckets.
var funnelPhases = []struct {
	key   string
	order int
}{
	{"applied", 1},
	{"in_progress", 2},
	{"offer", 3},
}

// GetFunnel returns the phase-based application funnel.
//
// Buckets are the fixed pipeline phases (Applied → In Progress → Offer). A job
// "reaches" a phase when its current status maps to that phase or higher, or
// when it recorded a stage whose template belongs to that phase — so progress
// still counts for jobs that were later rejected. The In Progress phase carries
// a drill-down of the user's own in-progress stage templates. Rejection is a
// terminal status, reported separately (getRejectedSummary), not as a bucket.
func (r *AnalyticsRepository) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	const phaseQuery = `
		WITH applied_jobs AS (
			SELECT j.id, j.status
			FROM jobs j
			WHERE j.user_id = $1 AND j.applied_at IS NOT NULL
		),
		job_reach AS (
			SELECT
				aj.id,
				GREATEST(
					1,
					CASE aj.status WHEN 'offer' THEN 3 WHEN 'on_hold' THEN 2 ELSE 1 END,
					COALESCE(MAX(CASE st.phase WHEN 'offer' THEN 3 WHEN 'in_progress' THEN 2 ELSE 1 END), 1)
				) AS reached
			FROM applied_jobs aj
			LEFT JOIN job_stages js ON js.job_id = aj.id
			LEFT JOIN stage_templates st ON st.id = js.stage_template_id
			GROUP BY aj.id, aj.status
		)
		SELECT
			COUNT(*) AS applied,
			COUNT(*) FILTER (WHERE reached >= 2) AS in_progress,
			COUNT(*) FILTER (WHERE reached >= 3) AS offer
		FROM job_reach
	`

	var applied, inProgress, offer int
	if err := r.pool.QueryRow(ctx, phaseQuery, userID).Scan(&applied, &inProgress, &offer); err != nil {
		return nil, err
	}

	// No applications → empty funnel (the frontend shows an empty state).
	if applied == 0 {
		return &model.FunnelAnalytics{}, nil
	}

	subStages, err := r.getInProgressBreakdown(ctx, userID)
	if err != nil {
		return nil, err
	}

	rejected, err := r.getRejectedSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &model.FunnelAnalytics{
		Stages:   buildPhaseFunnel(applied, inProgress, offer, subStages),
		Rejected: rejected,
	}, nil
}

// getInProgressBreakdown lists the user's in-progress stage templates that at
// least one applied job reached — the drill-down under the In Progress phase.
func (r *AnalyticsRepository) getInProgressBreakdown(ctx context.Context, userID string) ([]model.FunnelStage, error) {
	const query = `
		SELECT st.name, st."order", COUNT(DISTINCT js.job_id) AS cnt
		FROM stage_templates st
		JOIN job_stages js ON js.stage_template_id = st.id
		JOIN jobs j ON j.id = js.job_id AND j.user_id = $1 AND j.applied_at IS NOT NULL
		WHERE st.user_id = $1 AND st.phase = 'in_progress'
		GROUP BY st.id, st.name, st."order"
		HAVING COUNT(DISTINCT js.job_id) > 0
		ORDER BY st."order", st.name
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []model.FunnelStage
	for rows.Next() {
		var s model.FunnelStage
		if err := rows.Scan(&s.StageName, &s.StageOrder, &s.Count); err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}
	return stages, rows.Err()
}

// buildPhaseFunnel turns the three phase counts into funnel buckets, computing
// conversion/drop-off relative to the previous phase and attaching the
// in-progress drill-down. Pure — unit-tested.
func buildPhaseFunnel(applied, inProgress, offer int, inProgressSub []model.FunnelStage) []model.FunnelStage {
	counts := []int{applied, inProgress, offer}
	stages := make([]model.FunnelStage, 0, len(funnelPhases))
	prev := 0
	for i, p := range funnelPhases {
		s := model.FunnelStage{StageName: p.key, StageOrder: p.order, Count: counts[i]}
		if i == 0 || prev == 0 {
			s.ConversionRate = 100
			s.DropOffRate = 0
		} else {
			s.ConversionRate = roundRate(float64(counts[i]) / float64(prev) * 100)
			s.DropOffRate = roundRate(float64(prev-counts[i]) / float64(prev) * 100)
		}
		if p.key == "in_progress" {
			s.SubStages = inProgressSub
		}
		stages = append(stages, s)
		prev = counts[i]
	}
	return stages
}

func roundRate(v float64) float64 {
	return math.Round(v*100) / 100
}

// getRejectedSummary groups rejected applications by the furthest stage they
// reached. Applications rejected with no tracked stages fall into the
// synthetic "Applied" bucket (order 1) — same convention as the funnel.
func (r *AnalyticsRepository) getRejectedSummary(ctx context.Context, userID string) (*model.RejectedSummary, error) {
	query := `
		WITH rejected AS (
			SELECT j.id
			FROM jobs j
			WHERE j.user_id = $1 AND j.applied_at IS NOT NULL AND j.status = 'rejected'
		),
		last_stage AS (
			SELECT DISTINCT ON (js.job_id)
				js.job_id, st.name, st."order"
			FROM job_stages js
			JOIN stage_templates st ON st.id = js.stage_template_id
			JOIN rejected r ON r.id = js.job_id
			ORDER BY js.job_id, st."order" DESC
		)
		SELECT
			COALESCE(ls.name, 'Applied') AS stage_name,
			COALESCE(ls."order", 1) AS stage_order,
			COUNT(*) AS count
		FROM rejected r
		LEFT JOIN last_stage ls ON ls.job_id = r.id
		GROUP BY 1, 2
		ORDER BY 2
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := &model.RejectedSummary{}
	for rows.Next() {
		var sc model.RejectedStageCount
		if err := rows.Scan(&sc.StageName, &sc.StageOrder, &sc.Count); err != nil {
			return nil, err
		}
		summary.ByStage = append(summary.ByStage, sc)
		summary.Total += sc.Count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if summary.Total == 0 {
		return nil, nil
	}
	return summary, nil
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
