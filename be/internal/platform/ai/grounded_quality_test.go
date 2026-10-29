package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestGroundedLetterSemanticRejections(t *testing.T) {
	for _, claim := range []string{"I personally generated 3 million in revenue.", "I mentored engineers.", "I have C systems programming experience."} {
		t.Run(claim, func(t *testing.T) {
			calls := 0
			client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				calls++
				var body string
				if calls == 1 {
					d := letterFixtureDraft("unused", "f0", "")
					d.Paragraphs[0][0].Text = claim
					data, _ := json.Marshal(d)
					body = string(data)
				} else {
					v := letterFixtureVerdict(letterFixtureDraft(claim, "f0", "Reduced costs by 3 million dollars."))
					no := false
					v.Sentences[0].Supported = &no
					v.Sentences[0].Kind = "candidate_fact"
					body = letterFixtureJSON(v)
				}
				r := mockResponse(body)
				r.StopReason = anthropic.StopReasonEndTurn
				return r, nil
			})
			_, err := client.GenerateCoverLetter(t.Context(), "Acme", "", "", "Revenue, C systems and mentoring.", letterFixtureSource("Team result (not a personal achievement):\nReduced costs by 3 million dollars."))
			if !errors.Is(err, ErrInvalidCoverLetterResponse) || calls != 2 {
				t.Fatalf("semantic review did not fail closed: calls=%d err=%v", calls, err)
			}
		})
	}
}
func TestGroundedLetterCancellationSharesDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	calls := 0
	var first time.Time
	client := newTestClient(func(callCtx context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		deadline, ok := callCtx.Deadline()
		if !ok {
			t.Fatal("missing deadline")
		}
		if calls == 1 {
			first = deadline
			body := letterFixtureJSON(letterFixtureDraft("I built Python APIs.", "f0", "I built Python APIs."))
			r := mockResponse(body)
			r.StopReason = anthropic.StopReasonEndTurn
			return r, nil
		}
		if !deadline.Equal(first) {
			t.Fatal("verifier reset the request deadline")
		}
		cancel()
		<-callCtx.Done()
		return nil, callCtx.Err()
	})
	_, err := client.GenerateCoverLetter(ctx, "Acme", "", "", "APIs", letterFixtureSource("I built Python APIs."))
	if !errors.Is(err, context.Canceled) || calls != 2 {
		t.Fatalf("cancellation did not propagate: %v", err)
	}
}
func TestGroundedLetterSelectsDifferentSupportedEvidence(t *testing.T) {
	source := `{"experiences":[{"id":"one","position":"Engineer","company":"Acme","description":"Built cloud APIs."},{"id":"two","position":"Volunteer","company":"Community","description":"Organized community workshops."}]}`
	outputs := []string{}
	for _, job := range []string{"cloud infrastructure", "community workshops and mentoring"} {
		calls := 0
		client := newTestClient(func(_ context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
			calls++
			body := ""
			if calls == 1 {
				text := "I built cloud APIs at Acme, which provides a foundation for the infrastructure work described."
				id := "f0"
				if strings.Contains(job, "community") {
					text = "I organized community workshops, which offers transferable experience for your community focus. I am interested in developing mentoring skills."
					id = "f1"
				}
				quote := "Built cloud APIs."
				if id == "f1" {
					quote = "Organized community workshops."
				}
				b, _ := json.Marshal(letterFixtureDraft(text, id, quote))
				body = string(b)
			} else {
				d := letterFixtureDraft("unused", "f0", "unused")
				body = letterFixtureJSON(letterFixtureVerdict(d))
			}
			r := mockResponse(body)
			r.StopReason = anthropic.StopReasonEndTurn
			return r, nil
		})
		r, err := client.GenerateCoverLetter(t.Context(), "Acme", "", "", job, source)
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, r.Paragraphs[1])
	}
	if outputs[0] == outputs[1] || strings.Contains(outputs[1], "I mentored") {
		t.Fatal("Job relevance or missing-experience framing lost")
	}
}
func TestGroundingPreservesMultilineMetricContext(t *testing.T) {
	description := "Team result (not my personal achievement):\nReduced costs by 3 million dollars, not revenue growth."
	source := `{"experiences":[{"id":"e","company":"Acme","position":"Engineer","start_date":"2020","description":` + string(mustGroundingJSON(t, description)) + `}]}`
	c, err := resumeGroundingCatalog(source, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Facts) != 1 || !strings.Contains(c.Facts[0].Text, description) {
		t.Fatal("qualifier split from metric")
	}
	result, err := resolveTailoringPlan(c, tailoringPlan{Summary: &tailoringSelection{FactIDs: []string{"e0f0"}, Reason: "I mentored engineers"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Summary.After, description) || strings.Contains(result.Summary.Reason, "mentored") {
		t.Fatal("metric context or reason safety lost")
	}
	for _, ids := range [][]string{{"unknown"}, {"e0f0", "e0f0"}} {
		if _, err = resolveFacts(c, ids, "", ""); err == nil {
			t.Fatal("invalid fact IDs accepted")
		}
	}
}
func mustGroundingJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestGroundedTailoringLocalizedReasons(t *testing.T) {
	for _, tt := range []struct{ locale, want string }{{"en", "existing"}, {"ru", "подтверждённый"}, {"uk", "підтверджений"}, {"ua", "підтверджений"}} {
		t.Run(tt.locale, func(t *testing.T) {
			r := &ResumeTailoringSuggestions{Summary: &TailoringTextSuggestion{Reason: "invented mentoring"}, Experiences: []TailoringTextSuggestion{{Reason: "invented"}}, SkillOrder: &TailoringSkillSuggestion{Reason: "invented"}}
			localizeGroundedReasons(r, tt.locale)
			if !strings.Contains(r.Summary.Reason, tt.want) || r.Summary.Reason != r.Experiences[0].Reason || r.Summary.Reason != r.SkillOrder.Reason {
				t.Fatal("source-only localized reasons missing")
			}
		})
	}
}
func TestGroundedTailoringRejectsSemanticRewrite(t *testing.T) {
	source := `{"experiences":[{"id":"e","company":"Acme","position":"Engineer","description":"Team result, not personal: Reduced costs by 3 million dollars."}]}`
	calls := 0
	client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		body := `{"summary":null,"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":"Personally generated 3 million in revenue."}],"skill_order":null}`
		if calls == 2 {
			body = `{"supported":false,"unsupported_claims":["actor and metric semantics changed"]}`
		}
		r := mockResponse(body)
		r.StopReason = anthropic.StopReasonEndTurn
		return r, nil
	})
	_, err := client.TailorResume(t.Context(), "Revenue leadership", source, "en")
	if !errors.Is(err, ErrInvalidTailoringResponse) || calls != 2 {
		t.Fatalf("semantic tailoring error not rejected: %v", err)
	}
}

