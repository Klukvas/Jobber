package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func toolClient(t *testing.T, jobs *fakeJobs, userID string) *sdk.ClientSession {
	t.Helper()
	companies := &fakeCompanies{items: make(map[string][]*companyModel.CompanyDTO)}
	server := NewServer(jobs, companies, jobs, nil)
	return toolClientForServer(t, server, userID)
}

func toolClientForServer(t *testing.T, server *sdk.Server, userID string) *sdk.ClientSession {
	t.Helper()
	// In-memory transport has no HTTP auth. Inject identity at the same SDK
	// RequestExtra boundary used by the production Streamable HTTP transport.
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if call, ok := req.(*sdk.CallToolRequest); ok && userID != "" {
				call.Extra = &sdk.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: userID}}
			}
			return next(ctx, method, req)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	ct, st := sdk.NewInMemoryTransports()
	session, err := server.Connect(ctx, st, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	client, err := sdk.NewClient(&sdk.Implementation{Name: "tests", Version: "1"}, nil).Connect(ctx, ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func callTool(t *testing.T, client *sdk.ClientSession, name string, args any) *sdk.CallToolResult {
	t.Helper()
	result, err := client.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	return result
}

func decodeOutput[T any](t *testing.T, result *sdk.CallToolResult) T {
	t.Helper()
	require.False(t, result.IsError, "%v", result.Content)
	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var output T
	require.NoError(t, json.Unmarshal(encoded, &output))
	return output
}

func TestToolsList(t *testing.T) {
	store := newFakeResumeStore()
	jobs := newFakeJobs()
	server := NewServer(jobs, nil, jobs, nil,
		WithResumeTools(ResumeTools{Builder: store, Uploaded: &fakeUploadedResumes{}, Snapshots: &fakeResumeSnapshots{store: store}}),
		WithAITools(newFakeAITools().deps()))
	client := toolClientForServer(t, server, userA)
	listed, err := client.ListTools(t.Context(), nil)
	require.NoError(t, err)
	// Each pair specifies read-only and idempotent hints, respectively.
	expected := map[string][2]bool{
		"list_stages":             {true, true},
		"list_jobs":               {true, true},
		"get_job":                 {true, true},
		"create_job":              {false, false},
		"move_job":                {false, true},
		"archive_job":             {false, true},
		"list_resumes":            {true, true},
		"get_resume":              {true, true},
		"update_resume_section":   {false, false},
		"list_resume_snapshots":   {true, true},
		"restore_resume_snapshot": {false, false},
		"parse_job_page":          {false, false},
		"tailor_resume":           {false, false},
		"apply_tailoring":         {false, true},
		"generate_cover_letter":   {false, false},
		"check_match":             {false, false},
		"get_resume_pdf_link":     {false, true},
	}
	require.Len(t, listed.Tools, 17)
	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
		hints, found := expected[tool.Name]
		require.True(t, found, "unexpected tool %q", tool.Name)
		require.NotNil(t, tool.Annotations, tool.Name)
		require.NotNil(t, tool.Annotations.DestructiveHint, tool.Name)
		require.False(t, *tool.Annotations.DestructiveHint, tool.Name)
		require.NotNil(t, tool.Annotations.OpenWorldHint, tool.Name)
		require.False(t, *tool.Annotations.OpenWorldHint, tool.Name)
		require.Equal(t, hints[0], tool.Annotations.ReadOnlyHint, tool.Name)
		require.Equal(t, hints[1], tool.Annotations.IdempotentHint, tool.Name)
		if !isAITool(tool.Name) && !strings.Contains(tool.Name, "resume") {
			assertFieldDescriptions(t, tool.InputSchema)
			assertFieldDescriptions(t, tool.OutputSchema)
		}
	}
	expectedNames := make([]string, 0, len(expected))
	for name := range expected {
		expectedNames = append(expectedNames, name)
	}
	require.ElementsMatch(t, expectedNames, names)
}

func assertFieldDescriptions(t *testing.T, schema any) {
	t.Helper()
	encoded, err := json.Marshal(schema)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(encoded, &value))
	var walk func(any)
	walk = func(node any) {
		switch obj := node.(type) {
		case map[string]any:
			if properties, ok := obj["properties"].(map[string]any); ok {
				for _, field := range properties {
					require.NotEmpty(t, field.(map[string]any)["description"])
				}
			}
			for _, child := range obj {
				walk(child)
			}
		case []any:
			for _, child := range obj {
				walk(child)
			}
		}
	}
	walk(value)
}

