package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"log/slog"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/google/uuid"
)

type UploadedResumeReader interface {
	GetByID(context.Context, string, string) (*resumeModel.Resume, error)
}
type ResumeFileDownloader func(context.Context, *resumeModel.Resume) ([]byte, error)

// PDFTailoringService coordinates durable import and generation operations while
// keeping recognition corrections in the existing Builder editor.
type PDFTailoringService struct {
	tailoring *TailoringService
	imports   *ImportService
	uploads   UploadedResumeReader
	download  ResumeFileDownloader
}

func NewPDFTailoringService(tailoring *TailoringService, imports *ImportService, uploads UploadedResumeReader, download ResumeFileDownloader) *PDFTailoringService {
	return &PDFTailoringService{tailoring: tailoring, imports: imports, uploads: uploads, download: download}
}

func (s *PDFTailoringService) transaction(ctx context.Context, user string, fn func(ports.WorkflowTransaction) error) error {
	return s.tailoring.repo.RunTailoringTransaction(ctx, user, func(tx ports.TailoringTransaction) error {
		workflow, ok := tx.(ports.WorkflowTransaction)
		if !ok {
			return model.ErrProposalStoreUnavailable
		}
		return fn(workflow)
	})
}

func operationState(op *model.WorkflowOperation, fingerprint string) error {
	if op.Fingerprint != fingerprint {
		return model.ErrOperationConflict
	}
	switch op.Status {
	case "succeeded", "parsed":
		return nil
	case "failed":
		return operationError(op.ErrorCode)
	case "interrupted":
		return model.ErrOperationInterrupted
	default:
		if time.Since(op.CreatedAt) > 90*time.Second {
			return model.ErrOperationInterrupted
		}
		return model.ErrOperationPending
	}
}

func operationError(code string) error {
	switch code {
	case "AI_BUDGET_REACHED":
		return subModel.ErrAIBudgetReached
	case "CREDIT_UNAVAILABLE":
		return subModel.ErrCreditUnavailable
	case "AI_OPERATION_CONFLICT":
		return subModel.ErrAIOperationConflict
	case "AI_TIMEOUT":
		return model.ErrAITimeout
	case "AI_INVALID_RESPONSE":
		return model.ErrAIInvalidResponse
	default:
		return model.ErrAIRequestFailed
	}
}
func operationCode(err error) string {
	switch {
	case errors.Is(err, subModel.ErrAIBudgetReached):
		return "AI_BUDGET_REACHED"
	case errors.Is(err, subModel.ErrCreditUnavailable):
		return "CREDIT_UNAVAILABLE"
	case errors.Is(err, subModel.ErrAIOperationConflict):
		return "AI_OPERATION_CONFLICT"
	case errors.Is(err, model.ErrAITimeout):
		return "AI_TIMEOUT"
	case errors.Is(err, model.ErrAIInvalidResponse):
		return "AI_INVALID_RESPONSE"
	default:
		return "AI_REQUEST_FAILED"
	}
}

// Replay precedes source lookup: deleting an input or result never erases a
// consumed request, and a lost response can be recovered after plan changes.
func (s *PDFTailoringService) replay(ctx context.Context, user, job, kind, key, fingerprint string) (*model.WorkflowOperation, error) {
	var found *model.WorkflowOperation
	err := s.transaction(ctx, user, func(tx ports.WorkflowTransaction) error {
		op, err := tx.GetOperation(ctx, user, kind, key)
		if err != nil || op == nil {
			return err
		}
		if op.JobID != job {
			return model.ErrTailoringNotFound
		}
		if err := operationState(op, fingerprint); err != nil {
			return err
		}
		found = op
		return nil
	})
	return found, err
}

