package mcp

import (
	"fmt"
	"strings"
	"testing"

	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestToolValidation(t *testing.T) {
	tests := []struct {
		name, tool string
		args       map[string]any
	}{
		{"job uuid", "get_job", map[string]any{"job_id": "bad"}},
		{"nil uuid", "get_job", map[string]any{"job_id": uuid.Nil.String()}},
		{"move job uuid", "move_job", map[string]any{"job_id": "bad", "stage_id": stageA}},
		{"move stage uuid", "move_job", map[string]any{"job_id": jobA, "stage_id": "bad"}},
		{"archive uuid", "archive_job", map[string]any{"job_id": "bad", "archived": true}},
		{"archive bool required", "archive_job", map[string]any{"job_id": jobA}},
		{"empty title", "create_job", map[string]any{"title": "  "}},
		{"missing title", "create_job", map[string]any{}},
		{"long title", "create_job", map[string]any{"title": strings.Repeat("x", 256)}},
		{"long company", "create_job", map[string]any{"title": "Role", "company": strings.Repeat("x", 256)}},
		{"long description", "create_job", map[string]any{"title": "Role", "description": strings.Repeat("x", 20001)}},
		{"long url", "create_job", map[string]any{"title": "Role", "url": "https://example.com/" + strings.Repeat("x", 2048)}},
		{"unsafe url", "create_job", map[string]any{"title": "Role", "url": "javascript:alert(1)"}},
		{"url credentials", "create_job", map[string]any{"title": "Role", "url": "https://user:password@example.com/"}},
		{"create stage uuid", "create_job", map[string]any{"title": "Role", "stage_id": "bad"}},
		{"stage filter uuid", "list_jobs", map[string]any{"stage": "bad"}},
		{"stage_id filter uuid", "list_jobs", map[string]any{"stage_id": "bad"}},
		{"stage_id and alias disagree", "list_jobs", map[string]any{"stage_id": stageA, "stage": stageB}},
		{"limit max", "list_jobs", map[string]any{"limit": 101}},
		{"limit zero", "list_jobs", map[string]any{"limit": 0}},
		{"limit negative", "list_jobs", map[string]any{"limit": -1}},
		{"limit type", "list_jobs", map[string]any{"limit": "25"}},
		{"offset negative", "list_jobs", map[string]any{"offset": -1}},
		{"long query", "list_jobs", map[string]any{"query": strings.Repeat("x", 256)}},
	}
	jobs := newFakeJobs()
	client := toolClient(t, jobs, userA)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := jobs.calls.Load()
			result := callTool(t, client, test.tool, test.args)
			require.True(t, result.IsError, "%v", result.Content)
			require.Equal(t, before, jobs.calls.Load(), "invalid input reached the service")
		})
	}
}

func TestListJobsStagePaginationAndArchived(t *testing.T) {
	jobs := newFakeJobs()
	jobs.jobs = make(map[string]*jobModel.JobDTO)
	for index := 0; index < 335; index++ {
		id := fmt.Sprintf("%08d-3333-4333-8333-333333333333", index)
		stage := stageA
		if index%3 == 0 {
			stage = stageB
		}
		jobs.jobs[id] = &jobModel.JobDTO{ID: id, Title: "Engineer", CurrentStageTemplateID: ptr(stage), IsArchived: index%2 == 0}
		jobs.owners[id] = userA
	}
	client := toolClient(t, jobs, userA)
	before := jobs.listCalls.Load()
	result := callTool(t, client, "list_jobs", map[string]any{"stage_id": "urn:uuid:" + stageA, "archived": true, "query": "Engineer", "limit": 3, "offset": 2})
	require.Equal(t, before+1, jobs.listCalls.Load())
	require.Equal(t, stageA, jobs.lastOptions.StageTemplateID)
	output := decodeOutput[ListJobsOutput](t, result)
	require.Equal(t, 112, output.Total)
	require.Len(t, output.Jobs, 3)
	require.Equal(t, "00000008-3333-4333-8333-333333333333", output.Jobs[0].ID)
	for _, job := range output.Jobs {
		require.Equal(t, stageA, *job.StageID)
		require.True(t, job.Archived)
	}
	require.NotContains(t, result.Content[0].(*sdk.TextContent).Text, "description")
	// SQL filters and paginates; the adapter makes one service call for any size.
	all := decodeOutput[ListJobsOutput](t, callTool(t, client, "list_jobs", map[string]any{"stage_id": stageA, "limit": 100}))
	require.Equal(t, 111, all.Total)
	require.Len(t, all.Jobs, 100)
	empty := decodeOutput[ListJobsOutput](t, callTool(t, client, "list_jobs", map[string]any{"stage": stageA, "offset": 1000}))
	require.Empty(t, empty.Jobs)
	require.Equal(t, 111, empty.Total)
}

func TestListJobsStageAliasStillFilters(t *testing.T) {
	jobs := newFakeJobs()
	client := toolClient(t, jobs, userA)
	for _, args := range []map[string]any{
		{"stage": "urn:uuid:" + stageB},
		{"stage_id": stageB, "stage": strings.ToUpper(stageB)},
	} {
		result := callTool(t, client, "list_jobs", args)
		require.False(t, result.IsError, "%v", args)
		require.Equal(t, stageB, jobs.lastOptions.StageTemplateID, "%v", args)
	}
}

func TestCreateStageAndCompany(t *testing.T) {
	jobs := newFakeJobs()
	client := toolClient(t, jobs, userA)
	first := decodeOutput[JobOutput](t, callTool(t, client, "create_job", map[string]any{"title": "One", "company": "Acme", "stage_id": "urn:uuid:" + stageB}))
	second := decodeOutput[JobOutput](t, callTool(t, client, "create_job", map[string]any{"title": "Two", "company": " acme "}))
	require.Equal(t, *first.Job.CompanyID, *second.Job.CompanyID)
	require.Equal(t, stageB, *first.Job.StageID)
	require.Equal(t, stageA, *second.Job.StageID)
	jobs.mu.Lock()
	jobs.stages[userA] = nil
	jobs.mu.Unlock()
	result := callTool(t, client, "create_job", map[string]any{"title": "Three"})
	require.True(t, result.IsError)
	require.Contains(t, result.Content[0].(*sdk.TextContent).Text, "Stage not found")
}

func TestListJobsAlwaysUsesOneServiceCall(t *testing.T) {
	for _, count := range []int{0, 1, 1001} {
		t.Run(fmt.Sprintf("jobs=%d", count), func(t *testing.T) {
			jobs := newFakeJobs()
			jobs.jobs = make(map[string]*jobModel.JobDTO)
			for index := 0; index < count; index++ {
				id := fmt.Sprintf("%08d-3333-4333-8333-333333333333", index)
				jobs.jobs[id] = &jobModel.JobDTO{ID: id, Title: "Role", CurrentStageTemplateID: ptr(stageA)}
				jobs.owners[id] = userA
			}
			client := toolClient(t, jobs, userA)
			for _, stage := range []string{stageA, stageC} {
				before := jobs.listCalls.Load()
				result := callTool(t, client, "list_jobs", map[string]any{"stage": stage, "limit": 1})
				output := decodeOutput[ListJobsOutput](t, result)
				require.Equal(t, before+1, jobs.listCalls.Load())
				expected := count
				if stage == stageC {
					expected = 0
				}
				require.Equal(t, expected, output.Total)
				require.LessOrEqual(t, len(output.Jobs), 1)
			}
		})
	}
}
