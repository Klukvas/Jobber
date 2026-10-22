package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
)

var _ ports.TailoringRepository = (*ResumeBuilderRepository)(nil)
var _ ports.ResumeCreationRepository = (*ResumeBuilderRepository)(nil)
var _ ports.TailoringTransaction = (*tailoringTransaction)(nil)

func (r *ResumeBuilderRepository) GetTailoringJob(ctx context.Context, userID, jobID string) (*model.TailoringJob, error) {
	return r.getTailoringJob(ctx, userID, jobID, false)
}

func (r *ResumeBuilderRepository) RunResumeCreationTransaction(ctx context.Context, userID string, limit int, fn func(ports.ResumeBuilderRepository) error) error {
	return r.RunTailoringTransaction(ctx, userID, func(tx ports.TailoringTransaction) error {
		if err := tx.CheckResumeBuilderCapacity(ctx, userID, limit); err != nil {
			return err
		}
		return fn(tx)
	})
}

func (r *tailoringTransaction) CheckResumeBuilderCapacity(ctx context.Context, userID string, limit int) error {
	if limit < 0 {
		return nil
	}
	var used int
	if err := r.q.QueryRow(ctx, `SELECT COUNT(*) FROM resume_builders WHERE user_id = $1`, userID).Scan(&used); err != nil {
		return fmt.Errorf("count Builder Resumes: %w", err)
	}
	if used >= limit {
		return subModel.ErrLimitReached
	}
	return nil
}

func (r *ResumeBuilderRepository) getTailoringJob(ctx context.Context, userID, jobID string, lock bool) (*model.TailoringJob, error) {
	query := `SELECT j.id, j.title, COALESCE(j.description, ''), COALESCE(c.name, ''),
		j.resume_id, j.resume_builder_id, j.updated_at
		FROM jobs j LEFT JOIN companies c ON c.id = j.company_id AND c.user_id = j.user_id
		WHERE j.id = $1 AND j.user_id = $2`
	if lock {
		query += ` FOR UPDATE OF j`
	}
	job := &model.TailoringJob{}
	err := r.q.QueryRow(ctx, query, jobID, userID).Scan(&job.ID, &job.Title, &job.Description, &job.Company,
		&job.ResumeID, &job.ResumeBuilderID, &job.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrTailoringNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load tailoring job: %w", err)
	}
	return job, nil
}

// RunTailoringTransaction serializes Applies per user, including the quota recheck.
func (r *ResumeBuilderRepository) RunTailoringTransaction(ctx context.Context, userID string, fn func(ports.TailoringTransaction) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tailoring: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "resume-tailoring:"+userID); err != nil {
		return fmt.Errorf("lock tailoring user: %w", err)
	}
	txRepo := &ResumeBuilderRepository{pool: r.pool, q: tx}
	lockedRepo := &tailoringTransaction{ResumeBuilderRepository: txRepo}
	if err := fn(lockedRepo); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tailoring: %w", err)
	}
	return nil
}

type tailoringTransaction struct {
	*ResumeBuilderRepository
}

func (r *tailoringTransaction) GetTailoringJob(ctx context.Context, userID, jobID string) (*model.TailoringJob, error) {
	return r.getTailoringJob(ctx, userID, jobID, true)
}

func (r *tailoringTransaction) GetFullResume(ctx context.Context, userID, resumeID string) (*model.FullResumeDTO, error) {
	return r.loadFullResume(ctx, userID, resumeID)
}

// LockTailoringSource protects every part of the reviewed snapshot. Parent FOR UPDATE
// blocks FK inserts; section FOR SHARE locks block updates/deletes of existing entries.
func (r *tailoringTransaction) LockTailoringSource(ctx context.Context, userID, resumeID string) (*model.FullResumeDTO, error) {
	var ownedID string
	err := r.q.QueryRow(ctx, `SELECT id FROM resume_builders WHERE id = $1 AND user_id = $2 FOR UPDATE`, resumeID, userID).Scan(&ownedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrTailoringNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock source resume: %w", err)
	}
	for _, table := range []string{"resume_contacts", "resume_summaries", "resume_experiences", "resume_educations", "resume_skills", "resume_languages", "resume_certifications", "resume_projects", "resume_volunteering", "resume_custom_sections", "resume_section_orders"} {
		// Table names come exclusively from the static list above.
		rows, err := r.q.Query(ctx, `SELECT id FROM `+table+` WHERE resume_builder_id = $1 ORDER BY id FOR SHARE`, resumeID)
		if err != nil {
			return nil, fmt.Errorf("lock source sections: %w", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, fmt.Errorf("read locked section: %w", err)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("lock source entries: %w", err)
		}
	}
	return r.loadFullResume(ctx, userID, resumeID)
}

func (r *tailoringTransaction) LinkTailoredResume(ctx context.Context, userID, jobID, resumeID string) error {
	tag, err := r.q.Exec(ctx, `UPDATE jobs SET resume_id = NULL, resume_builder_id = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND user_id = $2`, jobID, userID, resumeID)
	if err != nil {
		return fmt.Errorf("link tailored resume: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return model.ErrTailoringNotFound
	}
	return nil
}
