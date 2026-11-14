package model

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func validateBindingTags(t *testing.T, value any) error {
	t.Helper()

	validate := validator.New()
	validate.SetTagName("binding")
	return validate.Struct(value)
}

// Over-long values used to pass binding and fail at the VARCHAR(255) columns,
// surfacing as a 500 instead of a validation error.
func TestCompanyRequestLengthLimits(t *testing.T) {
	atLimit := strings.Repeat("я", 255)
	overLimit := strings.Repeat("a", 256)

	tests := []struct {
		name    string
		request any
		wantErr bool
	}{
		{"create location at limit", CreateCompanyRequest{Name: "Acme", Location: &atLimit}, false},
		{"create location over limit", CreateCompanyRequest{Name: "Acme", Location: &overLimit}, true},
		{"update name at limit", UpdateCompanyRequest{Name: &atLimit}, false},
		{"update name over limit", UpdateCompanyRequest{Name: &overLimit}, true},
		{"update location over limit", UpdateCompanyRequest{Location: &overLimit}, true},
		{"update with nothing set", UpdateCompanyRequest{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBindingTags(t, tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
