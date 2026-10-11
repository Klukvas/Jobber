package service

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Create decides two things that used to be wrong or missing: which column a
// new card starts in, and what applied_at it carries. The frontend's "I already
// applied" toggle sends a date and nothing else — the card must leave the
// wishlist and keep the date the user picked.

// pipelineRepo serves a whole pipeline from List (in column order) and echoes
// requested columns from GetByID.
func pipelineRepo(columns ...*model.StageTemplate) *MockStageTemplateRepository {
	byID := make(map[string]*model.StageTemplate, len(columns))
	for _, c := range columns {
		byID[c.ID] = c
	}
	return &MockStageTemplateRepository{
		ListFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
			if limit < len(columns) {
				return columns[:limit], len(columns), nil
			}
			return columns, len(columns), nil
		},
		GetByIDFunc: func(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
			column, ok := byID[templateID]
			if !ok {
				return nil, model.ErrStageTemplateNotFound
			}
			return column, nil
		},
	}
}

func defaultPipelineColumns() []*model.StageTemplate {
	return []*model.StageTemplate{
		{ID: "c-wishlist", Name: "Wishlist", Order: 0, Kind: model.StageKindSaved},
		{ID: "c-applied", Name: "Applied", Order: 1, Kind: model.StageKindApplied},
		{ID: "c-screening", Name: "Screening", Order: 2, Kind: model.StageKindResponse},
		{ID: "c-offer", Name: "Offer", Order: 3, Kind: model.StageKindOffer},
		{ID: "c-rejected", Name: "Rejected", Order: 4, Kind: model.StageKindRejected},
	}
}

// capturingJobRepo records the card both creation paths were handed; the repo's
// InitialStage field records the history row, when one was written.
func capturingJobRepo() (*MockJobRepository, func() *model.Job) {
	var created *model.Job
	repo := &MockJobRepository{
		CreateFunc: func(ctx context.Context, job *model.Job) error {
			job.ID = "job-1"
			created = job
			return nil
		},
	}
	return repo, func() *model.Job { return created }
}

func TestJobService_Create_Placement(t *testing.T) {
	userID := "user-123"
	appliedOn := time.Date(2026, time.March, 2, 9, 30, 0, 0, time.FixedZone("CET", 3600))

	t.Run("places an already-applied card in the applied column and keeps the date", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		require.NoError(t, err)
		require.NotNil(t, created())
		require.NotNil(t, created().CurrentStageTemplateID)
		assert.Equal(t, "c-applied", *created().CurrentStageTemplateID)
		require.NotNil(t, dto.AppliedAt)
		assert.True(t, dto.AppliedAt.Equal(appliedOn), "the user's date is honored, not replaced by now")
		assert.Equal(t, time.UTC, dto.AppliedAt.Location(), "stored in UTC like every other timestamp")
	})

	t.Run("records the initial history row for the column the card starts in", func(t *testing.T) {
		repo, _ := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))

		_, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		require.NoError(t, err)
		require.NotNil(t, repo.InitialStage, "a card that starts past the wishlist has entered a column")
		assert.Equal(t, "c-applied", repo.InitialStage.StageTemplateID)
		assert.Equal(t, "active", repo.InitialStage.Status)
		assert.Equal(t, 0, repo.InitialStage.Order)
		assert.True(t, repo.InitialStage.StartedAt.Equal(appliedOn),
			"history starts when the user says they applied, so time-to-response measures from there")
	})

	t.Run("a plain save lands in the wishlist with no history and no applied_at", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{Title: "Engineer"})

		require.NoError(t, err)
		require.NotNil(t, created().CurrentStageTemplateID)
		assert.Equal(t, "c-wishlist", *created().CurrentStageTemplateID)
		assert.Nil(t, dto.AppliedAt)
		assert.Nil(t, repo.InitialStage, "nothing has happened to a wishlisted card yet")
	})

	t.Run("a plain save finds the wishlist behind an unclassified first column", func(t *testing.T) {
		// The wishlist is not where this board starts, and the first column
		// carries no meaning to read.
		custom := []*model.StageTemplate{
			{ID: "c-ideas", Name: "Ideas", Order: 0, Kind: model.StageKindOther},
			{ID: "c-wishlist", Name: "Wishlist", Order: 1, Kind: model.StageKindSaved},
			{ID: "c-applied", Name: "Applied", Order: 2, Kind: model.StageKindApplied},
		}
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(custom...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{Title: "Engineer"})

		require.NoError(t, err)
		assert.Equal(t, "c-wishlist", *created().CurrentStageTemplateID,
			"the column that means 'not applied yet' is where a bookmark belongs")
		assert.Nil(t, dto.AppliedAt)
		assert.Nil(t, repo.InitialStage)
	})

	t.Run("an explicit column wins over the applied date", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))
		explicit := "c-screening"

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:           "Engineer",
			StageTemplateID: &explicit,
			AppliedAt:       &appliedOn,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-screening", *created().CurrentStageTemplateID)
		require.NotNil(t, dto.AppliedAt)
		assert.True(t, dto.AppliedAt.Equal(appliedOn))
	})

	t.Run("an explicit column past the first one still stamps applied_at", func(t *testing.T) {
		repo, _ := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))
		explicit := "c-offer"

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:           "Engineer",
			StageTemplateID: &explicit,
		})

		require.NoError(t, err)
		assert.NotNil(t, dto.AppliedAt, "placing a card past the wishlist means it was applied to")
	})

	t.Run("falls back to the first post-application column when the pipeline has no applied column", func(t *testing.T) {
		// A custom pipeline: no "applied" kind at all.
		custom := []*model.StageTemplate{
			{ID: "c-leads", Name: "Leads", Order: 0, Kind: model.StageKindSaved},
			{ID: "c-notes", Name: "Notes to self", Order: 1, Kind: model.StageKindOther},
			{ID: "c-call", Name: "Recruiter call", Order: 2, Kind: model.StageKindResponse},
		}
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(custom...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-call", *created().CurrentStageTemplateID)
		require.NotNil(t, dto.AppliedAt, "the toggle is never silently dropped")
		assert.True(t, dto.AppliedAt.Equal(appliedOn))
	})

	t.Run("never places an applied card in a rejected column", func(t *testing.T) {
		custom := []*model.StageTemplate{
			{ID: "c-leads", Name: "Leads", Order: 0, Kind: model.StageKindSaved},
			{ID: "c-nope", Name: "No thanks", Order: 1, Kind: model.StageKindRejected},
			{ID: "c-misc", Name: "Misc", Order: 2, Kind: model.StageKindOther},
		}
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(custom...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		require.NoError(t, err)
		// The rejected column is skipped; the next one out of the wishlist wins.
		assert.Equal(t, "c-misc", *created().CurrentStageTemplateID)
		require.NotNil(t, dto.AppliedAt)
	})

	t.Run("a single-column pipeline keeps the card there but still records the date", func(t *testing.T) {
		only := []*model.StageTemplate{{ID: "c-only", Name: "Board", Order: 0, Kind: model.StageKindOther}}
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(only...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-only", *created().CurrentStageTemplateID)
		require.NotNil(t, dto.AppliedAt)
		assert.Nil(t, repo.InitialStage, "the card never left the first column")
	})

	t.Run("ignores a zero applied_at", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))
		zero := time.Time{}

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &zero,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-wishlist", *created().CurrentStageTemplateID)
		assert.Nil(t, dto.AppliedAt)
	})

	t.Run("normalizes an empty company id to no company", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(defaultPipelineColumns()...))
		empty := ""

		_, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			CompanyID: &empty,
		})

		require.NoError(t, err)
		assert.Nil(t, created().CompanyID, "an empty id must never reach a UUID column")
	})

	t.Run("fails when the user has no pipeline", func(t *testing.T) {
		repo, _ := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo())

		_, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})
}

