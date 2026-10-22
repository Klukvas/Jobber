package interviewprep

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	quota "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Client interface {
	Interview(context.Context, string, []byte, func() error) ([]byte, error)
}
type Limits interface {
	ResourceLimit(context.Context, string, string) (int, error)
}
type Service struct {
	store  *Store
	client Client
	limits Limits
}

func NewService(store *Store, client Client, limits Limits) *Service {
	return &Service{store: store, client: client, limits: limits}
}
func (s *Service) Store() *Store   { return s.store }
func (s *Service) Available() bool { return s.client != nil }

const operationSelect = `SELECT id::text,kind,status,error_code,attempts,created_at,version_id::text,session_id::text,answer_id::text,charge_released FROM interview_operations WHERE user_id=$1 AND job_id=$2`

func scanOperation(row scanner) (Operation, error) {
	var o Operation
	err := row.Scan(&o.ID, &o.Kind, &o.Status, &o.ErrorCode, &o.Attempts, &o.CreatedAt, &o.VersionID, &o.SessionID, &o.AnswerID, &o.ChargeReleased)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}
func (s *Service) Generate(ctx context.Context, owner, job, resume, locale, key string) (Operation, error) {
	if !uuidValid(key) || !uuidValid(resume) || (locale != "en" && locale != "ru" && locale != "uk") {
		return Operation{}, ErrValidation
	}
	fp := digest(struct{ Job, Kind, Business string }{job, "plan", resume + ":" + locale})
	if prior, found, err := s.operationReplay(ctx, owner, job, "plan", key, fp); found || err != nil {
		return prior, err
	}
	snap, err := s.store.Snapshot(ctx, owner, job, resume, locale)
	if err != nil {
		return Operation{}, err
	}
	raw, _ := json.Marshal(AIInput{Snapshot: snap})
	if len(raw) > maxSourceInputBytes {
		return Operation{}, ErrInputTooLarge
	}
	return s.enqueue(ctx, owner, job, "plan", key, job+":"+resume+":"+locale+":"+snap.Hash, AIInput{Snapshot: snap}, nil, nil, nil)
}
func (s *Service) Review(ctx context.Context, owner, job, session, answer, key string) (Operation, error) {
	if !uuidValid(key) {
		return Operation{}, ErrValidation
	}
	ss, err := s.store.Session(ctx, owner, job, session)
	if err != nil {
		return Operation{}, err
	}
	var selected *Answer
	replayKind, replayBusiness := "summary", session
	if answer != "" {
		if !uuidValid(answer) {
			return Operation{}, ErrValidation
		}
		for _, a := range ss.Answers {
			if a.ID == answer {
				copy := a
				selected = &copy
			}
		}
		if selected == nil {
			return Operation{}, ErrNotFound
		}
		if selected.Skipped {
			return Operation{}, ErrValidation
		}
		replayKind = "feedback"
		replayBusiness = session + ":" + answer
	}
	fp := digest(struct{ Job, Kind, Business string }{job, replayKind, replayBusiness})
	if prior, found, err := s.operationReplay(ctx, owner, job, replayKind, key, fp); found || err != nil {
		return prior, err
	}
	version, err := s.store.Version(ctx, owner, job, ss.VersionID)
	if err != nil {
		return Operation{}, err
	}
	if version.Stale {
		return Operation{}, ErrSourceChanged
	}
	input := AIInput{Snapshot: Snapshot{JobID: job, ResumeID: version.ResumeBuilderID, ResumeTitle: version.ResumeTitle, JobTitle: version.JobTitle, Locale: version.Locale, Facts: version.Facts, Hash: version.SourceHash}}
	// Load the original bounded Job snapshot; public DTOs never contain private notes.
	var raw []byte
	if err := s.store.pool.QueryRow(ctx, `SELECT snapshot FROM interview_versions WHERE user_id=$1 AND job_id=$2 AND id=$3`, owner, job, version.ID).Scan(&raw); err != nil {
		return Operation{}, err
	}
	if err := json.Unmarshal(raw, &input.Snapshot); err != nil {
		return Operation{}, err
	}
	kind, business := "summary", session
	var answerID *string
	if answer != "" {
		if !uuidValid(answer) {
			return Operation{}, ErrValidation
		}
		kind = "feedback"
		business = answer
		answerID = &answer
		for _, q := range ss.Questions {
			if q.ID == selected.QuestionID {
				copy := questionContext(q)
				input.Question = &copy
			}
		}
		if input.Question == nil {
			return Operation{}, ErrConflict
		}
		input.Answers = []AIAnswer{{ID: selected.ID, QuestionID: selected.QuestionID, Text: selected.Text, Skipped: selected.Skipped}}
	} else {
		if ss.Status != "completed" {
			return Operation{}, ErrConflict
		}
		input = compactInput(input.Snapshot, ss.Questions, ss.Answers)
	}
	return s.enqueue(ctx, owner, job, kind, key, business, input, &version.ID, &session, answerID)
}
func (s *Service) enqueue(ctx context.Context, owner, job, kind, key, business string, input AIInput, version, session, answer *string) (Operation, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return Operation{}, err
	}
	if len(raw) > maxAIInputBytes {
		return Operation{}, ErrValidation
	}
	fingerprintBusiness := business
	if kind == "feedback" && session != nil {
		fingerprintBusiness = *session + ":" + business
	}
	if kind == "plan" {
		fingerprintBusiness = input.Snapshot.ResumeID + ":" + input.Snapshot.Locale
	}
	fingerprint := digest(struct{ Job, Kind, Business string }{job, kind, fingerprintBusiness})
	if prior, found, err := s.operationReplay(ctx, owner, job, kind, key, fingerprint); found || err != nil {
		return prior, err
	}
	limit, limitErr := s.limits.ResourceLimit(ctx, owner, "ai")
	var result Operation
	err = s.store.transaction(ctx, owner, job, func(tx pgx.Tx) error {
		id, err := receipt(ctx, tx, owner, "operation:"+kind, key, fingerprint)
		if err != nil {
			return err
		}
		if id != "" {
			result, err = scanOperation(tx.QueryRow(ctx, operationSelect+` AND id=$3`, owner, job, id))
			return err
		}
		existing, err := scanOperation(tx.QueryRow(ctx, operationSelect+` AND kind=$3 AND business_key=$4 AND (kind!='plan' OR status IN ('queued','running'))`, owner, job, kind, business))
		if err == nil {
			result = existing
			return saveReceipt(ctx, tx, owner, "operation:"+kind, key, fingerprint, result.ID)
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		if s.client == nil {
			return ErrUnavailable
		}
		var pending, recent int
		if err := tx.QueryRow(ctx, `SELECT count(*) FILTER(WHERE status IN ('queued','running')),count(*) FILTER(WHERE created_at>now()-interval '1 minute') FROM interview_operations WHERE user_id=$1`, owner).Scan(&pending, &recent); err != nil {
			return err
		}
		if pending >= 3 || recent >= 10 {
			return ErrRateLimited
		}
		if limitErr != nil {
			return limitErr
		}
		id = uuid.NewString()
		if err := quota.ReserveAIInTransaction(ctx, tx, owner, limit, id); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO interview_operations(id,user_id,job_id,kind,business_key,fingerprint,input,version_id,session_id,answer_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, owner, job, kind, business, fingerprint, raw, version, session, answer)
		if err != nil {
			return err
		}
		if err := saveReceipt(ctx, tx, owner, "operation:"+kind, key, fingerprint, id); err != nil {
			return err
		}
		result, err = scanOperation(tx.QueryRow(ctx, operationSelect+` AND id=$3`, owner, job, id))
		return err
	})
	return result, err
}
func (s *Service) Retry(ctx context.Context, owner, job, id string) (Operation, error) {
	limit, limitErr := s.limits.ResourceLimit(ctx, owner, "ai")
	var result Operation
	err := s.store.transaction(ctx, owner, job, func(tx pgx.Tx) error {
		var err error
		result, err = scanOperation(tx.QueryRow(ctx, operationSelect+` AND id=$3 FOR UPDATE`, owner, job, id))
		if err != nil {
			return err
		}
		if result.Status == "queued" || result.Status == "running" || result.Status == "succeeded" {
			return nil
		}
		if s.client == nil {
			return ErrUnavailable
		}
		if result.ErrorCode == "SOURCE_CHANGED" || result.Attempts >= 3 {
			return ErrConflict
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM interview_operations WHERE user_id=$1 AND status IN ('queued','running')`, owner).Scan(&count); err != nil {
			return err
		}
		if count >= 3 {
			return ErrRateLimited
		}
		if result.Kind == "plan" {
			var active bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM interview_operations a JOIN interview_operations b ON a.user_id=b.user_id AND a.business_key=b.business_key WHERE b.id=$1 AND a.id<>b.id AND a.kind='plan' AND a.status IN ('queued','running'))`, id).Scan(&active); err != nil {
				return err
			}
			if active {
				return ErrConflict
			}
		}
		// A never-called operation may have had its allowance refunded. Reacquire
		// that same receipt atomically; calls already made never charge twice.
		if result.ChargeReleased {
			if limitErr != nil {
				return limitErr
			}
			if err := quota.ReserveAIInTransaction(ctx, tx, owner, limit, result.ID); err != nil {
				return err
			}
		}
		// Retrying the same charged logical operation never inserts ai_usage.
		_, err = tx.Exec(ctx, `UPDATE interview_operations SET status='queued',error_code='',lease_token=NULL,lease_until=NULL,charge_released=false,updated_at=now() WHERE user_id=$1 AND job_id=$2 AND id=$3`, owner, job, id)
		result.Status = "queued"
		result.ErrorCode = ""
		return err
	})
	return result, err
}

func sessionAction(action, job, session string) string {
	return "session:" + action + ":" + job + ":" + session
}
func revisionFingerprint(revision int, value any) string {
	return digest(strconv.Itoa(revision) + ":" + digest(value))
}

func (s *Service) operationReplay(ctx context.Context, owner, job, kind, key, fp string) (Operation, bool, error) {
	var id, stored string
	err := s.store.pool.QueryRow(ctx, `SELECT object_id::text,fingerprint FROM interview_receipts WHERE user_id=$1 AND action=$2 AND request_key=$3`, owner, "operation:"+kind, key).Scan(&id, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, false, nil
	}
	if err != nil {
		return Operation{}, false, err
	}
	if fp != stored {
		return Operation{}, true, ErrConflict
	}
	result, err := scanOperation(s.store.pool.QueryRow(ctx, operationSelect+` AND id=$3`, owner, job, id))
	return result, true, err
}