func (s *PDFTailoringService) Import(ctx context.Context, user, job string, input *model.PDFImportRequest) (*model.PDFImportResult, error) {
	request := *input
	request.RequestKey = workflowObjectID(user, "import", input.RequestKey)
	req := &request
	fingerprint := hashTailoring(struct{ Job, Resume, Text string }{job, req.ResumeID, strings.TrimSpace(req.Text)})
	op, err := s.replay(ctx, user, job, "import", req.RequestKey, fingerprint)
	if err != nil {
		return nil, err
	}
	if op != nil {
		return s.finishImport(ctx, op)
	}
	if _, err := s.tailoring.repo.GetTailoringJob(ctx, user, job); err != nil {
		return nil, err
	}
	uploaded, err := s.uploads.GetByID(ctx, user, req.ResumeID)
	if err != nil {
		return nil, model.ErrTailoringNotFound
	}
	if s.imports == nil || s.imports.aiClient == nil {
		return nil, model.ErrAIUnavailable
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		data, err := s.download(ctx, uploaded)
		if err != nil {
			return nil, model.ErrPDFUnreadable
		}
		if len(data) > pdf.MaxPDFSize || !bytes.HasPrefix(data, []byte("%PDF-")) {
			return nil, model.ErrPDFUnreadable
		}
		text, err = s.imports.extractPDF(data)
		if err != nil || strings.TrimSpace(text) == "" {
			return nil, model.ErrPDFUnreadable
		}
	}
	if len(text) > 100000 {
		return nil, model.ErrTailoringValidation
	}
	ceiling, err := resumeBuilderCeiling(ctx, s.tailoring.limits, user)
	if err != nil {
		return nil, err
	}
	aiLimit, err := s.aiLimit(ctx, user)
	if err != nil {
		return nil, err
	}
	op = &model.WorkflowOperation{UserID: user, Kind: "import", Key: req.RequestKey, JobID: job, Fingerprint: fingerprint, Status: "pending", ObjectID: req.RequestKey, CreatedAt: time.Now().UTC()}
	op.AIReceiptID = uuid.NewString()
	fresh := false
	err = s.transaction(ctx, user, func(tx ports.WorkflowTransaction) error {
		existing, err := tx.GetOperation(ctx, user, "import", req.RequestKey)
		if err != nil {
			return err
		}
		if existing != nil {
			op = existing
			return operationState(existing, fingerprint)
		}
		if _, err := tx.GetTailoringJob(ctx, user, job); err != nil {
			return err
		}
		if err := tx.CheckResumeBuilderCapacity(ctx, user, ceiling); err != nil {
			return err
		}
		if err := tx.ChargeAI(ctx, user, op.AIReceiptID, aiLimit); err != nil {
			return err
		}
		fresh = true
		return tx.PutOperation(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	if !fresh {
		return s.finishImport(ctx, op)
	}
	// Finish bounded work after a disconnected client, so retrying can recover it.
	work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 60*time.Second)
	defer cancel()
	parsed, err := s.imports.aiClient.ParseResumeText(work, text)
	if err != nil || parsed == nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = model.ErrAITimeout
		} else {
			err = errors.Join(model.ErrAIRequestFailed, err)
		}
		s.fail(work, op, err)
		return nil, err
	}
	op.Status = "parsed"
	op.Result, _ = json.Marshal(parsed)
	if err := s.transaction(work, user, func(tx ports.WorkflowTransaction) error { return tx.PutOperation(work, op) }); err != nil {
		return nil, err
	}
	return s.finishImport(work, op)
}

