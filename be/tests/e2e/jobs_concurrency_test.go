//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrency regressions for the pipeline writes. Every one of these used to
// fail — not by erroring, but by silently losing a write:
//
//   - PATCH read the whole card, mutated it in memory and wrote every column
//     back, so a rename carried a stale applied_at / current column over a move
//     that had just committed, and two PATCHes overwrote each other's fields;
//   - Move decided "already in this column" from a copy read before the row
//     lock, so one of two concurrent moves could report success and do nothing;
//   - AddStage completed only the row its stale copy pointed at, leaving a card
//     with two active history rows.
//
// They run against the real migrated schema, where the row locks actually lock.
//
// Nothing in a worker goroutine touches *testing.T: require/assert end a test
// by calling FailNow, which is only valid on the goroutine running the test, so
// a failing request would abort the wrong goroutine and leave the test hanging
// or passing on a half-finished run. Workers return their status and error as
// data; every assertion happens back in the test goroutine.

// httpCall is one request a concurrent worker issues.
type httpCall struct {
	method string
	path   string
	body   map[string]interface{}
	token  string
}

// callResult is what a worker brings back.
type callResult struct {
	status int
	body   string
	err    error
}

// performCall issues one request without using *testing.T.
func performCall(call httpCall) callResult {
	var payload io.Reader
	if call.body != nil {
		raw, err := json.Marshal(call.body)
		if err != nil {
			return callResult{err: fmt.Errorf("marshal %s %s: %w", call.method, call.path, err)}
		}
		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(call.method, serverURL+call.path, payload)
	if err != nil {
		return callResult{err: fmt.Errorf("build %s %s: %w", call.method, call.path, err)}
	}
	req.Header.Set("Content-Type", "application/json")
	if call.token != "" {
		req.Header.Set("Authorization", call.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return callResult{err: fmt.Errorf("%s %s: %w", call.method, call.path, err)}
	}
	defer resp.Body.Close() //nolint:errcheck // body is fully read below

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return callResult{status: resp.StatusCode, err: fmt.Errorf("read %s %s: %w", call.method, call.path, err)}
	}
	return callResult{status: resp.StatusCode, body: string(raw)}
}

// runConcurrentCalls releases every call at the same instant, so the requests
// actually overlap instead of queuing behind each other's setup, and returns
// the results in the order the calls were given.
func runConcurrentCalls(calls ...httpCall) []callResult {
	results := make([]callResult, len(calls))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)
		go func(i int, call httpCall) {
			defer wg.Done()
			<-start
			results[i] = performCall(call)
		}(i, call)
	}
	close(start)
	wg.Wait()
	return results
}

// requireAllStatus asserts, in the test goroutine, that every concurrent call
// came back with the expected status. A lost write that surfaces as a 409 or a
// 500 is a failure of the thing these tests are about, so no result is allowed
// to go unchecked.
func requireAllStatus(t *testing.T, want int, results []callResult) {
	t.Helper()
	for i, res := range results {
		require.NoErrorf(t, res.err, "call %d", i)
		require.Equalf(t, want, res.status, "call %d: body %s", i, res.body)
	}
}

