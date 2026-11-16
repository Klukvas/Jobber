package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	importService "github.com/andreypavlenko/jobber/modules/jobimport/service"
	"github.com/stretchr/testify/require"
)

func resultCreditParserFixture(t *testing.T) (*fakeAITools, *guardedLedger, *importService.ImportService) {
	t.Helper()
	f := newFakeAITools()
	ledger := &guardedLedger{aiLedger: f.ledger, attempts: map[string]*guardedAttempt{}}
	useGuardedCredits(f, ledger)
	previous := http.DefaultTransport
	http.DefaultTransport = creditTransport(func(request *http.Request) (*http.Response, error) {
		f.providerCalls++
		return providerResponse(t, request, false), nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	provider := ai.NewAnthropicClient("fake-test-key")
	provider.SetCreditStore(ledger)
	return f, ledger, importService.NewImportService(provider, f.credits)
}

func requireRetainedProviderCost(t *testing.T, ledger *guardedLedger) {
	t.Helper()
	require.Len(t, ledger.attempts, 1)
	for _, attempt := range ledger.attempts {
		require.True(t, attempt.dispatched)
		require.True(t, attempt.settled)
		require.Positive(t, attempt.actualNanodollars)
		require.Positive(t, attempt.reservedNanodollars)
	}
}

func TestCreditScopeMCPCompletionFailureDoesNotPublishParsedResult(t *testing.T) {
	f, ledger, parser := resultCreditParserFixture(t)
	ledger.completeErr = errors.New("private completion failure")
	deps := f.deps()
	deps.Parser = parser
	jobs := newFakeJobs()
	client := toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithAITools(deps)), userA)
	result := callTool(t, client, "parse_job_page", ParseJobPageInput{PageURL: "https://example.com/job", PageText: "A valid posting"})
	require.True(t, result.IsError)
	require.Nil(t, result.StructuredContent, "a usable parsed draft must not be published before charging succeeds")
	encoded, err := json.Marshal(result.Content)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "Engineer")
	require.NotContains(t, string(encoded), "Posting data")
	require.NotContains(t, string(encoded), "private completion failure")
	require.Equal(t, 1, f.providerCalls)
	require.Equal(t, 1, ledger.completeCalls)
	require.Equal(t, 100, ledger.balance[userA])
	require.Equal(t, 1, ledger.finished, "one rejection refunds the reservation once")
	require.Empty(t, ledger.reserved)
	require.Len(t, ledger.resultOutcomes, 1)
	for _, successful := range ledger.resultOutcomes {
		require.False(t, successful)
	}
	requireRetainedProviderCost(t, ledger)
}

func TestCreditScopeCancelledAdmittedResultAndSavedDurableResult(t *testing.T) {
	for _, tc := range []struct {
		name         string
		durableSaved bool
	}{
		{name: "cancelled admitted response refunds user"},
		{name: "already saved durable success preserves charge", durableSaved: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, ledger, parser := resultCreditParserFixture(t)
			handlers := &tools{ai: AIDependencies{Credits: f.credits}}
			parent, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx, finish, err := handlers.aiScope(parent, userA, "parse_job_page", "job_parse", ParseJobPageInput{PageURL: "https://example.com/job", PageText: "A valid posting"})
			require.NoError(t, err)
			draft, err := parser.ParseJobPage(ctx, userA, &importModel.ParseJobRequest{PageURL: "https://example.com/job", PageText: "A valid posting"})
			require.NoError(t, err)
			require.Equal(t, "Engineer", draft.Title)
			require.True(t, ai.CreditRequestAdmitted(ctx))
			if tc.durableSaved {
				operation, _, found := ai.CurrentCreditOperation(ctx)
				require.True(t, found)
				require.NoError(t, ledger.finishResult(operation, true))
				ai.SetDurableCreditContext(ctx)
			}
			cancel()
			err = handlers.completeAIResult(ctx)
			if tc.durableSaved {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, context.Canceled)
				require.False(t, ai.CreditResultReady(ctx))
			}
			require.Zero(t, ledger.completeCalls, "cancelled response never reaches detached credit completion")
			finish()
			f.credits.FinishAIRequest(ctx)
			require.Equal(t, 1, ledger.finished, "repeated cleanup cannot refund or charge twice")
			wantBalance := 100
			if tc.durableSaved {
				wantBalance -= ai.OperationCosts["job_parse"]
			}
			require.Equal(t, wantBalance, ledger.balance[userA])
			if !tc.durableSaved {
				require.Empty(t, ledger.reserved)
			}
			requireRetainedProviderCost(t, ledger)
		})
	}
}
