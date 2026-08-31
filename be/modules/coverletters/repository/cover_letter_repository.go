package repository

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/jackc/pgx/v5"
)

// CoverLetterRepository implements ports.CoverLetterRepository.
type CoverLetterRepository struct {
	pool PgxDB
}

// NewCoverLetterRepository creates a new CoverLetterRepository.
func NewCoverLetterRepository(pool PgxDB) *CoverLetterRepository {
	return &CoverLetterRepository{pool: pool}
}

// Create creates a new cover letter.
func (r *CoverLetterRepository) Create(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
	query := `INSERT INTO cover_letters (user_id, resume_builder_id, job_id, title, template,
		recipient_name, recipient_title, company_name, company_address,
		greeting, paragraphs, closing, font_family, font_size, primary_color)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		cl.UserID, cl.ResumeBuilderID, cl.JobID, cl.Title, cl.Template,
		cl.RecipientName, cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
		cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
	).Scan(&cl.ID, &cl.CreatedAt, &cl.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create cover letter: %w", err)
	}

	return cl, nil
}

// VerifyOwnership reports whether the cover letter exists and belongs to the
// user. It returns ErrCoverLetterNotFound when the row is missing and
// ErrNotAuthorized when it is owned by someone else — letting callers keep a
// 403-vs-404 distinction while GetByID stays owner-scoped as a backstop.
func (r *CoverLetterRepository) VerifyOwnership(ctx context.Context, userID, id string) error {
	var ownerID string
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM cover_letters WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.ErrCoverLetterNotFound
		}
		return err
	}
	if ownerID != userID {
		return model.ErrNotAuthorized
	}
	return nil
}

// GetByID retrieves a cover letter by ID.
func (r *CoverLetterRepository) GetByID(ctx context.Context, userID, id string) (*model.CoverLetter, error) {
	query := `SELECT id, user_id, resume_builder_id, job_id, title, template, recipient_name, recipient_title,
		company_name, company_address, greeting, paragraphs, closing, font_family, font_size, primary_color, created_at, updated_at
		FROM cover_letters WHERE id = $1 AND user_id = $2`

	cl := &model.CoverLetter{}
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&cl.ID, &cl.UserID, &cl.ResumeBuilderID, &cl.JobID, &cl.Title, &cl.Template,
		&cl.RecipientName, &cl.RecipientTitle, &cl.CompanyName, &cl.CompanyAddress,
		&cl.Greeting, &cl.Paragraphs, &cl.Closing, &cl.FontFamily, &cl.FontSize, &cl.PrimaryColor,
		&cl.CreatedAt, &cl.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrCoverLetterNotFound
		}
		return nil, err
	}

	return cl, nil
}

// List retrieves all cover letters for a user.
func (r *CoverLetterRepository) List(ctx context.Context, userID string) ([]*model.CoverLetter, error) {
	query := `SELECT id, user_id, resume_builder_id, job_id, title, template, recipient_name, recipient_title,
		company_name, company_address, greeting, paragraphs, closing, font_family, font_size, primary_color, created_at, updated_at
		FROM cover_letters WHERE user_id = $1 ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cover letters: %w", err)
	}
	defer rows.Close()

	var letters []*model.CoverLetter
	for rows.Next() {
		cl := &model.CoverLetter{}
		if err := rows.Scan(
			&cl.ID, &cl.UserID, &cl.ResumeBuilderID, &cl.JobID, &cl.Title, &cl.Template,
			&cl.RecipientName, &cl.RecipientTitle, &cl.CompanyName, &cl.CompanyAddress,
			&cl.Greeting, &cl.Paragraphs, &cl.Closing, &cl.FontFamily, &cl.FontSize, &cl.PrimaryColor,
			&cl.CreatedAt, &cl.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cover letter: %w", err)
		}
		letters = append(letters, cl)
	}

	return letters, nil
}

// Update updates a cover letter.
func (r *CoverLetterRepository) Update(ctx context.Context, cl *model.CoverLetter) (*model.CoverLetter, error) {
	query := `UPDATE cover_letters SET
		title = $1, resume_builder_id = $2, job_id = $3, template = $4, recipient_name = $5,
		recipient_title = $6, company_name = $7, company_address = $8,
		greeting = $9, paragraphs = $10, closing = $11, font_family = $12, font_size = $13, primary_color = $14
		WHERE id = $15 AND user_id = $16 RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		cl.Title, cl.ResumeBuilderID, cl.JobID, cl.Template, cl.RecipientName,
		cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
		cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
		cl.ID, cl.UserID,
	).Scan(&cl.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrCoverLetterNotFound
		}
		return nil, fmt.Errorf("failed to update cover letter: %w", err)
	}

	return cl, nil
}

// Delete deletes a cover letter.
func (r *CoverLetterRepository) Delete(ctx context.Context, userID, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM cover_letters WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete cover letter: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return model.ErrCoverLetterNotFound
	}
	return nil
}
