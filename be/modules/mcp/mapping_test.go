package mcp

import (
	"encoding/json"
	"testing"
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/require"
)

func TestGetJobFullDetails(t *testing.T) {
	jobs := newFakeJobs()
	timestamp := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	job := *jobs.jobs[jobA]
	job.Source = ptr("Manual")
	job.IsFavorite = true
	job.AppliedAt = &timestamp
	job.CurrentStageName = ptr("Saved")
	job.CurrentStageID = ptr("history")
	job.CreatedAt, job.UpdatedAt, job.LastActivityAt = timestamp, timestamp, timestamp
	job.Resume = &jobModel.ResumeNestedDTO{ID: "resume", Name: "Builder Resume", Type: "builder"}
	comment := &commentModel.CommentDTO{ID: "comment", JobID: jobA, Content: "Note", CreatedAt: timestamp, UpdatedAt: timestamp}
	job.JobComments = []*commentModel.CommentDTO{comment}
	job.StageComments = []*commentModel.CommentDTO{comment}
	jobs.jobs[jobA] = &job
	result := callTool(t, toolClient(t, jobs, userA), "get_job", GetJobInput{JobID: jobA})
	output := decodeOutput[JobOutput](t, result)
	require.Equal(t, "Manual", *output.Source)
	require.Equal(t, "history", *output.StageHistoryID)
	require.Equal(t, "builder", output.Resume.Type)
	require.Equal(t, jobA, output.JobComments[0].JobID)
	require.Equal(t, "Note", output.StageComments[0].Content)
	require.True(t, output.Job.Favorite)
	require.Equal(t, timestamp.Format(time.RFC3339Nano), *output.Job.AppliedAt)
	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"job_description_untrusted"`)
	require.NotContains(t, string(encoded), `"description"`)
	require.Equal(t, "Engineer", jobs.jobs[jobA].Title)
}
