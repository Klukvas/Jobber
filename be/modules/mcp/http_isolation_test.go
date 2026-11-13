package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/gin-gonic/gin"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var otherPAT = "jbr_pat_" + strings.Repeat("B", 43)

type sharedTokens struct {
	identities map[string]patModel.AuthenticationInfo
	calls      atomic.Int64
}

func (s *sharedTokens) AuthenticateWithInfo(_ context.Context, token string) (*patModel.AuthenticationInfo, error) {
	s.calls.Add(1)
	info, ok := s.identities[token]
	if !ok {
		return nil, patModel.ErrInvalidToken
	}
	return &info, nil
}

type allowPaid struct{}

func (allowPaid) RequirePaidPlan(context.Context, string) error { return nil }

func sharedRouter() (*gin.Engine, *fakeJobs, *sharedTokens) {
	jobs := newFakeJobs()
	jobs.jobs[jobB] = &jobModel.JobDTO{ID: jobB, Title: "Private B Job", CurrentStageTemplateID: ptr(stageC)}
	jobs.owners[jobB] = userB
	tokens := &sharedTokens{identities: map[string]patModel.AuthenticationInfo{
		validPAT: {UserID: userA, TokenID: tokenA}, otherPAT: {UserID: userB, TokenID: jobB},
	}}
	router := gin.New()
	server := NewServer(jobs, nil, jobs, nil)
	registerHTTP(router, server, tokens, allowPaid{}, testHTTPConfig(), zap.NewNop())
	return router, jobs, tokens
}

func httpTool(t *testing.T, router http.Handler, token, name string, args any) *sdk.CallToolResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
	require.NoError(t, err)
	response := rpcHTTP(router, http.MethodPost, "Bearer "+token, string(body))
	require.Equal(t, 200, response.Code, response.Body.String())
	var envelope struct {
		Result *sdk.CallToolResult `json:"result"`
		Error  any                 `json:"error"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.Nil(t, envelope.Error)
	require.NotNil(t, envelope.Result)
	return envelope.Result
}

func assertHTTPIdentity(t *testing.T, router http.Handler, token, ownID, foreignID, ownStage, foreignStage string) {
	t.Helper()
	listed := decodeOutput[ListJobsOutput](t, httpTool(t, router, token, "list_jobs", map[string]any{}))
	require.Len(t, listed.Jobs, 1)
	require.Equal(t, ownID, listed.Jobs[0].ID)
	for _, name := range []string{"get_job", "move_job", "archive_job"} {
		args := map[string]any{"job_id": foreignID}
		if name == "move_job" {
			args["stage_id"] = ownStage
		}
		if name == "archive_job" {
			args["archived"] = true
		}
		result := httpTool(t, router, token, name, args)
		require.True(t, result.IsError)
		require.Contains(t, result.Content[0].(*sdk.TextContent).Text, "Job not found")
	}
	for _, name := range []string{"create_job", "move_job"} {
		args := map[string]any{"stage_id": foreignStage}
		if name == "create_job" {
			args["title"] = "Foreign Stage"
			args["company"] = "Must not persist"
		} else {
			args["job_id"] = ownID
		}
		result := httpTool(t, router, token, name, args)
		require.True(t, result.IsError)
		require.Contains(t, result.Content[0].(*sdk.TextContent).Text, "Stage not found")
	}
}

func TestHTTPSharedServerAlternatingIdentities(t *testing.T) {
	router, jobs, tokens := sharedRouter()
	for index := 0; index < 4; index++ {
		assertHTTPIdentity(t, router, validPAT, jobA, jobB, stageA, stageC)
		assertHTTPIdentity(t, router, otherPAT, jobB, jobA, stageC, stageA)
	}
	require.Equal(t, int64(48), tokens.calls.Load())
	require.False(t, jobs.jobs[jobA].IsArchived)
	require.False(t, jobs.jobs[jobB].IsArchived)
}

func TestHTTPSharedServerConcurrentIdentities(t *testing.T) {
	router, _, _ := sharedRouter()
	for index := 0; index < 8; index++ {
		t.Run("A", func(t *testing.T) { t.Parallel(); assertHTTPIdentity(t, router, validPAT, jobA, jobB, stageA, stageC) })
		t.Run("B", func(t *testing.T) { t.Parallel(); assertHTTPIdentity(t, router, otherPAT, jobB, jobA, stageC, stageA) })
	}
}
