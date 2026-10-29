package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var ErrInvalidTailoringResponse = errors.New("invalid resume tailoring response")

type TailoringEvidence struct {
	Section string `json:"section"`
	EntryID string `json:"entry_id,omitempty"`
	Quote   string `json:"quote"`
}

type TailoringTextSuggestion struct {
	ExperienceID string              `json:"experience_id,omitempty"`
	After        string              `json:"after"`
	Reason       string              `json:"reason"`
	Evidence     []TailoringEvidence `json:"evidence"`
}

type TailoringSkillSuggestion struct {
	SkillIDs []string            `json:"skill_ids"`
	Reason   string              `json:"reason"`
	Evidence []TailoringEvidence `json:"evidence"`
}

type ResumeTailoringSuggestions struct {
	Summary     *TailoringTextSuggestion  `json:"summary"`
	Experiences []TailoringTextSuggestion `json:"experiences"`
	SkillOrder  *TailoringSkillSuggestion `json:"skill_order"`
}

// TailorResume proposes edits using only the supplied resume facts. Service-level
// validation checks source quotes, identifiers, lengths and skill permutations.
func (c *AnthropicClient) TailorResume(ctx context.Context, jobContext, resumeFacts, locale string) (*ResumeTailoringSuggestions, error) {
	catalog, err := resumeGroundingCatalog(resumeFacts, true)
	if err != nil {
		return nil, fmt.Errorf("resume facts: %w", ErrInvalidTailoringResponse)
	}
	if err = validateGroundingInput(catalog, 60000); err != nil {
		return nil, fmt.Errorf("resume facts: %w", ErrInvalidTailoringResponse)
	}
	factsJSON, err := tailoringCatalogJSON(catalog)
	if err != nil {
		return nil, err
	}
	if len(factsJSON) > 60000 {
		return nil, fmt.Errorf("encoded resume facts: %w", ErrInvalidTailoringResponse)
	}
	systemPrompt := `Adapt a resume using ONLY facts in the supplied catalog. The Job is relevance context, NEVER candidate evidence.
For summary select COMPLETE source units; code preserves their actor, employer, dates and text.
For experience descriptions write meaningful, concise Job-relevant rewrites of ONLY the SAME fact.text. entry_context is read-only metadata stored outside the description: NEVER copy its role/employer header or start/end dates into after. Preserve every qualifier, negation, metric meaning, workload, technology and inline client/employer attribution for each retained claim.
For experience AFTER use conservative syntactic reframing and reordering so the most Job-relevant supported work comes first; you may omit unrelated independent sentences. Preserve each retained action/object/named-tool/qualifier bundle in full, including attribution and qualifications. A conservative rephrase may change sentence structure while keeping its exact factual scope. If an adjacent sentence or line qualifies the claim (team/observer attribution, negation, conditions or causal limits), retain that qualification together with the claim as one complete unit. Do not abbreviate specific tools into generic categories, broaden actions or objects, or add connective factual claims. A useful reorder or focus selection is a meaningful edit. Do not replace specific technical detail by broader infrastructure terms. Never add interpretive "demonstrating expertise/ownership" claims or infer leadership. You may omit unrelated independent work; never change actor, scope, seniority or responsibility.
Return a compact plan with experience AFTER text, never source quotes or UUIDs. Fact IDs are resolved by code to exact source text and evidence.
For summary select relevant summary/experience fact IDs. For an experience use only that experience's fact IDs and its short experience ID.
For skill_order rank the most relevant EXISTING skill IDs first; code appends remaining existing skills in source order.
Do not output unknown or repeated IDs. Omit unchanged selections. Prefer 1-2 useful experience edits plus summary/skill emphasis, not every entry. Do not return an empty plan when relevant supported facts can be emphasized.
Return ONLY JSON: {"summary":null|{"fact_ids":["s0","e0f0"],"reason":"relevance"},"experiences":[{"experience_id":"e0","fact_ids":["e0f0"],"after":"meaningful source-grounded rewrite"}],"skill_order":null|{"skill_ids":["k1","k0"],"reason":"relevance"}}.
Reasons explain Job relevance without adding candidate facts. Preserve the resume language. Write reasons in ` + localeToLanguage(locale) + `.` + antiInjectionClause
	response, err := c.callGroundedAPI(ctx, anthropic.MessageNewParams{
		MaxTokens: 1500,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(
			"job_context:\n" + sanitizeForLLM(jobContext, 32000) + "\nresume_facts:\n" + sanitizeForLLM(factsJSON, 60000)))},
	})
	if err != nil {
		return nil, fmt.Errorf("request resume tailoring: %w", err)
	}
	if response == nil || response.StopReason != "end_turn" || len(response.Content) == 0 {
		return nil, fmt.Errorf("incomplete resume tailoring response: %w", ErrInvalidTailoringResponse)
	}
	text := extractTextFromResponse(response.Content)
	if !strings.HasPrefix(strings.TrimSpace(text), "{") || len(text) > 200000 {
		return nil, fmt.Errorf("resume tailoring response too large: %w", ErrInvalidTailoringResponse)
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	var plan tailoringPlan
	if err := decoder.Decode(&plan); err != nil {
		return nil, fmt.Errorf("parse resume tailoring response: %w", ErrInvalidTailoringResponse)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing resume tailoring response: %w", ErrInvalidTailoringResponse)
	}
	result, err := resolveTailoringPlan(catalog, plan)
	if err != nil {
		return nil, fmt.Errorf("resolve tailoring plan: %w", ErrInvalidTailoringResponse)
	}
	localizeGroundedReasons(result, locale)
	if err = c.verifyTailoringRewrites(ctx, catalog, result); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("verify tailoring claims: %w", errors.Join(ErrInvalidTailoringResponse, err))
	}
	return result, nil
}
