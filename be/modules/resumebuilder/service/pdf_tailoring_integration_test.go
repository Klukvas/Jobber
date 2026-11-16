//go:build integration

package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type workflowUploadReader struct{ user, id string }

func (r workflowUploadReader) GetByID(_ context.Context, user, id string) (*resumeModel.Resume, error) {
	if user != r.user || id != r.id {
		return nil, errors.New("not owned")
	}
	return &resumeModel.Resume{ID: id, UserID: user, Title: "Original PDF"}, nil
}

// Each call gets the credit scope supplied by HTTP/MCP in production. In
// particular, concurrent calls and replays must never share mutable scope state.
type creditPDFWorkflow struct {
	*PDFTailoringService
	credits *subService.SubscriptionService
}

func (s *creditPDFWorkflow) Import(ctx context.Context, user, job string, req *model.PDFImportRequest) (*model.PDFImportResult, error) {
	ctx = ai.CreditContext(ctx, func() string { return user }, "", "resume_import")
	defer s.credits.FinishAIRequest(ctx)
	return s.PDFTailoringService.Import(ctx, user, job, req)
}

func (s *creditPDFWorkflow) GenerateIdempotent(ctx context.Context, user, job, source, locale, key string) (*model.TailoringProposalDTO, error) {
	ctx = ai.CreditContext(ctx, func() string { return user }, "", "tailor")
	defer s.credits.FinishAIRequest(ctx)
	return s.PDFTailoringService.GenerateIdempotent(ctx, user, job, source, locale, key)
}

func pdfWorkflow(t *testing.T, f *tailoringIntegrationFixture, parse func(context.Context, string) (*ai.ParsedResume, error)) (*creditPDFWorkflow, *tailoringTestAI, string) {
	t.Helper()
	uploaded := uuid.NewString()
	importer := NewImportServiceWithDeps(f.repo, &MockResumeTextParser{ParseResumeTextFunc: parse}, f.limits, func([]byte) (string, error) { return "Synthetic PDF resume text", nil })
	client := &tailoringTestAI{result: &ai.ResumeTailoringSuggestions{Summary: &ai.TailoringTextSuggestion{After: "Engineer building reliable Go services.", Reason: "Emphasize engineering", Evidence: []ai.TailoringEvidence{{Section: "summary", Quote: "Built reliable Go services."}}}}}
	tailoring := NewTailoringService(f.repo, f.repo, client, repository.NewPostgresTailoringProposalStore(f.repo), f.limits)
	credits := subService.NewSubscriptionService(subRepo.NewSubscriptionRepository(f.pool), nil, subService.BillingConfig{})
	workflow := NewPDFTailoringService(tailoring, importer, workflowUploadReader{f.userID, uploaded}, func(context.Context, *resumeModel.Resume) ([]byte, error) { return []byte("%PDF-synthetic"), nil })
	return &creditPDFWorkflow{PDFTailoringService: workflow, credits: credits}, client, uploaded
}
func parsedPDF(context.Context, string) (*ai.ParsedResume, error) {
	return &ai.ParsedResume{FullName: "Recognized Example", Summary: "Built reliable Go services.", Skills: []ai.ParsedSkill{{Name: "Go", Level: "expert"}}}, nil
}
func workflowUsage(t *testing.T, f *tailoringIntegrationFixture) int {
	t.Helper()
	var count int
	// Failed operations keep their receipt even after user credits are refunded.
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM ai_credit_operations WHERE user_id=$1`, f.userID).Scan(&count))
	return count
}

func workflowCharges(t *testing.T, f *tailoringIntegrationFixture, charged, reserved, released int) {
	t.Helper()
	var actualCharged, actualReserved, actualReleased, usedCredits, legacyUsage int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT
		count(*) FILTER (WHERE charge_state='charged'),
		count(*) FILTER (WHERE charge_state='reserved'),
		count(*) FILTER (WHERE charge_state='released'),
		COALESCE(sum(credits) FILTER (WHERE charge_state<>'released'),0)
		FROM ai_credit_operations WHERE user_id=$1 AND kind IN ('resume_import','tailor')`, f.userID).Scan(&actualCharged, &actualReserved, &actualReleased, &usedCredits))
	assert.Equal(t, charged, actualCharged, "usable results are charged exactly once")
	assert.Equal(t, reserved, actualReserved, "only in-flight work holds credits")
	assert.Equal(t, released, actualReleased, "rejected results refund their reservation")
	assert.Equal(t, (charged+reserved)*7, usedCredits)
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.userID).Scan(&legacyUsage))
	assert.Zero(t, legacyUsage, "credit-scoped PDF workflows must not use legacy usage")
}

