package service

import (
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// appliedAtAfterMove moved to job_service_applied.go together with the rest of
// the "when is a card applied to" rule; its tests live in
// job_service_applied_test.go.

func TestLockedJobState_InColumn(t *testing.T) {
	column := "c1"
	withColumn := &lockedJobState{CurrentStageTemplateID: &column}

	assert.True(t, withColumn.InColumn("c1"))
	assert.False(t, withColumn.InColumn("c2"))
	assert.False(t, (&lockedJobState{}).InColumn("c1"), "a card with no column is never already there")
}

func TestApplyStageStatusUpdate(t *testing.T) {
	now := time.Date(2026, time.May, 5, 8, 0, 0, 0, time.UTC)
	chosen := time.Date(2026, time.May, 1, 8, 0, 0, 0, time.UTC)
	status := func(s string) *string { return &s }

	t.Run("stamps completed_at when a stage is completed without one", func(t *testing.T) {
		stage := &model.JobStage{Status: "active"}

		applyStageStatusUpdate(stage, &model.UpdateStageRequest{Status: status("completed")}, now)

		assert.Equal(t, "completed", stage.Status)
		require.NotNil(t, stage.CompletedAt)
		assert.True(t, stage.CompletedAt.Equal(now))
	})

	t.Run("an explicit completed_at wins", func(t *testing.T) {
		stage := &model.JobStage{Status: "active"}

		applyStageStatusUpdate(stage, &model.UpdateStageRequest{
			Status:      status("completed"),
			CompletedAt: &chosen,
		}, now)

		require.NotNil(t, stage.CompletedAt)
		assert.True(t, stage.CompletedAt.Equal(chosen))
	})

	t.Run("going back to an open status clears completed_at", func(t *testing.T) {
		stage := &model.JobStage{Status: "completed", CompletedAt: &chosen}

		applyStageStatusUpdate(stage, &model.UpdateStageRequest{Status: status("active")}, now)

		assert.Equal(t, "active", stage.Status)
		assert.Nil(t, stage.CompletedAt)
	})

	t.Run("skipped and cancelled keep their timestamp", func(t *testing.T) {
		for _, s := range []string{"skipped", "cancelled"} {
			stage := &model.JobStage{Status: "active", CompletedAt: &chosen}

			applyStageStatusUpdate(stage, &model.UpdateStageRequest{Status: status(s)}, now)

			assert.Equal(t, s, stage.Status)
			require.NotNil(t, stage.CompletedAt, "%s is a final status", s)
		}
	})

	t.Run("a request with no status leaves the row's status alone", func(t *testing.T) {
		stage := &model.JobStage{Status: "active"}

		applyStageStatusUpdate(stage, &model.UpdateStageRequest{CompletedAt: &chosen}, now)

		assert.Equal(t, "active", stage.Status)
		require.NotNil(t, stage.CompletedAt)
	})
}
