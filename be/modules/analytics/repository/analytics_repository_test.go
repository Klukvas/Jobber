package repository

import (
	"context"
	"testing"

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

	t.Run("returns overview analytics successfully", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"total_applications",
			"active_applications",
			"closed_applications",
			"response_rate",
			"avg_days_to_first_response",
		}).AddRow(10, 5, 5, 50.0, 3.5)

		mock.ExpectQuery("WITH app_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetOverview(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, 10, result.TotalApplications)
		assert.Equal(t, 5, result.ActiveApplications)
		assert.Equal(t, 5, result.ClosedApplications)
		assert.Equal(t, 50.0, result.ResponseRate)
		assert.Equal(t, 3.5, result.AvgDaysToFirstResponse)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns zero values for empty data", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"total_applications",
			"active_applications",
			"closed_applications",
			"response_rate",
			"avg_days_to_first_response",
		}).AddRow(0, 0, 0, 0.0, 0.0)

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

	t.Run("returns funnel stages successfully", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"stage_name",
			"stage_order",
			"app_count",
			"conversion_rate",
			"drop_off_rate",
		}).
			AddRow("Applied", 1, 100, 100.0, 0.0).
			AddRow("Phone Screen", 2, 50, 50.0, 50.0).
			AddRow("Interview", 3, 25, 50.0, 50.0).
			AddRow("Offer", 4, 10, 40.0, 60.0)

		mock.ExpectQuery("WITH total_apps AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result.Stages, 4)

		assert.Equal(t, "Applied", result.Stages[0].StageName)
		assert.Equal(t, 100, result.Stages[0].Count)
		assert.Equal(t, 100.0, result.Stages[0].ConversionRate)

		assert.Equal(t, "Phone Screen", result.Stages[1].StageName)
		assert.Equal(t, 50, result.Stages[1].Count)
		assert.Equal(t, 50.0, result.Stages[1].ConversionRate)
		assert.Equal(t, 50.0, result.Stages[1].DropOffRate)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty stages for user without data", func(t *testing.T) {
		rows := pgxmock.NewRows([]string{
			"stage_name",
			"stage_order",
			"app_count",
			"conversion_rate",
			"drop_off_rate",
		})

		mock.ExpectQuery("WITH total_apps AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Stages)

		require.NoError(t, mock.ExpectationsWereMet())
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
