package interviewprep

import (
	"context"
	"encoding/json"
	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func syntheticPlan(facts []Fact) Plan {
	evidence := []Evidence{}
	if len(facts) > 0 {
		evidence = []Evidence{{FactID: facts[0].ID, Quote: facts[0].Text}}
	}
	questions := []Question{}
	for i, kind := range []string{"technical", "behavioral", "situational"} {
		q := Question{ID: []string{"q1", "q2", "q3"}[i], Kind: kind, Question: "How would you approach this work?", Purpose: "Explain your reasoning", Difficulty: "medium", Advice: "If true, describe your real experience; otherwise clarify the missing facts.", Evidence: evidence}
		if kind == "behavioral" {
			q.STAR = &STAR{Situation: "Describe a real situation", Task: "What was your task?", Action: "What did you do?", Result: "What happened?"}
		}
		questions = append(questions, q)
	}
	return Plan{Requirements: []Requirement{{Requirement: "Go", Assessment: "Clarify your real experience", Evidence: evidence}}, Questions: questions, Gaps: []Gap{{Priority: "high", Gap: "Prepare examples", Action: "Write one accurate example"}}, InterviewerQuestions: []string{"How does the team review work?"}}
}
func syntheticReview(facts []Fact, followup bool) Review {
	score := 2
	evidence := []Evidence{}
	answerFacts := []Fact{}
	for _, f := range facts {
		if strings.HasPrefix(f.ID, "answer:") {
			answerFacts = append(answerFacts, f)
		}
	}
	if len(answerFacts) > 0 {
		quote := []rune(answerFacts[0].Text)
		if len(quote) > 1000 {
			quote = quote[:1000]
		}
		evidence = []Evidence{{FactID: answerFacts[0].ID, Quote: string(quote)}}
	} else {
		score = 0
	}
	r := Review{Criteria: []Criterion{}, Strengths: []string{}, Improvements: []string{"Add specific real details"}, Revisit: []string{"Practice a concrete example"}}
	for _, name := range []string{"content", "specificity", "structure"} {
		r.Criteria = append(r.Criteria, Criterion{Name: name, Score: &score, Explanation: "The supplied answer gives partial evidence; explain your own contribution.", Evidence: evidence})
	}
	if followup {
		r.Followup = &Followup{Question: "What did you personally change?", Purpose: "Clarify individual contribution"}
	}
	return r
}
func TestValidatePlan(t *testing.T) {
	facts := []Fact{{ID: "skills:known", Section: "skills", Text: "Go"}}
	valid, _ := json.Marshal(syntheticPlan(facts))
	cases := []struct {
		name   string
		change func() []byte
		want   bool
	}{
		{"valid", func() []byte { return valid }, true},
		{"unknown field", func() []byte {
			return []byte(strings.Replace(string(valid), `"requirements":`, `"extra":true,"requirements":`, 1))
		}, false},
		{"trailing json", func() []byte { return append(append([]byte(nil), valid...), []byte(` {}`)...) }, false},
		{"unknown evidence", func() []byte { return []byte(strings.ReplaceAll(string(valid), "skills:known", "hidden:id")) }, false},
		{"fabricated quote", func() []byte {
			return []byte(strings.ReplaceAll(string(valid), `"quote":"Go"`, `"quote":"I led 100 engineers"`))
		}, false},
		{"invalid utf8", func() []byte { return []byte(strings.Replace(string(valid), "How", string([]byte{255}), 1)) }, false},
		{"missing required kind", func() []byte { return []byte(strings.ReplaceAll(string(valid), "situational", "technical")) }, false},
		{"missing star", func() []byte { p := syntheticPlan(facts); p.Questions[1].STAR = nil; b, _ := json.Marshal(p); return b }, false},
		{"oversized advice", func() []byte {
			p := syntheticPlan(facts)
			p.Questions[0].Advice = strings.Repeat("x", 2001)
			b, _ := json.Marshal(p)
			return b
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { _, err := ValidatePlan(tc.change(), facts); require.Equal(t, tc.want, err == nil) })
	}
}
func TestValidateReview(t *testing.T) {
	fact := []Fact{{ID: "answer:known", Text: "I investigated latency"}}
	valid := syntheticReview(fact, true)
	for _, tc := range []struct {
		name   string
		mutate func(*Review)
		allow  bool
		want   bool
	}{
		{"valid", func(*Review) {}, true, true},
		{"followup recursion", func(*Review) {}, false, false},
		{"missing score", func(r *Review) { r.Criteria[0].Score = nil }, true, false},
		{"out of range", func(r *Review) { n := 5; r.Criteria[0].Score = &n }, true, false},
		{"duplicate criterion", func(r *Review) { r.Criteria[0].Name = "structure" }, true, false},
		{"bad quote", func(r *Review) { r.Criteria[0].Evidence = []Evidence{{FactID: "answer:known", Quote: "invented"}} }, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(valid)
			var r Review
			require.NoError(t, json.Unmarshal(b, &r))
			tc.mutate(&r)
			b, _ = json.Marshal(r)
			_, err := ValidateReview(b, fact, tc.allow)
			require.Equal(t, tc.want, err == nil)
		})
	}
}

func TestValidatePlanFakeProviderDiagnostics(t *testing.T) {
	facts := []Fact{{ID: "skills:known", Section: "skills", Text: "Go"}}
	valid, err := json.Marshal(syntheticPlan(facts))
	require.NoError(t, err)
	for _, tc := range []struct {
		name, old, replacement, stage, rule, path string
		truncated                                 bool
	}{
		{name: "success"},
		{name: "malformed", old: string(valid), replacement: `{"requirements":`, stage: "interview_decode", rule: "json", path: "$"},
		{name: "missing", old: `"requirements":`, replacement: `"ignored":`, stage: "interview_decode", rule: "required", path: "$.requirements"},
		{name: "null array", old: `"evidence":[{"fact_id":"skills:known","quote":"Go"}]`, replacement: `"evidence":null`, stage: "interview_decode", rule: "null", path: "$.requirements[0].evidence"},
		{name: "null text", old: `"requirement":"Go"`, replacement: `"requirement":null`, stage: "interview_decode", rule: "null", path: "$.requirements[0].requirement"},
		{name: "wrong type", old: `"requirement":"Go"`, replacement: `"requirement":42`, stage: "interview_decode", rule: "type", path: "$.requirements[0].requirement"},
		{name: "unknown evidence", old: `"fact_id":"skills:known"`, replacement: `"fact_id":"foreign"`, stage: "interview_plan", rule: "unknown_id", path: "$.requirements[0].evidence[0].fact_id"},
		{name: "inexact quote", old: `"quote":"Go"`, replacement: `"quote":"Rust"`, stage: "interview_plan", rule: "exact_quote", path: "$.requirements[0].evidence[0].quote"},
		{name: "enum", old: `"difficulty":"medium"`, replacement: `"difficulty":"extreme"`, stage: "interview_plan", rule: "enum", path: "$.questions[0].difficulty"},
		{name: "truncated", truncated: true, stage: "interview_transport", rule: "incomplete", path: "$"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := string(valid)
			if tc.old != "" {
				body = strings.Replace(body, tc.old, tc.replacement, 1)
			}
			client := ai.NewTestClient(func(context.Context, anthropic.MessageNewParams) (*anthropic.Message, error) {
				stop := anthropic.StopReasonEndTurn
				if tc.truncated {
					stop = anthropic.StopReasonMaxTokens
				}
				return &anthropic.Message{StopReason: stop, Content: []anthropic.ContentBlockUnion{{Type: "text", Text: body}}}, nil
			})
			raw, err := client.Interview(t.Context(), "plan", []byte(`{"snapshot":{"locale":"en"}}`), nil)
			if err == nil {
				_, err = ValidatePlan(raw, facts)
			}
			if tc.stage == "" {
				require.NoError(t, err)
				return
			}
			var diagnostic *ai.ResponseValidationError
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.stage, diagnostic.Stage)
			require.Equal(t, tc.rule, diagnostic.Rule)
			require.Equal(t, tc.path, diagnostic.Path)
			if !tc.truncated {
				require.ErrorIs(t, err, ErrInvalidAI)
			}
			require.NotContains(t, err.Error(), "skills:known")
		})
	}
}

