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
