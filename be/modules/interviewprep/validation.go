package interviewprep

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

func strictJSON(raw []byte, target any) error {
	if !utf8.Valid(raw) || len(raw) == 0 || len(raw) > 180000 {
		return ErrInvalidAI
	}
	if !requiredFields(raw, reflect.TypeOf(target).Elem()) {
		return ErrInvalidAI
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidAI
	}
	if decoder.Decode(new(any)) != io.EOF {
		return ErrInvalidAI
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
	if len(p.Requirements) < 1 || len(p.Requirements) > 15 || len(p.Questions) < 3 || len(p.Questions) > 12 || p.Gaps == nil || len(p.Gaps) > 12 || len(p.InterviewerQuestions) < 1 || len(p.InterviewerQuestions) > 8 {
		return p, ErrInvalidAI
	}
	for _, r := range p.Requirements {
		if !validText(r.Requirement, 1000, true) || !validText(r.Assessment, 1500, true) || !evidenceValid(r.Evidence, facts) {
			return p, ErrInvalidAI
		}
	}
	ids := map[string]bool{}
	kinds := map[string]bool{}
	for _, q := range p.Questions {
		if !validText(q.ID, 64, true) || ids[q.ID] || strings.HasPrefix(q.ID, "followup:") || !validText(q.Question, 1500, true) || !validText(q.Purpose, 1000, true) || !validText(q.Advice, 2000, true) || !evidenceValid(q.Evidence, facts) {
			return p, ErrInvalidAI
		}
		ids[q.ID] = true
		kinds[q.Kind] = true
		if q.Kind != "technical" && q.Kind != "behavioral" && q.Kind != "situational" {
			return p, ErrInvalidAI
		}
		if q.Difficulty != "easy" && q.Difficulty != "medium" && q.Difficulty != "hard" {
			return p, ErrInvalidAI
		}
		if q.Kind == "behavioral" && q.STAR == nil {
			return p, ErrInvalidAI
		}
		if q.STAR != nil {
			if !validText(q.STAR.Situation, 1000, true) || !validText(q.STAR.Task, 1000, true) || !validText(q.STAR.Action, 1000, true) || !validText(q.STAR.Result, 1000, true) {
				return p, ErrInvalidAI
			}
		}
	}
	if !kinds["technical"] || !kinds["behavioral"] || !kinds["situational"] {
		return p, ErrInvalidAI
	}
	for _, g := range p.Gaps {
		if (g.Priority != "high" && g.Priority != "medium" && g.Priority != "low") || !validText(g.Gap, 1000, true) || !validText(g.Action, 1500, true) {
			return p, ErrInvalidAI
		}
	}
	for _, q := range p.InterviewerQuestions {
		if !validText(q, 1000, true) {
			return p, ErrInvalidAI
		}
	}
	contexts := compactInput(Snapshot{}, p.Questions, nil).Questions
	contextRaw, _ := json.Marshal(contexts)
	if len(contextRaw) > maxQuestionContextBytes {
		return Plan{}, ErrInvalidAI
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

func requiredFields(raw []byte, kind reflect.Type) bool {
	if kind.Kind() == reflect.Pointer {
		if string(raw) == "null" {
			return true
		}
		return requiredFields(raw, kind.Elem())
	}
	switch kind.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || fields == nil {
			return false
		}
		for i := 0; i < kind.NumField(); i++ {
			field := kind.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			child, ok := fields[name]
			if !ok || !requiredFields(child, field.Type) {
				return false
			}
		}
	case reflect.Slice:
		var children []json.RawMessage
		if json.Unmarshal(raw, &children) != nil {
			return false
		}
		for _, child := range children {
			if !requiredFields(child, kind.Elem()) {
				return false
			}
		}
	}
	return true
}
