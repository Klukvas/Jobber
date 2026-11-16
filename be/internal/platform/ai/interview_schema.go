package ai

import "sort"

// The provider enforces JSON syntax and shape; domain validators independently
// enforce counts, lengths, exact source quotes, IDs and factual support.
func interviewResponseSchema(kind string) map[string]any {
	text := map[string]any{"type": "string"}
	array := func(item map[string]any) map[string]any { return map[string]any{"type": "array", "items": item} }
	object := func(properties map[string]any) map[string]any {
		required := make([]string, 0, len(properties))
		for key := range properties {
			required = append(required, key)
		}
		sort.Strings(required)
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	}
	nullable := func(value map[string]any) map[string]any {
		return map[string]any{"anyOf": []any{value, map[string]any{"type": "null"}}}
	}
	enum := func(values ...string) map[string]any { return map[string]any{"type": "string", "enum": values} }
	evidence := array(object(map[string]any{"fact_id": text, "quote": text}))
	if kind == "plan" {
		star := nullable(object(map[string]any{"situation": text, "task": text, "action": text, "result": text}))
		question := object(map[string]any{"id": text, "kind": enum("technical", "behavioral", "situational"), "question": text, "purpose": text, "difficulty": enum("easy", "medium", "hard"), "advice": text, "evidence": evidence, "star": star})
		return object(map[string]any{
			"requirements":          array(object(map[string]any{"requirement": text, "assessment": text, "evidence": evidence})),
			"questions":             array(question),
			"gaps":                  array(object(map[string]any{"priority": enum("high", "medium", "low"), "gap": text, "action": text})),
			"interviewer_questions": array(text),
		})
	}
	return object(map[string]any{
		"criteria":  array(object(map[string]any{"name": enum("content", "specificity", "structure"), "score": map[string]any{"type": "integer"}, "explanation": text, "evidence": evidence})),
		"strengths": array(text), "improvements": array(text), "revisit": array(text),
		"followup": nullable(object(map[string]any{"question": text, "purpose": text})),
	})
}