func TestToolsHappyPaths(t *testing.T) {
	jobs := newFakeJobs()
	client := toolClient(t, jobs, userA)
	stages := decodeOutput[ListStagesOutput](t, callTool(t, client, "list_stages", map[string]any{}))
	require.Equal(t, stageA, stages.Stages[0].ID)
	listed := decodeOutput[ListJobsOutput](t, callTool(t, client, "list_jobs", map[string]any{}))
	require.Len(t, listed.Jobs, 1)
	require.Equal(t, 1, listed.Total)
	require.Equal(t, 25, jobs.lastOptions.Limit)
	require.Equal(t, "active", jobs.lastOptions.Status)
	full := decodeOutput[JobOutput](t, callTool(t, client, "get_job", map[string]any{"job_id": "urn:uuid:" + jobA}))
	require.Contains(t, *full.Description, "private posting")
	require.Equal(t, userA, jobs.lastUser)
	moved := decodeOutput[JobOutput](t, callTool(t, client, "move_job", map[string]any{"job_id": jobA, "stage_id": "urn:uuid:" + stageB}))
	require.Equal(t, stageB, *moved.Job.StageID)
	archived := decodeOutput[JobOutput](t, callTool(t, client, "archive_job", ArchiveJobInput{JobID: jobA, Archived: true}))
	require.True(t, archived.Job.Archived)
	restored := decodeOutput[JobOutput](t, callTool(t, client, "archive_job", ArchiveJobInput{JobID: jobA, Archived: false}))
	require.False(t, restored.Job.Archived)
	created := decodeOutput[JobOutput](t, callTool(t, client, "create_job", map[string]any{"title": " New role ", "company": "Acme", "description": "posting", "url": "https://example.com/job"}))
	require.Equal(t, "New role", created.Job.Title)
	require.NotNil(t, created.Job.CompanyID)
	require.Equal(t, stageA, *jobs.lastCreate.StageTemplateID)
	require.Equal(t, "posting", *created.Description)
}

func TestToolsOwnershipAndNotFound(t *testing.T) {
	for _, userID := range []string{userB, userA} {
		t.Run(userID, func(t *testing.T) {
			jobs := newFakeJobs()
			client := toolClient(t, jobs, userID)
			target := jobA
			if userID == userA {
				target = "77777777-7777-4777-8777-777777777777"
			}
			for _, name := range []string{"get_job", "move_job", "archive_job"} {
				args := map[string]any{"job_id": target}
				if name == "move_job" {
					args["stage_id"] = stageB
				}
				if name == "archive_job" {
					args["archived"] = true
				}
				result := callTool(t, client, name, args)
				require.True(t, result.IsError)
				require.Contains(t, result.Content[0].(*sdk.TextContent).Text, "Job not found")
			}
			require.False(t, jobs.jobs[jobA].IsArchived)
			require.Equal(t, stageA, *jobs.jobs[jobA].CurrentStageTemplateID)
			if userID == userB {
				listed := decodeOutput[ListJobsOutput](t, callTool(t, client, "list_jobs", map[string]any{}))
				require.Empty(t, listed.Jobs)
				stages := decodeOutput[ListStagesOutput](t, callTool(t, client, "list_stages", map[string]any{}))
				require.Equal(t, "Private B", stages.Stages[0].Name)
			}
		})
	}
}

func TestToolErrors(t *testing.T) {
	t.Run("limit", func(t *testing.T) {
		jobs := newFakeJobs()
		jobs.createError = subModel.ErrLimitReached
		result := callTool(t, toolClient(t, jobs, userA), "create_job", map[string]any{"title": "Role"})
		require.True(t, result.IsError)
		require.Contains(t, result.Content[0].(*sdk.TextContent).Text, "Upgrade")
	})
	t.Run("missing identity", func(t *testing.T) {
		jobs := newFakeJobs()
		result := callTool(t, toolClient(t, jobs, ""), "list_jobs", map[string]any{})
		require.True(t, result.IsError)
		require.Zero(t, jobs.calls.Load())
	})
	t.Run("unexpected error logging", func(t *testing.T) {
		core, observed := observer.New(zap.ErrorLevel)
		handlers := &tools{logger: zap.New(core)}
		ctx := context.WithValue(t.Context(), requestIDKey{}, "request-123")
		err := handlers.toolError(ctx, "list_jobs", errors.New("private-token-and-posting"))
		require.Equal(t, "Something went wrong. Please try again later.", err.Error())
		require.Len(t, observed.All(), 1)
		require.Equal(t, "request-123", observed.All()[0].ContextMap()["request_id"])
		encoded, marshalErr := json.Marshal(observed.All()[0].ContextMap())
		require.NoError(t, marshalErr)
		require.NotContains(t, string(encoded), "private-token-and-posting")
	})
}
