//go:build integration

package interviewprep

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDispatchGuardBlocksHideUntilBodyWritten(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	snapshot, err := f.store.Snapshot(ctx, f.owner, f.job, f.resume, "en")
	require.NoError(t, err)
	release, err := f.store.guardDispatch(ctx, Operation{Owner: f.owner, JobID: f.job}, snapshot)
	require.NoError(t, err)
	defer release()
	attempted := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(attempted)
		_, e := f.pool.Exec(ctx, `UPDATE resume_skills SET name='Hidden after dispatch' WHERE resume_builder_id=$1`, f.resume)
		done <- e
	}()
	<-attempted
	select {
	case e := <-done:
		require.NoError(t, e)
		t.Fatal("source changed while body dispatch guard held")
	case <-time.After(80 * time.Millisecond):
	}
	require.NoError(t, release())
	require.NoError(t, <-done)
	_, err = f.store.guardDispatch(ctx, Operation{Owner: f.owner, JobID: f.job}, snapshot)
	require.ErrorIs(t, err, ErrSourceChanged)
}
func TestAdmissionBudgetRejectsBeforePersistenceAndStillSummarizes(t *testing.T) {
	f := setup(t, 1)
	ctx := t.Context()
	_, err := f.pool.Exec(ctx, `UPDATE jobs SET description=$1 WHERE id=$2 AND user_id=$3`, strings.Repeat("\x01", 7000), f.job, f.owner)
	require.NoError(t, err)
	_, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	h, err := f.store.History(ctx, f.owner, f.job, 0, true)
	require.NoError(t, err)
	ss, err := f.store.StartSession(ctx, f.owner, f.job, h.Versions[0].ID, uuid.NewString())
	require.NoError(t, err)
	snapshot, err := loadSnapshot(ctx, f.pool, f.owner, f.job, ss.VersionID)
	require.NoError(t, err)
	rejected := 0
	for _, q := range ss.Questions {
		request := SaveAnswer{Key: uuid.NewString(), QuestionID: q.ID, Revision: ss.Revision, Text: strings.Repeat("\x01", 12000)}
		_, err = f.store.SaveAnswer(ctx, f.owner, f.job, ss.ID, request)
		if err == nil {
			ss, err = f.store.Session(ctx, f.owner, f.job, ss.ID)
			require.NoError(t, err)
			continue
		}
		rejected++
		require.ErrorIs(t, err, ErrCapacity)
		unchanged, e := f.store.Session(ctx, f.owner, f.job, ss.ID)
		require.NoError(t, e)
		require.Equal(t, ss.Revision, unchanged.Revision)
		require.Len(t, unchanged.Answers, len(ss.Answers))
		request.Key = uuid.NewString()
		request.Text = ""
		request.Skip = true
		ss, err = f.store.SaveAnswer(ctx, f.owner, f.job, ss.ID, request)
		require.NoError(t, err)
	}
	require.Positive(t, rejected)
	ss, err = f.store.Action(ctx, f.owner, f.job, ss.ID, SessionAction{Key: uuid.NewString(), Action: "finish", Revision: ss.Revision})
	require.NoError(t, err)
	input := compactInput(snapshot, ss.Questions, ss.Answers)
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	require.Less(t, len(raw), maxAIInputBytes)
	require.NotContains(t, string(raw), `"feedback"`)
	require.Contains(t, string(raw), ss.Questions[0].Question)
	op, err := f.svc.Review(ctx, f.owner, f.job, ss.ID, "", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	var status string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT status FROM interview_operations WHERE id=$1`, op.ID).Scan(&status))
	require.Equal(t, "succeeded", status)
	ss, err = f.store.Session(ctx, f.owner, f.job, ss.ID)
	require.NoError(t, err)
	require.NotNil(t, ss.Summary)

}
func TestFeedbackReplayCannotCrossOwnedSessions(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	h, err := f.store.History(ctx, f.owner, f.job, 0, true)
	require.NoError(t, err)
	first, err := f.store.StartSession(ctx, f.owner, f.job, h.Versions[0].ID, uuid.NewString())
	require.NoError(t, err)
	second, err := f.store.StartSession(ctx, f.owner, f.job, h.Versions[0].ID, uuid.NewString())
	require.NoError(t, err)
	first, err = f.store.SaveAnswer(ctx, f.owner, f.job, first.ID, SaveAnswer{Key: uuid.NewString(), QuestionID: first.Questions[0].ID, Revision: 0, Text: "My real answer"})
	require.NoError(t, err)
	key := uuid.NewString()
	_, err = f.svc.Review(ctx, f.owner, f.job, first.ID, first.Answers[0].ID, key)
	require.NoError(t, err)
	_, err = f.svc.Review(ctx, f.owner, f.job, second.ID, first.Answers[0].ID, key)
	require.ErrorIs(t, err, ErrNotFound)
}
func TestTelemetryReportsCommittedSourceChanged(t *testing.T) {
	f := setup(t, 3)
	ctx := context.Background()
	f.client.hook = func() {
		_, err := f.pool.Exec(ctx, `UPDATE resume_skills SET name='Changed after dispatch' WHERE resume_builder_id=$1`, f.resume)
		require.NoError(t, err)
	}
	_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	var log bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&log, nil)))
	defer slog.SetDefault(previous)
	require.NoError(t, f.svc.RunOnce(ctx))
	require.Contains(t, log.String(), `"outcome":"SOURCE_CHANGED"`)
	require.NotContains(t, log.String(), "Changed after dispatch")
}

func TestWorkerNormalizesJSONBBeforeProviderBudget(t *testing.T) {
	f := setup(t, 1)
	ctx := t.Context()
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	var databaseInput []byte
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT input FROM interview_operations WHERE user_id=$1 AND job_id=$2 AND id=$3`, f.owner, f.job, op.ID).Scan(&databaseInput))
	var input AIInput
	require.NoError(t, json.Unmarshal(databaseInput, &input))
	compact, err := json.Marshal(input)
	require.NoError(t, err)
	require.Greater(t, len(databaseInput), len(compact))
	require.NoError(t, f.svc.RunOnce(ctx))
	require.Equal(t, string(compact), f.client.lastInput)
}
