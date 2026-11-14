package model

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// Over-long values used to pass binding and fail at the VARCHAR columns,
// surfacing as a 500 instead of a validation error.
func TestCoverLetterRequestLengthLimits(t *testing.T) {
	validate := validator.New()
	validate.SetTagName("binding")
	title255 := strings.Repeat("я", 255)
	long256 := strings.Repeat("a", 256)
	address500 := strings.Repeat("a", 500)
	address501 := strings.Repeat("a", 501)

	tests := []struct {
		name    string
		request any
		wantErr bool
	}{
		{"create with defaults", CreateCoverLetterRequest{}, false},
		{"create title at limit", CreateCoverLetterRequest{Title: title255}, false},
		{"create title over limit", CreateCoverLetterRequest{Title: long256}, true},
		{"update with nothing set", UpdateCoverLetterRequest{}, false},
		{"update title over limit", UpdateCoverLetterRequest{Title: &long256}, true},
		{"update company over limit", UpdateCoverLetterRequest{CompanyName: &long256}, true},
		{"update address at limit", UpdateCoverLetterRequest{CompanyAddress: &address500}, false},
		{"update address over limit", UpdateCoverLetterRequest{CompanyAddress: &address501}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
