package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplicationRepository struct {
	pool *pgxpool.Pool
}

func NewApplicationRepository(pool *pgxpool.Pool) *ApplicationRepository {
	return &ApplicationRepository{pool: pool}
}

func (r *ApplicationRepository) Create(ctx context.Context, app *model.Application) error {
	query := `
		INSERT INTO applications (id, user_id, job_id, resume_id, name, current_stage_id, status, applied_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	app.ID = uuid.New().String()
	now := time.Now().UTC()
	app.CreatedAt = now
	app.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		app.ID, app.UserID, app.JobID, app.ResumeID, app.Name, app.CurrentStageID, app.Status, app.AppliedAt, app.CreatedAt, app.UpdatedAt,
	)
	return err
}

func (r *ApplicationRepository) GetByID(ctx context.Context, userID, appID string) (*model.Application, error) {
	query := `
		SELECT id, user_id, job_id, resume_id, name, current_stage_id, status, applied_at, created_at, updated_at
		FROM applications WHERE id = $1 AND user_id = $2
	`

	app := &model.Application{}
	err := r.pool.QueryRow(ctx, query, appID, userID).Scan(
		&app.ID, &app.UserID, &app.JobID, &app.ResumeID, &app.Name, &app.CurrentStageID, &app.Status, &app.AppliedAt, &app.CreatedAt, &app.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrApplicationNotFound
		}
		return nil, err
	}
	return app, nil
}

func (r *ApplicationRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.Application, int, error) {
	// Build optional status filter
	statusFilter := ""
	args := []any{userID}
	if opts.Status != "" {
		statusFilter = fmt.Sprintf(" AND a.status = $%d", len(args)+1)
		args = append(args, opts.Status)
	}

	// Get total count
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM applications a WHERE a.user_id = $1%s`, statusFilter)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY clause
	orderBy := "applied_at DESC" // default
	if opts.SortBy != "" {
		sortCol := ""
		switch opts.SortBy {
		case "last_activity":
			// We'll need to calculate this in a subquery
			sortCol = "last_activity_at"
		case "status":
			sortCol = "status"
		case "applied_at":
			sortCol = "applied_at"
		default:
			sortCol = "applied_at"
		}

		sortDir := "DESC"
		if strings.ToUpper(opts.SortDir) == "ASC" {
			sortDir = "ASC"
		}

		orderBy = fmt.Sprintf("%s %s", sortCol, sortDir)
	}

	// Get paginated results with last_activity calculation
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	query := fmt.Sprintf(`
		WITH last_activities AS (
			SELECT
				a.id as app_id,
				GREATEST(
					a.updated_at,
					COALESCE((SELECT MAX(created_at) FROM application_stages WHERE application_id = a.id), a.updated_at),
					COALESCE((SELECT MAX(created_at) FROM comments WHERE application_id = a.id), a.updated_at)
				) as last_activity_at
			FROM applications a
			WHERE a.user_id = $1%s
		)
		SELECT
			a.id, a.user_id, a.job_id, a.resume_id, a.name,
			a.current_stage_id, a.status, a.applied_at, a.created_at, a.updated_at
		FROM applications a
		JOIN last_activities la ON a.id = la.app_id
		WHERE a.user_id = $1%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, statusFilter, statusFilter, orderBy, limitIdx, offsetIdx)

	queryArgs := append(args, opts.Limit, opts.Offset)
	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var apps []*model.Application
	for rows.Next() {
		app := &model.Application{}
		if err := rows.Scan(&app.ID, &app.UserID, &app.JobID, &app.ResumeID, &app.Name, &app.CurrentStageID, &app.Status, &app.AppliedAt, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, 0, err
		}
		apps = append(apps, app)
	}
	return apps, total, rows.Err()
}

// ListEnriched returns enriched ApplicationDTOs via JOINs (single query, no N+1).
func (r *ApplicationRepository) ListEnriched(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.ApplicationDTO, int, error) {
	// Build optional status filter
	statusFilter := ""
	args := []any{userID}
	if opts.Status != "" {
		statusFilter = fmt.Sprintf(" AND a.status = $%d", len(args)+1)
		args = append(args, opts.Status)
	}

	// Build ORDER BY clause
	orderBy := "last_activity_at DESC" // default
	if opts.SortBy != "" {
		sortCol := ""
		switch opts.SortBy {
		case "last_activity":
			sortCol = "last_activity_at"
		case "status":
			sortCol = "a.status"
		case "applied_at":
			sortCol = "a.applied_at"
		default:
			sortCol = "last_activity_at"
		}

		sortDir := "DESC"
		if strings.ToUpper(opts.SortDir) == "ASC" {
			sortDir = "ASC"
		}

		orderBy = fmt.Sprintf("%s %s", sortCol, sortDir)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	query := fmt.Sprintf(`
		WITH stage_activity AS (
			SELECT application_id, MAX(created_at) as max_created
			FROM application_stages
			GROUP BY application_id
		),
		comment_activity AS (
			SELECT application_id, MAX(created_at) as max_created
			FROM comments
			GROUP BY application_id
		)
		SELECT
			a.id, a.name, a.status, a.applied_at, a.created_at, a.updated_at,
			a.current_stage_id,
			GREATEST(
				a.updated_at,
				COALESCE(sa.max_created, a.updated_at),
				COALESCE(ca.max_created, a.updated_at)
			) as last_activity_at,
			j.id, j.title,
			c.id, c.name, c.location, c.notes, c.is_favorite, c.created_at, c.updated_at,
			r.id, r.title,
			st.name as current_stage_name,
			COUNT(*) OVER() as total_count
		FROM applications a
		LEFT JOIN stage_activity sa ON sa.application_id = a.id
		LEFT JOIN comment_activity ca ON ca.application_id = a.id
		LEFT JOIN jobs j ON j.id = a.job_id
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN resumes r ON r.id = a.resume_id
		LEFT JOIN application_stages cur_stage ON cur_stage.id = a.current_stage_id
		LEFT JOIN stage_templates st ON st.id = cur_stage.stage_template_id
		WHERE a.user_id = $1%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, statusFilter, orderBy, limitIdx, offsetIdx)

	queryArgs := append(args, opts.Limit, opts.Offset)
	rows, err := r.pool.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var dtos []*model.ApplicationDTO
	var total int
	for rows.Next() {
		dto := &model.ApplicationDTO{}
		var lastActivity time.Time
		var jobID, jobTitle *string
		var companyID, companyName *string
		var companyLocation, companyNotes *string
		var companyIsFavorite *bool
		var companyCreatedAt, companyUpdatedAt *time.Time
		var resumeID, resumeTitle *string
		var currentStageName *string

		if err := rows.Scan(
			&dto.ID, &dto.Name, &dto.Status, &dto.AppliedAt, &dto.CreatedAt, &dto.UpdatedAt,
			&dto.CurrentStageID,
			&lastActivity,
			&jobID, &jobTitle,
			&companyID, &companyName, &companyLocation, &companyNotes, &companyIsFavorite, &companyCreatedAt, &companyUpdatedAt,
			&resumeID, &resumeTitle,
			&currentStageName,
			&total,
		); err != nil {
			return nil, 0, err
		}

		dto.LastActivityAt = lastActivity
		dto.CurrentStageName = currentStageName

		// Build nested Job + Company
		if jobID != nil {
			dto.Job = &model.JobNestedDTO{
				ID:    *jobID,
				Title: safeString(jobTitle),
			}
			if companyID != nil {
				dto.Job.Company = &companyModel.CompanyDTO{
					ID:         *companyID,
					Name:       safeString(companyName),
					Location:   companyLocation,
					Notes:      companyNotes,
					IsFavorite: safeBool(companyIsFavorite),
				}
				if companyCreatedAt != nil {
					dto.Job.Company.CreatedAt = *companyCreatedAt
				}
				if companyUpdatedAt != nil {
					dto.Job.Company.UpdatedAt = *companyUpdatedAt
				}
			}
		}

		// Build nested Resume
		if resumeID != nil {
			dto.Resume = &model.ResumeNestedDTO{
				ID:   *resumeID,
				Name: safeString(resumeTitle),
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

func safeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func (r *ApplicationRepository) Update(ctx context.Context, app *model.Application) error {
	query := `
		UPDATE applications SET current_stage_id = $3, status = $4, updated_at = $5
		WHERE id = $1 AND user_id = $2
	`

	app.UpdatedAt = time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, app.ID, app.UserID, app.CurrentStageID, app.Status, app.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrApplicationNotFound
	}
	return nil
}

func (r *ApplicationRepository) Delete(ctx context.Context, userID, appID string) error {
	query := `DELETE FROM applications WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, appID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrApplicationNotFound
	}
	return nil
}

func (r *ApplicationRepository) GetLastActivityAt(ctx context.Context, appID string) (time.Time, error) {
	query := `
		SELECT GREATEST(
			a.updated_at,
			COALESCE((SELECT MAX(created_at) FROM application_stages WHERE application_id = a.id), a.updated_at),
			COALESCE((SELECT MAX(created_at) FROM comments WHERE application_id = a.id), a.updated_at)
		) as last_activity_at
		FROM applications a
		WHERE a.id = $1
	`
	var lastActivity time.Time
	err := r.pool.QueryRow(ctx, query, appID).Scan(&lastActivity)
	return lastActivity, err
}
