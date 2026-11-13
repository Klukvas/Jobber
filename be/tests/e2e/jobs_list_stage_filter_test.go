//go:build integration

package e2e

import (
	"testing"
	"time"

	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
	jobRepo "github.com/andreypavlenko/jobber/modules/jobs/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type jobListFixture struct {
	userID       string
	savedStage   string
	appliedStage string
	foreignStage string
	jobIDs       []string
	timestamps   map[string]time.Time
}

type jobListCase struct {
	name    string
	options jobPorts.ListOptions
	wantIDs []string
	total   int
}

// Exercise the actual List SQL on the migrated schema, including the count
// fallback. A non-UTC session must not change timestamp-without-time-zone values
// relative to GetByID (the detail endpoint's repository path).
func TestIntegrationJobRepositoryListStageFilter(t *testing.T) {
	cleanupAll(t)
	fixture := seedJobListFixture(t)
	ctx := boundedCtx(t)
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) //nolint:errcheck // read-only test transaction
	_, err = tx.Exec(ctx, "SET LOCAL TIME ZONE 'America/New_York'")
	require.NoError(t, err)
	repository := jobRepo.NewJobRepository(tx)
	for _, test := range jobListCases(fixture) {
		t.Run(test.name, func(t *testing.T) {
			options := test.options
			options.SortBy, options.SortDir = "created_at", "desc"
			jobs, total, err := repository.List(ctx, fixture.userID, &options)
			require.NoError(t, err)
			require.Equal(t, test.total, total)
			var ids []string
			for _, job := range jobs {
				ids = append(ids, job.ID)
				detail, err := repository.GetByID(ctx, fixture.userID, job.ID)
				require.NoError(t, err)
				require.True(t, job.CreatedAt.Equal(detail.CreatedAt), "list/detail created_at differs")
				require.True(t, job.UpdatedAt.Equal(detail.UpdatedAt), "list/detail updated_at differs")
				expected := fixture.timestamps[job.ID]
				require.True(t, job.CreatedAt.Equal(expected), "stored wall-clock timestamp shifted")
				require.True(t, job.LastActivityAt.Equal(expected), "last_activity_at shifted")
			}
			require.Equal(t, test.wantIDs, ids)
		})
	}
}

func seedJobListFixture(t *testing.T) jobListFixture {
	t.Helper()
	userID := seedUser(t, "repository-list-stages@test.com", "securepass123")
	stages := seedPipeline(t, userID, "Saved", "Applied")
	fixture := jobListFixture{userID: userID, savedStage: stages["Saved"], appliedStage: stages["Applied"], timestamps: make(map[string]time.Time)}
	baseTime := time.Date(2026, time.November, 13, 10, 0, 0, 0, time.UTC)
	for index, row := range []struct {
		title, stage string
		archived     bool
	}{
		{"Alpha", fixture.savedStage, false}, {"Beta", fixture.savedStage, false},
		{"Gamma", fixture.appliedStage, false}, {"Delta", fixture.savedStage, true},
		{"Epsilon", fixture.appliedStage, true},
	} {
		timestamp := baseTime.Add(time.Duration(index) * time.Hour)
		id := seedListedJob(t, userID, row.stage, row.title, row.archived, timestamp)
		fixture.jobIDs = append(fixture.jobIDs, id)
		fixture.timestamps[id] = timestamp
	}
	foreignUser := seedUser(t, "repository-list-foreign@test.com", "securepass123")
	fixture.foreignStage = seedPipeline(t, foreignUser, "Private")["Private"]
	seedListedJob(t, foreignUser, fixture.foreignStage, "Foreign newest Job", false, baseTime.Add(24*time.Hour))
	return fixture
}

func seedListedJob(t *testing.T, userID, stageID, title string, archived bool, timestamp time.Time) string {
	t.Helper()
	// INSERT avoids the schema's BEFORE UPDATE timestamp trigger, so both
	// stored timestamps stay fixed and page ordering cannot depend on timing.
	id := uuid.NewString()
	_, err := pool.Exec(t.Context(), `INSERT INTO jobs
  (id, user_id, title, current_stage_template_id, is_archived, created_at, updated_at)
  VALUES ($1, $2, $3, $4, $5, $6::timestamp, $6::timestamp)`,
		id, userID, title, stageID, archived, timestamp)
	require.NoError(t, err)
	return id
}

func jobListCases(fixture jobListFixture) []jobListCase {
	ids, saved, applied := fixture.jobIDs, fixture.savedStage, fixture.appliedStage
	return []jobListCase{
		{"without Stage first page", jobPorts.ListOptions{Limit: 2}, []string{ids[2], ids[1]}, 3},
		{"without Stage second page", jobPorts.ListOptions{Limit: 2, Offset: 2}, []string{ids[0]}, 3},
		{"without Stage past end", jobPorts.ListOptions{Limit: 1, Offset: 99}, nil, 3},
		{"Saved first page", jobPorts.ListOptions{StageTemplateID: saved, Limit: 1}, []string{ids[1]}, 2},
		{"Saved second page", jobPorts.ListOptions{StageTemplateID: saved, Limit: 1, Offset: 1}, []string{ids[0]}, 2},
		{"Saved past end", jobPorts.ListOptions{StageTemplateID: saved, Limit: 1, Offset: 99}, nil, 2},
		{"Applied active", jobPorts.ListOptions{StageTemplateID: applied, Limit: 2}, []string{ids[2]}, 1},
		{"Saved archived", jobPorts.ListOptions{StageTemplateID: saved, Status: "archived", Limit: 1}, []string{ids[3]}, 1},
		{"Applied archived", jobPorts.ListOptions{StageTemplateID: applied, Status: "archived", Limit: 1}, []string{ids[4]}, 1},
		{"Saved including archived", jobPorts.ListOptions{StageTemplateID: saved, Status: "all", Limit: 2}, []string{ids[3], ids[1]}, 3},
		{"archived without Stage", jobPorts.ListOptions{Status: "archived", Limit: 2}, []string{ids[4], ids[3]}, 2},
		{"foreign Stage", jobPorts.ListOptions{StageTemplateID: fixture.foreignStage, Limit: 1}, nil, 0},
		{"nonexistent Stage", jobPorts.ListOptions{StageTemplateID: uuid.NewString(), Limit: 1, Offset: 99}, nil, 0},
	}
}
