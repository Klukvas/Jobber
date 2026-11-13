package mcp

import (
	"context"
	"errors"
	"strings"

	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
)

func (t *tools) createJob(ctx context.Context, userID string, input CreateJobInput) (JobOutput, error) {
	if err := validateCreate(input); err != nil {
		return JobOutput{}, err
	}
	stage, err := t.createStage(ctx, userID, input.StageID)
	if err != nil {
		return JobOutput{}, err
	}
	if t.limits == nil {
		return JobOutput{}, errors.New("job limit checker unavailable")
	}
	if err := t.limits.CheckLimit(ctx, userID, "jobs"); err != nil {
		return JobOutput{}, err
	}
	companyID, err := t.companyID(ctx, userID, strings.TrimSpace(input.Company))
	if err != nil {
		return JobOutput{}, err
	}
	// JobService.Create rechecks the limit and Stage ownership at the write.
	// CompanyService and JobService do not share a transaction; a concurrent
	// limit/Stage change can still reject the Job after Company creation.
	job, err := t.jobs.Create(ctx, userID, &jobModel.CreateJobRequest{
		Title: strings.TrimSpace(input.Title), CompanyID: companyID,
		URL: optionalText(input.URL), Description: optionalText(input.Description), StageTemplateID: &stage,
	})
	if err != nil {
		return JobOutput{}, err
	}
	return fullJob(job), nil
}

func (t *tools) createStage(ctx context.Context, userID, stage string) (string, error) {
	canonical := ""
	if stage != "" {
		id, err := canonicalID(stage, "stage_id")
		if err != nil {
			return "", err
		}
		canonical = id
	}
	stages, err := t.listStages(ctx, userID, ListStagesInput{})
	if err != nil {
		return "", err
	}
	for _, candidate := range stages.Stages {
		if canonical == "" || candidate.ID == canonical {
			return candidate.ID, nil
		}
	}
	return "", jobModel.ErrStageTemplateNotFound
}

func (t *tools) moveJob(ctx context.Context, userID string, input MoveJobInput) (JobOutput, error) {
	jobID, err := canonicalID(input.JobID, "job_id")
	if err != nil {
		return JobOutput{}, err
	}
	stageID, err := canonicalID(input.StageID, "stage_id")
	if err != nil {
		return JobOutput{}, err
	}
	job, err := t.jobs.Move(ctx, userID, jobID, &jobModel.MoveJobRequest{StageTemplateID: stageID})
	if err != nil {
		return JobOutput{}, err
	}
	return fullJob(job), nil
}

func (t *tools) archiveJob(ctx context.Context, userID string, input ArchiveJobInput) (JobOutput, error) {
	id, err := canonicalID(input.JobID, "job_id")
	if err != nil {
		return JobOutput{}, err
	}
	job, err := t.jobs.Update(ctx, userID, id, &jobModel.UpdateJobRequest{IsArchived: &input.Archived})
	if err != nil {
		return JobOutput{}, err
	}
	return fullJob(job), nil
}

func (t *tools) companyID(ctx context.Context, userID, name string) (*string, error) {
	if name == "" {
		return nil, nil
	}
	for offset := 0; ; offset += servicePageSize {
		companies, total, err := t.companies.List(ctx, userID, &companyPorts.ListOptions{Limit: servicePageSize, Offset: offset, SortBy: "name", SortDir: "asc"})
		if err != nil {
			return nil, err
		}
		for _, company := range companies {
			if strings.EqualFold(strings.TrimSpace(company.Name), name) {
				return &company.ID, nil
			}
		}
		if len(companies) == 0 || offset+len(companies) >= total {
			break
		}
	}
	company, err := t.companies.Create(ctx, userID, &companyModel.CreateCompanyRequest{Name: name})
	if err != nil {
		return nil, err
	}
	return &company.ID, nil
}
