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
		return nil, invalidTailoringResponse("input", "source", "$")
	}
	if err = validateGroundingInput(catalog, 60000); err != nil {
		return nil, invalidTailoringResponse("input", "source", "$")
	}
	factsJSON, err := tailoringCatalogJSON(catalog)
	if err != nil {
		return nil, err
	}
	if len(factsJSON) > 60000 {
		return nil, invalidTailoringResponse("input", "size", "$")
	}
	systemPrompt := `Adapt a resume using ONLY facts in the supplied catalog. The Job is relevance context, NEVER candidate evidence.
Your task is to make supported work easier for a reader of THIS Job to find. First identify the Job's main responsibilities, then prioritize matching documented work in the summary and each relevant experience.
An already relevant resume can still benefit from emphasis: move a complete matching achievement or activity ahead of general duties; reorder independent sentences without changing their words. This is useful tailoring and does not require inventing facts or stronger verbs.
For example, for a database-performance Job, SOURCE "Built HTTP APIs. Tuned PostgreSQL queries. No management duties." can become "Tuned PostgreSQL queries. Built HTTP APIs. No management duties." All assertions and qualifications are preserved. This example is not candidate evidence.
For summary select COMPLETE source units; code preserves their actor, employer, dates and text.
Use the summary to surface relevant documented experience. Even when the existing summary names the right tools, it may omit a concrete Job-relevant achievement found in experience; prefer that complete experience source unit over generic tool/role prose. Selecting the existing summary fact IDs alone reproduces the original summary and is NOT a change; omit that no-op. Selecting relevant experience fact IDs instead can provide a concrete evidence-based summary, including its original attribution. Do not combine it with an unchanged summary merely to make the summary longer.
For experience descriptions write meaningful, concise Job-relevant rewrites of ONLY the SAME fact.text. entry_context is read-only metadata stored outside the description: NEVER copy its role/employer header or start/end dates into after. Preserve every qualifier, negation, metric meaning, workload, technology and inline client/employer attribution for each retained claim.
For experience AFTER use conservative syntactic reframing and reordering so the most Job-relevant supported work comes first; you may omit unrelated independent sentences. Preserve each retained action/object/named-tool/qualifier bundle in full, including attribution and qualifications. A conservative rephrase may change sentence structure while keeping its exact factual scope. If an adjacent sentence or line qualifies the claim (team/observer attribution, negation, conditions or causal limits), retain that qualification together with the claim as one complete unit. Do not abbreviate specific tools into generic categories, broaden actions or objects, or add connective factual claims. A useful reorder or focus selection is a meaningful edit. Do not replace specific technical detail by broader infrastructure terms. Never add interpretive "demonstrating expertise/ownership" claims or infer leadership. You may omit unrelated independent work; never change actor, scope, seniority or responsibility.
Return a compact plan with experience AFTER text, never source quotes or UUIDs. Fact IDs are resolved by code to exact source text and evidence.
For summary select relevant summary/experience fact IDs with at most 5000 Unicode characters of combined full context including metadata. For an experience use only that experience's fact IDs and its short experience ID.
For skill_order rank the most relevant EXISTING skill IDs first; code appends remaining existing skills in source order. When fewer than two existing skills are provided, skill_order MUST be null because no reorder is possible.
Do not output unknown or repeated IDs. Omit unchanged selections. Prefer 1-2 useful experience edits plus summary/skill emphasis, not every entry. Before returning, compare each selection with its source: an identical AFTER, an existing-summary-only selection and an identity skill permutation do nothing. When relevant supported work can be brought forward, propose that conservative reorder instead of an unchanged or empty plan. Return an empty plan only when no useful supported emphasis is possible.
All three top-level fields are required. experiences must be an array, never null; summary and skill_order may be null. Every selection uses 1-6 unique fact_ids. after must be nonempty and <=10000 Unicode characters; reasons must be nonempty and <=1000. Do not output null strings or arrays.
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
		return nil, invalidTailoringResponse("transport", "incomplete", "$")
	}
	text := extractTextFromResponse(response.Content)
	if !strings.HasPrefix(strings.TrimSpace(text), "{") || len(text) > 200000 {
		return nil, invalidTailoringResponse("decode", "size_or_object", "$")
	}
	if err := validateTailoringPlanJSON([]byte(text)); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.DisallowUnknownFields()
	var plan tailoringPlan
	if err := decoder.Decode(&plan); err != nil {
		return nil, invalidTailoringResponse("decode", "schema", "$")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, invalidTailoringResponse("decode", "trailing_json", "$")
	}
	result, err := resolveTailoringPlan(catalog, plan)
	if err != nil {
		return nil, err
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
