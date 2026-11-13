package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/stretchr/testify/require"
)

func TestJobService_ListStageFilter(t *testing.T) {
	for _, stage := range []string{"", "44444444-4444-4444-8444-444444444444"} {
		t.Run("stage="+stage, func(t *testing.T) {
			options := ports.ListOptions{StageTemplateID: stage, Limit: 1, Offset: 3, Search: "Engineer", Status: "archived"}
			calls := 0
			repo := &MockJobRepository{ListFunc: func(_ context.Context, userID string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
				calls++
				require.Equal(t, "owner", userID)
				require.Equal(t, options, *opts)
				return []*model.JobDTO{{ID: "job", CurrentStageTemplateID: &stage}}, 40, nil
			}}
			service := newServiceWithTemplates(repo, &MockStageTemplateRepository{})
			copy := options
			jobs, total, err := service.List(t.Context(), "owner", &options)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
			require.Equal(t, 40, total)
			require.Equal(t, 1, calls)
			require.Equal(t, copy, options)
		})
	}
}
