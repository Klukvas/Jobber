package mcp

import (
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
)

func compactJob(job *jobModel.JobDTO) JobItem {
	item := JobItem{
		ID: job.ID, Title: job.Title, CompanyID: job.CompanyID, Company: job.CompanyName,
		URL: job.URL, StageID: job.CurrentStageTemplateID, StageName: job.CurrentStageName,
		Archived: job.IsArchived, Favorite: job.IsFavorite,
		CreatedAt: job.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: job.UpdatedAt.Format(time.RFC3339Nano),
	}
	if job.AppliedAt != nil {
		timestamp := job.AppliedAt.Format(time.RFC3339Nano)
		item.AppliedAt = &timestamp
	}
	return item
}

func fullJob(job *jobModel.JobDTO) JobOutput {
	output := JobOutput{Job: compactJob(job), Description: job.Description, Source: job.Source,
		StageHistoryID: job.CurrentStageID, LastActivityAt: job.LastActivityAt.Format(time.RFC3339Nano),
		JobComments: mapComments(job.JobComments), StageComments: mapComments(job.StageComments)}
	if job.Resume != nil {
		output.Resume = &ResumeReference{ID: job.Resume.ID, Name: job.Resume.Name, Type: job.Resume.Type}
	}
	return output
}

func mapComments(comments []*commentModel.CommentDTO) []Comment {
	output := make([]Comment, 0, len(comments))
	for _, comment := range comments {
		output = append(output, Comment{ID: comment.ID, JobID: comment.JobID, StageID: comment.StageID, Content: comment.Content,
			CreatedAt: comment.CreatedAt.Format(time.RFC3339Nano), UpdatedAt: comment.UpdatedAt.Format(time.RFC3339Nano)})
	}
	return output
}
