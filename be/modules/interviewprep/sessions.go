package interviewprep

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) StartSession(ctx context.Context, owner, job, version, key string) (Session, error) {
	if !uuidValid(version) || !uuidValid(key) {
		return Session{}, ErrValidation
	}
	var id string
	fingerprint := digest(job + ":" + version)
	err := s.transaction(ctx, owner, job, func(tx pgx.Tx) error {
		var err error
		id, err = receipt(ctx, tx, owner, "start_session", key, fingerprint)
		if err != nil || id != "" {
			return err
		}
		v, err := scanVersion(tx.QueryRow(ctx, versionSelect+` AND id=$3`, owner, job, version))
		if err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM interview_sessions WHERE user_id=$1 AND job_id=$2 AND status!='completed'`, owner, job).Scan(&count); err != nil {
			return err
		}
		if count >= 10 {
			return ErrRateLimited
		}
		q, err := json.Marshal(v.Plan.Questions)
		if err != nil {
			return err
		}
		id = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO interview_sessions(id,user_id,job_id,version_id,questions) VALUES($1,$2,$3,$4,$5)`, id, owner, job, version, q)
		if err != nil {
			return err
		}
		return saveReceipt(ctx, tx, owner, "start_session", key, fingerprint, id)
	})
	if err != nil {
		return Session{}, err
	}
	return s.Session(ctx, owner, job, id)
}

type SaveAnswer struct {
	Key        string `json:"key"`
	QuestionID string `json:"question_id"`
	Text       string `json:"text"`
	Skip       bool   `json:"skip"`
	Revision   int    `json:"revision"`
}

func (s *Store) SaveAnswer(ctx context.Context, owner, job, session string, request SaveAnswer) (Session, error) {
	if !uuidValid(request.Key) || !validText(request.Text, 12000, !request.Skip) || request.Revision < 0 || (request.Skip && request.Text != "") {
		return Session{}, ErrValidation
	}
	action := sessionAction("answer", job, session)
	fingerprint := revisionFingerprint(request.Revision, request)
	err := s.transaction(ctx, owner, job, func(tx pgx.Tx) error {
		id, err := receipt(ctx, tx, owner, action, request.Key, fingerprint)
		if err != nil || id != "" {
			return err
		}
		ss, err := scanSession(tx.QueryRow(ctx, sessionSelect+` AND id=$3 FOR UPDATE`, owner, job, session))
		if err != nil {
			return err
		}
		if ss.Status != "active" || ss.Revision != request.Revision {
			return ErrConflict
		}
		answers, err := loadAnswers(ctx, tx, owner, session)
		if err != nil {
			return err
		}
		if len(answers) >= 24 {
			return ErrConflict
		}
		answered := map[string]bool{}
		for _, a := range answers {
			answered[a.QuestionID] = true
		}
		next := ""
		for _, q := range ss.Questions {
			if !answered[q.ID] {
				next = q.ID
				break
			}
		}
		if next == "" || next != request.QuestionID {
			return ErrConflict
		}
		answerID := uuid.NewString()
		snapshot, err := loadSnapshot(ctx, tx, owner, job, ss.VersionID)
		if err != nil {
			return err
		}
		prospective := append(append([]Answer(nil), answers...), Answer{ID: answerID, QuestionID: request.QuestionID, Text: request.Text, Skipped: request.Skip})
		if remainingBudget(snapshot, ss.Questions, prospective) < 0 {
			return ErrCapacity
		}
		_, err = tx.Exec(ctx, `INSERT INTO interview_answers(id,user_id,session_id,question_id,answer_text,skipped) VALUES($1,$2,$3,$4,$5,$6)`, answerID, owner, session, request.QuestionID, request.Text, request.Skip)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE interview_sessions SET revision=revision+1 WHERE user_id=$1 AND job_id=$2 AND id=$3`, owner, job, session)
		if err != nil {
			return err
		}
		return saveReceipt(ctx, tx, owner, action, request.Key, fingerprint, session)
	})
	if err != nil {
		return Session{}, err
	}
	return s.Session(ctx, owner, job, session)
}

type SessionAction struct {
	Key      string `json:"key"`
	Action   string `json:"action"`
	Revision int    `json:"revision"`
}

func (s *Store) Action(ctx context.Context, owner, job, session string, request SessionAction) (Session, error) {
	if !uuidValid(request.Key) || request.Revision < 0 || (request.Action != "pause" && request.Action != "resume" && request.Action != "finish") {
		return Session{}, ErrValidation
	}
	action := sessionAction(request.Action, job, session)
	fp := revisionFingerprint(request.Revision, request)
	err := s.transaction(ctx, owner, job, func(tx pgx.Tx) error {
		id, err := receipt(ctx, tx, owner, action, request.Key, fp)
		if err != nil || id != "" {
			return err
		}
		ss, err := scanSession(tx.QueryRow(ctx, sessionSelect+` AND id=$3 FOR UPDATE`, owner, job, session))
		if err != nil {
			return err
		}
		if ss.Revision != request.Revision || ss.Status == "completed" {
			return ErrConflict
		}
		status := "paused"
		if request.Action == "resume" {
			if ss.Status != "paused" {
				return ErrConflict
			}
			status = "active"
		}
		if request.Action == "finish" {
			status = "completed"
		}
		_, err = tx.Exec(ctx, `UPDATE interview_sessions SET status=$4,revision=revision+1 WHERE user_id=$1 AND job_id=$2 AND id=$3`, owner, job, session, status)
		if err != nil {
			return err
		}
		return saveReceipt(ctx, tx, owner, action, request.Key, fp, session)
	})
	if err != nil {
		return Session{}, err
	}
	return s.Session(ctx, owner, job, session)
}