// countActiveStages returns how many history rows of a card are still open.
func countActiveStages(t *testing.T, jobID string) int {
	t.Helper()
	var count int
	err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM job_stages WHERE job_id = $1 AND status = 'active'`, jobID).Scan(&count)
	require.NoError(t, err)
	return count
}

// jobRow reads the card's pipeline state straight from the database.
func jobRow(t *testing.T, jobID string) (currentTemplateID string, appliedAt *string, title, source string) {
	t.Helper()
	var src *string
	err := pool.QueryRow(context.Background(),
		`SELECT current_stage_template_id, applied_at::text, title, source FROM jobs WHERE id = $1`, jobID).
		Scan(&currentTemplateID, &appliedAt, &title, &src)
	require.NoError(t, err)
	if src != nil {
		source = *src
	}
	return currentTemplateID, appliedAt, title, source
}

func TestIntegrationConcurrentPatchDoesNotRevertAMove(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "concurrent-patch-move@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
	jobID := job["id"].(string)

	results := runConcurrentCalls(
		httpCall{
			method: http.MethodPost,
			path:   "/api/v1/jobs/" + jobID + "/move",
			body:   map[string]interface{}{"stage_template_id": cols["Applied"]},
			token:  token,
		},
		httpCall{
			method: http.MethodPatch,
			path:   "/api/v1/jobs/" + jobID,
			body:   map[string]interface{}{"title": "Senior Engineer"},
			token:  token,
		},
	)
	requireAllStatus(t, http.StatusOK, results)

	currentTemplate, appliedAt, title, _ := jobRow(t, jobID)
	assert.Equal(t, cols["Applied"], currentTemplate, "the rename must not drag the card back to its old column")
	assert.NotNil(t, appliedAt, "the rename must not wipe the applied timestamp the move stamped")
	assert.Equal(t, "Senior Engineer", title, "the rename itself must stick")
	assert.Equal(t, 1, countActiveStages(t, jobID))
}

func TestIntegrationConcurrentPatchesKeepBothFields(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "concurrent-patches@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
	jobID := job["id"].(string)

	results := runConcurrentCalls(
		httpCall{
			method: http.MethodPatch,
			path:   "/api/v1/jobs/" + jobID,
			body:   map[string]interface{}{"title": "Staff Engineer"},
			token:  token,
		},
		httpCall{
			method: http.MethodPatch,
			path:   "/api/v1/jobs/" + jobID,
			body:   map[string]interface{}{"source": "LinkedIn"},
			token:  token,
		},
	)
	requireAllStatus(t, http.StatusOK, results)

	_, _, title, source := jobRow(t, jobID)
	assert.Equal(t, "Staff Engineer", title, "each PATCH writes only its own fields")
	assert.Equal(t, "LinkedIn", source)
}

func TestIntegrationConcurrentMovesLeaveOneActiveStage(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "concurrent-moves@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
	jobID := job["id"].(string)

	targets := []string{cols["Applied"], cols["Screening"], cols["Technical Interview"]}
	moves := make([]httpCall, 0, len(targets))
	for _, target := range targets {
		moves = append(moves, httpCall{
			method: http.MethodPost,
			path:   "/api/v1/jobs/" + jobID + "/move",
			body:   map[string]interface{}{"stage_template_id": target},
			token:  token,
		})
	}

	// Repeat the contention: which move wins the row lock differs from round to
	// round, and the invariant has to survive every ordering — including the one
	// where a move lands in the column the card already sits in.
	const rounds = 5
	for round := 0; round < rounds; round++ {
		requireAllStatus(t, http.StatusOK, runConcurrentCalls(moves...))

		currentTemplate, appliedAt, _, _ := jobRow(t, jobID)
		assert.Containsf(t, targets, currentTemplate, "round %d: the card ends up in one of the requested columns", round)
		assert.NotNilf(t, appliedAt, "round %d: leaving the first column stamps applied_at", round)
		assert.Equalf(t, 1, countActiveStages(t, jobID), "round %d: a card has at most one open history row", round)

		// Every move that wrote appended history; the orders stay unique.
		var distinctOrders, rows int
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT COUNT(DISTINCT "order"), COUNT(*) FROM job_stages WHERE job_id = $1`, jobID).
			Scan(&distinctOrders, &rows))
		assert.Equalf(t, rows, distinctOrders, "round %d: history orders must not collide", round)
	}

	// The stamp is written once and then carried, however many moves raced.
	_, appliedAt, _, _ := jobRow(t, jobID)
	require.NotNil(t, appliedAt)
}

