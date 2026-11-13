package mcp

import (
	"context"
	"sort"

	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
)

const servicePageSize = 100

func (t *tools) listStages(ctx context.Context, userID string, _ ListStagesInput) (ListStagesOutput, error) {
	stages := make([]Stage, 0)
	for offset := 0; ; offset += servicePageSize {
		page, total, err := t.jobs.ListStageTemplates(ctx, userID, servicePageSize, offset)
		if err != nil {
			return ListStagesOutput{}, err
		}
		for _, stage := range page {
			stages = append(stages, Stage{ID: stage.ID, Name: stage.Name, Position: stage.Order})
		}
		if len(page) == 0 || offset+len(page) >= total {
			break
		}
	}
	sort.SliceStable(stages, func(i, j int) bool { return stages[i].Position < stages[j].Position })
	return ListStagesOutput{Stages: stages}, nil
}

func (t *tools) getJob(ctx context.Context, userID string, input GetJobInput) (JobOutput, error) {
	id, err := canonicalID(input.JobID, "job_id")
	if err != nil {
		return JobOutput{}, err
	}
	job, err := t.jobs.GetByID(ctx, userID, id)
	if err != nil {
		return JobOutput{}, err
	}
	return fullJob(job), nil
}

func (t *tools) listJobs(ctx context.Context, userID string, input ListJobsInput) (ListJobsOutput, error) {
	opts, err := listOptions(input)
	if err != nil {
		return ListJobsOutput{}, err
	}
	jobs, total, err := t.jobs.List(ctx, userID, opts)
	if err != nil {
		return ListJobsOutput{}, err
	}
	items := make([]JobItem, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, compactJob(job))
	}
	return ListJobsOutput{Jobs: items, Total: total}, nil
}

func listOptions(input ListJobsInput) (*jobPorts.ListOptions, error) {
	limit := 25
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit < 1 || limit > 100 || input.Offset < 0 {
		return nil, validationError("limit must be 1 to 100 and offset must be nonnegative.")
	}
	if err := validateText(input.Query, "query", 255, false); err != nil {
		return nil, err
	}
	stage := ""
	if input.Stage != "" {
		id, err := canonicalID(input.Stage, "stage")
		if err != nil {
			return nil, err
		}
		stage = id
	}
	status := "active"
	if input.Archived {
		status = "archived"
	}
	return &jobPorts.ListOptions{Limit: limit, Offset: input.Offset, Search: input.Query, Status: status, StageTemplateID: stage, SortBy: "created_at", SortDir: "desc"}, nil
}
