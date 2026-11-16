//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	analyticsRepo "github.com/andreypavlenko/jobber/modules/analytics/repository"
	analyticsSvc "github.com/andreypavlenko/jobber/modules/analytics/service"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	jobRepo "github.com/andreypavlenko/jobber/modules/jobs/repository"
	jobSvc "github.com/andreypavlenko/jobber/modules/jobs/service"
	shareRepo "github.com/andreypavlenko/jobber/modules/sharing/repository"
	shareSvc "github.com/andreypavlenko/jobber/modules/sharing/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This suite deliberately refuses the local UI database. Migrate the dedicated
// database before running with R3_DATA_TEST_DATABASE_URL and -tags=integration.
func r3Pool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	dsn := os.Getenv("R3_DATA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set R3_DATA_TEST_DATABASE_URL to a migrated jobber_r3_data_test database")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	var name string
	require.NoError(t, db.QueryRow(ctx, `SELECT current_database()`).Scan(&name))
	require.Equal(t, "jobber_r3_data_test", name, "refusing to modify a non-test database")
	userID := uuid.NewString()
	_, err = db.Exec(ctx, `INSERT INTO users (id,email,name,password_hash) VALUES ($1,$2,'Regression','test-only')`, userID, userID+"@regression.test")
	require.NoError(t, err)
	return db, userID
}

func r3Template(t *testing.T, db *pgxpool.Pool, userID, name string, order int, kind jobModel.StageKind) *jobModel.StageTemplate {
	t.Helper()
	template := &jobModel.StageTemplate{UserID: userID, Name: name, Order: order, Kind: kind}
	require.NoError(t, jobRepo.NewStageTemplateRepository(db).Create(context.Background(), template))
	return template
}

func TestRegressionR3TemplateOrder(t *testing.T) {
	db, userID := r3Pool(t)
	ctx := context.Background()
	repo := jobRepo.NewStageTemplateRepository(db)
	first := r3Template(t, db, userID, "First", 0, jobModel.StageKindSaved)
	second := r3Template(t, db, userID, "Second", 1, jobModel.StageKindApplied)
	third := r3Template(t, db, userID, "Third", 2, jobModel.StageKindInterview)
	// Reproduce legacy ties/gaps. created_at and id define the stable tie order.
	_, err := db.Exec(ctx, `UPDATE stage_templates SET "order" = 7, created_at = CASE WHEN id = $2 THEN '2020-01-01'::timestamp ELSE '2020-01-02'::timestamp END WHERE user_id=$1`, userID, first.ID)
	require.NoError(t, err)
	inserted := r3Template(t, db, userID, "Inserted", 1, jobModel.StageKindOther)
	all, _, err := repo.List(ctx, userID, 1000, 0)
	require.NoError(t, err)
	require.Len(t, all, 4)
	assert.Equal(t, first.ID, all[0].ID)
	assert.Equal(t, inserted.ID, all[1].ID)
	for i, v := range all {
		assert.Equal(t, i, v.Order)
	}
	require.NoError(t, repo.Delete(ctx, userID, inserted.ID))
	all, _, err = repo.List(ctx, userID, 1000, 0)
	require.NoError(t, err)
	before := []string{all[0].ID, all[1].ID, all[2].ID}
	require.ErrorIs(t, repo.Reorder(ctx, userID, []string{first.ID, first.ID, third.ID}), jobModel.ErrReorderMismatch)
	all, _, err = repo.List(ctx, userID, 1000, 0)
	require.NoError(t, err)
	for i, v := range all {
		assert.Equal(t, before[i], v.ID)
		assert.Equal(t, i, v.Order)
	}

	// All four mutation paths compete for the same user's serialization lock.
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- repo.Create(ctx, &jobModel.StageTemplate{UserID: userID, Name: fmt.Sprintf("Concurrent %d", i), Order: 1})
		}(i)
	}
	wg.Add(3)
	go func() {
		defer wg.Done()
		position := 0
		_, err := repo.Update(ctx, userID, third.ID, &jobModel.StageTemplatePatch{Order: &position})
		errs <- err
	}()
	go func() { defer wg.Done(); errs <- repo.Delete(ctx, userID, second.ID) }()
	go func() {
		defer wg.Done()
		err := repo.Reorder(ctx, userID, before)
		if errors.Is(err, jobModel.ErrReorderMismatch) {
			err = nil
		}
		errs <- err
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	all, _, err = repo.List(ctx, userID, 1000, 0)
	require.NoError(t, err)
	require.Len(t, all, 14)
	for i, v := range all {
		assert.Equal(t, i, v.Order)
	}
}

