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

	t.Run("returns error when query fails", func(t *testing.T) {
		mock.ExpectQuery("WITH first_template AS").
			WithArgs(userID).
			WillReturnError(assert.AnError)

		result, err := repo.GetOverview(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	overviewColumns := []string{
		"active_applications",
		"closed_applications",
		"rejected_applications",
		"response_rate",
		"avg_days_to_first_response",
	}

	// Total is derived from the other two buckets, so the identity holds for
	// every shape of data — including the mixed active/archived mix that used
	// to print Total 23 next to Active 23 and Closed 3.
	t.Run("total always equals active plus closed", func(t *testing.T) {
		tests := []struct {
			name      string
			active    int
			closed    int
			wantTotal int
		}{
			{name: "no cards at all", active: 0, closed: 0, wantTotal: 0},
			{name: "only active cards", active: 23, closed: 0, wantTotal: 23},
			{name: "only archived cards", active: 0, closed: 3, wantTotal: 3},
			{name: "mixed active and archived", active: 23, closed: 3, wantTotal: 26},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mock.ExpectQuery("WITH first_template AS").
					WithArgs(userID).
					WillReturnRows(pgxmock.NewRows(overviewColumns).
						AddRow(tt.active, tt.closed, 0, 50.0, 3.5))

				result, err := repo.GetOverview(context.Background(), userID)

				require.NoError(t, err)
				assert.Equal(t, tt.active, result.ActiveApplications)
				assert.Equal(t, tt.closed, result.ClosedApplications)
				assert.Equal(t, tt.wantTotal, result.TotalApplications)
				assert.Equal(t,
					result.ActiveApplications+result.ClosedApplications,
					result.TotalApplications,
				)
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	})

	t.Run("returns overview analytics successfully", func(t *testing.T) {
		mock.ExpectQuery("WITH first_template AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows(overviewColumns).AddRow(5, 5, 0, 50.0, 3.5))

		result, err := repo.GetOverview(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, 10, result.TotalApplications)
		assert.Equal(t, 5, result.ActiveApplications)
		assert.Equal(t, 5, result.ClosedApplications)
		assert.Equal(t, 0, result.RejectedApplications)
		assert.Equal(t, 50.0, result.ResponseRate)
		assert.Equal(t, 3.5, result.AvgDaysToFirstResponse)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns zero values for empty data", func(t *testing.T) {
		mock.ExpectQuery("WITH first_template AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows(overviewColumns).AddRow(0, 0, 0, 0.0, 0.0))

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

	t.Run("returns error when the stage query fails", func(t *testing.T) {
		mock.ExpectQuery("WITH templates AS").
			WithArgs(userID).
			WillReturnError(assert.AnError)

		result, err := repo.GetFunnel(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, result)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("builds a positional stage funnel over the user's stage templates", func(t *testing.T) {
		// The user's own columns, in order, with reached counts.
		mock.ExpectQuery("WITH templates AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"stage_name", "stage_order", "reached_count"}).
				AddRow("Applied", 1, 100).
				AddRow("Interview", 2, 50).
				AddRow("Offer", 3, 10))

		result, err := repo.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result.Stages, 3)

		// Buckets are the user's own columns.
		assert.Equal(t, "Applied", result.Stages[0].StageName)
		assert.Equal(t, 1, result.Stages[0].StageOrder)
		assert.Equal(t, 100, result.Stages[0].Count)
		assert.Equal(t, 100.0, result.Stages[0].ConversionRate)

		assert.Equal(t, "Interview", result.Stages[1].StageName)
		assert.Equal(t, 50, result.Stages[1].Count)
		assert.Equal(t, 50.0, result.Stages[1].ConversionRate) // 50/100
		assert.Equal(t, 50.0, result.Stages[1].DropOffRate)

		assert.Equal(t, "Offer", result.Stages[2].StageName)
		assert.Equal(t, 10, result.Stages[2].Count)
		assert.Equal(t, 20.0, result.Stages[2].ConversionRate) // 10/50

		// No status → no rejected block in the single-axis model.
		assert.Nil(t, result.Rejected)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns an empty funnel when the user has no stage templates", func(t *testing.T) {
		mock.ExpectQuery("WITH templates AS").
			WithArgs(userID).
			WillReturnRows(pgxmock.NewRows([]string{"stage_name", "stage_order", "reached_count"}))

		result, err := repo.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Stages)
		assert.Nil(t, result.Rejected)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBuildStageFunnel(t *testing.T) {
	type rates struct {
		conversion float64
		dropOff    float64
	}

	tests := []struct {
		name   string
		counts []stageCount
		want   []rates
	}{
		{
			name: "conversion and drop-off are relative to the previous column",
			counts: []stageCount{
				{name: "Applied", order: 1, count: 100},
				{name: "Interview", order: 2, count: 40},
				{name: "Offer", order: 3, count: 10},
			},
			want: []rates{{100, 0}, {40, 60}, {25, 75}},
		},
		{
			name: "an empty pipeline reports nothing rather than a perfect 100%",
			counts: []stageCount{
				{name: "Applied", order: 1, count: 0},
				{name: "Interview", order: 2, count: 0},
				{name: "Offer", order: 3, count: 0},
			},
			want: []rates{{0, 0}, {0, 0}, {0, 0}},
		},
		{
			name: "a zero denominator mid-funnel does not divide by zero",
			counts: []stageCount{
				{name: "Applied", order: 1, count: 5},
				{name: "Interview", order: 2, count: 0},
				{name: "Offer", order: 3, count: 0},
			},
			want: []rates{{100, 0}, {0, 100}, {0, 0}},
		},
		{
			name: "a non-monotonic count can never produce >100% or a negative drop-off",
			counts: []stageCount{
				{name: "Wishlist", order: 1, count: 5},
				{name: "Applied", order: 2, count: 17},
				{name: "Interview", order: 3, count: 3},
			},
			want: []rates{{100, 0}, {100, 0}, {17.65, 82.35}},
		},
		{
			name: "no columns yields no buckets",
			// A brand new account has no stage templates at all.
			counts: nil,
			want:   nil,
		},
		{
			name:   "a single column is its own base",
			counts: []stageCount{{name: "Applied", order: 1, count: 3}},
			want:   []rates{{100, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stages := buildStageFunnel(tt.counts)

			require.Len(t, stages, len(tt.want))
			for i, want := range tt.want {
				assert.Equal(t, tt.counts[i].name, stages[i].StageName)
				assert.Equal(t, tt.counts[i].count, stages[i].Count)
				assert.Equal(t, want.conversion, stages[i].ConversionRate, "conversion at %d", i)
				assert.Equal(t, want.dropOff, stages[i].DropOffRate, "drop-off at %d", i)
				assert.GreaterOrEqual(t, stages[i].ConversionRate, 0.0)
				assert.LessOrEqual(t, stages[i].ConversionRate, 100.0)
				assert.GreaterOrEqual(t, stages[i].DropOffRate, 0.0)
				assert.LessOrEqual(t, stages[i].DropOffRate, 100.0)
			}
		})
	}
}

func TestClampPercent(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "below range", in: -240, want: 0},
		{name: "lower bound", in: 0, want: 0},
		{name: "inside range", in: 42.5, want: 42.5},
		{name: "upper bound", in: 100, want: 100},
		{name: "above range", in: 340, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clampPercent(tt.in))
		})
	}
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

		mock.ExpectQuery("resume_stats AS").
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

		mock.ExpectQuery("resume_stats AS").
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

		mock.ExpectQuery("source_stats AS").
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

		mock.ExpectQuery("source_stats AS").
			WithArgs(userID).
			WillReturnRows(rows)

		result, err := repo.GetSourceAnalytics(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result.Sources)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
