package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	subRepository "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	"github.com/jackc/pgx/v5"
)

func (r *tailoringTransaction) GetOperation(ctx context.Context, user, kind, key string) (*model.WorkflowOperation, error) {
	return r.getOperation(ctx, user, kind, key)
}
func (r *ResumeBuilderRepository) getOperation(ctx context.Context, user, kind, key string) (*model.WorkflowOperation, error) {
	op := &model.WorkflowOperation{}
	err := r.q.QueryRow(ctx, `SELECT user_id,kind,request_key,job_id,fingerprint,status,COALESCE(object_id::text,''),payload,error_code,created_at,COALESCE(ai_receipt_id::text,'') FROM resume_tailoring_operations WHERE user_id=$1 AND kind=$2 AND request_key=$3`, user, kind, key).Scan(&op.UserID, &op.Kind, &op.Key, &op.JobID, &op.Fingerprint, &op.Status, &op.ObjectID, &op.Result, &op.ErrorCode, &op.CreatedAt, &op.AIReceiptID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load tailoring receipt: %w", err)
	}
	return op, nil
}
func (r *tailoringTransaction) PutOperation(ctx context.Context, op *model.WorkflowOperation) error {
	_, err := r.q.Exec(ctx, `INSERT INTO resume_tailoring_operations(user_id,kind,request_key,job_id,fingerprint,status,object_id,payload,error_code,provider_started,deadline_at,created_at,ai_receipt_id) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8,$9,$10,$11,$12,NULLIF($13,'')::uuid) ON CONFLICT(user_id,kind,request_key) DO UPDATE SET status=EXCLUDED.status,job_id=EXCLUDED.job_id,object_id=EXCLUDED.object_id,payload=EXCLUDED.payload,error_code=EXCLUDED.error_code,fingerprint=EXCLUDED.fingerprint,updated_at=now()`, op.UserID, op.Kind, op.Key, op.JobID, op.Fingerprint, op.Status, op.ObjectID, op.Result, op.ErrorCode, op.Kind == "import" || op.Kind == "generate", op.CreatedAt.Add(90*time.Second), op.CreatedAt, op.AIReceiptID)
	if err != nil {
		return fmt.Errorf("save tailoring receipt: %w", err)
	}
	if op.AIReceiptID != "" && (op.Kind == "import" || op.Kind == "generate") && (op.Status == "succeeded" || op.Status == "failed") {
		if credit, _, ok := ai.CurrentCreditOperation(ctx); ok {
			tx, ok := r.q.(pgx.Tx)
			if !ok {
				return model.ErrProposalStoreUnavailable
			}
			if credit.Key != op.AIReceiptID {
				return model.ErrOperationConflict
			}
			return subRepository.FinalizeCreditResultInTransaction(ctx, tx, op.UserID, op.AIReceiptID, op.Status == "succeeded", op.ErrorCode)
		}
	}
	return nil
}
func (r *tailoringTransaction) FindImport(ctx context.Context, user, job, source string) (*model.WorkflowOperation, error) {
	// Search across Jobs too: selecting the same imported copy elsewhere must not
	// bypass its mandatory recognition review.
	var key string
	err := r.q.QueryRow(ctx, `SELECT request_key FROM resume_tailoring_operations WHERE user_id=$1 AND kind='import' AND object_id=$2 AND status='succeeded' LIMIT 1`, user, source).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find imported source: %w", err)
	}
	return r.GetOperation(ctx, user, "import", key)
}
func (r *tailoringTransaction) ChargeAI(ctx context.Context, user, receipt string, limit int) error {
	tx, ok := r.q.(pgx.Tx)
	if !ok {
		return model.ErrProposalStoreUnavailable
	}
	// Only a locked, server-owned parsed receipt may restore a saved result
	// on its original credit period. This path never authorizes provider dispatch.
	var parsed bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM resume_tailoring_operations WHERE user_id=$1 AND ai_receipt_id=$2 AND kind='import' AND status='parsed')`, user, receipt).Scan(&parsed); err != nil {
		return err
	}
	if parsed {
		if _, ok := ai.CreditOperationFromContext(ctx, user, receipt); ok {
			ai.MarkCreditAdmission(ctx)
			return subRepository.RestoreSavedResultCharge(ctx, tx, user, receipt)
		}
	}
	return subRepository.ReserveAIInTransaction(ctx, tx, user, limit, receipt)
}

// PostgresTailoringProposalStore keeps full authoritative proposals in the same
// durable operation table. Missing migrations fail closed; Redis is not used.
type PostgresTailoringProposalStore struct{ repo *ResumeBuilderRepository }

func NewPostgresTailoringProposalStore(repo *ResumeBuilderRepository) *PostgresTailoringProposalStore {
	return &PostgresTailoringProposalStore{repo: repo}
}
func (s *PostgresTailoringProposalStore) Save(ctx context.Context, p *model.TailoringProposal) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tx, err := s.repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE resume_tailoring_operations SET payload=$4,status='succeeded',updated_at=now() WHERE user_id=$1 AND kind='generate' AND request_key=$2 AND job_id=$3 AND status='pending'`, p.UserID, p.ProposalID, p.Job.ID, data)
	if err != nil {
		return fmt.Errorf("save tailoring proposal: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return model.ErrProposalStoreUnavailable
	}
	if op, _, ok := ai.CurrentCreditOperation(ctx); ok {
		if err := subRepository.FinalizeCreditResultInTransaction(ctx, tx, op.UserID, op.Key, true, ""); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	ai.SetDurableCreditContext(ctx)
	return nil
}
func (s *PostgresTailoringProposalStore) Get(ctx context.Context, user, key string) (*model.TailoringProposal, error) {
	op, err := s.repo.getOperation(ctx, user, "generate", key)
	if err != nil {
		return nil, err
	}
	if op == nil || op.Status != "succeeded" {
		return nil, model.ErrProposalExpired
	}
	var proposal model.TailoringProposal
	if err := json.Unmarshal(op.Result, &proposal); err != nil {
		return nil, model.ErrProposalStoreUnavailable
	}
	if !time.Now().Before(proposal.ExpiresAt) {
		return nil, model.ErrProposalExpired
	}
	return &proposal, nil
}

func (s *PostgresTailoringProposalStore) Durable() bool { return true }