func TestGroundedLetterManyValidCitations(t *testing.T) {
	catalog := groundingCatalog{Facts: []groundedFact{}}
	for i := 0; i < 12; i++ {
		catalog.Facts = append(catalog.Facts, groundedFact{ID: fmt.Sprintf("f%d", i), Section: "experience", Text: "Built reliable source APIs.", SourceText: "Built reliable source APIs."})
	}
	d := letterFixtureDraft("unused", "f11", "")
	if _, _, err := resolveLetterDraft(catalog, d); err != nil {
		t.Fatal("complete source inventory was capped", err)
	}
	d.Paragraphs[1][0].UnitID = "foreign"
	if _, _, err := resolveLetterDraft(catalog, d); err == nil {
		t.Fatal("unknown unit accepted")
	}
}
func TestGroundedVerifierRejectsPerEditArray(t *testing.T) {
	response := mockResponse(`[{"supported":true,"unsupported_claims":[]}]`)
	response.StopReason = anthropic.StopReasonEndTurn
	var verdict coverLetterVerdict
	if err := decodeGroundedResponse(response, &verdict); err == nil {
		t.Fatal("per-edit array bypassed aggregate verdict")
	}
	config := groundingVerdictFormat()
	if config.Format.Schema["type"] != "object" {
		t.Fatal("aggregate object not constrained at provider")
	}
}

