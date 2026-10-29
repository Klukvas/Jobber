package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

var ErrInvalidCoverLetterResponse = errors.New("invalid cover letter response")

type letterSupport struct {
	FactID string `json:"fact_id"`
	Quote  string `json:"quote"`
}
type letterSentence struct {
	Kind     string          `json:"kind"`
	UnitID   string          `json:"unit_id,omitempty"`
	Text     string          `json:"text,omitempty"`
	Supports []letterSupport `json:"supports,omitempty"`
}

func (s *letterSentence) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	var kind string
	if err := json.Unmarshal(fields["kind"], &kind); err != nil {
		return err
	}
	allowed := map[string]bool{"kind": true}
	switch kind {
	case "source_anchor":
		allowed["unit_id"] = true
	case "interest", "job_context":
		allowed["text"] = true
	default:
		return ErrInvalidCoverLetterResponse
	}
	for key := range fields {
		if !allowed[key] {
			return ErrInvalidCoverLetterResponse
		}
	}
	type plain letterSentence
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = letterSentence(value)
	return nil
}

type coverLetterDraft struct {
	Paragraphs [][]letterSentence `json:"paragraphs"`
}
type auditedLetterSupport struct {
	Quote  string `json:"quote"`
	FactID string `json:"fact_id"`
}
type auditedLetterSentence struct {
	ID            int                    `json:"id"`
	Text          string                 `json:"text"`
	GeneratorKind string                 `json:"generator_kind"`
	Supports      []auditedLetterSupport `json:"supports"`
}
type letterSentenceVerdict struct {
	Sentences []struct {
		ID        int    `json:"id"`
		Kind      string `json:"kind"`
		Supported *bool  `json:"supported"`
	} `json:"sentences"`
	Issues []struct {
		ID     int    `json:"id"`
		Reason string `json:"reason"`
	} `json:"issues"`
}
type coverLetterVerdict struct {
	Supported         *bool    `json:"supported"`
	UnsupportedClaims []string `json:"unsupported_claims"`
}

// groundedQualityParams keeps the model and raw thinking setting in the serialized
// request, so the production transport and measured test hook use the same body.
func groundedQualityParams(params anthropic.MessageNewParams) anthropic.MessageNewParams {
	params.Model = anthropic.Model("claude-sonnet-5-5")
	params.OutputConfig.Effort = anthropic.OutputConfigEffortLow
	params.SetExtraFields(map[string]any{"thinking": map[string]any{"type": "between_tools"}})
	return params
}

