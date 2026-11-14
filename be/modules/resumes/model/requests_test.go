package model

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// An over-long title used to pass binding and fail at the VARCHAR(255) column,
// surfacing as a 500 instead of a validation error.
func TestUpdateResumeRequestTitleLength(t *testing.T) {
	validate := validator.New()
	validate.SetTagName("binding")
	atLimit := strings.Repeat("я", 255)
	overLimit := strings.Repeat("a", 256)

	assert.NoError(t, validate.Struct(UpdateResumeRequest{Title: &atLimit}))
	assert.NoError(t, validate.Struct(UpdateResumeRequest{}))
	assert.Error(t, validate.Struct(UpdateResumeRequest{Title: &overLimit}))
}
