package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// uniqueViolationCode is PostgreSQL's SQLSTATE for a unique constraint breach.
const uniqueViolationCode = "23505"

type ResumeRepository struct {
	pool PgxDB
}

func NewResumeRepository(pool PgxDB) *ResumeRepository {
	return &ResumeRepository{pool: pool}
}

const insertResumeSQL = `
		INSERT INTO resumes (id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

// countCountableResumesSQL counts the rows that consume a plan slot — see
// model.CountableResumeCondition for what is left out and why.
const countCountableResumesSQL = `
		SELECT COUNT(*) FROM resumes
		WHERE user_id = $1
		  AND ` + model.CountableResumeCondition

// claimPlaceholderSQL turns a presign-era placeholder into the finalized
// resume, in place.
//
// The presign-era upload flow wrote its row up front, so a customer who
// uploaded then and finalizes now has a row already sitting under the id — with
// a real object behind it. An INSERT can only collide with that, and the
// caller's re-lookup saw the same placeholder and answered 404: the upload was
// unfinalizable forever, with a perfectly good PDF in the bucket.
//
// Narrow on purpose. `id` and `user_id` together mean the row is this caller's
// own — the storage key was derived from both — and the placeholder condition
// means it is a row that never became a resume. A real resume under the same id,
// or anybody else's row, matches nothing here and falls through to the INSERT,
// which reports the collision exactly as before.
//
// `created_at` is deliberately not written: the row keeps the moment it was
// first stored. It is returned so the caller's struct describes the row as it
// actually stands, `updated_at` included — which is also what stops the claimed
// row from still matching the placeholder condition.
const claimPlaceholderSQL = `
		UPDATE resumes
		SET title = $3, file_url = $4, storage_type = $5, storage_key = $6,
		    is_active = $7, updated_at = $8
		WHERE id = $1
		  AND user_id = $2
		  AND ` + model.UnfinalizedUploadCondition + `
		RETURNING created_at, updated_at
	`

// claimArgs is the argument list claimPlaceholderSQL expects, in order.
func claimArgs(resume *model.Resume) []any {
	return []any{
		resume.ID, resume.UserID, resume.Title, resume.FileURL,
		resume.StorageType, resume.StorageKey, resume.IsActive,
		resume.UpdatedAt,
	}
}

// lockUserResumesSQL serialises every slot decision for one user.
//
// `pg_advisory_xact_lock` is held to the end of the transaction and released by
// the commit, so a second finalize either waits for the first to land or is
// counted against a total that already includes it. The key is derived from the
// user id, so two different customers never wait on each other.
const lockUserResumesSQL = `SELECT pg_advisory_xact_lock(hashtextextended('resumes:' || $1::text, 0))`

// insertArgs is the argument list insertResumeSQL expects, in order.
func insertArgs(resume *model.Resume) []any {
	return []any{
		resume.ID, resume.UserID, resume.Title, resume.FileURL,
		resume.StorageType, resume.StorageKey, resume.IsActive,
		resume.CreatedAt, resume.UpdatedAt,
	}
}

// stampNewResume fills in the identity and timestamps a fresh row needs.
//
// Writes to the caller's struct on purpose, and the two Create methods say so:
// the service builds its response from the same value, and the id and the two
// timestamps are decided here. Returning a copy would leave every caller
// holding a row with a zero id and zero timestamps.
//
// The id is only generated when the caller has not chosen one: the upload flow
// hands out an id with the presigned URL and finalizes under it, which is what
// makes a repeated finalization recognisable as a duplicate rather than a
// second resume.
func stampNewResume(resume *model.Resume) {
	if resume.ID == "" {
		resume.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	resume.CreatedAt = now
	resume.UpdatedAt = now
}

// asWriteError translates a duplicate primary key into the sentinel the
// upload-finalization path reads. It retries with a fixed id, so the caller has
// to tell "already done" apart from a genuine write failure.
func asWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return model.ErrResumeAlreadyExists
	}
	return err
}

// Create writes a complete resume. `resume` is filled in as it is stored: its
// id, if it had none, and its timestamps.
func (r *ResumeRepository) Create(ctx context.Context, resume *model.Resume) error {
	stampNewResume(resume)

	if _, err := r.pool.Exec(ctx, insertResumeSQL, insertArgs(resume)...); err != nil {
		return asWriteError(err)
	}
	return nil
}

// CreateFinalizedUpload inserts a verified upload, but only while the customer
// is still under `maxResumes`. A negative limit means the plan is unlimited.
//
// The limit used to be checked by the subscription service and then acted on by
// a separate INSERT. Between those two statements nothing held: two uploads
// finalizing at once — two tabs, a double-tap, a retry racing its own first
// attempt — both read "two of three used" and both wrote, leaving a free plan
// with four resumes and no way for the product to notice. The count and the
// write are one transaction here, serialised per user by an advisory lock, so
// the second one counts a total that already includes the first.
//
// A row left behind by the presign-era upload flow is *claimed* rather than
// collided with — see claimPlaceholderSQL. It is the same slot decision either
// way: a placeholder is not counted, and the resume it becomes is, so both
// paths sit inside this transaction and behind this lock.
//
// What this does *not* change: ownership (the row carries the authenticated
// user id its caller built the storage key from, and the claim matches on both)
// and idempotency (a repeated finalization of the same upload still surfaces as
// ErrResumeAlreadyExists, which the caller answers with the row that already
// exists — and it is answered before this is ever reached).
//
// Like Create, this fills `resume` in as it is stored.
func (r *ResumeRepository) CreateFinalizedUpload(ctx context.Context, resume *model.Resume, maxResumes int) error {
	stampNewResume(resume)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin resume creation for user %s: %w", resume.UserID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	if _, err := tx.Exec(ctx, lockUserResumesSQL, resume.UserID); err != nil {
		return fmt.Errorf("failed to lock resume slots for user %s: %w", resume.UserID, err)
	}

	if maxResumes >= 0 {
		var used int
		if err := tx.QueryRow(ctx, countCountableResumesSQL, resume.UserID).Scan(&used); err != nil {
			return fmt.Errorf("failed to count resumes for user %s: %w", resume.UserID, err)
		}
		if used >= maxResumes {
			return model.ErrResumeLimitReached
		}
	}

	// Claim a placeholder this upload already owns, if there is one. Inside the
	// same transaction and behind the same lock as the count, so a claim is a
	// slot decision like any other: the row it takes over was not being
	// counted, and the resume it becomes is.
	err = tx.QueryRow(ctx, claimPlaceholderSQL, claimArgs(resume)...).
		Scan(&resume.CreatedAt, &resume.UpdatedAt)
	switch {
	case err == nil:
		// Claimed. Nothing left to insert.
	case errors.Is(err, pgx.ErrNoRows):
		// Nothing to claim, which is the ordinary case: no build since the
		// presign era writes a placeholder.
		if _, err := tx.Exec(ctx, insertResumeSQL, insertArgs(resume)...); err != nil {
			return asWriteError(err)
		}
	default:
		// A failed UPDATE is a write failure, not "nothing to claim". Falling
		// through to the INSERT would turn it into a duplicate-key error, and
		// the caller answers that with a 404.
		return fmt.Errorf("failed to claim upload placeholder %s: %w", resume.ID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit resume %s: %w", resume.ID, err)
	}
	return nil
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
			COALESCE(COUNT(j.id), 0) as applications_count
		FROM resumes r
		LEFT JOIN jobs j ON r.id = j.resume_id AND j.applied_at IS NOT NULL
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return model.ErrResumeInUse
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrResumeNotFound
	}
	return nil
}
