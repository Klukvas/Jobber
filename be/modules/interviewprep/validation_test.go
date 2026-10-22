package interviewprep

import (
	"encoding/json"
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
