package service

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// "Has this card been applied to?" is answered by the target column's kind, not
// by where the user happens to have dragged that column. These tests pin both
// directions of the bug that position-based rule had: an "Applied" column
// reordered to the front stopped stamping applied_at, and a "Wishlist" column
// reordered to the middle started stamping it for a bookmark.

func TestColumnImpliesApplied(t *testing.T) {
	tests := []struct {
		name   string
		column *model.StageTemplate
		want   bool
	}{
		{
			name:   "an applied column reordered to the front still means applied",
			column: &model.StageTemplate{Name: "Applied", Order: 0, Kind: model.StageKindApplied},
			want:   true,
		},
		{
			name:   "a wishlist column reordered to the middle still means not applied",
			column: &model.StageTemplate{Name: "Wishlist", Order: 2, Kind: model.StageKindSaved},
			want:   false,
		},
		{
			name:   "a rejection can only be reached by applying",
			column: &model.StageTemplate{Name: "Rejected", Order: 0, Kind: model.StageKindRejected},
			want:   true,
		},
		{
			name:   "an interview implies the application went out",
			column: &model.StageTemplate{Name: "Onsite loop", Order: 1, Kind: model.StageKindInterview},
			want:   true,
		},
		{
			name:   "an unclassified column keeps the positional rule: first column is not applied",
			column: &model.StageTemplate{Name: "Ideas", Order: 0, Kind: model.StageKindOther},
			want:   false,
		},
		{
			name:   "an unclassified column keeps the positional rule: past the first one is",
			column: &model.StageTemplate{Name: "Follow-up", Order: 3, Kind: model.StageKindOther},
			want:   true,
		},
		{
			name:   "a legacy row with no kind at all falls back to its position",
			column: &model.StageTemplate{Name: "Written before kinds", Order: 2},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, columnImpliesApplied(tt.column))
		})
	}
}

func TestAppliedAtAfterMove(t *testing.T) {
	now := time.Date(2026, time.April, 9, 10, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, time.January, 3, 8, 0, 0, 0, time.UTC)

	t.Run("stamps when the target column means applied, whatever its position", func(t *testing.T) {
		target := &model.StageTemplate{Name: "Applied", Order: 0, Kind: model.StageKindApplied}

		got := appliedAtAfterMove(nil, target, now)

		require.NotNil(t, got)
		assert.True(t, got.Equal(now))
	})

	t.Run("does not stamp when the target column means not applied yet", func(t *testing.T) {
		target := &model.StageTemplate{Name: "Wishlist", Order: 4, Kind: model.StageKindSaved}

		assert.Nil(t, appliedAtAfterMove(nil, target, now))
	})

	t.Run("keeps an existing date when moving back into the wishlist", func(t *testing.T) {
		target := &model.StageTemplate{Name: "Wishlist", Order: 0, Kind: model.StageKindSaved}

		got := appliedAtAfterMove(&earlier, target, now)

		require.NotNil(t, got, "a move never clears a date the user already has")
		assert.True(t, got.Equal(earlier))
	})

	t.Run("never re-stamps an existing date", func(t *testing.T) {
		target := &model.StageTemplate{Name: "Offer", Order: 5, Kind: model.StageKindOffer}

		got := appliedAtAfterMove(&earlier, target, now)

		require.NotNil(t, got)
		assert.True(t, got.Equal(earlier))
	})
}

// kindTemplateRepo yields one column with an explicit name/order/kind for the
// pre-transaction GetByID every pipeline write does.
func kindTemplateRepo(name string, order int, kind model.StageKind) *MockStageTemplateRepository {
	return &MockStageTemplateRepository{
		GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: tid, UserID: uid, Name: name, Order: order, Kind: kind}, nil
		},
	}
}

