package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResumeRepository struct {
	pool *pgxpool.Pool
}

func NewResumeRepository(pool *pgxpool.Pool) *ResumeRepository {
	return &ResumeRepository{pool: pool}
}

func (r *ResumeRepository) Create(ctx context.Context, resume *model.Resume) error {
	query := `
		INSERT INTO resumes (id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// Only generate ID if not already set (S3 upload flow sets it)
	if resume.ID == "" {
		resume.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	resume.CreatedAt = now
	resume.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, resume.CreatedAt, resume.UpdatedAt,
	)
	return err
}

func (r *ResumeRepository) GetByID(ctx context.Context, userID, resumeID string) (*model.Resume, error) {
	query := `
		SELECT id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at
		FROM resumes WHERE id = $1 AND user_id = $2
	`

	resume := &model.Resume{}
	err := r.pool.QueryRow(ctx, query, resumeID, userID).Scan(
		&resume.ID, &resume.UserID, &resume.Title, &resume.FileURL, &resume.StorageType, &resume.StorageKey, &resume.IsActive, &resume.CreatedAt, &resume.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrResumeNotFound
		}
		return nil, err
	}
	return resume, nil
}

func (r *ResumeRepository) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM resumes WHERE user_id = $1`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine ORDER BY clause
	orderClause := "r.created_at DESC"
	switch sortBy {
	case "created_at":
		if sortDir == "asc" {
			orderClause = "r.created_at ASC"
		} else {
			orderClause = "r.created_at DESC"
		}
	case "title":
		if sortDir == "desc" {
			orderClause = "r.title DESC"
		} else {
			orderClause = "r.title ASC"
		}
	case "is_active":
		if sortDir == "asc" {
			orderClause = "r.is_active ASC, r.created_at DESC"
		} else {
			orderClause = "r.is_active DESC, r.created_at DESC"
		}
	}

	// Get paginated results with applications count
	query := `
		SELECT 
			r.id, 
			r.user_id, 
			r.title, 
			r.file_url, 
			r.storage_type,
			r.storage_key,
			r.is_active, 
			r.created_at, 
			r.updated_at,
			COALESCE(COUNT(a.id), 0) as applications_count
		FROM resumes r
		LEFT JOIN applications a ON r.id = a.resume_id
		WHERE r.user_id = $1
		GROUP BY r.id, r.user_id, r.title, r.file_url, r.storage_type, r.storage_key, r.is_active, r.created_at, r.updated_at
		ORDER BY ` + orderClause + `
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var resumesWithCounts []*ports.ResumeWithCount
	for rows.Next() {
		resume := &model.Resume{}
		var applicationsCount int
		if err := rows.Scan(
			&resume.ID,
			&resume.UserID,
			&resume.Title,
			&resume.FileURL,
			&resume.StorageType,
			&resume.StorageKey,
			&resume.IsActive,
			&resume.CreatedAt,
			&resume.UpdatedAt,
			&applicationsCount,
		); err != nil {
			return nil, 0, err
		}
		resumesWithCounts = append(resumesWithCounts, &ports.ResumeWithCount{
			Resume:            resume,
			ApplicationsCount: applicationsCount,
		})
	}
	return resumesWithCounts, total, rows.Err()
}

func (r *ResumeRepository) Update(ctx context.Context, resume *model.Resume) error {
	query := `
		UPDATE resumes SET title = $3, file_url = $4, storage_type = $5, storage_key = $6, is_active = $7, updated_at = $8
		WHERE id = $1 AND user_id = $2
	`

	resume.UpdatedAt = time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, resume.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrResumeNotFound
	}
	return nil
}

func (r *ResumeRepository) Delete(ctx context.Context, userID, resumeID string) error {
	query := `DELETE FROM resumes WHERE id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, resumeID, userID)
	if err != nil {
		// Check if error is foreign key constraint violation (resume used in applications)
		// PostgreSQL error code 23503 = foreign_key_violation
		if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "23503") {
			return model.ErrResumeInUse
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrResumeNotFound
	}
	return nil
}