func TestSavedColumn(t *testing.T) {
	// Where a bookmark goes: the wishlist by meaning, the first column only
	// when the board has no wishlist.
	tests := []struct {
		name    string
		columns []*model.StageTemplate
		want    string
	}{
		{
			name: "takes the wishlist wherever the user dragged it",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindApplied},
				{ID: "b", Kind: model.StageKindResponse},
				{ID: "c", Kind: model.StageKindSaved},
			},
			want: "c",
		},
		{
			name: "takes the first of several wishlist columns",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindOther},
				{ID: "b", Kind: model.StageKindSaved},
				{ID: "c", Kind: model.StageKindSaved},
			},
			want: "b",
		},
		{
			name: "falls back to the first column when the board has no wishlist",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindApplied},
				{ID: "b", Kind: model.StageKindRejected},
			},
			want: "a",
		},
		{
			name: "treats a legacy column with no kind as unclassified, not as a wishlist",
			columns: []*model.StageTemplate{
				{ID: "a"},
				{ID: "b", Kind: model.StageKindSaved},
			},
			want: "b",
		},
		{
			name:    "falls back to the only column",
			columns: []*model.StageTemplate{{ID: "a", Kind: model.StageKindOther}},
			want:    "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, savedColumn(tt.columns).ID)
		})
	}
}

func TestAppliedColumn(t *testing.T) {
	// The chain is deterministic: applied → first response/interview/offer →
	// second column → the only column.
	tests := []struct {
		name    string
		columns []*model.StageTemplate
		want    string
	}{
		{
			name: "prefers the applied column even when a later one would match",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindSaved},
				{ID: "b", Kind: model.StageKindInterview},
				{ID: "c", Kind: model.StageKindApplied},
			},
			want: "c",
		},
		{
			name: "takes the earliest post-application column otherwise",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindSaved},
				{ID: "b", Kind: model.StageKindOther},
				{ID: "c", Kind: model.StageKindInterview},
				{ID: "d", Kind: model.StageKindOffer},
			},
			want: "c",
		},
		{
			name: "falls back to the second column",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindSaved},
				{ID: "b", Kind: model.StageKindOther},
			},
			want: "b",
		},
		{
			name: "skips a rejected column in the positional fallback",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindSaved},
				{ID: "b", Kind: model.StageKindRejected},
				{ID: "c", Kind: model.StageKindOther},
			},
			want: "c",
		},
		{
			name: "returns the initial column when every other one is a rejection",
			columns: []*model.StageTemplate{
				{ID: "a", Kind: model.StageKindSaved},
				{ID: "b", Kind: model.StageKindRejected},
			},
			want: "a",
		},
		{
			name:    "falls back to the only column",
			columns: []*model.StageTemplate{{ID: "a", Kind: model.StageKindOther}},
			want:    "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, appliedColumn(tt.columns).ID)
		})
	}
}