// Quality operations have explicit per-stage attempts. Other AI operations keep
// their existing transport policy; no automatic repair or SDK retry is hidden here.
func (c *AnthropicClient) callGroundedAPI(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params = groundedQualityParams(params)
	return c.budgetedMessage(ctx, params, func() (*anthropic.Message, error) { return c.callUnbudgetedGroundedAPI(ctx, params) })
}
func (c *AnthropicClient) callUnbudgetedGroundedAPI(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	if c.callAPIFunc != nil {
		return c.callAPIFunc(ctx, params)
	}
	return c.client.Messages.New(ctx, params, option.WithMaxRetries(0), option.WithHTTPClient(noReplayHTTPClient()))
}
func decodeGroundedResponse(response *anthropic.Message, target any) error {
	if response == nil || response.StopReason != "end_turn" || len(response.Content) == 0 {
		return ErrInvalidCoverLetterResponse
	}
	text := extractTextFromResponse(response.Content)
	if len(text) > 20000 {
		return ErrInvalidCoverLetterResponse
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidCoverLetterResponse
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return ErrInvalidCoverLetterResponse
	}
	return nil
}
func (c *AnthropicClient) generateGroundedCoverLetter(ctx context.Context, company, recipient, recipientTitle, job, resume string) (*CoverLetterContent, error) {
	catalog, err := resumeGroundingCatalog(resume, false)
	if err != nil {
		return nil, err
	}
	if err = validateGroundingInput(catalog, 60000); err != nil {
		return nil, fmt.Errorf("candidate facts: %w", err)
	}
	type unitView struct {
		ID     string `json:"unit_id"`
		FactID string `json:"fact_id"`
		Words  int    `json:"rendered_words"`
	}
	units := []unitView{}
	for _, f := range letterAnchorFacts(catalog) {
		units = append(units, unitView{ID: f.ID, FactID: f.ID, Words: len(strings.Fields(renderLetterAnchor(f)))})
	}
	// The production caller provides a structured Builder Resume. Nonempty raw
	// text still has no typed, complete source unit and is rejected explicitly.
	if len(catalog.Facts) > 0 && len(units) == 0 {
		return nil, ErrInvalidCoverLetterResponse
	}
	input, err := json.Marshal(struct {
		Company        string         `json:"company"`
		Recipient      string         `json:"recipient"`
		RecipientTitle string         `json:"recipient_title"`
		Job            string         `json:"job"`
		Facts          []groundedFact `json:"candidate_facts"`
		Units          []unitView     `json:"source_units"`
	}{Company: truncateString(company, 200), Recipient: truncateString(recipient, 200), RecipientTitle: truncateString(recipientTitle, 200), Job: sanitizeForLLM(job, 10000), Facts: catalog.Facts, Units: units})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	response, err := c.callGroundedAPI(ctx, anthropic.MessageNewParams{
		MaxTokens: 1500,
		System: []anthropic.TextBlockParam{{Text: `Write a polished professional cover letter as EXACTLY three cohesive paragraphs, each containing 1-3 segment objects. Aim for 150-200 words and never exceed 250 rendered words. Return ONLY JSON:
{"paragraphs":[[{"kind":"interest","text":"honest role interest"}],[{"kind":"source_anchor","unit_id":"f2"}],[{"kind":"interest","text":"future interest and invitation to discuss"}]]}.
The opening and closing are fluent, Job-specific interest or job_context prose. The body selects ONE relevant complete experience unit from source_units; two only when the complete letter remains concise and readable. Unit IDs reference candidate_facts; rendered_words includes the role/employer frame. Code renders each complete original experience description verbatim, preserving its governing subject, client, technologies and all qualifications. Do not reproduce its text, quotes, fragments or supports in the source_anchor object: only kind and unit_id are allowed. Never modify or combine historical claim fragments. Do not choose an oversized unit or try to truncate it.
All candidate history/ability/skill assertions MUST come from a source_anchor. Free text may ONLY be future interest/application intentions or target-role context. It must not contain past or current candidate assertions, including modifiers such as "my background", "my experience", "my ability", "I bring", or claims of proficiency. A future wish to contribute is fine; a future wish to apply a claimed past capability is a factual assertion and forbidden in free text. Job requirements are never candidate evidence.
Produce a connected professional letter with a relevant opening, one documented achievement anchor in the body and a natural future-facing closing. This is not a copy of the whole CV or a list of quotations. Do not add generic filler or wrap a fact in new scope, causal links, first-person ownership or cross-component technology claims. Code supplies the conventional greeting and closing; return paragraph objects only.` + antiInjectionClause + letterCredentialInstructions(catalog)}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(string(input)))},
	})
	if err != nil {
		return nil, err
	}
	var draft coverLetterDraft
	if err = decodeGroundedResponse(response, &draft); err != nil {
		return nil, err
	}
	content, sentences, err := resolveLetterDraft(catalog, draft)
	if err != nil {
		return nil, err
	}
	if name := strings.Join(strings.Fields(recipient), " "); name != "" {
		content.Greeting = "Dear " + truncateString(name, 160) + ","
	}
	if !letterNumbersSupported(catalog, job, *content) {
		return nil, ErrInvalidCoverLetterResponse
	}
	audit, err := json.Marshal(struct {
		Facts     []groundedFact          `json:"candidate_facts"`
		Sentences []auditedLetterSentence `json:"sentences"`
	}{catalog.Facts, sentences})
	if err != nil {
		return nil, err
	}
	response, err = c.callGroundedAPI(ctx, anthropic.MessageNewParams{
		MaxTokens: 768, OutputConfig: letterSentenceVerdictFormat(),
		System: []anthropic.TextBlockParam{{Text: `Independently classify and verify EVERY output sentence. Generator kinds are untrusted, never exemptions. source_anchor is a complete code-rendered source description with an impersonal role/employer bridge; classify it as candidate_fact and audit the full text, including team/client/observer qualifications. It may contain multiple complete original sentences. Free interest/job_context segments must not hide any historical assertion; independently reclassify the whole rendered segment. Return JSON {"sentences":[{"id":0,"kind":"candidate_fact|interest|job_context","supported":true|false}],"issues":[{"id":0,"reason":"concrete unsupported assertion"}]}.
Return EXACTLY one verdict for EVERY input sentence id, no missing/duplicate ids. candidate_fact means any assertion of candidate past history, current skill, ability or experience; mixed fact+interest/job sentences are candidate_fact. Pure future desire/application intention is interest; assertions only about the target employer/role are job_context. A future wish to apply a listed language or join a client-facing team does not claim prior language use or team experience.
For each candidate_fact sentence verify each assertion against its cited supports AND the entire original fact found by fact_id in candidate_facts, including actor/client/subproject context and qualifications. No supports means unsupported. The full candidate_facts catalog is also supplied to detect contradictions or cherry-picked restrictions; it must not be pooled to transfer technologies, responsibility, ownership or metrics between activities, employers, clients or subprojects.
A skill-list source supports listing the skill, not professional experience/proficiency. Container/cloud use alone does not prove distributed deployments. Preserve named tools, actions, objects, metric type/unit/direction/time/workload, negation, conditions, attribution and causal restrictions. Team achievement is not personal leadership; cost savings are not revenue. General good-engineer virtues, schema responsibility, testing/documentation standards and broad expertise need explicit evidence.
Do not require all source work to appear. Inspect the full sentence for hidden assertions, not only its quoted part. A quote's presence does not prove entailment. Pure intentions, cautious relevance and target-role context need no candidate-history evidence. Report at most three concrete problems. Any unsupported factual assertion makes its sentence false. Do not repair.` + antiInjectionClause + letterCredentialInstructions(catalog)}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(string(audit)))},
	})
	if err != nil {
		return nil, err
	}
	var verdict letterSentenceVerdict
	if err = decodeGroundedResponse(response, &verdict); err != nil {
		return nil, err
	}
	if err = validateLetterSentenceVerdict(sentences, verdict); err != nil {
		return nil, err
	}
	return content, nil
}
func validLetterKind(kind string) bool {
	return kind == "candidate_fact" || kind == "interest" || kind == "job_context"
}
func hasLetterExperience(catalog groundingCatalog) bool {
	for _, fact := range catalog.Facts {
		if fact.Section == "experience" && strings.TrimSpace(fact.SourceText) != "" {
			return true
		}
	}
	return false
}
func letterAnchorFacts(catalog groundingCatalog) []groundedFact {
	experienced := hasLetterExperience(catalog)
	facts := []groundedFact{}
	for _, fact := range catalog.Facts {
		eligible := fact.Section == "experience"
		if !experienced {
			switch fact.Section {
			case "summary", "skills", "education", "languages", "certifications", "projects", "volunteering", "custom":
				eligible = true
			default:
				eligible = false
			}
		}
		if eligible && strings.TrimSpace(fact.SourceText) != "" {
			facts = append(facts, fact)
		}
	}
	return facts
}
func letterCredentialInstructions(catalog groundingCatalog) string {
	if hasLetterExperience(catalog) || len(letterAnchorFacts(catalog)) == 0 {
		return ""
	}
	return "\nThis profile has no eligible work experience. Instead of an experience/achievement unit, select ONE complete summary, listed skill, education, language, certification, project, volunteering or custom-section source entry from source_units for the body, using the same source_anchor object. Code renders the whole original entry with section labels and all actors, URLs, proficiency, degree, date, expiry, coursework and other qualifications. Listed skills and languages establish only what is explicitly listed, not professional use or proficiency beyond the explicit level. Education does not imply an awarded degree; preserve current-study and degree restrictions. Certification dates do not imply current validity. Project, volunteer and custom-section activities do not imply employment, personal ownership or leadership. Never combine entries into invented employment, accomplishments or technology experience. All free prose remains subject to the same independent factual audit."
}
func renderLetterAnchor(f groundedFact) string {
	switch f.Section {
	case "summary":
		return "My résumé summary states: " + f.SourceText
	case "skills":
		return "My résumé lists this skill: " + f.SourceText
	case "education":
		return "My résumé lists this education: " + f.SourceText
	case "languages":
		return "My résumé lists this language: " + f.SourceText
	case "certifications":
		return "My résumé lists this certification: " + f.SourceText
	case "projects":
		return "My résumé lists this project: " + f.SourceText
	case "volunteering":
		return "My résumé lists this volunteering: " + f.SourceText
	case "custom":
		return "My résumé includes this section: " + f.SourceText
	}
	bridge := "My résumé describes the following work: "
	switch {
	case f.ExperienceRole != "" && f.ExperienceCompany != "":
		bridge = "In my role as " + f.ExperienceRole + " at " + f.ExperienceCompany + ", the work included: "
	case f.ExperienceCompany != "":
		bridge = "At " + f.ExperienceCompany + ", the work included: "
	case f.ExperienceRole != "":
		bridge = "In my role as " + f.ExperienceRole + ", the work included: "
	}
	return bridge + f.SourceText
}
func resolveLetterDraft(catalog groundingCatalog, draft coverLetterDraft) (*CoverLetterContent, []auditedLetterSentence, error) {
	if len(draft.Paragraphs) != 3 {
		return nil, nil, ErrInvalidCoverLetterResponse
	}
	byID := map[string]groundedFact{}
	eligible := map[string]bool{}
	for _, f := range catalog.Facts {
		byID[f.ID] = f
	}
	for _, f := range letterAnchorFacts(catalog) {
		eligible[f.ID] = true
	}
	content := &CoverLetterContent{Greeting: "Dear Hiring Manager,", Closing: "Sincerely,", Paragraphs: []string{}}
	audit := []auditedLetterSentence{}
	anchors := map[string]bool{}
	for pi, paragraph := range draft.Paragraphs {
		if len(paragraph) < 1 || len(paragraph) > 3 {
			return nil, nil, ErrInvalidCoverLetterResponse
		}
		parts := []string{}
		for _, segment := range paragraph {
			item := auditedLetterSentence{ID: len(audit), GeneratorKind: segment.Kind, Supports: []auditedLetterSupport{}}
			switch segment.Kind {
			case "source_anchor":
				f, ok := byID[segment.UnitID]
				if !ok || !eligible[f.ID] || segment.Text != "" || len(segment.Supports) != 0 || anchors[f.ID] || pi != 1 || len(anchors) >= 2 {
					return nil, nil, ErrInvalidCoverLetterResponse
				}
				anchors[f.ID] = true
				item.Text = renderLetterAnchor(f)
				item.Supports = append(item.Supports, auditedLetterSupport{FactID: f.ID, Quote: f.SourceText})
			case "interest", "job_context":
				if segment.UnitID != "" || len(segment.Supports) != 0 || strings.TrimSpace(segment.Text) == "" || len([]rune(segment.Text)) > 1000 || strings.ContainsRune(segment.Text, '\x00') {
					return nil, nil, ErrInvalidCoverLetterResponse
				}
				item.Text = strings.TrimSpace(segment.Text)
			default:
				return nil, nil, ErrInvalidCoverLetterResponse
			}
			audit = append(audit, item)
			parts = append(parts, item.Text)
		}
		text := strings.Join(parts, " ")
		if len([]rune(text)) > 4000 {
			return nil, nil, ErrInvalidCoverLetterResponse
		}
		content.Paragraphs = append(content.Paragraphs, text)
	}
	if len(eligible) > 0 && len(anchors) == 0 {
		return nil, nil, ErrInvalidCoverLetterResponse
	}
	if len(strings.Fields(content.Greeting+" "+strings.Join(content.Paragraphs, " ")+" "+content.Closing)) > 250 {
		return nil, nil, ErrInvalidCoverLetterResponse
	}
	return content, audit, nil
}
func validateLetterSentenceVerdict(sentences []auditedLetterSentence, verdict letterSentenceVerdict) error {
	if len(verdict.Sentences) != len(sentences) || verdict.Issues == nil || len(verdict.Issues) != 0 {
		return ErrInvalidCoverLetterResponse
	}
	seen := map[int]bool{}
	for _, item := range verdict.Sentences {
		if item.ID < 0 || item.ID >= len(sentences) || seen[item.ID] || !validLetterKind(item.Kind) || item.Supported == nil || !*item.Supported {
			return ErrInvalidCoverLetterResponse
		}
		seen[item.ID] = true
		if item.Kind == "candidate_fact" && len(sentences[item.ID].Supports) == 0 {
			return ErrInvalidCoverLetterResponse
		}
	}
	return nil
}
func letterSentenceVerdictFormat() anthropic.OutputConfigParam {
	sentence := map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}, "kind": map[string]any{"type": "string", "enum": []string{"candidate_fact", "interest", "job_context"}}, "supported": map[string]any{"type": "boolean"}}, "required": []string{"id", "kind", "supported"}, "additionalProperties": false}
	issue := map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}, "reason": map[string]any{"type": "string"}}, "required": []string{"id", "reason"}, "additionalProperties": false}
	return anthropic.OutputConfigParam{Format: anthropic.JSONOutputFormatParam{Schema: map[string]any{"type": "object", "properties": map[string]any{"sentences": map[string]any{"type": "array", "items": sentence}, "issues": map[string]any{"type": "array", "items": issue}}, "required": []string{"sentences", "issues"}, "additionalProperties": false}}}
}

var groundingNumbers = regexp.MustCompile(`[0-9]+(?:[.,][0-9]+)?`)

func letterNumbersSupported(catalog groundingCatalog, job string, letter CoverLetterContent) bool {
	allowed := map[string]bool{}
	source := job
	for _, f := range catalog.Facts {
		source += "\n" + f.Text + " " + f.Actor
	}
	for _, n := range groundingNumbers.FindAllString(source, -1) {
		allowed[n] = true
	}
	text := strings.Join(letter.Paragraphs, "\n")
	for _, n := range groundingNumbers.FindAllString(text, -1) {
		if !allowed[n] {
			return false
		}
	}
	return true
}

func groundingVerdictFormat() anthropic.OutputConfigParam {
	return anthropic.OutputConfigParam{Format: anthropic.JSONOutputFormatParam{Schema: map[string]any{
		"type": "object", "properties": map[string]any{"supported": map[string]any{"type": "boolean"}, "unsupported_claims": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}, "required": []string{"supported", "unsupported_claims"}, "additionalProperties": false,
	}}}
}
