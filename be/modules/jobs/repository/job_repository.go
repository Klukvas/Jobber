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
	"github.com/jackc/pgx/v5/pgconn"
)

// JobRepository implements ports.JobRepository
type JobRepository struct {
	pool PgxDB
}

// NewJobRepository creates a new job repository
func NewJobRepository(pool PgxDB) *JobRepository {
	return &JobRepository{pool: pool}
}

// execer is the write surface shared by the pool and a transaction, so the
// INSERT statements can run either standalone or inside Begin/Commit.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Create creates a new job.
func (r *JobRepository) Create(ctx context.Context, job *model.Job) error {
	return insertJob(ctx, r.pool, job)
}

// CreateWithInitialStage creates a job that starts past the user's first
// pipeline column together with the stage-history row for that column, in one
// transaction, and points the card at it.
//
// A card created straight into "Applied" has entered that column, so the
// history has to say so: without the row the funnel sees the card only through
// its current column and time-to-response has no start timestamp to measure
// from. On success stage.ID and job.CurrentStageID are populated.
func (r *JobRepository) CreateWithInitialStage(ctx context.Context, job *model.Job, stage *model.JobStage) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create-job transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	if err := insertJob(ctx, tx, job); err != nil {
		return err
	}

	stage.ID = uuid.New().String()
	stage.JobID = job.ID
	if stage.CreatedAt.IsZero() {
		stage.CreatedAt = job.CreatedAt
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NULL, $7)`,
		stage.ID, stage.JobID, stage.StageTemplateID, stage.Status, stage.Order, stage.StartedAt, stage.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert initial job stage: %w", err)
	}

	// Separate statement: the FK in each direction forbids writing both
	// pointers in a single INSERT.
	if _, err := tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = $2 WHERE id = $1`,
		job.ID, stage.ID,
	); err != nil {
		return fmt.Errorf("point job at initial stage: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create-job transaction: %w", err)
	}

	job.CurrentStageID = &stage.ID
	return nil
}

