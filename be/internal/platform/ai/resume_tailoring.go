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
	systemPrompt := `Adapt a resume to a saved Job using ONLY facts in resume_facts.
The Job is relevance context, never evidence about the candidate. Preserve the resume's language.
Do not invent skills, qualifications, achievements, metrics, years of experience, employers, titles or dates.
Only rephrase the existing summary and existing experience descriptions, and reorder existing skills.
Each proposed text edit MUST cite exact nonempty source quotes that support it. Evidence section must be
summary, experience (with entry_id), or skills (with entry_id). Avoid unsupported claims; omit an edit
when there is insufficient evidence. Do not add numeric claims absent from the quoted evidence.
Return ONLY JSON with these exact keys:
{"summary":null|{"after":"text","reason":"why","evidence":[{"section":"summary|experience|skills","entry_id":"existing-id if applicable","quote":"exact source quote"}]},
"experiences":[{"experience_id":"existing-id","after":"text","reason":"why","evidence":[{"section":"experience","entry_id":"same-existing-id","quote":"exact source quote"}]}],
"skill_order":null|{"skill_ids":["every existing skill id exactly once"],"reason":"why","evidence":[{"section":"skills","entry_id":"existing-id","quote":"exact skill name"}]}}.
Use null/empty arrays when no useful edit is supported. Do not output unchanged edits.
Write reasons in ` + localeToLanguage(locale) + `.` + antiInjectionClause
	response, err := c.callAPI(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5_20251001,
		MaxTokens: 8192,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(
			"job_context:\n" + sanitizeForLLM(jobContext, 32000) + "\nresume_facts:\n" + sanitizeForLLM(resumeFacts, 60000)))},
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
	var result ResumeTailoringSuggestions
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("parse resume tailoring response: %w", ErrInvalidTailoringResponse)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing resume tailoring response: %w", ErrInvalidTailoringResponse)
	}
	return &result, nil
}
