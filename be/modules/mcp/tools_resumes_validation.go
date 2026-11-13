package mcp

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin/binding"
)

func validateResumeOperation(input UpdateResumeSectionInput) (string, error) {
	switch input.Section {
	case "contact", "summary":
		if input.Op != "upsert" {
			return "", validationError("Contact and summary require op upsert.")
		}
	case "experience", "education", "skill", "language", "certification", "project", "volunteering", "custom":
		if input.Op != "create" && input.Op != "update" {
			return "", validationError("List sections require op create or update.")
		}
	default:
		return "", validationError("Choose a supported Builder Resume section.")
	}
	if input.Op == "update" {
		return canonicalID(input.EntryID, "entry_id")
	}
	if input.EntryID != "" {
		return "", validationError("entry_id is allowed only for update.")
	}
	return "", nil
}

// Reflect only at the JSON boundary to reuse the existing models' exact field names
// and binding rules, while excluding the REST API's ordering capability.
func decodeResumeData[T any](data map[string]any) (*T, error) {
	if len(data) == 0 {
		return nil, validationError("data must contain section fields.")
	}
	fields := make(map[string]bool)
	schema := reflect.TypeFor[T]()
	for i := 0; i < schema.NumField(); i++ {
		name, _, _ := strings.Cut(schema.Field(i).Tag.Get("json"), ",")
		if name != "sort_order" {
			fields[name] = true
		}
	}
	for name, value := range data {
		if !fields[name] {
			return nil, validationError("data contains an unsupported field; IDs and reordering are not allowed.")
		}
		if value == nil {
			return nil, validationError("data fields cannot be null; use an empty string to clear text.")
		}
		if text, ok := value.(string); ok && !validResumeText(text) {
			return nil, validationError("data contains invalid text or control characters.")
		}
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, validationError("data must be a valid section object.")
	}
	request := new(T)
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		return nil, validationError("data fields must have the correct types for this section.")
	}
	if err := binding.Validator.ValidateStruct(request); err != nil {
		return nil, validationError("A section field exceeds its Resume Builder length limit.")
	}
	return request, nil
}

func validResumeText(text string) bool {
	if !utf8.ValidString(text) {
		return false
	}
	for _, char := range text {
		if unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' {
			return false
		}
	}
	return true
}

func resumeSectionInputSchema() map[string]any {
	// Derive descriptions from the input contract; SDK v1.8's jsonschema tags
	// provide descriptions only, so enums must be set explicitly.
	properties := make(map[string]any)
	schema := reflect.TypeFor[UpdateResumeSectionInput]()
	required := make([]string, 0, schema.NumField())
	for i := 0; i < schema.NumField(); i++ {
		field := schema.Field(i)
		name, options, _ := strings.Cut(field.Tag.Get("json"), ",")
		kind := "string"
		if name == "data" {
			kind = "object"
		}
		properties[name] = map[string]any{"type": kind, "description": field.Tag.Get("jsonschema")}
		if options != "omitempty" {
			required = append(required, name)
		}
	}
	properties["section"].(map[string]any)["enum"] = []string{
		"contact", "summary", "experience", "education", "skill", "language",
		"certification", "project", "volunteering", "custom",
	}
	properties["op"].(map[string]any)["enum"] = []string{"upsert", "create", "update"}
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}
