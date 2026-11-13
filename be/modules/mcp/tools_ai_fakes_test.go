package mcp

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	cvModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subPorts "github.com/andreypavlenko/jobber/modules/subscriptions/ports"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const builderA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const proposalA = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
const tailoredA = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

// The actual subscription service checks the balance and finishes receipts.
// Only repository persistence and provider execution are replaced.
type aiLedger struct {
	subPorts.SubscriptionRepository
	mu                 sync.Mutex
	balance            map[string]int
	operations         []subModel.CreditOperation
	reserved           map[string]subModel.CreditOperation
	completed          map[string]bool
	finished           int
	finishedOperations []subModel.CreditOperation
}

func newAILedger() *aiLedger {
	return &aiLedger{balance: map[string]int{userA: 100, userB: 100}, reserved: map[string]subModel.CreditOperation{}, completed: map[string]bool{}}
}
func (l *aiLedger) GetByUserID(_ context.Context, owner string) (*subModel.Subscription, error) {
	return &subModel.Subscription{UserID: owner, Plan: "pro", Status: "active"}, nil
}
func (l *aiLedger) CountUserCoverLetters(context.Context, string) (int, error) { return 0, nil }
func (l *aiLedger) CreditBalance(_ context.Context, owner string) (subModel.CreditBalance, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return subModel.CreditBalance{Remaining: l.balance[owner]}, nil
}
func (l *aiLedger) FinishCreditOperation(_ context.Context, owner, key string, expected ...subModel.CreditOperation) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.finished++
	if len(expected) != 1 || expected[0].UserID != owner || expected[0].Key != key {
		return errors.New("unbound cleanup")
	}
	l.finishedOperations = append(l.finishedOperations, expected[0])
	if actual, ok := l.reserved[key]; ok && actual != expected[0] {
		return errors.New("cleanup fingerprint mismatch")
	}
	if op, ok := l.reserved[key]; ok && !l.completed[key] {
		if op.UserID != owner {
			return errors.New("foreign receipt")
		}
		l.balance[owner] += op.Credits
		delete(l.reserved, key)
	}
	return nil
}
func (l *aiLedger) invoke(ctx context.Context, owner string, success bool) error {
	op, ok := ai.CreditOperationFromContext(ctx, owner, "")
	if !ok {
		return subModel.ErrCreditUnavailable
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.balance[owner] < op.Credits {
		return subModel.ErrCreditUnavailable
	}
	l.operations = append(l.operations, op)
	l.reserved[op.Key] = op
	l.balance[owner] -= op.Credits
	if !success {
		return errors.New("provider unavailable before dispatch: private detail")
	}
	l.completed[op.Key] = true
	return nil
}

type fakeAITools struct {
	ledger          *aiLedger
	credits         *subService.SubscriptionService
	fail            bool
	cached          bool
	providerCalls   int
	creates         int
	applies         int
	pdfCalls        int
	uploadKeys      []string
	uploaded        []byte
	expiry          time.Duration
	lastOwner       string
	lastDescription string
	lastLocale      string
	saved           cvModel.UpdateCoverLetterRequest
}

func newFakeAITools() *fakeAITools {
	ledger := newAILedger()
	return &fakeAITools{ledger: ledger, credits: subService.NewSubscriptionService(ledger, nil, subService.BillingConfig{})}
}
func (f *fakeAITools) deps() AIDependencies {
	return AIDependencies{Credits: f.credits, Parser: f, Builders: f, Tailoring: f, Letters: f, LetterAI: letterAIFake{f}, MatchCache: f, Matcher: f, PDF: f, Storage: f}
}
func (f *fakeAITools) invoke(ctx context.Context, owner string) error {
	f.lastOwner = owner
	if err := f.credits.CheckLimit(ctx, owner, "ai"); err != nil {
		return err
	}
	f.providerCalls++
	return f.ledger.invoke(ctx, owner, !f.fail)
}
func (f *fakeAITools) ParseJobPage(ctx context.Context, owner string, req *importModel.ParseJobRequest) (*importModel.ParseJobResponse, error) {
	if err := f.invoke(ctx, owner); err != nil {
		return nil, err
	}
	return &importModel.ParseJobResponse{Title: "Untrusted role", Description: ptr(req.PageText)}, nil
}
func (f *fakeAITools) Get(_ context.Context, owner, id string) (*rbModel.FullResumeDTO, error) {
	if owner != userA || id != builderA {
		return nil, rbModel.ErrResumeBuilderNotFound
	}
	return &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{ID: builderA, Title: "Source"}}, nil
}
func (f *fakeAITools) Generate(ctx context.Context, owner, job, resume, locale string) (*rbModel.TailoringProposalDTO, error) {
	f.lastLocale = locale
	if err := f.invoke(ctx, owner); err != nil {
		return nil, err
	}
	return &rbModel.TailoringProposalDTO{ProposalID: proposalA, SourceResume: rbModel.TailoringResource{ID: resume}, Job: rbModel.TailoringResource{ID: job}, Changes: []rbModel.TailoringChange{{ID: "summary", Kind: "summary"}}}, nil
}
func (f *fakeAITools) Apply(_ context.Context, owner, job, proposal string, req *rbModel.ApplyTailoringRequest) (*rbModel.ApplyTailoringResult, bool, error) {
	if owner != userA || proposal != proposalA {
		return nil, false, rbModel.ErrTailoringNotFound
	}
	f.applies++
	return &rbModel.ApplyTailoringResult{JobID: job, ResumeBuilder: &rbModel.ResumeBuilderDTO{ID: tailoredA, Title: req.Title}}, false, nil
}
func (f *fakeAITools) Create(_ context.Context, owner string, req *cvModel.CreateCoverLetterRequest) (*cvModel.CoverLetterDTO, error) {
	f.creates++
	f.lastOwner = owner
	return &cvModel.CoverLetterDTO{ID: proposalA, JobID: req.JobID, ResumeBuilderID: req.ResumeBuilderID}, nil
}
func (f *fakeAITools) Update(_ context.Context, owner, id string, req *cvModel.UpdateCoverLetterRequest) (*cvModel.CoverLetterDTO, error) {
	f.saved = *req
	return &cvModel.CoverLetterDTO{ID: id}, nil
}

