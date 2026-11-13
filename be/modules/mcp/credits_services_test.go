package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	cvHandler "github.com/andreypavlenko/jobber/modules/coverletters/handler"
	cvModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	cvService "github.com/andreypavlenko/jobber/modules/coverletters/service"
	rbHandler "github.com/andreypavlenko/jobber/modules/resumebuilder/handler"
	rbService "github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subHandler "github.com/andreypavlenko/jobber/modules/subscriptions/handler"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stagedCreditCase struct {
	name                   string
	rejectStage, failStage int
	insufficient           bool
	calls, attempts        int
	refunded               bool
}

func stagedCreditCases() []stagedCreditCase {
	return []stagedCreditCase{
		{name: "two stages charge once", calls: 2, attempts: 2},
		{name: "insufficient credits", insufficient: true, refunded: true},
		{name: "first dispatch rejected refunds", rejectStage: 1, refunded: true},
		{name: "first provider fails keeps charge", failStage: 1, calls: 1, attempts: 1},
		{name: "second dispatch rejected keeps first charge", rejectStage: 2, calls: 1, attempts: 1},
		{name: "second provider fails keeps charge", failStage: 2, calls: 2, attempts: 2},
	}
}

type stagedCreditFixture struct {
	fake      *fakeAITools
	ledger    *guardedLedger
	workflow  *creditWorkflow
	letters   *creditLetterRepository
	tailoring *rbService.PDFTailoringService
	letterAI  *cvService.AIService
	deps      AIDependencies
	calls     int
}

func newStagedCreditFixture(t *testing.T, kind string, tc stagedCreditCase) *stagedCreditFixture {
	t.Helper()
	fake := newFakeAITools()
	fake.ledger.balance[userA] = ai.OperationCosts[kind]
	if tc.insufficient {
		fake.ledger.balance[userA]--
	}
	ledger := &guardedLedger{aiLedger: fake.ledger, attempts: map[string]*guardedAttempt{}, rejectStage: tc.rejectStage}
	fixture := &stagedCreditFixture{fake: fake, ledger: ledger, workflow: newCreditWorkflow(ledger), letters: &creditLetterRepository{letters: map[string]*cvModel.CoverLetter{}}}
	previous := http.DefaultTransport
	http.DefaultTransport = creditTransport(func(request *http.Request) (*http.Response, error) {
		fixture.calls++
		return stagedProviderResponse(t, request, kind, fixture.calls, tc.failStage), nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	provider := ai.NewAnthropicClient("fake-test-key")
	provider.SetCreditStore(ledger)
	tailoring := rbService.NewTailoringService(fixture.workflow, fixture.workflow, provider, fixture.workflow, fake.credits)
	fixture.tailoring = rbService.NewPDFTailoringService(tailoring, nil, nil, nil)
	fixture.letterAI = cvService.NewAIService(fixture.letters, fixture.workflow, provider, fake.credits)
	fixture.deps = fake.deps()
	fixture.deps.Builders = rbService.NewResumeBuilderService(fixture.workflow, fake.credits)
	fixture.deps.Tailoring = fixture.tailoring
	fixture.deps.Letters = cvService.NewCoverLetterService(fixture.letters, fake.credits)
	fixture.deps.LetterAI = fixture.letterAI
	return fixture
}

func stagedProviderResponse(t *testing.T, request *http.Request, kind string, stage, failStage int) *http.Response {
	t.Helper()
	require.Equal(t, "/v1/messages", request.URL.Path)
	var params struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
	}
	require.NoError(t, json.NewDecoder(request.Body).Decode(&params))
	expectedTokens := 1500
	if stage == 2 {
		expectedTokens = map[string]int{"tailor": 512, "cover_letter": 768}[kind]
	}
	require.Equal(t, "claude-sonnet-5-5", params.Model)
	require.Equal(t, expectedTokens, params.MaxTokens)
	if stage == failStage {
		return providerResponse(t, request, true)
	}
	text := stagedProviderText(kind, stage)
	payload := map[string]any{"id": "msg_staged", "type": "message", "role": "assistant", "model": params.Model, "stop_reason": "end_turn", "usage": map[string]int{"input_tokens": 10, "output_tokens": 20}, "content": []map[string]string{{"type": "text", "text": text}}}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(encoded))), Request: request}
}
func stagedProviderText(kind string, stage int) string {
	if kind == "tailor" {
		if stage == 1 {
			return `{"summary":null,"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":"Developed reliable Go APIs."}],"skill_order":null}`
		}
		return `{"supported":true,"unsupported_claims":[]}`
	}
	// No resume_id is valid: the real web AI service has no candidate facts and
	// generates only future-facing interest, still using its independent verifier.
	if stage == 1 {
		return `{"paragraphs":[[{"kind":"interest","text":"I would like to apply for this role."}],[{"kind":"interest","text":"I would welcome the opportunity to discuss your team."}],[{"kind":"interest","text":"Thank you for considering this application."}]]}`
	}
	return `{"sentences":[{"id":0,"kind":"interest","supported":true},{"id":1,"kind":"interest","supported":true},{"id":2,"kind":"interest","supported":true}],"issues":[]}`
}

