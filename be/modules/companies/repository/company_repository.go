package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CompanyRepository implements ports.CompanyRepository
type CompanyRepository struct {
	pool *pgxpool.Pool
}

// NewCompanyRepository creates a new company repository
func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

// Create creates a new company
func (r *CompanyRepository) Create(ctx context.Context, company *model.Company) error {
	query := `
		INSERT INTO companies (id, user_id, name, location, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	company.ID = uuid.New().String()
	now := time.Now().UTC()
	company.CreatedAt = now
	company.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		company.ID,
		company.UserID,
		company.Name,
		company.Location,
		company.Notes,
		company.CreatedAt,
		company.UpdatedAt,
	)

	return err
}

// GetByID retrieves a company by ID
func (r *CompanyRepository) GetByID(ctx context.Context, userID, companyID string) (*model.Company, error) {
	query := `
		SELECT id, user_id, name, location, notes, created_at, updated_at
		FROM companies
		WHERE id = $1 AND user_id = $2
	`

	company := &model.Company{}
	err := r.pool.QueryRow(ctx, query, companyID, userID).Scan(
		&company.ID,
		&company.UserID,
		&company.Name,
		&company.Location,
		&company.Notes,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCompanyNotFound
		}
		return nil, err
	}

	return company, nil
}

// GetByIDEnriched retrieves a company by ID with enriched fields
func (r *CompanyRepository) GetByIDEnriched(ctx context.Context, userID, companyID string) (*model.CompanyDTO, error) {
	query := `
		WITH company_apps AS (
			SELECT 
				c.id as company_id,
				a.id as app_id,
				a.status,
				a.updated_at as app_updated,
				COALESCE((SELECT MAX(created_at) FROM application_stages WHERE application_id = a.id), a.updated_at) as stage_activity,
				COALESCE((SELECT MAX(created_at) FROM comments WHERE application_id = a.id), a.updated_at) as comment_activity,
				COALESCE((SELECT COUNT(*) FROM application_stages WHERE application_id = a.id), 0) as stages_count
			FROM companies c
			LEFT JOIN jobs j ON j.company_id = c.id AND j.user_id = c.user_id
			LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = j.user_id
			WHERE c.id = $1 AND c.user_id = $2
		)
		SELECT 
			c.id,
			c.name,
			c.location,
			c.notes,
			c.created_at,
			c.updated_at,
			COALESCE(COUNT(DISTINCT ca.app_id), 0) as applications_count,
			COALESCE(COUNT(DISTINCT ca.app_id) FILTER (WHERE ca.status = 'active'), 0) as active_applications_count,
			MAX(GREATEST(ca.app_updated, ca.stage_activity, ca.comment_activity)) as last_activity_at,
			COALESCE(MAX(ca.stages_count), 0) as max_stages
		FROM companies c
		LEFT JOIN company_apps ca ON ca.company_id = c.id
		WHERE c.id = $1 AND c.user_id = $2
		GROUP BY c.id, c.name, c.location, c.notes, c.created_at, c.updated_at
	`

	var dto model.CompanyDTO
	var maxStages int
	err := r.pool.QueryRow(ctx, query, companyID, userID).Scan(
		&dto.ID,
		&dto.Name,
		&dto.Location,
		&dto.Notes,
		&dto.CreatedAt,
		&dto.UpdatedAt,
		&dto.ApplicationsCount,
		&dto.ActiveApplicationsCount,
		&dto.LastActivityAt,
		&maxStages,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCompanyNotFound
		}
		return nil, err
	}

	// Derive status
	dto.DerivedStatus = r.deriveStatus(dto.ApplicationsCount, dto.ActiveApplicationsCount, maxStages)

	return &dto, nil
}

// List retrieves companies for a user with pagination and enriched fields
func (r *CompanyRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM companies WHERE user_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY clause
	orderBy := "c.name ASC" // default
	if opts.SortBy != "" {
		sortCol := ""
		switch opts.SortBy {
		case "name":
			sortCol = "c.name"
		case "last_activity":
			sortCol = "last_activity_at"
		case "applications_count":
			sortCol = "applications_count"
		default:
			sortCol = "c.name"
		}

		sortDir := "ASC"
		if strings.ToUpper(opts.SortDir) == "DESC" {
			sortDir = "DESC"
		}

		orderBy = fmt.Sprintf("%s %s", sortCol, sortDir)
	}

	// Get paginated results with enriched fields
	query := fmt.Sprintf(`
		WITH company_apps AS (
			SELECT 
				c.id as company_id,
				a.id as app_id,
				a.status,
				a.updated_at as app_updated,
				COALESCE((SELECT MAX(created_at) FROM application_stages WHERE application_id = a.id), a.updated_at) as stage_activity,
				COALESCE((SELECT MAX(created_at) FROM comments WHERE application_id = a.id), a.updated_at) as comment_activity,
				COALESCE((SELECT COUNT(*) FROM application_stages WHERE application_id = a.id), 0) as stages_count
			FROM companies c
			LEFT JOIN jobs j ON j.company_id = c.id AND j.user_id = c.user_id
			LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = j.user_id
			WHERE c.user_id = $1
		)
		SELECT 
			c.id,
			c.name,
			c.location,
			c.notes,
			c.created_at,
			c.updated_at,
			COALESCE(COUNT(DISTINCT ca.app_id), 0) as applications_count,
			COALESCE(COUNT(DISTINCT ca.app_id) FILTER (WHERE ca.status = 'active'), 0) as active_applications_count,
			MAX(GREATEST(ca.app_updated, ca.stage_activity, ca.comment_activity)) as last_activity_at,
			COALESCE(MAX(ca.stages_count), 0) as max_stages
		FROM companies c
		LEFT JOIN company_apps ca ON ca.company_id = c.id
		WHERE c.user_id = $1
		GROUP BY c.id, c.name, c.location, c.notes, c.created_at, c.updated_at
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, orderBy)

	rows, err := r.pool.Query(ctx, query, userID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var companies []*model.CompanyDTO
	for rows.Next() {
		dto := &model.CompanyDTO{}
		var maxStages int
		if err := rows.Scan(
			&dto.ID,
			&dto.Name,
			&dto.Location,
			&dto.Notes,
			&dto.CreatedAt,
			&dto.UpdatedAt,
			&dto.ApplicationsCount,
			&dto.ActiveApplicationsCount,
			&dto.LastActivityAt,
			&maxStages,
		); err != nil {
			return nil, 0, err
		}

		// Derive status
		dto.DerivedStatus = r.deriveStatus(dto.ApplicationsCount, dto.ActiveApplicationsCount, maxStages)

		companies = append(companies, dto)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return companies, total, nil
}

// GetRelatedJobsAndApplicationsCount gets counts of related jobs and applications
func (r *CompanyRepository) GetRelatedJobsAndApplicationsCount(ctx context.Context, userID, companyID string) (jobsCount, appsCount int, err error) {
	query := `
		SELECT 
			COALESCE(COUNT(DISTINCT j.id), 0) as jobs_count,
			COALESCE(COUNT(DISTINCT a.id), 0) as applications_count
		FROM companies c
		LEFT JOIN jobs j ON j.company_id = c.id AND j.user_id = c.user_id
		LEFT JOIN applications a ON a.job_id = j.id AND a.user_id = j.user_id
		WHERE c.id = $1 AND c.user_id = $2
	`

	err = r.pool.QueryRow(ctx, query, companyID, userID).Scan(&jobsCount, &appsCount)
	return
}

// deriveStatus derives company status based on application data
func (r *CompanyRepository) deriveStatus(appsCount, activeAppsCount, maxStages int) string {
	if appsCount == 0 {
		return string(model.CompanyStatusIdle)
	}
	if maxStages > 1 {
		return string(model.CompanyStatusInterviewing)
	}
	if activeAppsCount > 0 {
		return string(model.CompanyStatusActive)
	}
	return string(model.CompanyStatusIdle)
}

// Update updates a company
func (r *CompanyRepository) Update(ctx context.Context, company *model.Company) error {
	query := `
		UPDATE companies
		SET name = $3, location = $4, notes = $5, updated_at = $6
		WHERE id = $1 AND user_id = $2
	`

	company.UpdatedAt = time.Now().UTC()

	result, err := r.pool.Exec(ctx, query,
		company.ID,
		company.UserID,
		company.Name,
		company.Location,
		company.Notes,
		company.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrCompanyNotFound
	}

	return nil
}

// Delete deletes a company
func (r *CompanyRepository) Delete(ctx context.Context, userID, companyID string) error {
	query := `DELETE FROM companies WHERE id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, companyID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return model.ErrCompanyNotFound
	}

	return nil
}
