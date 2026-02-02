package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
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
		INSERT INTO jobs (id, user_id, company_id, title, source, url, notes, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	job.ID = uuid.New().String()
	job.Status = "active" // Always create as active
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
		job.Status,
		job.CreatedAt,
		job.UpdatedAt,
	)

	return err
}

// GetByID retrieves a job by ID
func (r *JobRepository) GetByID(ctx context.Context, userID, jobID string) (*model.Job, error) {
	query := `
		SELECT id, user_id, company_id, title, source, url, notes, status, created_at, updated_at
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
		&job.Status,
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

// List retrieves jobs for a user with pagination, filtering, and sorting
func (r *JobRepository) List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
	// Default to active status if not specified
	if status == "" {
		status = "active"
	}

	// Build WHERE clause
	whereClause := "j.user_id = $1"
	if status != "all" {
		whereClause += " AND j.status = '" + status + "'"
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM jobs j WHERE ` + whereClause
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine ORDER BY clause
	orderBy := "j.created_at DESC" // default
	if sortBy != "" {
		switch sortBy {
		case "created_at":
			if sortOrder == "asc" {
				orderBy = "j.created_at ASC"
			} else {
				orderBy = "j.created_at DESC"
			}
		case "title":
			// Case-insensitive sorting
			if sortOrder == "asc" {
				orderBy = "LOWER(j.title) ASC"
			} else {
				orderBy = "LOWER(j.title) DESC"
			}
		case "company_name":
			// Handle NULL company names - put them last regardless of sort order
			// Case-insensitive sorting using LOWER()
			// c.name IS NULL check ensures NULLs are always last
			if sortOrder == "asc" {
				orderBy = "(CASE WHEN c.name IS NULL THEN 1 ELSE 0 END), LOWER(c.name) ASC"
			} else {
				orderBy = "(CASE WHEN c.name IS NULL THEN 1 ELSE 0 END), LOWER(c.name) DESC"
			}
		default:
			orderBy = "j.created_at DESC"
		}
	}

	// Get paginated results with enriched data
	query := `
		SELECT 
			j.id, 
			j.user_id, 
			j.company_id, 
			j.title, 
			j.source, 
			j.url, 
			j.notes, 
			j.status,
			j.created_at, 
			j.updated_at,
			c.name as company_name,
			COALESCE(COUNT(a.id), 0) as applications_count
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		LEFT JOIN applications a ON j.id = a.job_id
		WHERE ` + whereClause + `
		GROUP BY j.id, j.user_id, j.company_id, j.title, j.source, j.url, j.notes, j.status, j.created_at, j.updated_at, c.name
		ORDER BY ` + orderBy + `
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*model.JobDTO
	for rows.Next() {
		var companyName *string
		var applicationsCount int
		job := &model.Job{}
		
		if err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.CompanyID,
			&job.Title,
			&job.Source,
			&job.URL,
			&job.Notes,
			&job.Status,
			&job.CreatedAt,
			&job.UpdatedAt,
			&companyName,
			&applicationsCount,
		); err != nil {
			return nil, 0, err
		}
		
		dto := job.ToDTO()
		dto.CompanyName = companyName
		dto.ApplicationsCount = applicationsCount
		jobs = append(jobs, dto)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// Update updates a job
func (r *JobRepository) Update(ctx context.Context, job *model.Job) error {
	query := `
		UPDATE jobs
		SET company_id = $3, title = $4, source = $5, url = $6, notes = $7, status = $8, updated_at = $9
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
		job.Status,
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
