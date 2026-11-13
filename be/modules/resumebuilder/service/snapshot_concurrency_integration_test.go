//go:build integration

package service

import (
	"context"
	"sort"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotPostgresConcurrentDedupe(t *testing.T) {
	const workers = 6
	f := newTailoringIntegrationFixture(t, workers+2)
	service := NewSnapshotService(repository.NewSnapshotRepository(f.pool))
	actions := make([]func(context.Context) error, workers)
	for i := range actions {
		actions[i] = func(ctx context.Context) error {
			return service.SnapshotBeforeWrite(ctx, f.userID, f.sourceID, "mcp")
		}
	}
	runSnapshotPostgresContenders(t, f, actions)
	items, err := service.ListSnapshots(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	require.Len(t, items, 1, "real transactions must serialize the window check and insert")
	assert.Equal(t, "mcp", items[0].Source)
}

func TestSnapshotPostgresConcurrentReversedReorder(t *testing.T) {
	for _, operation := range []string{"snapshot", "restore"} {
		t.Run(operation, func(t *testing.T) {
			f := newTailoringIntegrationFixture(t, 4)
			service := NewSnapshotService(repository.NewSnapshotRepository(f.pool))
			builder := NewResumeBuilderService(f.repo, f.limits)
			request := reversedSnapshotSectionOrder(t, f)
			snapshotID := ""
			if operation == "restore" {
				require.NoError(t, service.SnapshotBeforeWrite(t.Context(), f.userID, f.sourceID, "mcp"))
				items, err := service.ListSnapshots(t.Context(), f.userID, f.sourceID)
				require.NoError(t, err)
				snapshotID = items[0].ID
			}
			actions := []func(context.Context) error{
				func(ctx context.Context) error {
					if operation == "restore" {
						_, err := service.RestoreSnapshot(ctx, f.userID, snapshotID)
						return err
					}
					return service.SnapshotBeforeWrite(ctx, f.userID, f.sourceID, "mcp")
				},
				func(ctx context.Context) error {
					_, err := builder.UpdateSectionOrder(ctx, f.userID, f.sourceID, request)
					return err
				},
			}
			runSnapshotPostgresContenders(t, f, actions)
			orders, err := f.repo.ListSectionOrders(t.Context(), f.sourceID)
			require.NoError(t, err)
			assert.Len(t, orders, len(request.Sections))
		})
	}
}

func reversedSnapshotSectionOrder(t *testing.T, f *tailoringIntegrationFixture) *model.BatchUpdateSectionOrderRequest {
	t.Helper()
	orders, err := f.repo.ListSectionOrders(t.Context(), f.sourceID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(orders), 2)
	sort.Slice(orders, func(i, j int) bool { return orders[i].ID > orders[j].ID })
	request := &model.BatchUpdateSectionOrderRequest{Sections: make([]model.UpdateSectionOrderRequest, len(orders))}
	for i, order := range orders {
		request.Sections[i] = model.UpdateSectionOrderRequest{
			SectionKey: order.SectionKey, SortOrder: i, IsVisible: order.IsVisible, Column: order.Column,
		}
	}
	return request
}
