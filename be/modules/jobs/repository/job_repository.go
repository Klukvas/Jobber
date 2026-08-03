package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobRepository implements ports.JobRepository
type JobRepository struct {
	pool *pgxpool.Pool
}

// NewJobRepository creates a new job repository
func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

// Create creates a new job
func (r *JobRepository) Create(ctx context.Context, job *model.Job) error {
	query := `
		INSERT INTO jobs (id, user_id, company_id, title, source, url, notes, description,
		                  status, applied_at, resume_id, resume_builder_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	job.ID = uuid.New().String()
	if job.Status == "" {
		job.Status = string(model.StatusSaved)
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		job.ID,
		job.UserID,
		job.CompanyID,
		job.Title,
		job.Source,
		job.URL,
		job.Notes,
		job.Description,
		job.Status,
		job.AppliedAt,
		job.ResumeID,
		job.ResumeBuilderID,
		job.CreatedAt,
		job.UpdatedAt,
	)

	return err
}

// GetByID retrieves a job by ID
func (r *JobRepository) GetByID(ctx context.Context, userID, jobID string) (*model.Job, error) {
	query := `
		SELECT id, user_id, company_id, title, source, url, notes, description, status,
		       is_favorite, applied_at, resume_id, resume_builder_id, current_stage_id,
		       created_at, updated_at
		FROM jobs
		WHERE id = $1 AND user_id = $2
	`

	job := &model.Job{}
	err := r.pool.QueryRow(ctx, query, jobID, userID).Scan(
		&job.ID,
		&job.UserID,
		&job.CompanyID,
		&job.Title,
		&job.Source,
		&job.URL,
		&job.Notes,
		&job.Description,
		&job.Status,
		&job.IsFavorite,
		&job.AppliedAt,
		&job.ResumeID,
		&job.ResumeBuilderID,
		&job.CurrentStageID,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrJobNotFound
		}
		return nil, err
	}

	return job, nil
}

// statusFilter builds the status WHERE fragment.
// "" and the legacy "active" mean "everything except archived" (backward
// compatible with the deployed Chrome extension and pre-merge clients),
// "all" means no filter, otherwise an exact pipeline status match.
func statusFilter(status string, args *[]any) string {
	switch status {
	case "", "active":
		return " AND j.status != 'archived'"
	case "all":
		return ""
	default:
		*args = append(*args, status)
		return fmt.Sprintf(" AND j.status = $%d", len(*args))
	}
}

// List retrieves enriched jobs for a user with pagination, filtering, and sorting.
// Single query (no N+1): company/resume/current-stage joins, last-activity CTEs
// and COUNT(*) OVER() for the total.
func (r *JobRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
	args := []any{userID}
	filter := statusFilter(opts.Status, &args)

	sortDir := "DESC"
	if strings.EqualFold(opts.SortDir, "asc") {
		sortDir = "ASC"
	}

	orderBy := "last_activity_at " + sortDir // default
	switch opts.SortBy {
	case "created_at":
		orderBy = "j.created_at " + sortDir
	case "title":
		orderBy = "LOWER(j.title) " + sortDir
	case "company_name":
		orderBy = "(CASE WHEN c.name IS NULL THEN 1 ELSE 0 END), LOWER(c.name) " + sortDir
	case "status":
		orderBy = "j.status " + sortDir
	case "applied_at":
		orderBy = "j.applied_at " + sortDir + " NULLS LAST"
	case "", "last_activity":
		// default set above
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	query := fmt.Sprintf(`
		WITH stage_activity AS (
			SELECT job_id, MAX(created_at) as max_created
			FROM job_stages
			GROUP BY job_id
		),
		comment_activity AS (
			SELECT job_id, MAX(created_at) as max_created
			FROM comments
			GROUP BY job_id
		)
		SELECT
			j.id, j.company_id, j.title, j.source, j.url, j.notes, j.description,
			j.status, j.is_favorite, j.applied_at, j.current_stage_id,
			j.created_at, j.updated_at,
			GREATEST(
				j.updated_at,
				COALESCE(sa.max_created, j.updated_at),
				COALESCE(ca.max_created, j.updated_at)
			) as last_activity_at,
			c.name as company_name,
			r.id, r.title,
			rb.id, rb.title,
			st.name as current_stage_name,
			COUNT(*) OVER() as total_count
		FROM jobs j
		LEFT JOIN stage_activity sa ON sa.job_id = j.id
		LEFT JOIN comment_activity ca ON ca.job_id = j.id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN resumes r ON r.id = j.resume_id
		LEFT JOIN resume_builders rb ON rb.id = j.resume_builder_id
		LEFT JOIN job_stages cur_stage ON cur_stage.id = j.current_stage_id
		LEFT JOIN stage_templates st ON st.id = cur_stage.stage_template_id
		WHERE j.user_id = $1%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, filter, orderBy, limitIdx, offsetIdx)

	queryArgs := append(args, opts.Limit, opts.Offset)
	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var dtos []*model.JobDTO
	var total int
	for rows.Next() {
		dto := &model.JobDTO{}
		var lastActivity time.Time
		var companyName *string
		var resumeID, resumeTitle *string
		var resumeBuilderID, resumeBuilderTitle *string
		var currentStageName *string

		if err := rows.Scan(
			&dto.ID, &dto.CompanyID, &dto.Title, &dto.Source, &dto.URL, &dto.Notes, &dto.Description,
			&dto.Status, &dto.IsFavorite, &dto.AppliedAt, &dto.CurrentStageID,
			&dto.CreatedAt, &dto.UpdatedAt,
			&lastActivity,
			&companyName,
			&resumeID, &resumeTitle,
			&resumeBuilderID, &resumeBuilderTitle,
			&currentStageName,
			&total,
		); err != nil {
			return nil, 0, err
		}

		dto.LastActivityAt = lastActivity
		dto.CompanyName = companyName
		dto.CurrentStageName = currentStageName

		// Resume (uploaded or builder — mutually exclusive)
		if resumeID != nil {
			dto.Resume = &model.ResumeNestedDTO{
				ID:   *resumeID,
				Name: safeString(resumeTitle),
				Type: "uploaded",
			}
		} else if resumeBuilderID != nil {
			dto.Resume = &model.ResumeNestedDTO{
				ID:   *resumeBuilderID,
				Name: safeString(resumeBuilderTitle),
				Type: "builder",
			}
		}

		dtos = append(dtos, dto)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return dtos, total, nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Update updates a job
func (r *JobRepository) Update(ctx context.Context, job *model.Job) error {
	query := `
		UPDATE jobs
		SET company_id = $3, title = $4, source = $5, url = $6, notes = $7, description = $8,
		    status = $9, applied_at = $10, resume_id = $11, resume_builder_id = $12,
		    current_stage_id = $13, updated_at = $14
		WHERE id = $1 AND user_id = $2
	`

	job.UpdatedAt = time.Now().UTC()

	result, err := r.pool.Exec(ctx, query,
		job.ID,
		job.UserID,
		job.CompanyID,
		job.Title,
		job.Source,
		job.URL,
		job.Notes,
		job.Description,
		job.Status,
		job.AppliedAt,
		job.ResumeID,
		job.ResumeBuilderID,
		job.CurrentStageID,
		job.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrJobNotFound
	}

	return nil
}

// ToggleFavorite toggles the favorite status of a job
func (r *JobRepository) ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error) {
	query := `UPDATE jobs SET is_favorite = NOT is_favorite WHERE id = $1 AND user_id = $2 RETURNING is_favorite`

	var isFavorite bool
	err := r.pool.QueryRow(ctx, query, jobID, userID).Scan(&isFavorite)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, model.ErrJobNotFound
		}
		return false, err
	}

	return isFavorite, nil
}

// Delete deletes a job
func (r *JobRepository) Delete(ctx context.Context, userID, jobID string) error {
	query := `DELETE FROM jobs WHERE id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, jobID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrJobNotFound
	}

	return nil
}

// GetLastActivityAt returns the most recent activity timestamp for a job
// (job update, stage creation or comment creation)
func (r *JobRepository) GetLastActivityAt(ctx context.Context, jobID string) (time.Time, error) {
	query := `
		SELECT GREATEST(
			j.updated_at,
			COALESCE((SELECT MAX(created_at) FROM job_stages WHERE job_id = j.id), j.updated_at),
			COALESCE((SELECT MAX(created_at) FROM comments WHERE job_id = j.id), j.updated_at)
		) as last_activity_at
		FROM jobs j
		WHERE j.id = $1
	`
	var lastActivity time.Time
	err := r.pool.QueryRow(ctx, query, jobID).Scan(&lastActivity)
	return lastActivity, err
}
