package interviewprep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

func strictJSON(raw []byte, target any) error {
	if !utf8.Valid(raw) {
		return invalidInterview("decode", "utf8", "$")
	}
	if len(raw) == 0 || len(raw) > 180000 {
		return invalidInterview("decode", "size", "$")
	}
	if !json.Valid(raw) {
		return invalidInterview("decode", "json", "$")
	}
	if err := requiredFieldDiagnostic(raw, reflect.TypeOf(target).Elem(), "$"); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalidInterview("decode", "schema", "$")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return invalidInterview("decode", "trailing_json", "$")
	}
	return nil
}
func evidenceValid(items []Evidence, facts []Fact) bool {
	if items == nil || len(items) > 8 {
		return false
	}
	for _, e := range items {
		if !validText(e.Quote, 1000, true) {
			return false
		}
		found := false
		for _, fact := range facts {
			if fact.ID == e.FactID && strings.Contains(fact.Text, e.Quote) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func ValidatePlan(raw []byte, facts []Fact) (Plan, error) {
	var p Plan
	if err := strictJSON(raw, &p); err != nil {
		return p, err
	}
	for _, list := range []struct {
		path            string
		count, min, max int
	}{
		{"$.requirements", len(p.Requirements), 1, 15}, {"$.questions", len(p.Questions), 3, 12},
		{"$.gaps", len(p.Gaps), 0, 12}, {"$.interviewer_questions", len(p.InterviewerQuestions), 1, 8},
	} {
		if list.count < list.min || list.count > list.max {
			return p, invalidInterview("plan", "count", list.path)
		}
	}
	for i, r := range p.Requirements {
		path := fmt.Sprintf("$.requirements[%d]", i)
		if err := interviewTexts(path, []boundedInterviewText{{"requirement", r.Requirement, 1000}, {"assessment", r.Assessment, 1500}}); err != nil {
			return p, err
		}
		if err := interviewEvidence(r.Evidence, facts, path+".evidence"); err != nil {
			return p, err
		}
	}
	ids := map[string]bool{}
	kinds := map[string]bool{}
	for i, q := range p.Questions {
		path := fmt.Sprintf("$.questions[%d]", i)
		if err := interviewTexts(path, []boundedInterviewText{{"id", q.ID, 64}, {"question", q.Question, 1500}, {"purpose", q.Purpose, 1000}, {"advice", q.Advice, 2000}}); err != nil {
			return p, err
		}
		if ids[q.ID] || strings.HasPrefix(q.ID, "followup:") {
			return p, invalidInterview("plan", "unique_id", path+".id")
		}
		ids[q.ID] = true
		if q.Kind != "technical" && q.Kind != "behavioral" && q.Kind != "situational" {
			return p, invalidInterview("plan", "enum", path+".kind")
		}
		kinds[q.Kind] = true
		if q.Difficulty != "easy" && q.Difficulty != "medium" && q.Difficulty != "hard" {
			return p, invalidInterview("plan", "enum", path+".difficulty")
		}
		if err := interviewEvidence(q.Evidence, facts, path+".evidence"); err != nil {
			return p, err
		}
		if q.Kind == "behavioral" && q.STAR == nil {
			return p, invalidInterview("plan", "required", path+".star")
		}
		if q.STAR != nil {
			if err := interviewTexts(path+".star", []boundedInterviewText{{"situation", q.STAR.Situation, 1000}, {"task", q.STAR.Task, 1000}, {"action", q.STAR.Action, 1000}, {"result", q.STAR.Result, 1000}}); err != nil {
				return p, err
			}
		}
	}
	if !kinds["technical"] || !kinds["behavioral"] || !kinds["situational"] {
		return p, invalidInterview("plan", "required_kinds", "$.questions")
	}
	for i, g := range p.Gaps {
		path := fmt.Sprintf("$.gaps[%d]", i)
		if g.Priority != "high" && g.Priority != "medium" && g.Priority != "low" {
			return p, invalidInterview("plan", "enum", path+".priority")
		}
		if err := interviewTexts(path, []boundedInterviewText{{"gap", g.Gap, 1000}, {"action", g.Action, 1500}}); err != nil {
			return p, err
		}
	}
	for i, q := range p.InterviewerQuestions {
		if !validText(q, 1000, true) {
			return p, invalidInterview("plan", "text", fmt.Sprintf("$.interviewer_questions[%d]", i))
		}
	}
	contexts := compactInput(Snapshot{}, p.Questions, nil).Questions
	contextRaw, err := json.Marshal(contexts)
	if err != nil || len(contextRaw) > maxQuestionContextBytes {
		return Plan{}, invalidInterview("plan", "context_bytes", "$.questions")
	}
	return p, nil
}
func ValidateReview(raw []byte, facts []Fact, allowFollowup bool) (Review, error) {
	var r Review
	if err := strictJSON(raw, &r); err != nil {
		return r, err
	}
	if len(r.Criteria) != 3 {
		return r, ErrInvalidAI
	}
	names := map[string]bool{}
	for _, c := range r.Criteria {
		if names[c.Name] || (c.Name != "content" && c.Name != "specificity" && c.Name != "structure") || c.Score == nil || *c.Score < 0 || *c.Score > 4 || !validText(c.Explanation, 2000, true) || !evidenceValid(c.Evidence, facts) {
			return r, ErrInvalidAI
		}
		if *c.Score > 0 {
			answerEvidence := false
			for _, e := range c.Evidence {
				if strings.HasPrefix(e.FactID, "answer:") {
					answerEvidence = true
				}
			}
			if !answerEvidence {
				return r, ErrInvalidAI
			}
		}
		names[c.Name] = true
	}
	for _, list := range [][]string{r.Strengths, r.Improvements, r.Revisit} {
		if list == nil || len(list) > 10 {
			return r, ErrInvalidAI
		}
		for _, text := range list {
			if !validText(text, 1500, true) {
				return r, ErrInvalidAI
			}
		}
	}
	hasAnswer := false
	for _, fact := range facts {
		if strings.HasPrefix(fact.ID, "answer:") && strings.TrimSpace(fact.Text) != "" {
			hasAnswer = true
		}
	}
	if !hasAnswer && len(r.Strengths) > 0 {
		return r, ErrInvalidAI
	}
	if r.Followup != nil && (!allowFollowup || !validText(r.Followup.Question, 180, true) || !validText(r.Followup.Purpose, 120, true)) {
		return r, ErrInvalidAI
	}
	if r.Followup != nil {
		contextRaw, _ := json.Marshal(QuestionContext{ID: "followup:00000000-0000-4000-8000-000000000000", Kind: "followup", Question: r.Followup.Question, Purpose: r.Followup.Purpose})
		if len(contextRaw) > 2048 {
			return r, ErrInvalidAI
		}
	}
	return r, nil
}

func reviewFacts(input AIInput) []Fact {
	facts := append([]Fact(nil), input.Snapshot.Facts...)
	for _, a := range input.Answers {
		if !a.Skipped {
			facts = append(facts, Fact{ID: "answer:" + a.ID, Section: "answer", Text: a.Text})
		}
	}
	return facts
}
