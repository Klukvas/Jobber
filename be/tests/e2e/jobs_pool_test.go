//go:build integration

package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	commentRepo "github.com/andreypavlenko/jobber/modules/comments/repository"
	companyRepo "github.com/andreypavlenko/jobber/modules/companies/repository"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	jobRepo "github.com/andreypavlenko/jobber/modules/jobs/repository"
	jobService "github.com/andreypavlenko/jobber/modules/jobs/service"
	rbRepo "github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	resumeRepo "github.com/andreypavlenko/jobber/modules/resumes/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A pipeline write holds one pooled connection — and the card's row lock — for
// the whole transaction. Anything it reads through a pool-backed repository in
// that window asks the pool for a *second* connection while the lock holder is
// the one that needs it to finish, so as many simultaneous writers as there are
// connections is enough to starve the pool with nobody able to make progress.
//
// The suite's shared pool (10 connections) hides that: a stray read just gets
// another connection. These tests pin the service to a one-connection pool, so
// every read that belongs inside the transaction has to be served by the
// transaction's own connection or block forever, and bound each call with a
// context deadline so a block fails the test instead of parking the suite.

// singleConnCallTimeout bounds every call issued against the one-connection
// pool. Each of these calls is a handful of statements against a local
// container, so a deadline this generous can only be reached by a write that is
// waiting for a connection it can never get.
const singleConnCallTimeout = 10 * time.Second

// singleConnJobService builds a JobService on a fresh one-connection pool
// against the same container as the suite's shared pool.
//
// Every repository is wired to that same pool on purpose: a pool read from any
// of them inside a transaction must surface as a block, not as a lucky second
// connection.
func singleConnJobService(t *testing.T) *jobService.JobService {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(pool.Config().ConnString())
	require.NoError(t, err)
	cfg.MaxConns = 1

	single, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(single.Close)

	return jobService.NewJobService(
		single,
		jobRepo.NewJobRepository(single),
		jobRepo.NewJobStageRepository(single),
		jobRepo.NewStageTemplateRepository(single),
		companyRepo.NewCompanyRepository(single),
		resumeRepo.NewResumeRepository(single),
		rbRepo.NewResumeBuilderRepository(single),
		commentRepo.NewCommentRepository(single),
		nil, // logger: the service falls back to a no-op one
		nil, // limit checker: only Create consults it
		nil, // cache invalidator: only the job PATCH/delete paths use it
	)
}

// boundedCtx returns a context that expires well before a blocked connection
// acquire could hang the suite.
func boundedCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), singleConnCallTimeout)
	t.Cleanup(cancel)
	return ctx
}

// addStageOnSingleConn appends a history row through the constrained service.
func addStageOnSingleConn(t *testing.T, svc *jobService.JobService, userID, jobID, columnID string) *model.JobStageDTO {
	t.Helper()
	dto, err := svc.AddStage(boundedCtx(t), userID, jobID, &model.AddStageRequest{StageTemplateID: columnID})
	require.NoError(t, err, "AddStage must finish on the connection its transaction already holds")
	require.NotNil(t, dto)
	return dto
}

// classifyColumn sets a column's semantic kind through the API — what the user
// does in the stage UI for a column they created themselves.
func classifyColumn(t *testing.T, token, columnID, kind string) {
	t.Helper()
	resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+columnID,
		map[string]interface{}{"kind": kind}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// cardPipelineState is the card's pipeline state read straight from the
// database: the history row and column it points at, and whether the
// application counts as out.
type cardPipelineState struct {
	CurrentStageID    *string
	CurrentTemplateID *string
	AppliedAt         *string
}

func readCardPipelineState(t *testing.T, jobID string) cardPipelineState {
	t.Helper()
	var state cardPipelineState
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT current_stage_id, current_stage_template_id, applied_at::text FROM jobs WHERE id = $1`, jobID).
		Scan(&state.CurrentStageID, &state.CurrentTemplateID, &state.AppliedAt))
	return state
}

func TestIntegrationStageWritesRunOnASingleConnection(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "stage-pool@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)

	// "Screening Call" is a column the user invented and has not classified, and
	// it sits first, so neither its kind nor its position makes a card count as
	// applied. The card therefore keeps a NULL applied_at through every append
	// below — which is what lets the re-activation at the end prove the
	// column's kind was read inside the transaction.
	cols := seedPipelineWithKinds(t, userID,
		pipelineColumn{Name: "Screening Call", Kind: "other"},
		pipelineColumn{Name: "Wishlist", Kind: "saved"},
	)

	job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
	jobID := job["id"].(string)

	svc := singleConnJobService(t)

	// A bookmarked card carries no history yet, so the first append is what
	// gives it an open row. The two after it run with the card already pointing
	// at one — the state in which the write has to resolve that row's column
	// for its log line while holding the lock.
	firstCall := addStageOnSingleConn(t, svc, userID, jobID, cols["Screening Call"])
	assert.Equal(t, "Screening Call", firstCall.StageName)
	backToWishlist := addStageOnSingleConn(t, svc, userID, jobID, cols["Wishlist"])
	assert.Equal(t, "Wishlist", backToWishlist.StageName)
	secondCall := addStageOnSingleConn(t, svc, userID, jobID, cols["Screening Call"])
	assert.Equal(t, "Screening Call", secondCall.StageName)
	assert.Equal(t, 1, countActiveStages(t, jobID))

	require.Nil(t, readCardPipelineState(t, jobID).AppliedAt,
		"neither column means the application is out yet")

	// The user classifies the column: from here on, landing in it means the
	// application is out.
	classifyColumn(t, token, cols["Screening Call"], "interview")

	// Re-activating the first call closes the open row, moves the card back and
	// — because the column now implies it — stamps applied_at. All three
	// decisions are taken from the column read inside the transaction.
	active := "active"
	dto, err := svc.UpdateStage(boundedCtx(t), userID, jobID, firstCall.ID,
		&model.UpdateStageRequest{Status: &active})
	require.NoError(t, err, "UpdateStage must finish on the connection its transaction already holds")
	assert.Equal(t, "Screening Call", dto.StageName, "the DTO is still enriched with the column name")

	assert.Equal(t, 1, countActiveStages(t, jobID), "the re-activated row is the card's only open one")
	secondStatus, _ := stageState(t, secondCall.ID)
	assert.Equal(t, "completed", secondStatus, "the row that was open got closed")

	state := readCardPipelineState(t, jobID)
	require.NotNil(t, state.CurrentStageID)
	require.NotNil(t, state.CurrentTemplateID)
	assert.Equal(t, firstCall.ID, *state.CurrentStageID, "the card points at the row it was moved back to")
	assert.Equal(t, cols["Screening Call"], *state.CurrentTemplateID)
	assert.NotNil(t, state.AppliedAt,
		"the column's kind, read inside the transaction, decides applied_at on a re-activation")
}