func TestRegressionR3CompletionAndLastHistory(t *testing.T) {
	for _, pointer := range []bool{true, false} {
		for _, applied := range []bool{true, false} {
			t.Run(fmt.Sprintf("pointer=%t/applied=%t", pointer, applied), func(t *testing.T) {
				db, userID := r3Pool(t)
				ctx := context.Background()
				wishlist := r3Template(t, db, userID, "Saved", 0, jobModel.StageKindSaved)
				interview := r3Template(t, db, userID, "Onsite", 1, jobModel.StageKindInterview)
				jobID, stageID := uuid.NewString(), uuid.NewString()
				original := time.Date(2024, 4, 3, 2, 1, 0, 0, time.UTC)
				var appliedAt *time.Time
				if applied {
					appliedAt = &original
				}
				_, err := db.Exec(ctx, `INSERT INTO jobs (id,user_id,title,current_stage_template_id,applied_at) VALUES ($1,$2,'Test',$3,$4)`, jobID, userID, interview.ID, appliedAt)
				require.NoError(t, err)
				_, err = db.Exec(ctx, `INSERT INTO job_stages (id,job_id,stage_template_id,status,"order",started_at,completed_at) VALUES ($1,$2,$3,'completed',0,$4,$4)`, stageID, jobID, interview.ID, original)
				require.NoError(t, err)
				if pointer {
					_, err = db.Exec(ctx, `UPDATE jobs SET current_stage_id=$2 WHERE id=$1`, jobID, stageID)
					require.NoError(t, err)
				}
				svc := jobSvc.NewJobService(db, jobRepo.NewJobRepository(db), jobRepo.NewJobStageRepository(db), jobRepo.NewStageTemplateRepository(db), nil, nil, nil, nil, nil, nil, nil)
				completed := "completed"
				later := original.Add(time.Hour)
				for _, stamp := range []*time.Time{nil, &later} {
					got, err := svc.UpdateStage(ctx, userID, jobID, stageID, &jobModel.UpdateStageRequest{Status: &completed, CompletedAt: stamp})
					require.NoError(t, err)
					require.NotNil(t, got.CompletedAt)
					assert.True(t, original.Equal(*got.CompletedAt))
				}
				require.NoError(t, svc.DeleteStage(ctx, userID, jobID, stageID))
				got, err := jobRepo.NewJobRepository(db).GetByID(ctx, userID, jobID)
				require.NoError(t, err)
				assert.Nil(t, got.CurrentStageID)
				require.NotNil(t, got.CurrentStageTemplateID)
				assert.Equal(t, wishlist.ID, *got.CurrentStageTemplateID)
				if applied {
					require.NotNil(t, got.AppliedAt)
					assert.True(t, original.Equal(*got.AppliedAt))
				} else {
					assert.Nil(t, got.AppliedAt)
				}
			})
		}
	}
}

func TestRegressionR3ArchivedAnalyticsAndSharedSnapshot(t *testing.T) {
	db, userID := r3Pool(t)
	ctx := context.Background()
	wishlist := r3Template(t, db, userID, "Saved", 0, jobModel.StageKindSaved)
	interview := r3Template(t, db, userID, "Onsite", 1, jobModel.StageKindInterview)
	resumeID := uuid.NewString()
	_, err := db.Exec(ctx, `INSERT INTO resumes (id,user_id,title,storage_type) VALUES ($1,$2,'Resume','external')`, resumeID, userID)
	require.NoError(t, err)
	activeID, archivedID := uuid.NewString(), uuid.NewString()
	_, err = db.Exec(ctx, `INSERT INTO jobs (id,user_id,title,current_stage_template_id,resume_id,source,is_archived,applied_at) VALUES ($1,$3,'Active',$4,$6,'Referral',false,'2024-01-01'),($2,$3,'Archived',$5,$6,'Referral',true,'2024-01-01')`, activeID, archivedID, userID, wishlist.ID, interview.ID, resumeID)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO job_stages (job_id,stage_template_id,status,"order",started_at,completed_at) VALUES ($1,$2,'completed',0,'2024-01-03','2024-01-06')`, archivedID, interview.ID)
	require.NoError(t, err)
	repo := analyticsRepo.NewAnalyticsRepository(db)
	overview, err := repo.GetOverview(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 2, overview.TotalApplications)
	assert.Equal(t, 1, overview.ActiveApplications)
	assert.Equal(t, 1, overview.ClosedApplications)
	assert.Equal(t, 50.0, overview.ResponseRate)
	assert.Equal(t, 2.0, overview.AvgDaysToFirstResponse)
	funnel, err := repo.GetFunnel(ctx, userID)
	require.NoError(t, err)
	require.Len(t, funnel.Stages, 2)
	assert.Equal(t, 2, funnel.Stages[0].Count)
	assert.Equal(t, 1, funnel.Stages[1].Count)
	timing, err := repo.GetStageTime(ctx, userID)
	require.NoError(t, err)
	require.Len(t, timing.Stages, 1)
	assert.Equal(t, 1, timing.Stages[0].ApplicationsCount)
	assert.Equal(t, 3.0, timing.Stages[0].AvgDays)
	resumes, err := repo.GetResumeEffectiveness(ctx, userID)
	require.NoError(t, err)
	require.Len(t, resumes.Resumes, 1)
	assert.Equal(t, 2, resumes.Resumes[0].ApplicationsCount)
	assert.Equal(t, 1, resumes.Resumes[0].ResponsesCount)
	assert.Equal(t, 1, resumes.Resumes[0].InterviewsCount)
	assert.Equal(t, 50.0, resumes.Resumes[0].ResponseRate)
	sources, err := repo.GetSourceAnalytics(ctx, userID)
	require.NoError(t, err)
	require.Len(t, sources.Sources, 1)
	assert.Equal(t, 2, sources.Sources[0].ApplicationsCount)
	assert.Equal(t, 1, sources.Sources[0].ResponsesCount)
	assert.Equal(t, 50.0, sources.Sources[0].ConversionRate)
	shares := shareSvc.NewSharingService(shareRepo.NewSharingRepository(db), analyticsSvc.NewAnalyticsService(repo))
	share, err := shares.Create(ctx, userID)
	require.NoError(t, err)
	public, err := shares.GetPublicByToken(ctx, share.Token)
	require.NoError(t, err)
	assert.Equal(t, 50.0, public.Snapshot.Overview.ResponseRate)
	assert.Equal(t, 1, public.Snapshot.Funnel[1].Count)
	// All-archived users retain their history and rates as well.
	_, err = db.Exec(ctx, `UPDATE jobs SET is_archived=true WHERE id=$1`, activeID)
	require.NoError(t, err)
	overview, err = repo.GetOverview(ctx, userID)
	require.NoError(t, err)
	assert.Zero(t, overview.ActiveApplications)
	assert.Equal(t, 2, overview.ClosedApplications)
	assert.Equal(t, 50.0, overview.ResponseRate)
}
