package mcp

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

func canonicalID(value, field string) (string, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return "", validationError(field + " must be a valid UUID.")
	}
	return id.String(), nil
}

func validateText(value, field string, max int, required bool) error {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return validationError(field + " is too long or contains invalid text.")
	}
	if required && strings.TrimSpace(value) == "" {
		return validationError(field + " is required.")
	}
	return nil
}

func validateCreate(input CreateJobInput) error {
	for _, field := range []struct {
		value, name string
		max         int
		required    bool
	}{
		{input.Title, "title", 255, true}, {input.Company, "company", 255, false},
		{input.URL, "url", 2048, false}, {input.Description, "description", 20000, false},
	} {
		if err := validateText(field.value, field.name, field.max, field.required); err != nil {
			return err
		}
	}
	if input.URL != "" {
		parsed, err := url.Parse(input.URL)
		if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
			return validationError("url must be an absolute HTTP or HTTPS URL without credentials.")
		}
	}
	return nil
}

func optionalText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