func TestMove_AppliedAtFollowsTheColumnKind(t *testing.T) {
	userID := stagesUserID
	jobID := stagesJobID

	t.Run("an applied column reordered to the front still stamps applied_at", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		expMaxOrder(mock, 1)
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), jobID, "tmpl-applied", 1, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		// applied_at (4th arg) is stamped even though the column sits at order 0.
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), "tmpl-applied", notNilTimePtrArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, kindTemplateRepo("Applied", 0, model.StageKindApplied), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: "tmpl-applied"})

		require.NoError(t, err)
		assert.NotNil(t, dto.AppliedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a wishlist column reordered to the middle does not stamp applied_at", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-notes"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		expMaxOrder(mock, 2)
		expInsertMovedStage(mock)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), "tmpl-wishlist", nilTimePtrArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, kindTemplateRepo("Wishlist", 3, model.StageKindSaved), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: "tmpl-wishlist"})

		require.NoError(t, err)
		assert.Nil(t, dto.AppliedAt, "a bookmark column never invents an application date")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("moving back into the wishlist keeps the date the card already had", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-screening"
		stamped := time.Date(2026, time.February, 2, 9, 0, 0, 0, time.UTC)
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl, AppliedAt: &stamped}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, &stamped)
		expCompleteOpen(mock)
		expMaxOrder(mock, 4)
		expInsertMovedStage(mock)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), "tmpl-wishlist", timePtrArg(stamped), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, kindTemplateRepo("Wishlist", 0, model.StageKindSaved), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: "tmpl-wishlist"})

		require.NoError(t, err)
		require.NotNil(t, dto.AppliedAt, "moving a card back is not a reason to forget it was applied to")
		assert.True(t, dto.AppliedAt.Equal(stamped))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// expInsertMovedStage is Move's history insert with every argument left open.
func expInsertMovedStage(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
	return m.ExpectExec(`INSERT INTO job_stages`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

// notNilTimePtrMatcher matches any non-nil *time.Time argument — "a timestamp
// was written", without pinning the instant the service generated.
type notNilTimePtrMatcher struct{}

func (notNilTimePtrMatcher) Match(v interface{}) bool {
	p, ok := v.(*time.Time)
	return ok && p != nil
}

func notNilTimePtrArg() notNilTimePtrMatcher { return notNilTimePtrMatcher{} }

func TestJobService_Create_AppliedPlacementFollowsTheColumnKind(t *testing.T) {
	userID := "user-123"

	// A reordered pipeline: the user dragged Applied to the front and keeps
	// their wishlist in the middle.
	reordered := []*model.StageTemplate{
		{ID: "c-applied", Name: "Applied", Order: 0, Kind: model.StageKindApplied},
		{ID: "c-screening", Name: "Screening", Order: 1, Kind: model.StageKindResponse},
		{ID: "c-wishlist", Name: "Wishlist", Order: 2, Kind: model.StageKindSaved},
	}

	t.Run("an applied column at order 0 records the application in the history", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(reordered...))
		explicit := "c-applied"

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:           "Engineer",
			StageTemplateID: &explicit,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-applied", *created().CurrentStageTemplateID)
		require.NotNil(t, dto.AppliedAt, "the column means applied, wherever it sits")
		require.NotNil(t, repo.InitialStage, "the application is an event the funnel has to see")
		assert.Equal(t, "c-applied", repo.InitialStage.StageTemplateID)
		assert.Equal(t, "active", repo.InitialStage.Status)
	})

	t.Run("a wishlist column past the first one is still just a bookmark", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(reordered...))
		explicit := "c-wishlist"

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:           "Engineer",
			StageTemplateID: &explicit,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-wishlist", *created().CurrentStageTemplateID)
		assert.Nil(t, dto.AppliedAt, "position must not turn a bookmark into an application")
		assert.Nil(t, repo.InitialStage, "nothing has happened to a wishlisted card yet")
	})

	t.Run("a date supplied for a wishlist column is still honored", func(t *testing.T) {
		appliedOn := time.Date(2026, time.March, 20, 7, 0, 0, 0, time.UTC)
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(reordered...))
		explicit := "c-wishlist"

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:           "Engineer",
			StageTemplateID: &explicit,
			AppliedAt:       &appliedOn,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-wishlist", *created().CurrentStageTemplateID, "the explicit column still wins")
		require.NotNil(t, dto.AppliedAt, "the date the user picked is never dropped")
		assert.True(t, dto.AppliedAt.Equal(appliedOn))
	})

	t.Run("a bare save lands in the wishlist even when the board starts with Applied", func(t *testing.T) {
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(reordered...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{Title: "Bookmarked role"})

		require.NoError(t, err)
		assert.Equal(t, "c-wishlist", *created().CurrentStageTemplateID,
			"a bookmark belongs in the wishlist wherever the user dragged it")
		assert.Nil(t, dto.AppliedAt, "reordering the board must not invent an application date")
		assert.Nil(t, repo.InitialStage, "nothing has happened to a wishlisted card yet")
	})

	t.Run("an already-applied card still goes to the applied column", func(t *testing.T) {
		appliedOn := time.Date(2026, time.March, 2, 9, 30, 0, 0, time.UTC)
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(reordered...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:     "Engineer",
			AppliedAt: &appliedOn,
		})

		require.NoError(t, err)
		assert.Equal(t, "c-applied", *created().CurrentStageTemplateID,
			"the toggle still places the card by kind, not by the wishlist's position")
		require.NotNil(t, dto.AppliedAt)
		assert.True(t, dto.AppliedAt.Equal(appliedOn))
		require.NotNil(t, repo.InitialStage, "the application is an event the funnel has to see")
	})

	t.Run("a bare save into a pipeline with no wishlist at all counts as applied", func(t *testing.T) {
		// No saved column anywhere: the first column is the only placement
		// available, and it means the application is out.
		noWishlist := []*model.StageTemplate{
			{ID: "c-applied", Name: "Applied", Order: 0, Kind: model.StageKindApplied},
			{ID: "c-screening", Name: "Screening", Order: 1, Kind: model.StageKindResponse},
		}
		repo, created := capturingJobRepo()
		svc := newServiceWithTemplates(repo, pipelineRepo(noWishlist...))

		dto, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{Title: "Engineer"})

		require.NoError(t, err)
		assert.Equal(t, "c-applied", *created().CurrentStageTemplateID, "the first column is the fallback")
		require.NotNil(t, dto.AppliedAt,
			"the card visibly sits in Applied, so the date must not disagree with it")
		require.NotNil(t, repo.InitialStage)
	})
}