// insertJob writes the job row and stamps its generated id and timestamps.
func insertJob(ctx context.Context, db execer, job *model.Job) error {
	query := `
		INSERT INTO jobs (id, user_id, company_id, title, source, url, description,
		                  is_archived, applied_at, resume_id, resume_builder_id,
		                  current_stage_template_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	job.ID = uuid.New().String()
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	_, err := db.Exec(ctx, query,
		job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Description,
		job.IsArchived, job.AppliedAt, job.ResumeID, job.ResumeBuilderID,
		job.CurrentStageTemplateID, job.CreatedAt, job.UpdatedAt,
	)

	return err
}

// GetByID retrieves a job by ID
func (r *JobRepository) GetByID(ctx context.Context, userID, jobID string) (*model.Job, error) {
	query := `
		SELECT id, user_id, company_id, title, source, url, description,
		       is_favorite, is_archived, applied_at, resume_id, resume_builder_id,
		       current_stage_template_id, current_stage_id, created_at, updated_at
		FROM jobs
		WHERE id = $1 AND user_id = $2
	`

	job := &model.Job{}
	err := r.pool.QueryRow(ctx, query, jobID, userID).Scan(
		&job.ID, &job.UserID, &job.CompanyID, &job.Title, &job.Source, &job.URL, &job.Description,
		&job.IsFavorite, &job.IsArchived, &job.AppliedAt, &job.ResumeID, &job.ResumeBuilderID,
		&job.CurrentStageTemplateID, &job.CurrentStageID, &job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrJobNotFound
		}
		return nil, err
	}

	return job, nil
}

// archivedFilter builds the archived WHERE fragment from the (legacy) status
// filter value. "" and the legacy "active" mean "everything except archived"
// (backward compatible with the deployed Chrome extension), "all" means no
// filter, "archived" means only archived. Any other value is treated as
// "not archived" (statuses no longer exist).
func archivedFilter(status string) string {
	switch status {
	case "all":
		return ""
	case "archived":
		return " AND j.is_archived = true"
	default:
		return " AND j.is_archived = false"
	}
}

// searchFilter builds a case-insensitive search WHERE fragment matching the
// job title or the linked company name. LIKE wildcards in the user input are
// escaped so the term is matched literally. Empty search yields no fragment.
func searchFilter(search string, args *[]any) string {
	term := strings.TrimSpace(search)
	if term == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term)
	*args = append(*args, "%"+escaped+"%")
	idx := len(*args)
	return fmt.Sprintf(" AND (j.title ILIKE $%d OR c.name ILIKE $%d)", idx, idx)
}

// companyFilter restricts the list to one company. The id is bound as a
// parameter and cast to uuid, so a malformed value can only ever be a bad
// argument — the handler rejects those before we get here.
func companyFilter(companyID string, args *[]any) string {
	if companyID == "" {
		return ""
	}
	*args = append(*args, companyID)
	return fmt.Sprintf(" AND j.company_id = $%d::uuid", len(*args))
}

// List retrieves enriched jobs for a user with pagination, filtering, and sorting.
// Single query (no N+1): company/resume/current-column joins, last-activity CTEs
// and COUNT(*) OVER() for the total.
func (r *JobRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
	args := []any{userID}
	filter := archivedFilter(opts.Status)
	filter += searchFilter(opts.Search, &args)
	filter += companyFilter(opts.CompanyID, &args)

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
	case "stage":
		orderBy = "(CASE WHEN st.\"order\" IS NULL THEN 1 ELSE 0 END), st.\"order\" " + sortDir
	case "applied_at":
		orderBy = "j.applied_at " + sortDir + " NULLS LAST"
	case "", "last_activity":
		// default set above
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	query := fmt.Sprintf(`
		WITH stage_activity AS (
			SELECT js.job_id, MAX(js.created_at) as max_created
			FROM job_stages js
			JOIN jobs j ON j.id = js.job_id
			WHERE j.user_id = $1
			GROUP BY js.job_id
		),
		comment_activity AS (
			SELECT cm.job_id, MAX(cm.created_at) as max_created
			FROM comments cm
			JOIN jobs j ON j.id = cm.job_id
			WHERE j.user_id = $1
			GROUP BY cm.job_id
		)
		SELECT
			j.id, j.company_id, j.title, j.source, j.url, j.description,
			j.is_favorite, j.is_archived, j.applied_at,
			j.current_stage_template_id, j.current_stage_id,
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
		LEFT JOIN stage_templates st ON st.id = j.current_stage_template_id
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
			&dto.ID, &dto.CompanyID, &dto.Title, &dto.Source, &dto.URL, &dto.Description,
			&dto.IsFavorite, &dto.IsArchived, &dto.AppliedAt,
			&dto.CurrentStageTemplateID, &dto.CurrentStageID,
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
			dto.Resume = &model.ResumeNestedDTO{ID: *resumeID, Name: safeString(resumeTitle), Type: "uploaded"}
		} else if resumeBuilderID != nil {
			dto.Resume = &model.ResumeNestedDTO{ID: *resumeBuilderID, Name: safeString(resumeBuilderTitle), Type: "builder"}
		}

		dtos = append(dtos, dto)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// COUNT(*) OVER() is computed per returned row, so a page past the end
	// yields no rows and therefore no total. Ask for the real count instead of
	// reporting 0 for a non-empty list.
	if len(dtos) == 0 && opts.Offset > 0 {
		countQuery := `
			SELECT COUNT(*)
			FROM jobs j
			LEFT JOIN companies c ON j.company_id = c.id
			WHERE j.user_id = $1` + filter
		if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	return dtos, total, nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Update writes exactly the fields the patch carries and returns the resulting
// row.
//
// Field-scoped on purpose. The old whole-row UPDATE wrote every column from a
// copy of the job read moments earlier, which made two concurrent PATCHes
// overwrite each other's unrelated edits and let a plain "rename the card"
// request stomp applied_at and current_stage_template_id — pipeline state that
// Move/AddStage own and maintain under a row lock. Those columns are not
// listed here at all.
//
// Column names come from compile-time literals; every value is a bound
// parameter.
func (r *JobRepository) Update(ctx context.Context, userID, jobID string, patch *model.JobPatch) (*model.Job, error) {
	// $1/$2 scope the row to its owner; updated_at is always written, so the
	// SET list is never empty.
	args := []any{jobID, userID, time.Now().UTC()}
	sets := []string{"updated_at = $3"}

	set := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if patch.CompanyID != nil {
		set("company_id", nullableID(*patch.CompanyID))
	}
	if patch.ResumeID != nil {
		set("resume_id", nullableID(*patch.ResumeID))
	}
	if patch.ResumeBuilderID != nil {
		set("resume_builder_id", nullableID(*patch.ResumeBuilderID))
	}
	if patch.Title != nil {
		set("title", *patch.Title)
	}
	if patch.Source != nil {
		set("source", *patch.Source)
	}
	if patch.URL != nil {
		set("url", *patch.URL)
	}
	if patch.Description != nil {
		set("description", *patch.Description)
	}
	if patch.IsArchived != nil {
		set("is_archived", *patch.IsArchived)
	}

	query := fmt.Sprintf(`
		UPDATE jobs SET %s
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, company_id, title, source, url, description,
		          is_favorite, is_archived, applied_at, resume_id, resume_builder_id,
		          current_stage_template_id, current_stage_id, created_at, updated_at
	`, strings.Join(sets, ", "))

	job := &model.Job{}
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&job.ID, &job.UserID, &job.CompanyID, &job.Title, &job.Source, &job.URL, &job.Description,
		&job.IsFavorite, &job.IsArchived, &job.AppliedAt, &job.ResumeID, &job.ResumeBuilderID,
		&job.CurrentStageTemplateID, &job.CurrentStageID, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrJobNotFound
		}
		return nil, err
	}

	return job, nil
}

// nullableID turns the API's "an explicit empty string clears the link"
// convention into a SQL NULL. The id columns are UUIDs, so writing an empty
// string instead would fail the cast and surface as a 500.
func nullableID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
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
func (r *JobRepository) GetLastActivityAt(ctx context.Context, userID, jobID string) (time.Time, error) {
	query := `
		SELECT GREATEST(
			j.updated_at,
			COALESCE((SELECT MAX(created_at) FROM job_stages WHERE job_id = j.id), j.updated_at),
			COALESCE((SELECT MAX(created_at) FROM comments WHERE job_id = j.id), j.updated_at)
		) as last_activity_at
		FROM jobs j
		WHERE j.id = $1 AND j.user_id = $2
	`
	var lastActivity time.Time
	err := r.pool.QueryRow(ctx, query, jobID, userID).Scan(&lastActivity)
	return lastActivity, err
}