type letterAIFake struct{ f *fakeAITools }

func (a letterAIFake) Generate(ctx context.Context, owner, id, description string) (*ai.CoverLetterContent, error) {
	a.f.lastDescription = description
	if err := a.f.invoke(ctx, owner); err != nil {
		return nil, err
	}
	return &ai.CoverLetterContent{Greeting: "Hello", Paragraphs: []string{"Candidate facts"}, Closing: "Regards"}, nil
}
func (f *fakeAITools) GetCachedMatch(_ context.Context, owner string, req *matchModel.MatchScoreRequest) (*matchModel.MatchScoreResponse, error) {
	if owner != userA || req.JobID != jobA || req.ResumeID != builderA {
		return nil, rbModel.ErrTailoringNotFound
	}
	if f.cached {
		return &matchModel.MatchScoreResponse{OverallScore: 80, FromCache: true}, nil
	}
	return nil, matchModel.ErrCacheMiss
}
func (f *fakeAITools) CheckMatch(ctx context.Context, owner string, req *matchModel.MatchScoreRequest) (*matchModel.MatchScoreResponse, error) {
	if err := f.invoke(ctx, owner); err != nil {
		return nil, err
	}
	f.cached = true
	return &matchModel.MatchScoreResponse{OverallScore: 80}, nil
}
func (f *fakeAITools) HasFrontendPDF() bool { return true }
func (f *fakeAITools) GenerateResumePDFFromFrontend(_ context.Context, resume *rbModel.FullResumeDTO) ([]byte, error) {
	f.pdfCalls++
	return []byte("%PDF fake"), nil
}
func (f *fakeAITools) UploadPDF(_ context.Context, key string, body io.Reader) error {
	f.uploadKeys = append(f.uploadKeys, key)
	data, err := io.ReadAll(body)
	f.uploaded = data
	return err
}
func (f *fakeAITools) GeneratePresignedDownloadURL(_ context.Context, key, name string, expiry time.Duration) (string, error) {
	f.expiry = expiry
	return "https://storage.example/private.pdf", nil
}
func aiTestClient(t *testing.T, f *fakeAITools, owner string) *sdk.ClientSession {
	t.Helper()
	jobs := newFakeJobs()
	return toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(f.deps())), owner)
}
func requireToolError(t *testing.T, result *sdk.CallToolResult, contains string) {
	t.Helper()
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*sdk.TextContent).Text, contains)
}
