package service

import (
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateSectionDates(t *testing.T) {
	tests := []struct {
		name, start, end string
		invalid          bool
	}{
		{"empty", "", "", false},
		{"open start", "", "2024-01-01", false},
		{"open end", "2024-01-01", "", false},
		{"equal", "2024-01-01", "2024-01-01", false},
		{"legacy months", "2023-01", "2024-01", false},
		{"mixed precision", "2024-01", "2024-01-31", false},
		{"reversed", "2025-01-01", "2024-01-01", true},
		{"invalid month", "2024-13", "", true},
		{"invalid day", "2024-02-30", "", true},
		{"invalid legacy", "garbage", "", true},
		{"year zero", "0000-01-01", "", true},
		{"leap day", "2024-02-29", "2024-03-01", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSectionDates(tt.start, tt.end)
			if tt.invalid {
				require.ErrorIs(t, err, model.ErrInvalidDateRange)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
