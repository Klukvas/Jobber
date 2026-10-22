package interviewprep

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) transaction(ctx context.Context, owner, job string, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "interview:"+owner); err != nil {
		return err
	}
	var owned string
	if err := tx.QueryRow(ctx, `SELECT id FROM jobs WHERE id=$1 AND user_id=$2`, job, owner).Scan(&owned); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func receipt(ctx context.Context, tx pgx.Tx, owner, action, key, fingerprint string) (string, error) {
	var id, stored string
	err := tx.QueryRow(ctx, `SELECT object_id::text,fingerprint FROM interview_receipts WHERE user_id=$1 AND action=$2 AND request_key=$3`, owner, action, key).Scan(&id, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if stored != fingerprint {
		return "", ErrConflict
	}
	return id, nil
}
func saveReceipt(ctx context.Context, tx pgx.Tx, owner, action, key, fingerprint, id string) error {
	_, err := tx.Exec(ctx, `INSERT INTO interview_receipts(user_id,action,request_key,fingerprint,object_id) VALUES($1,$2,$3,$4,$5)`, owner, action, key, fingerprint, id)
	return err
}
func uuidValid(id string) bool { _, err := uuid.Parse(id); return err == nil }

const versionSelect = `SELECT id::text,job_id::text,provenance_resume_id::text,snapshot,plan,studied,source_hash,created_at FROM interview_versions WHERE user_id=$1 AND job_id=$2`

type scanner interface{ Scan(...any) error }

func scanVersion(row scanner) (Version, error) {
	var v Version
	var snapRaw, planRaw []byte
	err := row.Scan(&v.ID, &v.JobID, &v.ResumeBuilderID, &snapRaw, &planRaw, &v.Studied, &v.SourceHash, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	var snap Snapshot
	if err := json.Unmarshal(snapRaw, &snap); err != nil {
		return v, err
	}
	if err := json.Unmarshal(planRaw, &v.Plan); err != nil {
		return v, err
	}
	v.ResumeTitle = snap.ResumeTitle
	v.JobTitle = snap.JobTitle
	v.Locale = snap.Locale
	v.Facts = snap.Facts
	return v, nil
}
func (s *Store) Version(ctx context.Context, owner, job, id string) (Version, error) {
	v, err := scanVersion(s.pool.QueryRow(ctx, versionSelect+` AND id=$3`, owner, job, id))
	if err != nil {
		return v, err
	}
	current, err := s.Snapshot(ctx, owner, job, v.ResumeBuilderID, v.Locale)
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrValidation) {
		v.Stale = true
		return v, nil
	}
	if err != nil {
		return v, err
	}
	v.Stale = current.Hash != v.SourceHash
	return v, nil
}
func (s *Store) Studied(ctx context.Context, owner, job, id, question string, studied bool) (Version, error) {
	err := s.transaction(ctx, owner, job, func(tx pgx.Tx) error {
		v, err := scanVersion(tx.QueryRow(ctx, versionSelect+` AND id=$3 FOR UPDATE`, owner, job, id))
		if err != nil {
			return err
		}
		found := false
		for _, q := range v.Plan.Questions {
			if q.ID == question {
				found = true
			}
		}
		if !found {
			return ErrValidation
		}
		list := []string{}
		for _, q := range v.Studied {
			if q != question {
				list = append(list, q)
			}
		}
		if studied {
			list = append(list, question)
		}
		_, err = tx.Exec(ctx, `UPDATE interview_versions SET studied=$4 WHERE id=$3 AND user_id=$1 AND job_id=$2`, owner, job, id, list)
		return err
	})
	if err != nil {
		return Version{}, err
	}
	return s.Version(ctx, owner, job, id)
}

const sessionSelect = `SELECT id::text,version_id::text,job_id::text,status,revision,questions,summary,created_at FROM interview_sessions WHERE user_id=$1 AND job_id=$2`

func scanSession(row scanner) (Session, error) {
	var s Session
	var q, summary []byte
	err := row.Scan(&s.ID, &s.VersionID, &s.JobID, &s.Status, &s.Revision, &q, &summary, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(q, &s.Questions); err != nil {
		return s, err
	}
	if len(summary) > 0 {
		if err := json.Unmarshal(summary, &s.Summary); err != nil {
			return s, err
		}
	}
	s.Answers = []Answer{}
	return s, nil
}
func loadAnswers(ctx context.Context, tx interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, owner, session string) ([]Answer, error) {
	rows, err := tx.Query(ctx, `SELECT id::text,question_id,answer_text,skipped,feedback,created_at FROM interview_answers WHERE user_id=$1 AND session_id=$2 ORDER BY created_at,id LIMIT 25`, owner, session)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	answers := []Answer{}
	for rows.Next() {
		var a Answer
		var raw []byte
		if err := rows.Scan(&a.ID, &a.QuestionID, &a.Text, &a.Skipped, &raw, &a.CreatedAt); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &a.Feedback); err != nil {
				return nil, err
			}
		}
		answers = append(answers, a)
	}
	return answers, rows.Err()
}
func (s *Store) Session(ctx context.Context, owner, job, id string) (Session, error) {
	result, err := scanSession(s.pool.QueryRow(ctx, sessionSelect+` AND id=$3`, owner, job, id))
	if err != nil {
		return result, err
	}
	result.Answers, err = loadAnswers(ctx, s.pool, owner, id)
	if err != nil {
		return result, err
	}
	snapshot, err := loadSnapshot(ctx, s.pool, owner, job, result.VersionID)
	if err == nil {
		result.AnswerBytesRemaining = max(0, remainingBudget(snapshot, result.Questions, result.Answers))
	}
	return result, err
}
func (s *Store) History(ctx context.Context, owner, job string, offset int, available bool) (History, error) {
	h := History{Versions: []VersionSummary{}, Sessions: []SessionSummary{}, Operations: []Operation{}, AIAvailable: available}
	var owned string
	if err := s.pool.QueryRow(ctx, `SELECT id FROM jobs WHERE id=$1 AND user_id=$2`, job, owner).Scan(&owned); errors.Is(err, pgx.ErrNoRows) {
		return h, ErrNotFound
	} else if err != nil {
		return h, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,job_id::text,provenance_resume_id::text,snapshot->>'resume_title',snapshot->>'job_title',snapshot->>'locale',created_at FROM interview_versions WHERE user_id=$1 AND job_id=$2 ORDER BY created_at DESC,id DESC LIMIT 20 OFFSET $3`, owner, job, offset)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		var v VersionSummary
		e := rows.Scan(&v.ID, &v.JobID, &v.ResumeBuilderID, &v.ResumeTitle, &v.JobTitle, &v.Locale, &v.CreatedAt)
		if e != nil {
			rows.Close()
			return h, e
		}
		h.Versions = append(h.Versions, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return h, err
	}
	// Summary/history stays bounded; full answers are fetched only on session open.
	rows, err = s.pool.Query(ctx, `SELECT id::text,version_id::text,job_id::text,status,revision,created_at FROM interview_sessions WHERE user_id=$1 AND job_id=$2 ORDER BY created_at DESC,id DESC LIMIT 20 OFFSET $3`, owner, job, offset)
	if err != nil {
		return h, err
	}
	for rows.Next() {
		var v SessionSummary
		e := rows.Scan(&v.ID, &v.VersionID, &v.JobID, &v.Status, &v.Revision, &v.CreatedAt)
		if e != nil {
			rows.Close()
			return h, e
		}
		h.Sessions = append(h.Sessions, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return h, err
	}
	rows, err = s.pool.Query(ctx, operationSelect+` ORDER BY created_at DESC,id DESC LIMIT 30 OFFSET $3`, owner, job, offset)
	if err != nil {
		return h, err
	}
	defer rows.Close()
	for rows.Next() {
		o, e := scanOperation(rows)
		if e != nil {
			return h, e
		}
		h.Operations = append(h.Operations, o)
	}
	return h, rows.Err()
}
