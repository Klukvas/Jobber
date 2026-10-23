package service

import (
	"context"
	"encoding/json"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJobListNilRepositoryCollectionIsJSONArray(t *testing.T) {
	repo := &MockJobRepository{ListFunc: func(context.Context, string, *ports.ListOptions) ([]*model.JobDTO, int, error) { return nil, 0, nil }}
	svc := newServiceWithTemplates(repo, &MockStageTemplateRepository{})
	items, total, err := svc.List(context.Background(), "synthetic", &ports.ListOptions{})
	require.NoError(t, err)
	require.Zero(t, total)
	body, err := json.Marshal(items)
	require.NoError(t, err)
	require.Equal(t, "[]", string(body))
}