func TestIntegrationConcurrentAddStageLeavesOneActiveStage(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "concurrent-addstage@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
	jobID := job["id"].(string)

	adds := make([]httpCall, 0, 2)
	for _, target := range []string{cols["Screening"], cols["Technical Interview"]} {
		adds = append(adds, httpCall{
			method: http.MethodPost,
			path:   "/api/v1/jobs/" + jobID + "/stages",
			body:   map[string]interface{}{"stage_template_id": target},
			token:  token,
		})
	}

	const rounds = 5
	for round := 0; round < rounds; round++ {
		requireAllStatus(t, http.StatusCreated, runConcurrentCalls(adds...))

		assert.Equalf(t, 1, countActiveStages(t, jobID), "round %d: "+
			"the second add must complete the first one's row instead of adding a second active stage", round)

		// The card's pointer names a row that belongs to it and is the open one.
		var pointedStatus string
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT s.status FROM jobs j JOIN job_stages s ON s.id = j.current_stage_id WHERE j.id = $1`, jobID).
			Scan(&pointedStatus))
		assert.Equalf(t, "active", pointedStatus, "round %d: current_stage_id must point at the open row", round)
	}
}

func TestIntegrationMoveToTheSameColumnIsANoOp(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "move-noop@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	job := createTestJob(t, token, "Engineer", cols["Applied"], "")
	jobID := job["id"].(string)

	var before int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM job_stages WHERE job_id = $1`, jobID).Scan(&before))

	resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+jobID+"/move",
		map[string]interface{}{"stage_template_id": cols["Applied"]}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	var after int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM job_stages WHERE job_id = $1`, jobID).Scan(&after))
	assert.Equal(t, before, after, "moving a card where it already is appends no history")
}

// Move answers from the row as it stands after the write, so a rename that
// commits while the move is in flight is reported back instead of the stale
// title the move read before it took the lock — the frontend caches that
// response, so a stale copy makes the user's edit disappear on screen.
//
// What this test can assert is bounded by the race it runs. Four orderings are
// all legal here, including "the PATCH commits after the move already re-read",
// so the returned title is pinned to the two values it may legitimately carry
// rather than to the stored one. The pipeline columns are different: only the
// move writes them, so the response must report exactly what it wrote — which
// is what guards the re-read against losing the move's own change. The
// deterministic version of the stale-title regression lives in the service
// tests, where the repository can be made to hand out a renamed row mid-move.
func TestIntegrationMoveResponseMatchesTheStoredRow(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "move-response-fresh@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	// Includes a round that moves the card where it already is, so the no-op
	// path answers from a fresh read too.
	columns := []string{cols["Applied"], cols["Screening"], cols["Wishlist"]}

	for round := 0; round < 6; round++ {
		job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
		jobID := job["id"].(string)
		target := columns[round%len(columns)]
		renamed := fmt.Sprintf("Renamed in round %d", round)

		results := runConcurrentCalls(
			httpCall{
				method: http.MethodPost,
				path:   "/api/v1/jobs/" + jobID + "/move",
				body:   map[string]interface{}{"stage_template_id": target},
				token:  token,
			},
			httpCall{
				method: http.MethodPatch,
				path:   "/api/v1/jobs/" + jobID,
				body:   map[string]interface{}{"title": renamed},
				token:  token,
			},
		)
		requireAllStatus(t, http.StatusOK, results)

		var moveResponse struct {
			Title                  string  `json:"title"`
			CurrentStageTemplateID *string `json:"current_stage_template_id"`
		}
		require.NoError(t, json.Unmarshal([]byte(results[0].body), &moveResponse))

		storedColumn, _, storedTitle, _ := jobRow(t, jobID)
		assert.Equalf(t, renamed, storedTitle, "round %d: the rename itself must stick", round)
		assert.Containsf(t, []string{"Engineer", renamed}, moveResponse.Title,
			"round %d: the move invented a title that was never stored", round)
		require.NotNil(t, moveResponse.CurrentStageTemplateID)
		assert.Equalf(t, target, *moveResponse.CurrentStageTemplateID,
			"round %d: the re-read must not lose the move's own column change", round)
		assert.Equalf(t, storedColumn, *moveResponse.CurrentStageTemplateID,
			"round %d: the response must agree with the stored column", round)
	}
}
