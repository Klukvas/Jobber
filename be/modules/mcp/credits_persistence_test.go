package mcp

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	cvModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	cvPorts "github.com/andreypavlenko/jobber/modules/coverletters/ports"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

// Only persistence is synthetic: services, receipt replacement, provider stages,
// credit admission and request completion use the production implementations.
type creditWorkflow struct {
	rbPorts.WorkflowTransaction
	mu                 sync.Mutex
	ledger             *guardedLedger
	operations         map[string]*rbModel.WorkflowOperation
	originalReceipt    string
	replacementReceipt string
	source             *rbModel.FullResumeDTO
	proposal           *rbModel.TailoringProposal
}

func newCreditWorkflow(ledger *guardedLedger) *creditWorkflow {
	return &creditWorkflow{ledger: ledger, operations: map[string]*rbModel.WorkflowOperation{}, source: &rbModel.FullResumeDTO{
		ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{ID: builderA, Title: "Source"},
		Experiences:      []*rbModel.ExperienceDTO{{ID: tailoredA, Company: "Acme", Position: "Engineer", Description: "Built reliable Go APIs."}},
		SectionOrder:     []*rbModel.SectionOrderDTO{{SectionKey: "experience", IsVisible: true}},
	}}
}
func (r *creditWorkflow) VerifyOwnership(_ context.Context, owner, id string) error {
	if owner != userA || id != builderA {
		return rbModel.ErrResumeBuilderNotFound
	}
	return nil
}
func (r *creditWorkflow) GetFullResume(ctx context.Context, owner, id string) (*rbModel.FullResumeDTO, error) {
	if err := r.VerifyOwnership(ctx, owner, id); err != nil {
		return nil, err
	}
	return r.source, nil
}
func (r *creditWorkflow) GetTailoringJob(_ context.Context, owner, id string) (*rbModel.TailoringJob, error) {
	if owner != userA || id != jobA {
		return nil, rbModel.ErrTailoringNotFound
	}
	return &rbModel.TailoringJob{ID: jobA, Title: "Engineer", Description: "Go APIs"}, nil
}
func (r *creditWorkflow) LockTailoringSource(ctx context.Context, owner, id string) (*rbModel.FullResumeDTO, error) {
	return r.GetFullResume(ctx, owner, id)
}
func (r *creditWorkflow) RunTailoringTransaction(ctx context.Context, owner string, fn func(rbPorts.TailoringTransaction) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(r)
}
func (r *creditWorkflow) GetOperation(_ context.Context, owner, kind, key string) (*rbModel.WorkflowOperation, error) {
	op := r.operations[owner+":"+kind+":"+key]
	if op == nil {
		return nil, nil
	}
	copy := *op
	return &copy, nil
}
func (r *creditWorkflow) PutOperation(_ context.Context, op *rbModel.WorkflowOperation) error {
	copy := *op
	copy.Result = append([]byte(nil), op.Result...)
	r.operations[op.UserID+":"+op.Kind+":"+op.Key] = &copy
	return nil
}
func (r *creditWorkflow) FindImport(context.Context, string, string, string) (*rbModel.WorkflowOperation, error) {
	return nil, nil
}
func (r *creditWorkflow) ChargeAI(ctx context.Context, owner, receipt string, _ int) error {
	original, _, _ := ai.CurrentCreditOperation(ctx)
	r.originalReceipt = original.Key
	op, ok := ai.CreditOperationFromContext(ctx, owner, receipt)
	if !ok {
		return subModel.ErrCreditUnavailable
	}
	r.replacementReceipt = op.Key
	return r.ledger.reserveOperation(op)
}
func (r *creditWorkflow) Save(_ context.Context, proposal *rbModel.TailoringProposal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	encoded, err := json.Marshal(proposal)
	if err != nil {
		return err
	}
	op := r.operations[proposal.UserID+":generate:"+proposal.ProposalID]
	if op == nil {
		return rbModel.ErrProposalStoreUnavailable
	}
	copy := *op
	copy.Status = "succeeded"
	copy.Result = encoded
	r.operations[proposal.UserID+":generate:"+proposal.ProposalID] = &copy
	r.proposal = proposal
	return nil
}
func (r *creditWorkflow) Get(_ context.Context, owner, id string) (*rbModel.TailoringProposal, error) {
	if r.proposal == nil || r.proposal.UserID != owner || r.proposal.ProposalID != id {
		return nil, rbModel.ErrProposalExpired
	}
	return r.proposal, nil
}

type creditLetterRepository struct {
	cvPorts.CoverLetterRepository
	mu      sync.Mutex
	letters map[string]*cvModel.CoverLetter
}

func (r *creditLetterRepository) Create(_ context.Context, letter *cvModel.CoverLetter) (*cvModel.CoverLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *letter
	copy.ID = proposalA
	r.letters[copy.ID] = &copy
	return &copy, nil
}
func (r *creditLetterRepository) VerifyOwnership(ctx context.Context, owner, id string) error {
	_, err := r.GetByID(ctx, owner, id)
	return err
}
func (r *creditLetterRepository) GetByID(_ context.Context, owner, id string) (*cvModel.CoverLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	letter := r.letters[id]
	if letter == nil || letter.UserID != owner {
		return nil, cvModel.ErrCoverLetterNotFound
	}
	copy := *letter
	copy.Paragraphs = append([]string(nil), letter.Paragraphs...)
	return &copy, nil
}
func (r *creditLetterRepository) Update(_ context.Context, letter *cvModel.CoverLetter) (*cvModel.CoverLetter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *letter
	copy.Paragraphs = append([]string(nil), letter.Paragraphs...)
	r.letters[copy.ID] = &copy
	return &copy, nil
}
