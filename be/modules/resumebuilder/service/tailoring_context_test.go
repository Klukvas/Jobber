package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/anthropics/anthropic-sdk-go"
)

func TestTailoringCanonicalContextIsSameEntryOnly(t *testing.T) {
	exp := &model.ExperienceDTO{ID: "e1", Position: "Engineer", Company: "Acme", StartDate: "2020", Description: "Team result (not personal):\nReduced costs by 3 million, not revenue."}
	other := &model.ExperienceDTO{ID: "e2", Position: "Lead", Company: "Other", StartDate: "2022", Description: "Built APIs."}
	source := &model.FullResumeDTO{Experiences: []*model.ExperienceDTO{exp, other}}
	canonical := ai.GroundedExperienceContext(exp.Position, exp.Company, exp.StartDate, exp.EndDate, exp.Description)
	for _, tt := range []struct {
		name, kind, quote string
		wantErr           bool
	}{
		{name: "whole same entry", kind: "summary", quote: canonical},
		{name: "legacy verbatim description", kind: "summary", quote: exp.Description},
		{name: "fabricated employer", kind: "summary", quote: ai.GroundedExperienceContext(exp.Position, "Fabricated", exp.StartDate, exp.EndDate, exp.Description), wantErr: true},
		{name: "fabricated date", kind: "summary", quote: ai.GroundedExperienceContext(exp.Position, exp.Company, "2099", exp.EndDate, exp.Description), wantErr: true},
		{name: "cross entry actor", kind: "summary", quote: ai.GroundedExperienceContext(other.Position, other.Company, exp.StartDate, exp.EndDate, exp.Description), wantErr: true},
		{name: "canonical header fragment", kind: "summary", quote: "Engineer at Acme", wantErr: true},
		{name: "experience edits description only", kind: "experience", quote: canonical, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateTailoringEvidence(source, []ai.TailoringEvidence{{Section: "experience", EntryID: "e1", Quote: tt.quote}}, tt.kind, "e1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("validation=%v; expected error=%v", err, tt.wantErr)
			}
		})
	}
}

func TestTailoringMeaningfulExperienceRewriteThroughService(t *testing.T) {
	svc, repo, _, _, _ := tailoringFixture(t)
	repo.source.Experiences[0].Description = "Maintained an internal dashboard. Built reliable Go APIs."
	repo.source.Experiences[0].Position = "Engineer"
	repo.source.Experiences[0].Company = "Acme"
	repo.source.Experiences[0].StartDate = "2020"
	repo.source.Experiences[0].EndDate = "2022"
	before, _ := json.Marshal(repo.source)
	repo.job.Description = "SECRET_JOB_ONLY_REQUIREMENT"
	calls := 0
	svc.client = ai.NewTestClient(func(ctx context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 25*time.Second {
			t.Fatal("production deadline absent")
		}
		body := `{"summary":null,"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":"Built reliable Go APIs. Maintained an internal dashboard."}],"skill_order":null}`
		if calls == 1 {
			prompt := p.Messages[0].Content[0].OfText.Text
			if !strings.Contains(prompt, `"text":"Maintained an internal dashboard. Built reliable Go APIs."`) || !strings.Contains(prompt, `"entry_context":"Engineer at Acme [2020 - 2022]:\n"`) {
				t.Fatal("description facts were not separated from read-only metadata")
			}
		}
		if calls == 2 {
			var audit struct {
				Edits []struct {
					Source, After string
					Context       string `json:"entry_context"`
				} `json:"edits"`
			}
			if json.Unmarshal([]byte(p.Messages[0].Content[0].OfText.Text), &audit) != nil || len(audit.Edits) != 1 || audit.Edits[0].Source != repo.source.Experiences[0].Description || !strings.Contains(audit.Edits[0].Context, "2020") || strings.Contains(audit.Edits[0].After, "2020") {
				t.Fatal("verifier source and metadata boundaries were lost")
			}
			if strings.Contains(p.Messages[0].Content[0].OfText.Text, "SECRET_JOB_ONLY_REQUIREMENT") {
				t.Fatal("job contaminated verifier")
			}
			body = `{"supported":true,"unsupported_claims":[]}`
		}
		return &anthropic.Message{StopReason: anthropic.StopReasonEndTurn, Content: []anthropic.ContentBlockUnion{{Type: "text", Text: body}}}, nil
	})
	result, err := svc.Generate(t.Context(), "user", repo.job.ID, repo.source.ID, "ru")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 1 || result.Changes[0].Kind != "experience" || string(result.Changes[0].After) == string(result.Changes[0].Before) {
		t.Fatal("meaningful experience adaptation was lost")
	}
	if !strings.Contains(result.Changes[0].Reason, "подтверждённый") {
		t.Fatal("localized reason lost")
	}
	after, _ := json.Marshal(repo.source)
	if string(before) != string(after) {
		t.Fatal("generation mutated source actor, role, dates or description")
	}
	if calls != 2 {
		t.Fatal("rewrite verifier not used")
	}
}

func TestTailoringJobFocusPreservesCompleteRetainedClaims(t *testing.T) {
	for _, tt := range []struct{ name, job, after string }{
		{"backend focus", "Backend APIs", "Developed reliable Go APIs with PostgreSQL. Maintained an internal dashboard."},
		{"dashboard focus", "Internal dashboards", "Owned React modules that product teams used daily. Maintained an internal dashboard."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _, _ := tailoringFixture(t)
			source := "Owned React modules used daily by product teams. Built reliable Go APIs with PostgreSQL. Maintained an internal dashboard."
			repo.source.Experiences[0].Description = source
			repo.job.Description = tt.job
			calls := 0
			svc.client = ai.NewTestClient(func(_ context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				var body string
				if calls == 1 {
					b, _ := json.Marshal(map[string]any{"summary": nil, "experiences": []map[string]any{{"experience_id": "e0", "fact_ids": []string{"e0f0"}, "after": tt.after}}, "skill_order": nil})
					body = string(b)
				} else {
					var audit struct {
						Edits []struct{ Source, After string } `json:"edits"`
					}
					if json.Unmarshal([]byte(p.Messages[0].Content[0].OfText.Text), &audit) != nil || len(audit.Edits) != 1 || audit.Edits[0].Source != source || audit.Edits[0].After != tt.after {
						t.Fatal("focus escaped same-entry source")
					}
					body = `{"supported":true,"unsupported_claims":[]}`
				}
				return &anthropic.Message{StopReason: anthropic.StopReasonEndTurn, Content: []anthropic.ContentBlockUnion{{Type: "text", Text: body}}}, nil
			})
			result, err := svc.Generate(t.Context(), "user", repo.job.ID, repo.source.ID, "en")
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Changes) != 1 || result.Changes[0].Kind != "experience" || calls != 2 {
				t.Fatal("meaningful focus edit lost")
			}
			var after string
			if json.Unmarshal(result.Changes[0].After, &after) != nil || after != tt.after {
				t.Fatal("retained source sentences changed")
			}
		})
	}
}
