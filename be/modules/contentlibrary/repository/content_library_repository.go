package repository

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContentLibraryRepository implements ports.ContentLibraryRepository.
type ContentLibraryRepository struct {
	pool *pgxpool.Pool
}

// NewContentLibraryRepository creates a new ContentLibraryRepository.
func NewContentLibraryRepository(pool *pgxpool.Pool) *ContentLibraryRepository {
	return &ContentLibraryRepository{pool: pool}
}

// Create creates a new content library entry.
func (r *ContentLibraryRepository) Create(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
	query := `INSERT INTO content_library (user_id, title, content, category)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		entry.UserID, entry.Title, entry.Content, entry.Category,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create content library entry: %w", err)
	}

	return entry, nil
}

// GetByID retrieves a content library entry by ID.
func (r *ContentLibraryRepository) GetByID(ctx context.Context, id string) (*model.ContentLibraryEntry, error) {
	query := `SELECT id, user_id, title, content, category, created_at, updated_at
		FROM content_library WHERE id = $1`

	entry := &model.ContentLibraryEntry{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&entry.ID, &entry.UserID, &entry.Title, &entry.Content,
		&entry.Category, &entry.CreatedAt, &entry.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("content library entry not found: %w", err)
	}

	return entry, nil
}

// List retrieves all content library entries for a user.
func (r *ContentLibraryRepository) List(ctx context.Context, userID string) ([]*model.ContentLibraryEntry, error) {
	query := `SELECT id, user_id, title, content, category, created_at, updated_at
		FROM content_library WHERE user_id = $1 ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list content library entries: %w", err)
	}
	defer rows.Close()

	var entries []*model.ContentLibraryEntry
	for rows.Next() {
		entry := &model.ContentLibraryEntry{}
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Title, &entry.Content,
			&entry.Category, &entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan content library entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Update updates a content library entry.
func (r *ContentLibraryRepository) Update(ctx context.Context, entry *model.ContentLibraryEntry) (*model.ContentLibraryEntry, error) {
	query := `UPDATE content_library SET title = $1, content = $2, category = $3
		WHERE id = $4 RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		entry.Title, entry.Content, entry.Category, entry.ID,
	).Scan(&entry.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update content library entry: %w", err)
	}

	return entry, nil
}

// Delete deletes a content library entry.
func (r *ContentLibraryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM content_library WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete content library entry: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("content library entry not found")
	}
	return nil
}