func TestPDFWorkflowRecognitionCopyAndDurableReplay(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 1)
	s, client, uploaded := pdfWorkflow(t, f, parsedPDF)
	key := uuid.NewString()
	request := &model.PDFImportRequest{RequestKey: key, ResumeID: uploaded}
	imported, err := s.Import(t.Context(), f.userID, f.jobID, request)
	require.NoError(t, err)
	require.NotEqual(t, key, imported.ResumeBuilder.ID)
	again, err := s.Import(t.Context(), f.userID, f.jobID, request)
	require.NoError(t, err)
	assert.Equal(t, imported, again)
	assert.Equal(t, 1, workflowUsage(t, f))
	changed := *request
	changed.Text = "Different input"
	_, err = s.Import(t.Context(), f.userID, f.jobID, &changed)
	require.ErrorIs(t, err, model.ErrOperationConflict)
	source := imported.ResumeBuilder.ID
	_, required, err := s.SourceReview(t.Context(), f.userID, f.jobID, source)
	require.NoError(t, err)
	assert.True(t, required)
	_, err = s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", uuid.NewString())
	require.ErrorIs(t, err, model.ErrRecognitionRequired)
	assert.Equal(t, 1, workflowUsage(t, f))
	require.NoError(t, s.ConfirmRecognition(t.Context(), f.userID, f.jobID, imported.ImportID, source))
	// Every persisted correction invalidates the prior confirmation.
	require.NoError(t, f.repo.UpsertSummary(t.Context(), &model.Summary{ResumeBuilderID: source, Content: "Corrected: built reliable Go services."}))
	_, err = s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", uuid.NewString())
	require.ErrorIs(t, err, model.ErrRecognitionRequired)
	require.NoError(t, s.ConfirmRecognition(t.Context(), f.userID, f.jobID, imported.ImportID, source))
	client.result.Summary.Evidence[0].Quote = "Corrected: built reliable Go services."
	generationKey := uuid.NewString()
	proposal, err := s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", generationKey)
	require.NoError(t, err)
	replay, err := s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", generationKey)
	require.NoError(t, err)
	assert.Equal(t, proposal, replay)
	assert.Equal(t, 1, client.calls)
	assert.Equal(t, 2, workflowUsage(t, f))
	before, err := f.repo.GetFullResume(t.Context(), f.userID, source)
	require.NoError(t, err)
	apply := &model.ApplyTailoringRequest{Title: "Adapted copy", AcceptedChangeIDs: []string{proposal.Changes[0].ID}}
	result, replayed, err := s.Apply(t.Context(), f.userID, f.jobID, proposal.ProposalID, apply)
	require.NoError(t, err)
	assert.False(t, replayed)
	assert.NotEqual(t, source, result.ResumeBuilder.ID)
	// Expiry, a changed Job link and full slots must not prevent response recovery.
	_, err = f.pool.Exec(t.Context(), `UPDATE resume_tailoring_operations SET payload=jsonb_set(payload,'{expires_at}',to_jsonb('2000-01-01T00:00:00Z'::text)) WHERE user_id=$1 AND kind='generate'`, f.userID)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `UPDATE jobs SET resume_builder_id=NULL WHERE id=$1`, f.jobID)
	require.NoError(t, err)
	f.limits.maximum = 0
	recovered, replayed, err := s.Apply(t.Context(), f.userID, f.jobID, proposal.ProposalID, apply)
	require.NoError(t, err)
	assert.True(t, replayed)
	assert.Equal(t, result.ResumeBuilder.ID, recovered.ResumeBuilder.ID)
	job, err := f.repo.GetTailoringJob(t.Context(), f.userID, f.jobID)
	require.NoError(t, err)
	assert.Nil(t, job.ResumeBuilderID)
	after, err := f.repo.GetFullResume(t.Context(), f.userID, source)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	require.NoError(t, f.repo.Delete(t.Context(), f.userID, result.ResumeBuilder.ID))
	_, _, err = s.Apply(t.Context(), f.userID, f.jobID, proposal.ProposalID, apply)
	require.ErrorIs(t, err, model.ErrOperationDeleted)
	require.NoError(t, f.repo.Delete(t.Context(), f.userID, source))
	_, err = s.Import(t.Context(), f.userID, f.jobID, request)
	require.ErrorIs(t, err, model.ErrOperationDeleted)
	assert.Equal(t, 2, workflowUsage(t, f))
	workflowCharges(t, f, 2, 0, 0)
}

