package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/stretchr/testify/require"
)

type activityRepositoryFunc func(context.Context, string, int) ([]*model.ActivityDTO, error)

func (f activityRepositoryFunc) ListActivity(ctx context.Context, userID string, limit int) ([]*model.ActivityDTO, error) {
	return f(ctx, userID, limit)
}

func TestListActivityLimitsAndIsolation(t *testing.T) {
	for _, limit := range []int{-1, 0, 1, 20, 100, 101} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			called := false
			repo := activityRepositoryFunc(func(_ context.Context, userID string, gotLimit int) ([]*model.ActivityDTO, error) {
				called = true
				require.Equal(t, "owner", userID)
				require.Equal(t, limit, gotLimit)
				return []*model.ActivityDTO{{ID: "owned"}}, nil
			})
			svc := NewAPITokenService(nil, nil, nil, nil, WithActivityRepository(repo))
			items, err := svc.ListActivity(t.Context(), "owner", limit)
			if limit < 1 || limit > 100 {
				require.ErrorIs(t, err, model.ErrInvalidActivityLimit)
				require.False(t, called)
			} else {
				require.NoError(t, err, "activity needs no paid-plan checker")
				require.True(t, called)
				require.Equal(t, "owned", items[0].ID)
			}
		})
	}
}

func TestListActivityFailure(t *testing.T) {
	failure := errors.New("database failure")
	repo := activityRepositoryFunc(func(context.Context, string, int) ([]*model.ActivityDTO, error) { return nil, failure })
	svc := NewAPITokenService(nil, nil, nil, nil, WithActivityRepository(repo))
	_, err := svc.ListActivity(t.Context(), "owner", 20)
	require.ErrorIs(t, err, failure)
	_, err = NewAPITokenService(nil, nil, nil, nil).ListActivity(t.Context(), "owner", 20)
	require.ErrorContains(t, err, "repository unavailable")
}