func (s *PDFTailoringService) finishImport(ctx context.Context, op *model.WorkflowOperation) (*model.PDFImportResult, error) {
	if op.Status == "succeeded" {
		var result model.PDFImportResult
		if err := json.Unmarshal(op.Result, &result); err != nil {
			return nil, model.ErrProposalStoreUnavailable
		}
		if _, err := s.tailoring.resumes.GetByID(ctx, op.UserID, op.ObjectID); err != nil {
			if errors.Is(err, model.ErrResumeBuilderNotFound) {
				return nil, model.ErrOperationDeleted
			}
			return nil, err
		}
		return &result, nil
	}
	ceiling, err := resumeBuilderCeiling(ctx, s.tailoring.limits, op.UserID)
	if err != nil {
		return nil, err
	}
	var result *model.PDFImportResult
	err = s.transaction(ctx, op.UserID, func(tx ports.WorkflowTransaction) error {
		current, err := tx.GetOperation(ctx, op.UserID, "import", op.Key)
		if err != nil {
			return err
		}
		if current == nil {
			return model.ErrProposalStoreUnavailable
		}
		if current.Status == "succeeded" {
			return json.Unmarshal(current.Result, &result)
		}
		if current.Status != "parsed" {
			return operationState(current, op.Fingerprint)
		}
		if _, err := tx.GetTailoringJob(ctx, op.UserID, op.JobID); err != nil {
			return err
		}
		if err := tx.CheckResumeBuilderCapacity(ctx, op.UserID, ceiling); err != nil {
			return err
		}
		var parsed ai.ParsedResume
		if err := json.Unmarshal(current.Result, &parsed); err != nil {
			return err
		}
		title := strings.TrimSpace(parsed.FullName)
		if title == "" {
			title = "Imported Resume"
		}
		if len([]rune(title)) > 255 {
			title = string([]rune(title)[:255])
		}
		if _, err := writeParsedResume(ctx, tx, op.UserID, &parsed, title, op.ObjectID); err != nil {
			return err
		}
		full, err := tx.GetFullResume(ctx, op.UserID, op.ObjectID)
		if err != nil {
			return err
		}
		result = &model.PDFImportResult{ResumeBuilder: full.ResumeBuilderDTO, ImportID: op.Key, JobID: op.JobID}
		current.Status = "succeeded"
		current.Result, _ = json.Marshal(result)
		return tx.PutOperation(ctx, current)
	})
	return result, err
}

func (s *PDFTailoringService) ConfirmRecognition(ctx context.Context, user, job, importID, sourceID string) error {
	return s.transaction(ctx, user, func(tx ports.WorkflowTransaction) error {
		imported, err := tx.GetOperation(ctx, user, "import", importID)
		if err != nil {
			return err
		}
		if imported == nil || imported.Status != "succeeded" || imported.ObjectID != sourceID {
			return model.ErrTailoringNotFound
		}
		if _, err := tx.GetTailoringJob(ctx, user, job); err != nil {
			return err
		}
		source, err := tx.LockTailoringSource(ctx, user, sourceID)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(struct {
			SourceHash string `json:"source_hash"`
		}{hashTailoring(source)})
		return tx.PutOperation(ctx, &model.WorkflowOperation{UserID: user, Kind: "recognition", Key: importID, JobID: job, ObjectID: sourceID, Fingerprint: hashTailoring(source), Status: "succeeded", Result: payload, CreatedAt: time.Now().UTC()})
	})
}

func verifyRecognition(ctx context.Context, tx ports.WorkflowTransaction, user, job string, source *model.FullResumeDTO) error {
	imported, err := tx.FindImport(ctx, user, job, source.ID)
	if err != nil || imported == nil {
		return err
	}
	recognition, err := tx.GetOperation(ctx, user, "recognition", imported.Key)
	if err != nil {
		return err
	}
	if recognition == nil || recognition.JobID != job || recognition.Fingerprint != hashTailoring(source) {
		return model.ErrRecognitionRequired
	}
	return nil
}

func (s *PDFTailoringService) SourceReview(ctx context.Context, user, job, sourceID string) (string, bool, error) {
	var importID string
	var required bool
	err := s.transaction(ctx, user, func(tx ports.WorkflowTransaction) error {
		if _, err := tx.GetTailoringJob(ctx, user, job); err != nil {
			return err
		}
		source, err := tx.LockTailoringSource(ctx, user, sourceID)
		if err != nil {
			return err
		}
		imported, err := tx.FindImport(ctx, user, job, sourceID)
		if err != nil || imported == nil {
			return err
		}
		importID = imported.Key
		err = verifyRecognition(ctx, tx, user, job, source)
		if errors.Is(err, model.ErrRecognitionRequired) {
			required = true
			return nil
		}
		return err
	})
	return importID, required, err
}

