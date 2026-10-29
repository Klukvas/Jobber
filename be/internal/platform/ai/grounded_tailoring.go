package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type tailoringSelection struct {
	FactIDs      []string `json:"fact_ids"`
	After        string   `json:"after,omitempty"`
	Reason       string   `json:"reason"`
	ExperienceID string   `json:"experience_id,omitempty"`
}
type tailoringPlan struct {
	Summary     *tailoringSelection       `json:"summary"`
	Experiences []tailoringSelection      `json:"experiences"`
	SkillOrder  *TailoringSkillSuggestion `json:"skill_order"`
}

func resolveTailoringPlan(c groundingCatalog, p tailoringPlan) (*ResumeTailoringSuggestions, error) {
	result := &ResumeTailoringSuggestions{Experiences: []TailoringTextSuggestion{}}
	convert := func(s tailoringSelection, section, entry string) (TailoringTextSuggestion, error) {
		facts, err := resolveFacts(c, s.FactIDs, section, entry)
		if err != nil {
			return TailoringTextSuggestion{}, err
		}
		suggestion := TailoringTextSuggestion{Reason: "Emphasize existing source evidence relevant to this job.", ExperienceID: entry, Evidence: []TailoringEvidence{}}
		text := make([]groundedFact, 0, len(facts))
		for _, f := range facts {
			if entry != "" {
				f.Text = f.SourceText
			}
			text = append(text, f)
			suggestion.Evidence = append(suggestion.Evidence, TailoringEvidence{Section: f.Section, EntryID: f.EntryID, Quote: f.Text})
		}
		suggestion.After = groundedText(text)
		return suggestion, nil
	}
	if p.Summary != nil {
		if p.Summary.ExperienceID != "" || p.Summary.After != "" {
			return nil, fmt.Errorf("summary contains experience ID")
		}
		s, err := convert(*p.Summary, "", "")
		if err != nil {
			return nil, err
		}
		result.Summary = &s
	}
	seen := map[string]bool{}
	for _, edit := range p.Experiences {
		id, ok := c.Experiences[edit.ExperienceID]
		if !ok || seen[id] {
			return nil, fmt.Errorf("invalid experience selection")
		}
		seen[id] = true
		s, err := convert(edit, "experience", id)
		if err != nil {
			return nil, err
		}
		if edit.After != "" {
			s.After = edit.After
		}
		result.Experiences = append(result.Experiences, s)
	}
	if p.SkillOrder != nil {
		order := &TailoringSkillSuggestion{Reason: "Prioritize relevant existing skills; retain every remaining skill.", SkillIDs: []string{}, Evidence: []TailoringEvidence{}}
		byID := map[string]groundedFact{}
		for _, f := range c.Skills {
			byID[f.ID] = f
			byID[f.EntryID] = f
		}
		seen := map[string]bool{}
		if len(p.SkillOrder.SkillIDs) == 0 {
			return nil, fmt.Errorf("empty skill ranking")
		}
		for _, id := range p.SkillOrder.SkillIDs {
			f, ok := byID[id]
			if !ok || seen[f.EntryID] {
				return nil, fmt.Errorf("unknown or duplicate skill")
			}
			seen[f.EntryID] = true
			order.SkillIDs = append(order.SkillIDs, f.EntryID)
			if len(order.Evidence) < 20 {
				order.Evidence = append(order.Evidence, TailoringEvidence{Section: "skills", EntryID: f.EntryID, Quote: f.Text})
			}
		}
		// A partial rank promotes the selected known skills; the unchanged remainder
		// stays in source order. The strict service still receives a full permutation.
		for _, f := range c.Skills {
			if !seen[f.EntryID] {
				order.SkillIDs = append(order.SkillIDs, f.EntryID)
			}
		}
		result.SkillOrder = order
	}
	return result, nil
}
func tailoringCatalogJSON(c groundingCatalog) (string, error) {
	type factView struct {
		ID      string `json:"id"`
		Text    string `json:"text"`
		Section string `json:"section"`
		Context string `json:"entry_context,omitempty"`
	}
	view := struct {
		Facts       []factView     `json:"facts"`
		Experiences []string       `json:"experience_ids"`
		Skills      []groundedFact `json:"skills"`
	}{Facts: []factView{}, Experiences: []string{}, Skills: c.Skills}
	for _, f := range c.Facts {
		text := f.Text
		context := ""
		if f.Section == "experience" {
			text = f.SourceText
			context = strings.TrimSuffix(f.Text, f.SourceText)
		}
		view.Facts = append(view.Facts, factView{ID: f.ID, Text: text, Section: f.Section, Context: context})
	}
	for i := 0; i < len(c.Experiences); i++ {
		view.Experiences = append(view.Experiences, fmt.Sprintf("e%d", i))
	}
	data, err := json.Marshal(view)
	return string(data), err
}

