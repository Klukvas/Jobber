//go:build integration

package interviewprep

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type testLimits struct {
	pool    *pgxpool.Pool
	maximum int
}

func (l testLimits) ResourceLimit(ctx context.Context, _ string, _ string) (int, error) {
	var n int
	err := l.pool.QueryRow(ctx, `SELECT $1::int`, l.maximum).Scan(&n)
	return n, err
}

type testClient struct {
	mu        sync.Mutex
	calls     int
	lastInput string
	mode      string
	hook      func()
}

func (c *testClient) Interview(ctx context.Context, kind string, raw []byte, dispatched func() error) ([]byte, error) {
	if dispatched != nil {
		if err := dispatched(); err != nil {
			return nil, err
		}
	}
	c.mu.Lock()
	c.calls++
	c.lastInput = string(raw)
	mode, hook := c.mode, c.hook
	c.mu.Unlock()
	if hook != nil {
		hook()
	}
	if mode == "slow" {
		select {
		case <-time.After(8 * time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if mode == "fail" {
		return nil, errors.New("synthetic provider failure")
	}
	if mode == "timeout" {
		return nil, context.DeadlineExceeded
	}
	if mode == "malformed" {
		return []byte(`{"bad":true}`), nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var input AIInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}
	if kind == "plan" {
		return json.Marshal(syntheticPlan(input.Snapshot.Facts))
	}
	return json.Marshal(syntheticReview(reviewFacts(input), kind == "feedback" && input.Question.Kind != "followup"))
}
func (c *testClient) set(mode string) { c.mu.Lock(); defer c.mu.Unlock(); c.mode = mode }

type fixture struct {
	pool                      *pgxpool.Pool
	store                     *Store
	svc                       *Service
	client                    *testClient
	owner, other, job, resume string
	schema                    string
}

func setup(t *testing.T, maxConns int32) *fixture {
	t.Helper()
	dsn := os.Getenv("INTERVIEW_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTERVIEW_TEST_DATABASE_URL must point to isolated synthetic PostgreSQL")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "interview_test", cfg.ConnConfig.Database)
	require.Equal(t, uint16(55481), cfg.ConnConfig.Port)
	require.Equal(t, "127.0.0.1", cfg.ConnConfig.Host)
	admin, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	_, err = admin.Exec(t.Context(), `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public; ALTER EXTENSION "uuid-ossp" SET SCHEMA public`)
	require.NoError(t, err)
	schema := "interview_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(t.Context(), `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	cfg.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
		_, e := admin.Exec(context.WithoutCancel(t.Context()), `DROP SCHEMA `+schema+` CASCADE`)
		require.NoError(t, e)
		admin.Close()
	})
	f := &fixture{pool: pool, store: NewStore(pool), client: &testClient{}, owner: uuid.NewString(), other: uuid.NewString(), job: uuid.NewString(), resume: uuid.NewString(), schema: schema}
	files, err := filepath.Glob("../../migrations/*.up.sql")
	require.NoError(t, err)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "000049") {
			continue
		}
		raw, e := os.ReadFile(file)
		require.NoError(t, e)
		_, e = pool.Exec(t.Context(), string(raw))
		require.NoError(t, e, file)
	}
	if _, err := os.Stat("../../migrations/000048_comments_touch_updated_at_on_content_only.up.sql"); os.IsNotExist(err) {
		main48, e := os.ReadFile("testdata/current-main-000048.sql")
		require.NoError(t, e)
		_, e = pool.Exec(t.Context(), string(main48))
		require.NoError(t, e)
	}

	_, err = pool.Exec(t.Context(), `INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Synthetic Candidate','not-a-login'),($3,$4,'Other Synthetic','not-a-login')`, f.owner, f.owner+"@example.invalid", f.other, f.other+"@example.invalid")
	require.NoError(t, err)
	stage := uuid.NewString()
	_, err = pool.Exec(t.Context(), `INSERT INTO stage_templates(id,user_id,name,"order") VALUES($1,$2,'Saved',0)`, stage, f.owner)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO jobs(id,user_id,title,description,current_stage_template_id) VALUES($1,$2,'Synthetic Go Engineer','Build Go services and collaborate with a team',$3)`, f.job, f.owner, stage)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO resume_builders(id,user_id,title) VALUES($1,$2,'Synthetic Builder Resume')`, f.resume, f.owner)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `UPDATE jobs SET resume_builder_id=$1 WHERE id=$2`, f.resume, f.job)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO resume_skills(resume_builder_id,name) VALUES($1,'Go');`, f.resume)
	require.NoError(t, err)
	raw, err := os.ReadFile("../../migrations/000049_interview_preparation.up.sql")
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), string(raw))
	require.NoError(t, err)
	f.svc = NewService(f.store, f.client, testLimits{pool, -1})
	return f
}
func (f *fixture) generate(t *testing.T) Version {
	t.Helper()
	ctx := t.Context()
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	var id string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT version_id::text FROM interview_operations WHERE id=$1`, op.ID).Scan(&id))
	v, err := f.store.Version(ctx, f.owner, f.job, id)
	require.NoError(t, err)
	return v
}
func TestFullPracticeAndOwnership(t *testing.T) {
	f := setup(t, 5)
	ctx := t.Context()
	v := f.generate(t)
	_, err := f.store.Version(ctx, f.other, f.job, v.ID)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = f.store.History(ctx, f.other, f.job, 0, true)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = f.store.Studied(ctx, f.other, f.job, v.ID, "q1", true)
	require.ErrorIs(t, err, ErrNotFound)
	studied, err := f.store.Studied(ctx, f.owner, f.job, v.ID, "q1", true)
	require.NoError(t, err)
	require.Equal(t, []string{"q1"}, studied.Studied)
	session, key := Session{}, uuid.NewString()
	session, err = f.store.StartSession(ctx, f.owner, f.job, v.ID, key)
	require.NoError(t, err)
	replay, err := f.store.StartSession(ctx, f.owner, f.job, v.ID, key)
	require.NoError(t, err)
	require.Equal(t, session.ID, replay.ID)
	_, err = f.store.Session(ctx, f.other, f.job, session.ID)
	require.ErrorIs(t, err, ErrNotFound)
	req := SaveAnswer{Key: uuid.NewString(), QuestionID: "q1", Text: "I investigated latency and measured a change", Revision: session.Revision}
	saved, err := f.store.SaveAnswer(ctx, f.owner, f.job, session.ID, req)
	require.NoError(t, err)
	require.Len(t, saved.Answers, 1)
	replay, err = f.store.SaveAnswer(ctx, f.owner, f.job, session.ID, req)
	require.NoError(t, err)
	require.Len(t, replay.Answers, 1)
	req.Text = "different"
	_, err = f.store.SaveAnswer(ctx, f.owner, f.job, session.ID, req)
	require.ErrorIs(t, err, ErrConflict)
	_, err = f.svc.Review(ctx, f.other, f.job, session.ID, saved.Answers[0].ID, uuid.NewString())
	require.ErrorIs(t, err, ErrNotFound)
	f.client.set("fail")
	op, err := f.svc.Review(ctx, f.owner, f.job, session.ID, saved.Answers[0].ID, uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	still, err := f.store.Session(ctx, f.owner, f.job, session.ID)
	require.NoError(t, err)
	require.Equal(t, saved.Answers[0].Text, still.Answers[0].Text)
	require.Nil(t, still.Answers[0].Feedback)
	_, err = f.svc.Retry(ctx, f.other, f.job, op.ID)
	require.ErrorIs(t, err, ErrNotFound)
	f.client.set("")
	_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	session, err = f.store.Session(ctx, f.owner, f.job, session.ID)
	require.NoError(t, err)
	require.NotNil(t, session.Answers[0].Feedback)
	require.Len(t, session.Questions, 4)
	for _, question := range session.Questions[1:] {
		session, err = f.store.SaveAnswer(ctx, f.owner, f.job, session.ID, SaveAnswer{Key: uuid.NewString(), QuestionID: question.ID, Text: "I explained my personal contribution", Revision: session.Revision})
		require.NoError(t, err)
	}
	require.Equal(t, "followup:"+session.Answers[0].ID, session.Answers[3].QuestionID)
	session, err = f.store.Action(ctx, f.owner, f.job, session.ID, SessionAction{Key: uuid.NewString(), Action: "pause", Revision: session.Revision})
	require.NoError(t, err)
	session, err = f.store.Action(ctx, f.owner, f.job, session.ID, SessionAction{Key: uuid.NewString(), Action: "resume", Revision: session.Revision})
	require.NoError(t, err)
	session, err = f.store.Action(ctx, f.owner, f.job, session.ID, SessionAction{Key: uuid.NewString(), Action: "finish", Revision: session.Revision})
	require.NoError(t, err)
	_, err = f.svc.Review(ctx, f.owner, f.job, session.ID, "", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	session, err = f.store.Session(ctx, f.owner, f.job, session.ID)
	require.NoError(t, err)
	require.NotNil(t, session.Summary)
	require.Len(t, session.Summary.Criteria, 3)
	v2 := f.generate(t)
	require.NotEqual(t, v.ID, v2.ID)
	history, err := f.store.History(ctx, f.owner, f.job, 0, false)
	require.NoError(t, err)
	require.Len(t, history.Versions, 2)
	require.False(t, history.AIAvailable)
}
func TestAtomicQuotaDedupeAndReplay(t *testing.T) {
	f := setup(t, 1)
	ctx := t.Context()
	f.svc.limits = testLimits{f.pool, 1}
	var wg sync.WaitGroup
	ids := make(chan string, 10)
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op, e := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
			ids <- op.ID
			errs <- e
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	unique := map[string]bool{}
	for id := range ids {
		unique[id] = true
	}
	require.Len(t, unique, 1)
	var usage int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usage))
	require.Equal(t, 1, usage)
	// Legacy endpoint reservations use the exact same ledger and advisory lock.
	svc := subService.NewSubscriptionService(subRepo.NewSubscriptionRepository(f.pool), nil, subService.BillingConfig{})
	scoped := subService.AIRequestContext(ctx)
	require.ErrorIs(t, svc.CheckLimit(scoped, f.owner, "ai"), subModel.ErrLimitReached)
	// Once terminal, a different request may regenerate, subject to the same quota.
	require.NoError(t, f.svc.RunOnce(ctx))
	_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.ErrorIs(t, err, subModel.ErrLimitReached)
	key := uuid.NewString()
	f.svc.limits = testLimits{f.pool, -1}
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", key)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE resume_builders SET title='Changed title' WHERE id=$1`, f.resume)
	require.NoError(t, err)
	replay, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", key)
	require.NoError(t, err)
	require.Equal(t, op.ID, replay.ID)
	_, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "ru", key)
	require.ErrorIs(t, err, ErrConflict)
}
func TestVisibilityStalenessAndLateResult(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	for _, section := range []string{"custom", "custom_sections"} {
		_, err := f.pool.Exec(ctx, `DELETE FROM resume_section_orders WHERE resume_builder_id=$1`, f.resume)
		require.NoError(t, err)
		_, err = f.pool.Exec(ctx, `INSERT INTO resume_custom_sections(resume_builder_id,title,content) VALUES($1,'Hidden marker','PRIVATE NEVER SEND')`, f.resume)
		require.NoError(t, err)
		_, err = f.pool.Exec(ctx, `INSERT INTO resume_section_orders(resume_builder_id,section_key,is_visible) VALUES($1,$2,false) ON CONFLICT(resume_builder_id,section_key) DO UPDATE SET is_visible=false`, f.resume, section)
		require.NoError(t, err)
		snapshot, err := f.store.Snapshot(ctx, f.owner, f.job, f.resume, "en")
		require.NoError(t, err)
		raw, _ := json.Marshal(snapshot)
		require.NotContains(t, string(raw), "PRIVATE NEVER SEND")
		require.NotContains(t, string(raw), "Hidden marker")
		f.generate(t)
		require.NotContains(t, f.client.lastInput, "PRIVATE NEVER SEND")
		require.NotContains(t, f.client.lastInput, "Hidden marker")
	}
	v := f.generate(t)
	session, err := f.store.StartSession(ctx, f.owner, f.job, v.ID, uuid.NewString())
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE resume_skills SET name='Rust' WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	stale, err := f.store.Version(ctx, f.owner, f.job, v.ID)
	require.NoError(t, err)
	require.True(t, stale.Stale)
	session, err = f.store.SaveAnswer(ctx, f.owner, f.job, session.ID, SaveAnswer{Key: uuid.NewString(), QuestionID: "q1", Text: "My saved answer", Revision: 0})
	require.NoError(t, err)
	_, err = f.svc.Review(ctx, f.owner, f.job, session.ID, session.Answers[0].ID, uuid.NewString())
	require.ErrorIs(t, err, ErrSourceChanged)
	var code string
	// Mutation during the provider call cannot publish an already stale plan.
	f.client.hook = func() {
		_, e := f.pool.Exec(ctx, `UPDATE resume_skills SET name='Python' WHERE resume_builder_id=$1`, f.resume)
		require.NoError(t, e)
	}
	pending, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT error_code FROM interview_operations WHERE id=$1`, pending.ID).Scan(&code))
	require.Equal(t, "SOURCE_CHANGED", code)
	_, err = f.pool.Exec(ctx, `DELETE FROM resume_builders WHERE id=$1`, f.resume)
	require.NoError(t, err)
	stale, err = f.store.Version(ctx, f.owner, f.job, v.ID)
	require.NoError(t, err)
	require.True(t, stale.Stale)
	require.Equal(t, "Synthetic Builder Resume", stale.ResumeTitle)
}
func TestRestartLeaseFencingAndRetries(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	lease, err := f.svc.claim(ctx)
	require.NoError(t, err)
	require.Equal(t, op.ID, lease.ID)
	_, err = f.pool.Exec(ctx, `UPDATE interview_operations SET lease_until=now()-interval '1 second' WHERE id=$1`, op.ID)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	var status string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT status FROM interview_operations WHERE id=$1`, op.ID).Scan(&status))
	require.Equal(t, "interrupted", status)
	snapshot, err := f.store.Snapshot(ctx, f.owner, f.job, f.resume, "en")
	require.NoError(t, err)
	_, err = f.svc.complete(ctx, lease, AIInput{Snapshot: snapshot}, syntheticPlan(snapshot.Facts), Review{}, "")
	require.ErrorIs(t, err, ErrFenced)
	var count int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM interview_versions`).Scan(&count))
	require.Zero(t, count)
	f.client.set("malformed")
	for i := 0; i < 2; i++ {
		_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
		require.NoError(t, err)
		require.NoError(t, f.svc.RunOnce(ctx))
	}
	_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
	require.ErrorIs(t, err, ErrConflict)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage`).Scan(&count))
	require.Equal(t, 1, count)
	// New key after a terminal result can generate again; old exhausted retry
	// cannot race a new active plan for the same source.
	_, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
}
func TestUnavailableAndMigrationRollback(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	v := f.generate(t)
	session, err := f.store.StartSession(ctx, f.owner, f.job, v.ID, uuid.NewString())
	require.NoError(t, err)
	f.svc.client = nil
	session, err = f.store.SaveAnswer(ctx, f.owner, f.job, session.ID, SaveAnswer{Key: uuid.NewString(), QuestionID: "q1", Text: "Persist even while AI is unavailable", Revision: 0})
	require.NoError(t, err)
	_, err = f.svc.Review(ctx, f.owner, f.job, session.ID, session.Answers[0].ID, uuid.NewString())
	require.ErrorIs(t, err, ErrUnavailable)
	got, err := f.store.Session(ctx, f.owner, f.job, session.ID)
	require.NoError(t, err)
	require.Equal(t, session.Answers, got.Answers)
	raw, err := os.ReadFile("../../migrations/000049_interview_preparation.down.sql")
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, string(raw))
	require.NoError(t, err)
	var title string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT title FROM resume_builders WHERE id=$1`, f.resume).Scan(&title))
	require.Equal(t, "Synthetic Builder Resume", title)
	raw, err = os.ReadFile("../../migrations/000049_interview_preparation.up.sql")
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, string(raw))
	require.NoError(t, err)
}
func TestLegacyReservationReleaseAndConsume(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	svc := subService.NewSubscriptionService(subRepo.NewSubscriptionRepository(f.pool), nil, subService.BillingConfig{})
	scoped := subService.AIRequestContext(ctx)
	require.NoError(t, svc.CheckLimit(scoped, f.owner, "ai"))
	require.NoError(t, svc.CheckLimit(scoped, f.owner, "ai"))
	svc.FinishAIRequest(scoped)
	var count int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage`).Scan(&count))
	require.Zero(t, count)
	scoped = subService.AIRequestContext(ctx)
	require.NoError(t, svc.CheckLimit(scoped, f.owner, "ai"))
	require.NoError(t, svc.RecordAIUsage(scoped, f.owner))
	require.NoError(t, svc.RecordResumeAutofillUsage(scoped, f.owner))
	svc.FinishAIRequest(scoped)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage`).Scan(&count))
	require.Equal(t, 1, count)
}
func TestExpiredLegacyReservationAndMixedQuotaRace(t *testing.T) {
	f := setup(t, 5)
	ctx := t.Context()
	repo := subRepo.NewSubscriptionRepository(f.pool)
	id, err := repo.ReserveAI(ctx, f.owner, 1)
	require.NoError(t, err)
	f.svc.limits = testLimits{f.pool, 1}
	_, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.ErrorIs(t, err, subModel.ErrLimitReached)
	_, err = f.pool.Exec(ctx, `UPDATE ai_quota_reservations SET expires_at=now()-interval '1 second' WHERE id=$1`, id)
	require.NoError(t, err)
	require.ErrorIs(t, repo.CommitAIReservation(ctx, f.owner, id, "match_score"), subModel.ErrLimitReached)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); _, err := repo.ReserveAI(ctx, f.owner, 1); results <- err }()
	go func() {
		defer wg.Done()
		_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
		results <- err
	}()
	wg.Wait()
	close(results)
	success, limited := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, subModel.ErrLimitReached) {
			limited++
		} else {
			require.NoError(t, e)
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, limited)
}
func TestLegacyCommitFailureRetainsPendingReceipt(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	svc := subService.NewSubscriptionService(subRepo.NewSubscriptionRepository(f.pool), nil, subService.BillingConfig{})
	scope := subService.AIRequestContext(ctx)
	require.NoError(t, svc.CheckLimit(scope, f.owner, "ai"))
	_, err := f.pool.Exec(ctx, `ALTER TABLE ai_usage ADD CONSTRAINT synthetic_save_failure CHECK(false)`)
	require.NoError(t, err)
	require.Error(t, svc.RecordAIUsage(scope, f.owner))
	svc.FinishAIRequest(scope)
	var pending int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_quota_reservations WHERE user_id=$1`, f.owner).Scan(&pending))
	require.Equal(t, 1, pending)
	_, err = f.pool.Exec(ctx, `ALTER TABLE ai_usage DROP CONSTRAINT synthetic_save_failure`)
	require.NoError(t, err)
	require.NoError(t, svc.RecordAIUsage(scope, f.owner))
	require.NoError(t, svc.RecordAIUsage(scope, f.owner))
	svc.FinishAIRequest(scope)
	var count int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Equal(t, 1, count)
}
func TestIdenticalContentJobsHaveIndependentOperations(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	otherJob := uuid.NewString()
	_, err := f.pool.Exec(ctx, `INSERT INTO jobs(id,user_id,title,description,current_stage_template_id) SELECT $1,user_id,title,description,current_stage_template_id FROM jobs WHERE id=$2`, otherJob, f.job)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(2)
	operations := make(chan Operation, 2)
	errs := make(chan error, 2)
	for _, job := range []string{f.job, otherJob} {
		go func(job string) {
			defer wg.Done()
			op, e := f.svc.Generate(ctx, f.owner, job, f.resume, "en", uuid.NewString())
			operations <- op
			errs <- e
		}(job)
	}
	wg.Wait()
	close(operations)
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	ids := map[string]bool{}
	for op := range operations {
		ids[op.ID] = true
	}
	require.Len(t, ids, 2)
}
func TestNoCallRefundAndUnavailableRetry(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	f.svc.limits = testLimits{f.pool, 1}
	op, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE resume_skills SET name='Changed before provider call' WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	var count int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Zero(t, count)
	var code string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT error_code FROM interview_operations WHERE id=$1`, op.ID).Scan(&code))
	require.Equal(t, "SOURCE_CHANGED", code)
	require.Zero(t, f.client.calls)
	op, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	f.svc.client = nil
	require.NoError(t, f.svc.RunOnce(ctx))
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Zero(t, count)
	f.svc.client = f.client
	_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
	require.NoError(t, err)
	_, err = f.svc.Retry(ctx, f.owner, f.job, op.ID)
	require.NoError(t, err)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, f.svc.RunOnce(ctx))
	var versionID string
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT version_id::text FROM interview_operations WHERE id=$1`, op.ID).Scan(&versionID))
	ss, err := f.store.StartSession(ctx, f.owner, f.job, versionID, uuid.NewString())
	require.NoError(t, err)
	ss, err = f.store.SaveAnswer(ctx, f.owner, f.job, ss.ID, SaveAnswer{Key: uuid.NewString(), QuestionID: "q1", Text: "Saved answer", Revision: 0})
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `UPDATE resume_skills SET name='Changed after plan' WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	_, err = f.svc.Review(ctx, f.owner, f.job, ss.ID, ss.Answers[0].ID, uuid.NewString())
	require.ErrorIs(t, err, ErrSourceChanged)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Equal(t, 1, count)
}
func TestDuringCallSourceChangeKeepsSingleCharge(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	f.client.hook = func() {
		_, err := f.pool.Exec(ctx, `UPDATE resume_skills SET name='Changed during provider call' WHERE resume_builder_id=$1`, f.resume)
		require.NoError(t, err)
	}
	_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	var count int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM interview_versions`).Scan(&count))
	require.Zero(t, count)
	require.Equal(t, 1, f.client.calls)
}
func TestLeanBoundedHistoryPagination(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	v := f.generate(t)
	_, err := f.pool.Exec(ctx, `INSERT INTO interview_versions(id,user_id,job_id,source_resume_id,provenance_resume_id,source_hash,snapshot,plan) SELECT gen_random_uuid(),user_id,job_id,source_resume_id,provenance_resume_id,source_hash,snapshot,plan FROM interview_versions CROSS JOIN generate_series(1,21) WHERE id=$1`, v.ID)
	require.NoError(t, err)
	first, err := f.store.History(ctx, f.owner, f.job, 0, true)
	require.NoError(t, err)
	require.Len(t, first.Versions, 20)
	second, err := f.store.History(ctx, f.owner, f.job, 20, true)
	require.NoError(t, err)
	require.Len(t, second.Versions, 2)
	raw, err := json.Marshal(first)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"facts"`)
	var shape struct {
		Versions []map[string]json.RawMessage `json:"versions"`
		Sessions []map[string]json.RawMessage `json:"sessions"`
	}
	require.NoError(t, json.Unmarshal(raw, &shape))
	for _, item := range shape.Versions {
		require.NotContains(t, item, "plan")
		require.NotContains(t, item, "facts")
	}
	for _, item := range shape.Sessions {
		require.NotContains(t, item, "questions")
		require.NotContains(t, item, "summary")
		require.NotContains(t, item, "answers")
	}
	require.Less(t, len(raw), 20000)
}
func TestAutofillReservationPreservesUsageCategory(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	svc := subService.NewSubscriptionService(subRepo.NewSubscriptionRepository(f.pool), nil, subService.BillingConfig{})
	scope := subService.AIRequestContext(ctx)
	require.NoError(t, svc.CheckLimit(scope, f.owner, "ai"))
	require.NoError(t, svc.RecordResumeAutofillUsage(scope, f.owner))
	require.NoError(t, svc.RecordResumeAutofillUsage(scope, f.owner))
	svc.FinishAIRequest(scope)
	var usage string
	var count int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT usage_type FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usage))
	require.Equal(t, "resume_autofill_parse", usage)
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&count))
	require.Equal(t, 1, count)
}