func (s *PDFTailoringService) aiLimit(ctx context.Context, user string) (int, error) {
	provider, ok := s.tailoring.limits.(resourceLimitProvider)
	if !ok {
		return 0, model.ErrProposalStoreUnavailable
	}
	return provider.ResourceLimit(ctx, user, "ai")
}
func (s *PDFTailoringService) fail(ctx context.Context, op *model.WorkflowOperation, cause error) {
	op.Status = "failed"
	op.ErrorCode = operationCode(cause)
	if err := s.transaction(ctx, op.UserID, func(tx ports.WorkflowTransaction) error { return tx.PutOperation(ctx, op) }); err != nil { // A pending receipt fails closed on retry.
		slog.Error("persist tailoring workflow failure", "error", err)
	}
}
func (s *PDFTailoringService) Generate(ctx context.Context, user, job, source, locale string) (*model.TailoringProposalDTO, error) {
	return s.GenerateIdempotent(ctx, user, job, source, locale, uuid.NewString())
}
func (s *PDFTailoringService) GenerateIdempotent(ctx context.Context, user, job, source, locale, key string) (*model.TailoringProposalDTO, error) {
	key = workflowObjectID(user, "generate", key)
	fingerprint := hashTailoring(struct{ Job, Source, Locale string }{job, source, locale})
	op, err := s.replay(ctx, user, job, "generate", key, fingerprint)
	if err != nil {
		return nil, err
	}
	if op != nil {
		var proposal model.TailoringProposal
		if err := json.Unmarshal(op.Result, &proposal); err != nil {
			return nil, err
		}
		return &proposal.TailoringProposalDTO, nil
	}
	if _, err := s.tailoring.repo.GetTailoringJob(ctx, user, job); err != nil {
		return nil, err
	}
	if _, err := s.tailoring.resumes.GetFullResume(ctx, user, source); err != nil {
		return nil, model.ErrTailoringNotFound
	}

	if s.tailoring.client == nil {
		return nil, model.ErrAIUnavailable
	}
	aiLimit, err := s.aiLimit(ctx, user)
	if err != nil {
		return nil, err
	}
	op = &model.WorkflowOperation{UserID: user, Kind: "generate", Key: key, JobID: job, Fingerprint: fingerprint, Status: "pending", ObjectID: source, CreatedAt: time.Now().UTC()}
	var snapshot *model.FullResumeDTO
	var jobSnapshot *model.TailoringJob
	fresh := false
	err = s.transaction(ctx, user, func(tx ports.WorkflowTransaction) error {
		existing, err := tx.GetOperation(ctx, user, "generate", key)
		if err != nil {
			return err
		}
		if existing != nil {
			op = existing
			return operationState(existing, fingerprint)
		}
		jobSnapshot, err = tx.GetTailoringJob(ctx, user, job)
		if err != nil {
			return err
		}
		snapshot, err = tx.LockTailoringSource(ctx, user, source)
		if err != nil {
			return err
		}
		if err := verifyRecognition(ctx, tx, user, job, snapshot); err != nil {
			return err
		}
		if _, err := tailoringFacts(jobSnapshot, snapshot); err != nil {
			return err
		}
		fresh = true
		op.AIReceiptID = uuid.NewString()
		if err := tx.ChargeAI(ctx, user, op.AIReceiptID, aiLimit); err != nil {
			return err
		}
		return tx.PutOperation(ctx, op)
	})
	if err != nil {
		return nil, err
	}
	if !fresh {
		var p model.TailoringProposal
		if err := json.Unmarshal(op.Result, &p); err != nil {
			return nil, err
		}
		return &p.TailoringProposalDTO, nil
	}
	work, cancel := context.WithTimeout(context.WithoutCancel(ctx), 60*time.Second)
	defer cancel()
	proposal, err := s.tailoring.generateSnapshot(work, user, job, source, locale, key, false, jobSnapshot, snapshot)
	if err != nil {
		s.fail(work, op, err)
	}
	return proposal, err
}
func (s *PDFTailoringService) Apply(ctx context.Context, user, job, proposal string, req *model.ApplyTailoringRequest) (*model.ApplyTailoringResult, bool, error) {
	return s.tailoring.Apply(ctx, user, job, proposal, req)
}

// DownloadUploadedResume reuses the protected S3/external URL download path.
func DownloadUploadedResume(client *storage.S3Client) ResumeFileDownloader {
	return func(ctx context.Context, resume *resumeModel.Resume) ([]byte, error) {
		return storage.DownloadResumeFile(ctx, client, string(resume.StorageType), resume.StorageKey, resume.FileURL)
	}
}

// Client keys are owner scoped; global document IDs must not be attacker chosen.
func workflowObjectID(user, kind, key string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(user+":"+kind+":"+key)).String()
}
