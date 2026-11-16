//go:build integration

package service

import (
	"context"
	"github.com/anthropics/anthropic-sdk-go"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subSvc "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTailoringCreditResultAndReplayPostgres(t *testing.T) {
	for _, scenario := range []string{"success", "invalid response", "no changes", "save failure"} {
		t.Run(scenario, func(t *testing.T) {
			db, user := newImportCreditDB(t)
			ctx := t.Context()
			repo := repository.NewResumeBuilderRepository(db)
			builder := NewResumeBuilderService(repo, nil)
			source, err := builder.Create(ctx, user, &model.CreateResumeBuilderRequest{Title: "Source", TemplateID: DefaultTemplateID})
			require.NoError(t, err)
			text := "Experienced software engineer with 10 years of experience."
			require.NoError(t, repo.UpsertSummary(ctx, &model.Summary{ResumeBuilderID: source.ID, Content: text}))
			require.NoError(t, repo.CreateExperience(ctx, &model.Experience{ResumeBuilderID: source.ID, Position: "Go engineer", Company: "Cursopus QA", Description: "Built reliable Go services."}))
			stage, job := uuid.NewString(), uuid.NewString()
			_, err = db.Exec(ctx, `INSERT INTO stage_templates(id,user_id,name,"order",kind) VALUES($1,$2,'Wishlist',0,'saved')`, stage, user)
			require.NoError(t, err)
			_, err = db.Exec(ctx, `INSERT INTO jobs(id,user_id,title,description,current_stage_template_id,resume_builder_id) VALUES($1,$2,'Engineer','Build reliable services',$3,$4)`, job, user, stage, source.ID)
			require.NoError(t, err)
			require.NoError(t, subRepo.NewSubscriptionRepository(db).ConfigureAIBudget(ctx, 100_000_000))
			limits := subSvc.NewSubscriptionService(subRepo.NewSubscriptionRepository(db), nil, subSvc.BillingConfig{})
			calls := 0
			client := ai.NewTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				body := `{"summary":{"fact_ids":["e0f0"],"reason":"Emphasize documented Go experience"},"experiences":[],"skill_order":null}`
				if scenario == "invalid response" {
					body = `{"summary":null}`
				}
				if scenario == "no changes" {
					body = `{"summary":{"fact_ids":["s0"],"reason":"Existing summary is relevant"},"experiences":[],"skill_order":null}`
				}
				return &anthropic.Message{StopReason: "end_turn", Content: []anthropic.ContentBlockUnion{{Type: "text", Text: body}}}, nil
			})
			if scenario == "save failure" {
				_, err = db.Exec(ctx, `ALTER TABLE resume_tailoring_operations ADD CONSTRAINT fixture_result_save_failure CHECK(status<>'succeeded')`)
				require.NoError(t, err)
			}
			tailoring := NewTailoringService(repo, repo, client, repository.NewPostgresTailoringProposalStore(repo), limits)
			workflow := NewPDFTailoringService(tailoring, nil, nil, nil)
			credit := ai.CreditContext(ctx, func() string { return user }, uuid.NewString(), "tailor")
			key := uuid.NewString()
			proposal, err := workflow.GenerateIdempotent(credit, user, job, source.ID, "en", key)
			if scenario == "success" {
				require.NoError(t, err)
				require.Len(t, proposal.Changes, 1)
			} else {
				require.Error(t, err)
				if scenario == "no changes" {
					require.ErrorIs(t, err, model.ErrAINoChanges)
				}
			}
			limits.FinishAIRequest(credit)
			balance, readErr := subRepo.NewSubscriptionRepository(db).CreditBalance(ctx, user)
			require.NoError(t, readErr)
			if scenario == "success" {
				require.Equal(t, 23, balance.Remaining)
			} else {
				require.Equal(t, 30, balance.Remaining)
			}
			replay := ai.CreditContext(ctx, func() string { return user }, uuid.NewString(), "tailor")
			again, replayErr := workflow.GenerateIdempotent(replay, user, job, source.ID, "en", key)
			if scenario == "success" {
				require.NoError(t, replayErr)
				require.Equal(t, proposal.ProposalID, again.ProposalID)
				applied, _, applyErr := workflow.Apply(ctx, user, job, proposal.ProposalID, &model.ApplyTailoringRequest{Title: "Reviewed copy", AcceptedChangeIDs: []string{proposal.Changes[0].ID}})
				require.NoError(t, applyErr)
				require.NotEqual(t, source.ID, applied.ResumeBuilder.ID)
			} else {
				require.Error(t, replayErr)
				if scenario == "no changes" {
					require.ErrorIs(t, replayErr, model.ErrAINoChanges)
				}
			}
			limits.FinishAIRequest(replay)
			require.Equal(t, 1, calls, "replay must not call provider or charge again")
		})
	}
}