func TestTailoringRewriteCatalogSeparatesMetadataDates(t *testing.T) {
	source := `{"experiences":[{"id":"e","company":"Acme","position":"Engineer","start_date":"2024-06","description":"Built reliable Go APIs."}]}`
	catalog, err := resumeGroundingCatalog(source, true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := tailoringCatalogJSON(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var view struct {
		Facts []struct {
			Text    string `json:"text"`
			Context string `json:"entry_context"`
		} `json:"facts"`
	}
	if err = json.Unmarshal([]byte(data), &view); err != nil {
		t.Fatal(err)
	}
	if view.Facts[0].Text != "Built reliable Go APIs." || strings.Contains(view.Facts[0].Text, "2024") || !strings.Contains(view.Facts[0].Context, "2024-06") {
		t.Fatal("metadata contaminated allowed description text")
	}
	calls := 0
	client := newTestClient(func(_ context.Context, p anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		body := `{"summary":null,"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":"Developed reliable Go APIs."}],"skill_order":null}`
		if calls == 2 {
			var audit struct {
				Edits []struct {
					Source  string `json:"source"`
					Context string `json:"entry_context"`
				} `json:"edits"`
			}
			if json.Unmarshal([]byte(p.Messages[0].Content[0].OfText.Text), &audit) != nil || len(audit.Edits) != 1 || audit.Edits[0].Source != "Built reliable Go APIs." || !strings.Contains(audit.Edits[0].Context, "2024-06") {
				t.Fatal("description evidence and read-only metadata were not separated")
			}
			body = `{"supported":true,"unsupported_claims":[]}`
		}
		r := mockResponse(body)
		r.StopReason = anthropic.StopReasonEndTurn
		return r, nil
	})
	_, err = client.TailorResume(t.Context(), "Go APIs", source, "en")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTailoringMetadataDateRejectedBeforeSemanticCall(t *testing.T) {
	calls := 0
	client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		r := mockResponse(`{"summary":null,"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":"2024: Built reliable Go APIs."}],"skill_order":null}`)
		r.StopReason = anthropic.StopReasonEndTurn
		return r, nil
	})
	_, err := client.TailorResume(t.Context(), "Go", `{"experiences":[{"id":"e","start_date":"2024","description":"Built reliable Go APIs."}]}`, "en")
	if !errors.Is(err, ErrInvalidTailoringResponse) || calls != 1 {
		t.Fatalf("unsupported numeric metadata reached verifier: calls=%d err=%v", calls, err)
	}
}

func TestGroundedLetterFutureIntentDoesNotClaimListedSkillExperience(t *testing.T) {
	source := `{"experiences":[{"id":"e","company":"Acme","position":"Engineer","description":"Built Python APIs."}],"skills":[{"id":"s","name":"Go"}]}`
	d := letterFixtureDraft("unused", "f0", "")
	d.Paragraphs[2][0].Text = "I would welcome the opportunity to develop Go skills in a client-facing team."
	calls := 0
	client := newTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
		calls++
		body := letterFixtureJSON(d)
		if calls == 2 {
			body = letterFixtureJSON(letterFixtureVerdict(d))
		}
		r := mockResponse(body)
		r.StopReason = anthropic.StopReasonEndTurn
		return r, nil
	})
	result, err := client.GenerateCoverLetter(t.Context(), "Target", "", "", "Backend APIs", source)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(result.Paragraphs) != 3 || !strings.Contains(result.Paragraphs[1], "Built Python APIs.") || !strings.Contains(result.Paragraphs[2], "would welcome") {
		t.Fatal("controlled source prose or future intent lost")
	}
}
