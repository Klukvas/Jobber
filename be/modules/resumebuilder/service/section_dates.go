package service

import (
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// Empty dates are optional. Month precision remains supported for existing
// resumes; the editor sends full calendar dates.
func validateSectionDates(start, end string) error {
	parse := func(value string) (time.Time, error) {
		if value == "" {
			return time.Time{}, nil
		}
		layout := "2006-01-02"
		if len(value) == 7 {
			layout = "2006-01"
		}
		date, err := time.Parse(layout, value)
		if err != nil || date.Year() < 1 {
			return time.Time{}, model.ErrInvalidDateRange
		}
		return date, nil
	}
	first, err := parse(start)
	if err != nil {
		return err
	}
	last, err := parse(end)
	if err != nil {
		return err
	}
	if start != "" && end != "" && last.Before(first) {
		return model.ErrInvalidDateRange
	}
	return nil
}
