package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

func invalidTailoringResponse(stage, rule, path string) error {
	return InvalidResponse(ErrInvalidTailoringResponse, "tailoring_"+stage, rule, path)
}

func validateTailoringPlanJSON(raw []byte) error {
	if !utf8.Valid(raw) {
		return invalidTailoringResponse("decode", "utf8", "$")
	}
	if !json.Valid(raw) {
		return invalidTailoringResponse("decode", "json", "$")
	}
	root, err := tailoringObject(raw, "$", []string{"summary", "experiences", "skill_order"})
	if err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimSpace(root["summary"]), []byte("null")) {
		selection, err := tailoringObject(root["summary"], "$.summary", []string{"fact_ids", "reason"})
		if err != nil {
			return err
		}
		for _, field := range []string{"after", "experience_id"} {
			if _, present := selection[field]; present {
				return invalidTailoringResponse("decode", "unexpected_field", "$.summary")
			}
		}
		if err := tailoringIDs(selection["fact_ids"], "$.summary.fact_ids", 6); err != nil {
			return err
		}
		if err := tailoringString(selection["reason"], "$.summary.reason", 1000); err != nil {
			return err
		}
	}
	if bytes.Equal(bytes.TrimSpace(root["experiences"]), []byte("null")) {
		return invalidTailoringResponse("decode", "null", "$.experiences")
	}
	var experiences []json.RawMessage
	if json.Unmarshal(root["experiences"], &experiences) != nil {
		return invalidTailoringResponse("decode", "type", "$.experiences")
	}
	for i, raw := range experiences {
		path := fmt.Sprintf("$.experiences[%d]", i)
		selection, err := tailoringObject(raw, path, []string{"experience_id", "fact_ids", "after"})
		if err != nil {
			return err
		}
		if err := tailoringString(selection["experience_id"], path+".experience_id", 64); err != nil {
			return err
		}
		if err := tailoringIDs(selection["fact_ids"], path+".fact_ids", 6); err != nil {
			return err
		}
		if err := tailoringString(selection["after"], path+".after", 10000); err != nil {
			return err
		}
		if reason, ok := selection["reason"]; ok {
			if err := tailoringString(reason, path+".reason", 1000); err != nil {
				return err
			}
		}
	}
	if !bytes.Equal(bytes.TrimSpace(root["skill_order"]), []byte("null")) {
		order, err := tailoringObject(root["skill_order"], "$.skill_order", []string{"skill_ids", "reason"})
		if err != nil {
			return err
		}
		if err := tailoringIDs(order["skill_ids"], "$.skill_order.skill_ids", 0); err != nil {
			return err
		}
		if err := tailoringString(order["reason"], "$.skill_order.reason", 1000); err != nil {
			return err
		}
	}
	return nil
}

func tailoringObject(raw []byte, path string, required []string) (map[string]json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, invalidTailoringResponse("decode", "null", path)
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, invalidTailoringResponse("decode", "type", path)
	}
	for _, field := range required {
		if _, ok := object[field]; !ok {
			return nil, invalidTailoringResponse("decode", "required", path+"."+field)
		}
	}
	return object, nil
}

func tailoringString(raw []byte, path string, max int) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return invalidTailoringResponse("decode", "null", path)
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return invalidTailoringResponse("decode", "type", path)
	}
	if strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > max || strings.ContainsRune(value, '\x00') {
		return invalidTailoringResponse("decode", "text", path)
	}
	return nil
}

func tailoringIDs(raw []byte, path string, max int) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return invalidTailoringResponse("decode", "null", path)
	}
	var ids []json.RawMessage
	if json.Unmarshal(raw, &ids) != nil {
		return invalidTailoringResponse("decode", "type", path)
	}
	if len(ids) == 0 || (max > 0 && len(ids) > max) {
		return invalidTailoringResponse("decode", "count", path)
	}
	for i, id := range ids {
		if err := tailoringString(id, fmt.Sprintf("%s[%d]", path, i), 128); err != nil {
			return err
		}
	}
	return nil
}
