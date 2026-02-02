package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
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
	// Get total count
	countQuery := `SELECT COUNT(*) FROM applications WHERE user_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
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
			WHERE a.user_id = $1
		)
		SELECT 
			a.id, a.user_id, a.job_id, a.resume_id, a.name, 
			a.current_stage_id, a.status, a.applied_at, a.created_at, a.updated_at
		FROM applications a
		JOIN last_activities la ON a.id = la.app_id
		WHERE a.user_id = $1 
		ORDER BY %s
		LIMIT $2 OFFSET $3
	`, orderBy)

	rows, err := r.pool.Query(ctx, query, userID, opts.Limit, opts.Offset)
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
