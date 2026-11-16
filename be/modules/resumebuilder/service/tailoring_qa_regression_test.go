package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/require"
)

// Replay the saved synthetic QA input with representative no-op and useful
// plans. The original raw provider response was not retained. No provider is used.
func TestTailoringSavedQASnapshot(t *testing.T) {
	for _, useful := range []bool{false, true} {
		name := "unchanged selections"
		if useful {
			name = "supported emphasis saves a separate copy"
		}
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("testdata-tailoring-qa.json")
			require.NoError(t, err)
			var fixture struct {
				Source model.FullResumeDTO `json:"source"`
				Job    model.TailoringJob  `json:"job"`
			}
			require.NoError(t, json.Unmarshal(raw, &fixture))
			svc, repo, store, _, _ := tailoringFixture(t)
			repo.source, repo.job = &fixture.Source, &fixture.Job
			sourceBefore := hashTailoring(repo.source)
			var savedExperience *model.Experience
			repo.CreateExperienceFunc = func(_ context.Context, experience *model.Experience) error {
				copy := *experience
				savedExperience = &copy
				return nil
			}
			repo.GetFullResumeFunc = func(context.Context, string) (*model.FullResumeDTO, error) { return repo.source, nil }
			calls := 0
			svc.client = ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				body := `{"summary":{"fact_ids":["s0"],"reason":"Relevant source"},"experiences":[],"skill_order":{"skill_ids":["k0"],"reason":"Relevant skill"}}`
				if useful {
					after := "Reduced median API latency from 240 ms to 150 ms by indexing slow queries. Built Go REST APIs for shipment tracking. Wrote PostgreSQL queries and integration tests. Added structured logging for incident investigation. Used Docker, GitHub Actions, and Linux. Collaborated with three frontend engineers. No management duties."
					encoded, marshalErr := json.Marshal(after)
					require.NoError(t, marshalErr)
					body = `{"summary":{"fact_ids":["e0f0"],"reason":"Emphasize documented API work"},"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":` + string(encoded) + `}],"skill_order":null}`
					if calls == 2 {
						body = `{"supported":true,"unsupported_claims":[]}`
					}
				}
				return &anthropic.Message{StopReason: "end_turn", Content: []anthropic.ContentBlockUnion{{Type: "text", Text: body}}}, nil
			})
			proposal, err := svc.Generate(t.Context(), "user", repo.job.ID, repo.source.ID, "en")
			if !useful {
				require.ErrorIs(t, err, model.ErrAINoChanges)
				var diagnostic *ai.ResponseValidationError
				require.ErrorAs(t, err, &diagnostic)
				require.Equal(t, "unchanged_plan", diagnostic.Rule)
				require.Nil(t, store.proposal)
				require.Equal(t, 1, calls)
				return
			}
			require.NoError(t, err)
			require.Len(t, proposal.Changes, 2)
			require.Equal(t, 2, calls)
			ids := []string{proposal.Changes[0].ID, proposal.Changes[1].ID}
			applied, _, err := svc.Apply(t.Context(), "user", repo.job.ID, proposal.ProposalID, &model.ApplyTailoringRequest{Title: "Cursopus QA tailored copy", AcceptedChangeIDs: ids})
			require.NoError(t, err)
			require.NotEqual(t, fixture.Source.ID, applied.ResumeBuilder.ID)
			require.Equal(t, sourceBefore, hashTailoring(repo.source))
			require.NotNil(t, savedExperience)
			require.Equal(t, applied.ResumeBuilder.ID, savedExperience.ResumeBuilderID)
			require.NotEqual(t, fixture.Source.Experiences[0].Description, savedExperience.Description)
			require.Contains(t, savedExperience.Description, "240 ms to 150 ms")
			require.Contains(t, savedExperience.Description, "No management duties.")
			require.Contains(t, repo.copies, applied.ResumeBuilder.ID)
			require.Equal(t, applied.ResumeBuilder.ID, *repo.job.ResumeBuilderID)
		})
	}
}