func TestCreditScopeRealMultiStageServicesWebParity(t *testing.T) {
	for _, kind := range []string{"tailor", "cover_letter"} {
		t.Run(kind, func(t *testing.T) {
			for _, tc := range stagedCreditCases() {
				t.Run(tc.name, func(t *testing.T) {
					for _, surface := range []string{"mcp", "web"} {
						t.Run(surface, func(t *testing.T) {
							fixture := newStagedCreditFixture(t, kind, tc)
							if surface == "mcp" {
								fixture.callMCP(t, kind, tc)
							} else {
								fixture.callWeb(t, kind, tc)
							}
							fixture.assertAccounting(t, kind, tc)
						})
					}
				})
			}
		})
	}
}
func (f *stagedCreditFixture) callMCP(t *testing.T, kind string, tc stagedCreditCase) {
	t.Helper()
	jobs := newFakeJobs()
	client := toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(f.deps)), userA)
	name := "tailor_resume"
	var input any = TailorResumeInput{JobID: jobA, ResumeID: builderA}
	if kind == "cover_letter" {
		name = "generate_cover_letter"
		input = GenerateCoverLetterInput{JobID: jobA}
	}
	result := callTool(t, client, name, input)
	require.Equal(t, tc.name != "two stages charge once", result.IsError, "%v", result.Content)
	if result.IsError {
		require.NotContains(t, result.Content[0].(*sdk.TextContent).Text, "private provider detail")
		return
	}
	if kind == "tailor" {
		changes := decodeOutput[TailorResumeOutput](t, result).Proposal.Changes
		require.Len(t, changes, 1)
		require.Equal(t, "Built reliable Go APIs.", changes[0].Before)
		require.Equal(t, "Developed reliable Go APIs.", changes[0].After)
		return
	}
	letter, err := f.letters.GetByID(t.Context(), userA, decodeOutput[GenerateCoverLetterOutput](t, result).CoverLetterID)
	require.NoError(t, err)
	require.Len(t, letter.Paragraphs, 3)
}
func (f *stagedCreditFixture) callWeb(t *testing.T, kind string, tc stagedCreditCase) {
	t.Helper()
	router := gin.New()
	router.Use(subHandler.AIRequestScope(f.fake.credits))
	path := "/api/v1/jobs/" + jobA + "/resume-tailoring"
	body := `{"resume_builder_id":"` + builderA + `"}`
	if kind == "tailor" {
		handler := rbHandler.NewTailoringHandler(f.tailoring)
		router.POST("/api/v1/jobs/:id/resume-tailoring", func(c *gin.Context) { c.Set("user_id", userA); handler.Generate(c) })
	} else {
		_, err := f.deps.Letters.Create(t.Context(), userA, &cvModel.CreateCoverLetterRequest{JobID: ptr(jobA)})
		require.NoError(t, err)
		handler := cvHandler.NewAIHandler(f.letterAI, zap.NewNop())
		path = "/api/v1/cover-letters/ai/generate"
		body = `{"cover_letter_id":"` + proposalA + `","job_description":"Go APIs"}`
		router.POST(path, func(c *gin.Context) { c.Set("user_id", userA); handler.Generate(c) })
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", path, strings.NewReader(body)))
	if tc.name == "two stages charge once" {
		require.Equal(t, 200, response.Code, response.Body.String())
	} else {
		require.GreaterOrEqual(t, response.Code, 400)
	}
}
func (f *stagedCreditFixture) assertAccounting(t *testing.T, kind string, tc stagedCreditCase) {
	t.Helper()
	cost := ai.OperationCosts[kind]
	require.Equal(t, tc.calls, f.calls)
	require.Len(t, f.ledger.attempts, tc.attempts)
	balance := 0
	if tc.refunded {
		balance = cost
	}
	if tc.insufficient {
		balance--
	}
	require.Equal(t, balance, f.ledger.balance[userA])
	require.Equal(t, 100, f.ledger.balance[userB])
	require.Equal(t, 1, f.ledger.finished)
	require.Len(t, f.ledger.finishedOperations, 1)
	finished := f.ledger.finishedOperations[0]
	require.Equal(t, userA, finished.UserID)
	require.Equal(t, kind, finished.Kind)
	require.Equal(t, cost, finished.Credits)
	if tc.insufficient {
		require.Empty(t, f.ledger.operations)
	} else {
		require.Equal(t, []subModel.CreditOperation{finished}, f.ledger.operations)
	}
	for _, attempt := range f.ledger.attempts {
		require.Equal(t, finished, attempt.operation)
		require.True(t, attempt.dispatched)
	}
	if kind == "tailor" {
		require.NotEmpty(t, f.workflow.originalReceipt)
		require.NotEqual(t, f.workflow.originalReceipt, f.workflow.replacementReceipt)
		require.Equal(t, f.workflow.replacementReceipt, finished.Key)
		_, leaked := f.ledger.reserved[f.workflow.originalReceipt]
		require.False(t, leaked)
	}
	if !tc.insufficient && tc.refunded {
		require.Empty(t, f.ledger.reserved)
	}
	if tc.calls == 2 && tc.failStage == 0 {
		for _, attempt := range f.ledger.attempts {
			require.True(t, attempt.settled)
		}
	}
}
