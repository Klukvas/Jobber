package interviewprep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
)

func invalidInterview(stage, rule, path string) error {
	return ai.InvalidResponse(ErrInvalidAI, "interview_"+stage, rule, path)
}

// Walk only declared fields: diagnostics never include untrusted JSON key names.
func requiredFieldDiagnostic(raw []byte, kind reflect.Type, path string) error {
	raw = bytes.TrimSpace(raw)
	if kind.Kind() == reflect.Pointer {
		if bytes.Equal(raw, []byte("null")) {
			return nil
		}
		return requiredFieldDiagnostic(raw, kind.Elem(), path)
	}
	if bytes.Equal(raw, []byte("null")) {
		return invalidInterview("decode", "null", path)
	}
	switch kind.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || fields == nil {
			return invalidInterview("decode", "type", path)
		}
		for i := 0; i < kind.NumField(); i++ {
			field := kind.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			child, ok := fields[name]
			if !ok {
				return invalidInterview("decode", "required", path+"."+name)
			}
			if err := requiredFieldDiagnostic(child, field.Type, path+"."+name); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var children []json.RawMessage
		if json.Unmarshal(raw, &children) != nil {
			return invalidInterview("decode", "type", path)
		}
		for i, child := range children {
			if err := requiredFieldDiagnostic(child, kind.Elem(), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	default:
		if json.Unmarshal(raw, reflect.New(kind).Interface()) != nil {
			return invalidInterview("decode", "type", path)
		}
	}
	return nil
}

type boundedInterviewText struct {
	field, text string
	max         int
}

func interviewTexts(path string, fields []boundedInterviewText) error {
	for _, field := range fields {
		if !validText(field.text, field.max, true) {
			return invalidInterview("plan", "text", path+"."+field.field)
		}
	}
	return nil
}

func interviewEvidence(items []Evidence, facts []Fact, path string) error {
	if items == nil || len(items) > 8 {
		return invalidInterview("plan", "count", path)
	}
	for i, item := range items {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if !validText(item.Quote, 1000, true) {
			return invalidInterview("plan", "text", itemPath+".quote")
		}
		found := false
		for _, fact := range facts {
			if fact.ID == item.FactID {
				found = true
				if !strings.Contains(fact.Text, item.Quote) {
					return invalidInterview("plan", "exact_quote", itemPath+".quote")
				}
				break
			}
		}
		if !found {
			return invalidInterview("plan", "unknown_id", itemPath+".fact_id")
		}
	}
	return nil
}