func TestValidatePlanDiagnosticLimits(t *testing.T) {
	for _, tc := range []struct {
		name, rule, path string
		mutate           func(*Plan)
	}{
		{name: "evidence count", rule: "count", path: "$.requirements[0].evidence", mutate: func(p *Plan) { p.Requirements[0].Evidence = make([]Evidence, 9) }},
		{name: "advice length", rule: "text", path: "$.questions[0].advice", mutate: func(p *Plan) { p.Questions[0].Advice = strings.Repeat("я", 2001) }},
		{name: "duplicate ID", rule: "unique_id", path: "$.questions[1].id", mutate: func(p *Plan) { p.Questions[1].ID = p.Questions[0].ID }},
		{name: "STAR required", rule: "required", path: "$.questions[1].star", mutate: func(p *Plan) { p.Questions[1].STAR = nil }},
		{name: "required kinds", rule: "required_kinds", path: "$.questions", mutate: func(p *Plan) { p.Questions[2].Kind = "technical" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := syntheticPlan(nil)
			tc.mutate(&p)
			raw, err := json.Marshal(p)
			require.NoError(t, err)
			_, err = ValidatePlan(raw, nil)
			require.ErrorIs(t, err, ErrInvalidAI)
			var diagnostic *ai.ResponseValidationError
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.rule, diagnostic.Rule)
			require.Equal(t, tc.path, diagnostic.Path)
		})
	}
}