func TestAppliedColumn_NonRejectedFallback(t *testing.T) {
	t.Run("prefers an unclassified column over another wishlist one", func(t *testing.T) {
		columns := []*model.StageTemplate{
			{ID: "a", Kind: model.StageKindSaved},
			{ID: "b", Kind: model.StageKindSaved},
			{ID: "c", Kind: model.StageKindOther},
		}

		assert.Equal(t, "c", appliedColumn(columns).ID,
			"a second wishlist column is not a step out of the wishlist")
	})

	t.Run("prefers a wishlist column over a rejection", func(t *testing.T) {
		columns := []*model.StageTemplate{
			{ID: "a", Kind: model.StageKindRejected},
			{ID: "b", Kind: model.StageKindSaved},
		}

		assert.Equal(t, "b", appliedColumn(columns).ID,
			`"I applied" is not "I was rejected", even when the rejection comes first`)
	})

	t.Run("falls back to the first column only when every column is a rejection", func(t *testing.T) {
		columns := []*model.StageTemplate{
			{ID: "a", Kind: model.StageKindRejected},
			{ID: "b", Kind: model.StageKindRejected},
		}

		// The documented dead end: the card goes somewhere the user owns rather
		// than the create failing or a column being invented for them.
		assert.Equal(t, "a", appliedColumn(columns).ID)
	})

	t.Run("treats a legacy column with no kind as unclassified", func(t *testing.T) {
		columns := []*model.StageTemplate{
			{ID: "a", Kind: model.StageKindSaved},
			{ID: "b"},
		}

		assert.Equal(t, "b", appliedColumn(columns).ID)
	})
}