func (c *AnthropicClient) verifyTailoringRewrites(ctx context.Context, catalog groundingCatalog, result *ResumeTailoringSuggestions) error {
	type rewrite struct {
		Source  string `json:"source"`
		Context string `json:"entry_context,omitempty"`
		After   string `json:"after"`
	}
	rewrites := []rewrite{}
	for _, edit := range result.Experiences {
		source := ""
		for _, f := range catalog.Facts {
			if f.Section == "experience" && f.EntryID == edit.ExperienceID {
				source = f.SourceText
				break
			}
		}
		allowed := map[string]bool{}
		for _, n := range groundingNumbers.FindAllString(source, -1) {
			allowed[n] = true
		}
		for _, n := range groundingNumbers.FindAllString(edit.After, -1) {
			if !allowed[n] {
				return ErrInvalidTailoringResponse
			}
		}
		pairs := map[string]bool{}
		for _, pair := range groundedMetricPairs.FindAllString(source, -1) {
			pairs[normalizeGroundedMetric(pair)] = true
		}
		for _, pair := range groundedMetricPairs.FindAllString(edit.After, -1) {
			if !pairs[normalizeGroundedMetric(pair)] {
				return ErrInvalidTailoringResponse
			}
		}
	}

	for _, edit := range result.Experiences {
		for _, f := range catalog.Facts {
			if f.Section == "experience" && f.EntryID == edit.ExperienceID && edit.After != f.SourceText {
				rewrites = append(rewrites, rewrite{Source: f.SourceText, Context: strings.TrimSuffix(f.Text, f.SourceText), After: edit.After})
			}
		}
	}
	if len(rewrites) == 0 {
		return nil
	}
	data, err := json.Marshal(struct {
		Edits []rewrite `json:"edits"`
	}{Edits: rewrites})
	if err != nil {
		return err
	}
	response, err := c.callGroundedAPI(ctx, anthropic.MessageNewParams{
		MaxTokens: 512, OutputConfig: groundingVerdictFormat(),
		System: []anthropic.TextBlockParam{{Text: `Independently verify EVERY assertion of each resume AFTER against ONLY the SAME paired SOURCE. Sources belong to different employers and must never be pooled.
entry_context is read-only actor/employer/date metadata for understanding the paired source. AFTER is a description only: it must not copy metadata headers or dates absent from SOURCE.
Return ONE aggregate JSON OBJECT over ALL edits, never an array: {"supported":true|false,"unsupported_claims":["problem"]}. List at most three concise problems.
Check ONE-WAY entailment of each actual AFTER assertion by its paired SOURCE; do not require source completeness. For EACH RETAINED assertion, require the same actor, employer, seniority, ownership, action, object, named technology, metric type/unit/direction, timeframe, negation and restrictive qualifications. Every reported problem must identify a concrete AFTER assertion and the unsupported addition or changed restrictive qualification. Omitted independent source activities alone are never problems.
Team results must not become personal achievements; cost savings must not become revenue, requests or throughput. Related work does not prove mentoring, leadership, new skills or incident-response ownership.
Omitting an unrelated independent sentence or bullet is acceptable and is NOT a failure. Ownership, tools, users and impact belonging solely to OMITTED work need not appear. Example: SOURCE "Built APIs. Owned React dashboards used daily by product teams." AFTER "Built APIs." is valid: the entire dashboard claim was removed, so its ownership/users are not qualifications on the retained API claim. Do not demand a justification for omitting independent work.
Omitting a qualification, condition, negation or causal restriction on a retained claim is NOT valid. Example: SOURCE "Team result: reduced costs 3 million." AFTER "I generated 3 million revenue." is invalid.
Inspect all prose, including assertions without explicit citations. Do not repair. Return false for any unsupported or overstated assertion.` + antiInjectionClause}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(string(data)))},
	})
	if err != nil {
		return err
	}
	var verdict coverLetterVerdict
	if err = decodeGroundedResponse(response, &verdict); err != nil {
		return err
	}
	if verdict.Supported == nil || !*verdict.Supported || verdict.UnsupportedClaims == nil || len(verdict.UnsupportedClaims) != 0 {
		return ErrInvalidTailoringResponse
	}
	return nil
}
func localizeGroundedReasons(result *ResumeTailoringSuggestions, locale string) {
	reason := "Emphasize existing source evidence relevant to this job."
	switch locale {
	case "ru":
		reason = "Выделить подтверждённый опыт, релевантный вакансии."
	case "uk", "ua":
		reason = "Виділити підтверджений досвід, релевантний вакансії."
	}
	if result.Summary != nil {
		result.Summary.Reason = reason
	}
	for i := range result.Experiences {
		result.Experiences[i].Reason = reason
	}
	if result.SkillOrder != nil {
		result.SkillOrder.Reason = reason
	}
}

var groundedMetricPairs = regexp.MustCompile(`[\p{Sc}]\s*[+-]?[0-9]+(?:[.,][0-9]+)?|[+-]?[0-9]+(?:[.,][0-9]+)?\s*[%‰‱\p{Sc}]`)

func normalizeGroundedMetric(value string) string { return strings.Join(strings.Fields(value), "") }
