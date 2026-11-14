package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validateBindingTags(t *testing.T, value any) error {
	t.Helper()

	validate := validator.New()
	validate.SetTagName("binding")
	return validate.Struct(value)
}

func TestCreateJobRequestAppliedAtJSON(t *testing.T) {
	t.Run("parses an RFC3339 timestamp", func(t *testing.T) {
		var req CreateJobRequest

		err := json.Unmarshal([]byte(`{"title":"Engineer","applied_at":"2025-02-03T12:30:00+02:00"}`), &req)

		require.NoError(t, err)
		require.NotNil(t, req.AppliedAt)
		assert.True(t, req.AppliedAt.Equal(time.Date(2025, time.February, 3, 10, 30, 0, 0, time.UTC)))
	})

	t.Run("rejects an invalid timestamp", func(t *testing.T) {
		var req CreateJobRequest

		err := json.Unmarshal([]byte(`{"title":"Engineer","applied_at":"not-a-timestamp"}`), &req)

		assert.Error(t, err)
	})
}

func TestUpdateJobRequestTitleValidation(t *testing.T) {
	t.Run("accepts omitted title", func(t *testing.T) {
		require.NoError(t, validateBindingTags(t, UpdateJobRequest{}))
	})

	t.Run("accepts a title up to 255 characters", func(t *testing.T) {
		title := strings.Repeat("a", 255)
		require.NoError(t, validateBindingTags(t, UpdateJobRequest{Title: &title}))
	})

	t.Run("rejects a title longer than 255 characters", func(t *testing.T) {
		title := strings.Repeat("a", 256)
		assert.Error(t, validateBindingTags(t, UpdateJobRequest{Title: &title}))
	})
}

// An over-long name used to pass binding and fail at the VARCHAR(255) column,
// surfacing as a 500 instead of a validation error.
func TestUpdateStageTemplateRequestNameLength(t *testing.T) {
	atLimit := strings.Repeat("я", 255)
	overLimit := strings.Repeat("a", 256)

	assert.NoError(t, validateBindingTags(t, UpdateStageTemplateRequest{Name: &atLimit}))
	assert.NoError(t, validateBindingTags(t, UpdateStageTemplateRequest{}))
	assert.Error(t, validateBindingTags(t, UpdateStageTemplateRequest{Name: &overLimit}))
}

func TestJobRequestURLValidation(t *testing.T) {
	str := func(s string) *string { return &s }
	for _, tc := range []struct {
		url   *string
		valid bool
	}{
		{nil, true},
		{str(""), true},
		{str("https://example.com/jobs/123"), true},
		{str("http://example.com"), true},
		{str("not-a-url"), false},
		{str("javascript:alert(1)"), false},
		{str("ftp://example.com/file"), false},
	} {
		update := validateBindingTags(t, UpdateJobRequest{URL: tc.url})
		title := "Engineer"
		create := validateBindingTags(t, CreateJobRequest{Title: title, URL: tc.url})
		label := "<nil>"
		if tc.url != nil {
			label = *tc.url
		}
		if tc.valid {
			assert.NoError(t, update, "update %q", label)
			assert.NoError(t, create, "create %q", label)
		} else {
			assert.Error(t, update, "update %q", label)
			assert.Error(t, create, "create %q", label)
		}
	}
}
