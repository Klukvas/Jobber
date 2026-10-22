package interviewprep

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Run owns two bounded polling workers. The application cancels and waits for
// them on shutdown. HTTP cancellation never owns durable operation execution.
func (s *Service) Run(ctx context.Context) {
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := s.RunOnce(ctx); err != nil {
						slog.Error("interview worker storage failure", "outcome", "save_failed")
					}
				}
			}
		}()
	}
	workers.Wait()
}
func (s *Service) claim(ctx context.Context) (Operation, error) {
	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return Operation{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(488120321)`); err != nil {
		return Operation{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE interview_operations SET status='interrupted',error_code='INTERRUPTED',lease_token=NULL,lease_until=NULL,updated_at=now() WHERE status='running' AND lease_until<now()`); err != nil {
		return Operation{}, err
	}
	var running int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM interview_operations WHERE status='running'`).Scan(&running); err != nil {
		return Operation{}, err
	}
	if running >= 2 {
		if err := tx.Commit(ctx); err != nil {
			return Operation{}, err
		}
		return Operation{}, pgx.ErrNoRows
	}
	var o Operation
	err = tx.QueryRow(ctx, `SELECT id::text,user_id::text,job_id::text,kind,input,version_id::text,session_id::text,answer_id::text,attempts,provider_started FROM interview_operations WHERE status='queued' AND attempts<3 ORDER BY created_at,id FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&o.ID, &o.Owner, &o.JobID, &o.Kind, &o.Input, &o.VersionID, &o.SessionID, &o.AnswerID, &o.Attempts, &o.ProviderStarted)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return o, err
		}
		return o, pgx.ErrNoRows
	}
	if err != nil {
		return o, err
	}
	o.Token = uuid.NewString()
	o.Attempts++
	_, err = tx.Exec(ctx, `UPDATE interview_operations SET status='running',attempts=attempts+1,lease_token=$2,lease_until=now()+interval '90 seconds',updated_at=now() WHERE id=$1`, o.ID, o.Token)
	if err != nil {
		return o, err
	}
	return o, tx.Commit(ctx)
}
func (s *Service) RunOnce(ctx context.Context) error {
	operationCtx, cancelOperation := context.WithTimeout(ctx, 80*time.Second)
	defer cancelOperation()
	ctx = operationCtx
	o, err := s.claim(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	started := time.Now()
	input := AIInput{}
	code := ""
	var output []byte
	if err := json.Unmarshal(o.Input, &input); err != nil {
		code = "AI_INVALID_RESPONSE"
	} else {
		// PostgreSQL jsonb adds formatting. Match the exact lossless compact
		// representation measured by admission rather than charging whitespace
		// against the provider's input limit.
		o.Input, err = json.Marshal(input)
		if err != nil || len(o.Input) > maxAIInputBytes {
			code = "AI_INVALID_RESPONSE"
		}
	}
	if code == "" && s.client == nil {
		code = "AI_UNAVAILABLE"
	}
	if code == "" {
		tag, err := s.store.pool.Exec(ctx, `UPDATE interview_operations SET provider_started=true WHERE id=$1 AND user_id=$2 AND lease_token=$3 AND status='running' AND lease_until>now()`, o.ID, o.Owner, o.Token)
		if err != nil {
			code = "SAVE_FAILED"
		} else if tag.RowsAffected() != 1 {
			code = "INTERRUPTED"
		} else {
			// Durable send intent is conservative after a process crash.
			// Known pre-dispatch failures restore the prior flag below.
		}
	}
	if code == "" {
		release, guardErr := s.store.guardDispatch(ctx, o, input.Snapshot)
		if guardErr != nil {
			if errors.Is(guardErr, ErrSourceChanged) {
				code = "SOURCE_CHANGED"
			} else {
				code = "SAVE_FAILED"
			}
		} else {
			priorStarted := o.ProviderStarted
			o.ProviderStarted = true
			callCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			output, err = s.client.Interview(callCtx, o.Kind, o.Input, release)
			releaseErr := release()
			if err != nil {
				if errors.Is(err, ai.ErrInterviewNotDispatched) {
					o.ProviderStarted = priorStarted
					o.Attempts-- // No external attempt occurred; do not exhaust retries.
					code = "AI_UNAVAILABLE"
				} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(callCtx.Err(), context.DeadlineExceeded) {
					code = "AI_TIMEOUT"
				} else if ctx.Err() != nil {
					code = "INTERRUPTED"
				} else {
					code = "AI_FAILED"
				}
			} else if releaseErr != nil {
				code = "SAVE_FAILED"
			}
			cancel()
		}
	}
	var plan Plan
	var review Review
	if code == "" {
		if o.Kind == "plan" {
			plan, err = ValidatePlan(output, input.Snapshot.Facts)
		} else {
			review, err = ValidateReview(output, reviewFacts(input), o.Kind == "feedback" && input.Question != nil && input.Question.Kind != "followup")
		}
		if err != nil {
			code = "AI_INVALID_RESPONSE"
		}
	}
	// A cancelled application context still gets a bounded attempt to persist the
	// interrupted status. A crash leaves a lease that becomes interrupted on restart.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	outcome, err := s.complete(saveCtx, o, input, plan, review, code)
	if outcome == "" {
		outcome = "succeeded"
	}
	if errors.Is(err, ErrFenced) {
		outcome = "fenced"
		err = nil
	} else if err != nil {
		outcome = "save_failed"
	}
	slog.Info("interview AI operation", "operation_id", o.ID, "kind", o.Kind, "attempt", o.Attempts, "duration_ms", time.Since(started).Milliseconds(), "outcome", outcome)
	return err
}
func (s *Service) complete(ctx context.Context, o Operation, input AIInput, plan Plan, review Review, code string) (string, error) {
	err := s.store.transaction(ctx, o.Owner, o.JobID, func(tx pgx.Tx) error {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM interview_operations WHERE id=$1 AND user_id=$2 AND job_id=$3 AND status='running' AND lease_token=$4 AND lease_until>now() FOR UPDATE)`, o.ID, o.Owner, o.JobID, o.Token).Scan(&valid); err != nil {
			return err
		}
		if !valid {
			return ErrFenced
		}
		if code == "" {
			if err := lockSource(ctx, tx, o.Owner, o.JobID, input.Snapshot.ResumeID); err != nil {
				if errors.Is(err, ErrNotFound) {
					code = "SOURCE_CHANGED"
				} else {
					return err
				}
			}
			if code == "" {
				current, err := readSnapshot(ctx, tx, o.Owner, o.JobID, input.Snapshot.ResumeID, input.Snapshot.Locale)
				if errors.Is(err, ErrNotFound) || errors.Is(err, ErrValidation) {
					code = "SOURCE_CHANGED"
				} else if err != nil {
					return err
				} else if current.Hash != input.Snapshot.Hash {
					code = "SOURCE_CHANGED"
				}
			}
		}
		if code != "" && !o.ProviderStarted {
			if _, err := tx.Exec(ctx, `DELETE FROM ai_usage WHERE user_id=$1 AND id=$2 AND usage_type='match_score'`, o.Owner, o.ID); err != nil {
				return err
			}
		}
		released := code != "" && !o.ProviderStarted
		status := "succeeded"
		if code != "" {
			status = "failed"
			if code == "INTERRUPTED" {
				status = "interrupted"
			}
		} else {
			switch o.Kind {
			case "plan":
				id := uuid.NewString()
				snap, _ := json.Marshal(input.Snapshot)
				raw, _ := json.Marshal(plan)
				_, err := tx.Exec(ctx, `INSERT INTO interview_versions(id,user_id,job_id,source_resume_id,provenance_resume_id,source_hash,snapshot,plan) VALUES($1,$2,$3,$4,$4,$5,$6,$7)`, id, o.Owner, o.JobID, input.Snapshot.ResumeID, input.Snapshot.Hash, snap, raw)
				if err != nil {
					return err
				}
				o.VersionID = &id
			case "feedback":
				raw, _ := json.Marshal(review)
				tag, err := tx.Exec(ctx, `UPDATE interview_answers SET feedback=$4 WHERE user_id=$1 AND session_id=$2 AND id=$3`, o.Owner, o.SessionID, o.AnswerID, raw)
				if err != nil {
					return err
				}
				if tag.RowsAffected() != 1 {
					return ErrNotFound
				}
				if review.Followup != nil {
					ss, err := scanSession(tx.QueryRow(ctx, sessionSelect+` AND id=$3 FOR UPDATE`, o.Owner, o.JobID, o.SessionID))
					if err != nil {
						return err
					}
					// A late result never reopens a completed session or changes an answer.
					if ss.Status != "completed" && len(ss.Questions) < 24 {
						q := Question{ID: "followup:" + *o.AnswerID, Kind: "followup", Question: review.Followup.Question, Purpose: review.Followup.Purpose, Difficulty: "medium", Advice: "", Evidence: []Evidence{}, STAR: nil}
						questions := append(append([]Question(nil), ss.Questions...), q)
						ss.Answers, err = loadAnswers(ctx, tx, o.Owner, ss.ID)
						if err != nil {
							return err
						}
						if remainingBudget(input.Snapshot, questions, ss.Answers) < 0 {
							return ErrCapacity
						}
						raw, _ := json.Marshal(questions)
						if _, err := tx.Exec(ctx, `UPDATE interview_sessions SET questions=$4,revision=revision+1 WHERE user_id=$1 AND job_id=$2 AND id=$3`, o.Owner, o.JobID, o.SessionID, raw); err != nil {
							return err
						}
					}
				}
			case "summary":
				raw, _ := json.Marshal(review)
				if _, err := tx.Exec(ctx, `UPDATE interview_sessions SET summary=$4 WHERE user_id=$1 AND job_id=$2 AND id=$3 AND status='completed'`, o.Owner, o.JobID, o.SessionID, raw); err != nil {
					return err
				}
			}
		}

		_, err := tx.Exec(ctx, `UPDATE interview_operations SET status=$5,error_code=$6,version_id=$7,charge_released=$8,provider_started=$9,attempts=$10,lease_token=NULL,lease_until=NULL,updated_at=now() WHERE user_id=$1 AND job_id=$2 AND id=$3 AND lease_token=$4 AND status='running'`, o.Owner, o.JobID, o.ID, o.Token, status, code, o.VersionID, released, o.ProviderStarted, o.Attempts)
		return err
	})
	return code, err
}

// Start returns an idempotent cancel-and-wait function for application shutdown.
func (s *Service) Start(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() { defer workers.Done(); s.Run(ctx) }()
	return func() { cancel(); workers.Wait() }
}