func TestPDFWorkflowPendingUnreadableOwnershipAndLimits(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 4)
	var calls atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	s, _, uploaded := pdfWorkflow(t, f, func(ctx context.Context, text string) (*ai.ParsedResume, error) {
		calls.Add(1)
		close(entered)
		<-release
		return parsedPDF(ctx, text)
	})
	req := &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded}
	done := make(chan error, 1)
	go func() { _, err := s.Import(context.Background(), f.userID, f.jobID, req); done <- err }()
	<-entered
	_, err := s.Import(t.Context(), f.userID, f.jobID, req)
	require.ErrorIs(t, err, model.ErrOperationPending)
	close(release)
	require.NoError(t, <-done)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, 1, workflowUsage(t, f))
	foreign := *req
	foreign.RequestKey = uuid.NewString()
	foreign.ResumeID = uuid.NewString()
	_, err = s.Import(t.Context(), f.userID, f.jobID, &foreign)
	require.ErrorIs(t, err, model.ErrTailoringNotFound)
	foreign.ResumeID = uploaded
	_, err = s.Import(t.Context(), f.userID, uuid.NewString(), &foreign)
	require.ErrorIs(t, err, model.ErrTailoringNotFound)
	s.download = func(context.Context, *resumeModel.Resume) ([]byte, error) { return []byte("not a PDF"), nil }
	_, err = s.Import(t.Context(), f.userID, f.jobID, &foreign)
	require.ErrorIs(t, err, model.ErrPDFUnreadable)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, 1, workflowUsage(t, f))
	f.limits.maximum = 0
	foreign.Text = "Manual text"
	_, err = s.Import(t.Context(), f.userID, f.jobID, &foreign)
	require.ErrorIs(t, err, subModel.ErrLimitReached)
	assert.Equal(t, int32(1), calls.Load())
	workflowCharges(t, f, 1, 0, 0)
}

func TestPDFWorkflowFailureAndInterruptedRequestsDoNotRerun(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 3)
	var calls atomic.Int32
	s, _, uploaded := pdfWorkflow(t, f, func(context.Context, string) (*ai.ParsedResume, error) {
		calls.Add(1)
		return nil, context.DeadlineExceeded
	})
	req := &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded}
	_, err := s.Import(t.Context(), f.userID, f.jobID, req)
	require.ErrorIs(t, err, model.ErrAITimeout)
	_, err = s.Import(t.Context(), f.userID, f.jobID, req)
	require.ErrorIs(t, err, model.ErrAITimeout)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, 1, workflowUsage(t, f))
	_, err = f.pool.Exec(t.Context(), `UPDATE resume_tailoring_operations SET status='pending',created_at=$2 WHERE user_id=$1`, f.userID, time.Now().Add(-2*time.Minute))
	require.NoError(t, err)
	_, err = s.Import(t.Context(), f.userID, f.jobID, req)
	require.ErrorIs(t, err, model.ErrOperationInterrupted)
	assert.Equal(t, int32(1), calls.Load())
	workflowCharges(t, f, 0, 0, 1)
}

func TestPDFWorkflowRecognitionCanMoveAcrossJobs(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 3)
	s, _, uploaded := pdfWorkflow(t, f, parsedPDF)
	imported, err := s.Import(t.Context(), f.userID, f.jobID, &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded})
	require.NoError(t, err)
	source := imported.ResumeBuilder.ID
	require.NoError(t, s.ConfirmRecognition(t.Context(), f.userID, f.jobID, imported.ImportID, source))
	secondJob := uuid.NewString()
	_, err = f.pool.Exec(t.Context(), `INSERT INTO jobs(id,user_id,title,description,current_stage_template_id) SELECT $1,user_id,'Second Job',description,current_stage_template_id FROM jobs WHERE id=$2`, secondJob, f.jobID)
	require.NoError(t, err)
	require.NoError(t, s.ConfirmRecognition(t.Context(), f.userID, secondJob, imported.ImportID, source))
	_, required, err := s.SourceReview(t.Context(), f.userID, secondJob, source)
	require.NoError(t, err)
	assert.False(t, required)
	_, err = s.GenerateIdempotent(t.Context(), f.userID, secondJob, source, "en", uuid.NewString())
	require.NoError(t, err)
}

