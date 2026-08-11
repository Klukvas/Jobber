package repository

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsRepository_GetOverview(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewAnalyticsRepositoryWithPool(mock)
	userID := "user-123"

	t.Run("returns error when query fails", func(t *testing.T) {
		mock.ExpectQuery("WITH app_stats AS").
			WithArgs(userID).
			WillReturnError(assert.AnError)

		result, err := repo.GetOverview(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns overview analytics successfully", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"total_applications",
			"active_applications",
			"closed_applications",
			"rejected_applications",
			"response_rate",
			"avg_days_to_first_response",
		}).AddRow(10, 5, 5, 3, 50.0, 3.5)

		mock.ExpectQuery("WITH app_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetOverview(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, 10, result.TotalApplications)
		assert.Equal(t, 5, result.ActiveApplications)
		assert.Equal(t, 5, result.ClosedApplications)
		assert.Equal(t, 3, result.RejectedApplications)
		assert.Equal(t, 50.0, result.ResponseRate)
		assert.Equal(t, 3.5, result.AvgDaysToFirstResponse)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns zero values for empty data", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"total_applications",
			"active_applications",
			"closed_applications",
			"rejected_applications",
			"response_rate",
			"avg_days_to_first_response",
		}).AddRow(0, 0, 0, 0, 0.0, 0.0)

		mock.ExpectQuery("WITH app_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetOverview(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalApplications)
		assert.Equal(t, 0, result.ActiveApplications)
		assert.Equal(t, 0.0, result.ResponseRate)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAnalyticsRepository_GetFunnel(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewAnalyticsRepositoryWithPool(mock)
	userID := "user-123"

	t.Run("returns error when the phase query fails", func(t *testing.T) {
		mock.ExpectQuery("WITH applied_jobs AS").
			WithArgs(userID).
			WillReturnError(assert.AnError)

		result, err := repo.GetFunnel(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("builds phase funnel with an in-progress drill-down", func(t *testing.T) {
		// applied=100, in_progress=50, offer=10
		mock.ExpectQuery("WITH applied_jobs AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"applied", "in_progress", "offer"}).
				AddRow(100, 50, 10))
		// in-progress drill-down
		mock.ExpectQuery("SELECT st.name").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"name", "order", "cnt"}).
				AddRow("HR Interview", 1, 50).
				AddRow("Technical Interview", 2, 20))
		// rejected summary
		mock.ExpectQuery("WITH rejected AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"stage_name", "stage_order", "count"}).
				AddRow("Applied", 1, 3))

		result, err := repo.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result.Stages, 3)

		// Phase buckets carry the phase key for the frontend to localize.
		assert.Equal(t, "applied", result.Stages[0].StageName)
		assert.Equal(t, 100, result.Stages[0].Count)

		assert.Equal(t, "in_progress", result.Stages[1].StageName)
		assert.Equal(t, 50, result.Stages[1].Count)
		assert.Equal(t, 50.0, result.Stages[1].ConversionRate)
		assert.Equal(t, 50.0, result.Stages[1].DropOffRate)
		require.Len(t, result.Stages[1].SubStages, 2)
		assert.Equal(t, "HR Interview", result.Stages[1].SubStages[0].StageName)
		assert.Equal(t, 50, result.Stages[1].SubStages[0].Count)

		assert.Equal(t, "offer", result.Stages[2].StageName)
		assert.Equal(t, 10, result.Stages[2].Count)
		assert.Equal(t, 20.0, result.Stages[2].ConversionRate) // 10/50
		assert.Empty(t, result.Stages[2].SubStages)

		require.NotNil(t, result.Rejected)
		assert.Equal(t, 3, result.Rejected.Total)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns an empty funnel when there are no applications", func(t *testing.T) {
		// applied=0 → GetFunnel short-circuits, no drill-down/rejected queries.
		mock.ExpectQuery("WITH applied_jobs AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"applied", "in_progress", "offer"}).
				AddRow(0, 0, 0))

		result, err := repo.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Stages)
		assert.Nil(t, result.Rejected)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBuildPhaseFunnel(t *testing.T) {
	t.Run("computes conversion and drop-off between phases", func(t *testing.T) {
		stages := buildPhaseFunnel(100, 40, 10, nil)

		require.Len(t, stages, 3)
		assert.Equal(t, "applied", stages[0].StageName)
		assert.Equal(t, 100.0, stages[0].ConversionRate)

		assert.Equal(t, "in_progress", stages[1].StageName)
		assert.Equal(t, 40.0, stages[1].ConversionRate) // 40/100
		assert.Equal(t, 60.0, stages[1].DropOffRate)    // 60/100

		assert.Equal(t, "offer", stages[2].StageName)
		assert.Equal(t, 25.0, stages[2].ConversionRate) // 10/40
		assert.Equal(t, 75.0, stages[2].DropOffRate)    // 30/40
	})

	t.Run("attaches the drill-down only to the in_progress phase", func(t *testing.T) {
		sub := []model.FunnelStage{{StageName: "HR Interview", Count: 5}}
		stages := buildPhaseFunnel(10, 5, 0, sub)

		assert.Empty(t, stages[0].SubStages)
		assert.Equal(t, sub, stages[1].SubStages)
		assert.Empty(t, stages[2].SubStages)
	})

	t.Run("avoids divide-by-zero when the previous phase is empty", func(t *testing.T) {
		stages := buildPhaseFunnel(0, 0, 0, nil)
		for _, s := range stages {
			assert.Equal(t, 100.0, s.ConversionRate)
			assert.Equal(t, 0.0, s.DropOffRate)
		}
	})
}

func TestAnalyticsRepository_GetStageTime(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewAnalyticsRepositoryWithPool(mock)
	userID := "user-123"

	t.Run("returns stage time metrics successfully", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"stage_name",
			"stage_order",
			"avg_days",
			"min_days",
			"max_days",
			"applications_count",
		}).
			AddRow("Applied", 1, 2.5, 1.0, 5.0, 50).
			AddRow("Interview", 2, 7.0, 3.0, 14.0, 30)

		mock.ExpectQuery("WITH stage_durations AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetStageTime(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result.Stages, 2)

		assert.Equal(t, "Applied", result.Stages[0].StageName)
		assert.Equal(t, 2.5, result.Stages[0].AvgDays)
		assert.Equal(t, 1.0, result.Stages[0].MinDays)
		assert.Equal(t, 5.0, result.Stages[0].MaxDays)
		assert.Equal(t, 50, result.Stages[0].ApplicationsCount)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty for no stages", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"stage_name",
			"stage_order",
			"avg_days",
			"min_days",
			"max_days",
			"applications_count",
		})

		mock.ExpectQuery("WITH stage_durations AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetStageTime(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Stages)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAnalyticsRepository_GetResumeEffectiveness(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewAnalyticsRepositoryWithPool(mock)
	userID := "user-123"

	t.Run("returns resume effectiveness successfully", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"resume_id",
			"resume_title",
			"applications_count",
			"responses_count",
			"interviews_count",
			"response_rate",
		}).
			AddRow("resume-1", "Software Engineer Resume", 20, 10, 5, 50.0).
			AddRow("resume-2", "Senior Dev Resume", 15, 12, 8, 80.0)

		mock.ExpectQuery("WITH resume_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetResumeEffectiveness(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result.Resumes, 2)

		assert.Equal(t, "resume-1", result.Resumes[0].ResumeID)
		assert.Equal(t, "Software Engineer Resume", result.Resumes[0].ResumeTitle)
		assert.Equal(t, 20, result.Resumes[0].ApplicationsCount)
		assert.Equal(t, 10, result.Resumes[0].ResponsesCount)
		assert.Equal(t, 5, result.Resumes[0].InterviewsCount)
		assert.Equal(t, 50.0, result.Resumes[0].ResponseRate)

		assert.Equal(t, 80.0, result.Resumes[1].ResponseRate)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty for no resumes", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"resume_id",
			"resume_title",
			"applications_count",
			"responses_count",
			"interviews_count",
			"response_rate",
		})

		mock.ExpectQuery("WITH resume_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetResumeEffectiveness(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Resumes)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAnalyticsRepository_GetSourceAnalytics(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewAnalyticsRepositoryWithPool(mock)
	userID := "user-123"

	t.Run("returns source analytics successfully", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"source_name",
			"applications_count",
			"responses_count",
			"conversion_rate",
		}).
			AddRow("LinkedIn", 50, 25, 50.0).
			AddRow("Indeed", 30, 10, 33.33).
			AddRow("Unknown", 20, 5, 25.0)

		mock.ExpectQuery("WITH source_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetSourceAnalytics(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result.Sources, 3)

		assert.Equal(t, "LinkedIn", result.Sources[0].SourceName)
		assert.Equal(t, 50, result.Sources[0].ApplicationsCount)
		assert.Equal(t, 25, result.Sources[0].ResponsesCount)
		assert.Equal(t, 50.0, result.Sources[0].ConversionRate)

		assert.Equal(t, "Indeed", result.Sources[1].SourceName)
		assert.Equal(t, 33.33, result.Sources[1].ConversionRate)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty for no applications", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"source_name",
			"applications_count",
			"responses_count",
			"conversion_rate",
		})

		mock.ExpectQuery("WITH source_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetSourceAnalytics(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Sources)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
