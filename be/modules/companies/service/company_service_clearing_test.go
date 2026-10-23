package service

import (
	"context"
	"encoding/json"
	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompanyPatchExplicitClearAndOmission(t *testing.T) {
	for _, tc := range []struct{ name, body, location, notes string }{
		{"clear both", `{"location":"","notes":""}`, "", ""},
		{"omit notes", `{"location":""}`, "", "Keep note"},
		{"omit location", `{"notes":""}`, "Keep location", ""},
		{"omit both", `{"name":"Renamed"}`, "Keep location", "Keep note"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			location, notes := "Keep location", "Keep note"
			stored := &model.Company{ID: "company", UserID: "owner", Name: "Synthetic", Location: &location, Notes: &notes}
			repo := &MockCompanyRepository{
				GetByIDFunc:         func(context.Context, string, string) (*model.Company, error) { return stored, nil },
				UpdateFunc:          func(_ context.Context, company *model.Company) error { stored = company; return nil },
				GetByIDEnrichedFunc: func(context.Context, string, string) (*model.CompanyDTO, error) { return &model.CompanyDTO{}, nil },
			}
			var request model.UpdateCompanyRequest
			require.NoError(t, json.Unmarshal([]byte(tc.body), &request))
			_, err := NewCompanyService(repo).Update(context.Background(), "owner", "company", &request)
			require.NoError(t, err)
			require.NotNil(t, stored.Location)
			require.NotNil(t, stored.Notes)
			require.Equal(t, tc.location, *stored.Location)
			require.Equal(t, tc.notes, *stored.Notes)
		})
	}
}