func TestWorkflowObjectIDsAreOwnerAndActionScoped(t *testing.T) {
	key := uuid.NewString()
	assert.NotEqual(t, workflowObjectID("alice", "import", key), workflowObjectID("bob", "import", key))
	assert.NotEqual(t, workflowObjectID("alice", "import", key), workflowObjectID("alice", "generate", key))
	// Payload round trips must retain authoritative proposal snapshots.
	payload, err := json.Marshal(&model.TailoringProposal{UserID: "alice"})
	require.NoError(t, err)
	assert.Contains(t, string(payload), "alice")
}

func reserveWorkflowCredit(t *testing.T, f *tailoringIntegrationFixture, kind string) subModel.CreditOperation {
	t.Helper()
	op := subModel.CreditOperation{UserID: f.userID, Key: uuid.NewString(), Kind: kind, Fingerprint: "other-feature", Credits: ai.OperationCosts[kind]}
	tx, err := f.pool.Begin(t.Context())
	require.NoError(t, err)
	defer tx.Rollback(context.Background())
	require.NoError(t, subRepo.ReserveCreditOperationInTransaction(t.Context(), tx, op))
	require.NoError(t, tx.Commit(t.Context()))
	return op
}

func TestPDFWorkflowSharedAIQuotaAndCancellationRetention(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 4)
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	s, _, uploaded := pdfWorkflow(t, f, func(ctx context.Context, text string) (*ai.ParsedResume, error) {
		calls.Add(1)
		close(entered)
		<-release
		return parsedPDF(ctx, text)
	})
	// One weighted operation fits. Pending reservations share the same grant
	// across imports and tailoring, regardless of the old request-count limit.
	ledger := subRepo.NewSubscriptionRepository(f.pool)
	for _, kind := range []string{"interview_plan", "cover_letter", "match", "interview_summary"} {
		require.NoError(t, ledger.CompleteCreditOperation(t.Context(), reserveWorkflowCredit(t, f, kind)))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan *model.PDFImportResult, 1)
	fail := make(chan error, 1)
	req := &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded}
	go func() {
		result, err := s.Import(ctx, f.userID, f.jobID, req)
		done <- result
		fail <- err
	}()
	<-entered
	workflowCharges(t, f, 0, 1, 0)
	// A different request cannot exceed the same AI allowance while slots remain.
	_, err := s.Import(t.Context(), f.userID, f.jobID, &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded})
	require.ErrorIs(t, err, subModel.ErrCreditUnavailable)
	_, err = s.GenerateIdempotent(t.Context(), f.userID, f.jobID, f.sourceID, "en", uuid.NewString())
	require.ErrorIs(t, err, subModel.ErrCreditUnavailable)
	cancel()
	close(release)
	result := <-done
	require.NoError(t, <-fail)
	require.NotNil(t, result)
	assert.Equal(t, int32(1), calls.Load())
	_, err = f.repo.GetByID(t.Context(), f.userID, result.ResumeBuilder.ID)
	require.NoError(t, err, "cancelling never deletes the imported copy")
	var receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM resume_tailoring_operations o JOIN ai_credit_operations a ON a.operation_key=o.ai_receipt_id::text AND a.user_id=o.user_id WHERE o.user_id=$1 AND a.charge_state='charged' AND a.kind='resume_import' AND a.credits=7`, f.userID).Scan(&receipts))
	assert.Equal(t, 1, receipts)
	workflowCharges(t, f, 1, 0, 0)
	replayed, err := s.Import(t.Context(), f.userID, f.jobID, req)
	require.NoError(t, err, "a completed result is recoverable even with zero remaining credits")
	assert.Equal(t, result, replayed)
	// A new import is still blocked after the pending charge is finalized.
	_, err = s.Import(t.Context(), f.userID, f.jobID, &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded})
	require.ErrorIs(t, err, subModel.ErrCreditUnavailable)
	assert.Equal(t, int32(1), calls.Load())
	workflowCharges(t, f, 1, 0, 0)
}

func TestPDFWorkflowSameClientKeysAcrossOwners(t *testing.T) {
	first := newTailoringIntegrationFixture(t, 3)
	second := newTailoringIntegrationFixture(t, 3)
	a, _, uploadA := pdfWorkflow(t, first, parsedPDF)
	b, _, uploadB := pdfWorkflow(t, second, parsedPDF)
	key := uuid.NewString()
	importedA, err := a.Import(t.Context(), first.userID, first.jobID, &model.PDFImportRequest{RequestKey: key, ResumeID: uploadA})
	require.NoError(t, err)
	importedB, err := b.Import(t.Context(), second.userID, second.jobID, &model.PDFImportRequest{RequestKey: key, ResumeID: uploadB})
	require.NoError(t, err)
	assert.NotEqual(t, importedA.ResumeBuilder.ID, importedB.ResumeBuilder.ID)
	require.NoError(t, a.ConfirmRecognition(t.Context(), first.userID, first.jobID, importedA.ImportID, importedA.ResumeBuilder.ID))
	require.NoError(t, b.ConfirmRecognition(t.Context(), second.userID, second.jobID, importedB.ImportID, importedB.ResumeBuilder.ID))
	pa, err := a.GenerateIdempotent(t.Context(), first.userID, first.jobID, importedA.ResumeBuilder.ID, "en", key)
	require.NoError(t, err)
	pb, err := b.GenerateIdempotent(t.Context(), second.userID, second.jobID, importedB.ResumeBuilder.ID, "en", key)
	require.NoError(t, err)
	assert.NotEqual(t, pa.ProposalID, pb.ProposalID)
	_, _, err = a.Apply(t.Context(), first.userID, first.jobID, pb.ProposalID, &model.ApplyTailoringRequest{Title: "Foreign", AcceptedChangeIDs: []string{"summary"}})
	require.Error(t, err)
	assert.Equal(t, 2, workflowUsage(t, first))
	assert.Equal(t, 2, workflowUsage(t, second))
	workflowCharges(t, first, 2, 0, 0)
	workflowCharges(t, second, 2, 0, 0)
}

func TestPDFWorkflowEmptyAndExtractionErrorsHaveManualFallback(t *testing.T) {
	for _, name := range []string{"empty scan", "extraction error"} {
		t.Run(name, func(t *testing.T) {
			f := newTailoringIntegrationFixture(t, 3)
			s, _, uploaded := pdfWorkflow(t, f, parsedPDF)
			s.imports.extractPDF = func([]byte) (string, error) {
				if name == "extraction error" {
					return "", errors.New("broken PDF")
				}
				return "  ", nil
			}
			req := &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded}
			_, err := s.Import(t.Context(), f.userID, f.jobID, req)
			require.ErrorIs(t, err, model.ErrPDFUnreadable)
			assert.Equal(t, 0, workflowUsage(t, f))
			req.Text = "Verified manual resume text"
			imported, err := s.Import(t.Context(), f.userID, f.jobID, req)
			require.NoError(t, err)
			require.NotEmpty(t, imported.ResumeBuilder.ID)
			assert.Equal(t, 1, workflowUsage(t, f))
			workflowCharges(t, f, 1, 0, 0)
		})
	}
}

func TestPDFWorkflowGenerationFailureKeepsConfirmedCopy(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 3)
	s, client, uploaded := pdfWorkflow(t, f, parsedPDF)
	imported, err := s.Import(t.Context(), f.userID, f.jobID, &model.PDFImportRequest{RequestKey: uuid.NewString(), ResumeID: uploaded})
	require.NoError(t, err)
	source := imported.ResumeBuilder.ID
	require.NoError(t, s.ConfirmRecognition(t.Context(), f.userID, f.jobID, imported.ImportID, source))
	before, err := f.repo.GetFullResume(t.Context(), f.userID, source)
	require.NoError(t, err)
	client.err = context.DeadlineExceeded
	key := uuid.NewString()
	_, err = s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", key)
	require.ErrorIs(t, err, model.ErrAITimeout)
	_, err = s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", key)
	require.ErrorIs(t, err, model.ErrAITimeout)
	assert.Equal(t, 1, client.calls)
	assert.Equal(t, 2, workflowUsage(t, f))
	workflowCharges(t, f, 1, 0, 1)
	after, err := f.repo.GetFullResume(t.Context(), f.userID, source)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	_, required, err := s.SourceReview(t.Context(), f.userID, f.jobID, source)
	require.NoError(t, err)
	assert.False(t, required)
	// A failed result refunds its credits; a new logical request may succeed.
	client.err = nil
	_, err = s.GenerateIdempotent(t.Context(), f.userID, f.jobID, source, "en", uuid.NewString())
	require.NoError(t, err)
	assert.Equal(t, 2, client.calls)
	workflowCharges(t, f, 2, 0, 1)
}
