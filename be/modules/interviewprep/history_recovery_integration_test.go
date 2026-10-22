//go:build integration

package interviewprep

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOldFailedSummaryReceiptRecoveryBeyondHistoryWindow(t *testing.T) {
	f := setup(t, 3)
	ctx := t.Context()
	version := f.generate(t)
	session, err := f.store.StartSession(ctx, f.owner, f.job, version.ID, uuid.NewString())
	require.NoError(t, err)
	session, err = f.store.Action(ctx, f.owner, f.job, session.ID, SessionAction{Key: uuid.NewString(), Action: "finish", Revision: 0})
	require.NoError(t, err)
	key := uuid.NewString()
	operation, err := f.svc.Review(ctx, f.owner, f.job, session.ID, "", key)
	require.NoError(t, err)
	f.client.set("fail")
	require.NoError(t, f.svc.RunOnce(ctx))
	_, err = f.pool.Exec(ctx, `INSERT INTO interview_operations(id,user_id,job_id,kind,business_key,fingerprint,input,status,created_at) SELECT gen_random_uuid(),$1,$2,'plan',gen_random_uuid()::text,'synthetic','{}','succeeded',now()+interval '1 minute' FROM generate_series(1,31)`, f.owner, f.job)
	require.NoError(t, err)
	first, err := f.store.History(ctx, f.owner, f.job, 0, true)
	require.NoError(t, err)
	require.Len(t, first.Versions, 1)
	require.Len(t, first.Sessions, 1)
	require.Len(t, first.Operations, 30)
	for _, candidate := range first.Operations {
		require.NotEqual(t, operation.ID, candidate.ID)
	}
	older, err := f.store.History(ctx, f.owner, f.job, 20, true)
	require.NoError(t, err)
	found := false
	for _, candidate := range older.Operations {
		if candidate.ID == operation.ID {
			found = true
		}
	}
	require.True(t, found)
	replay, err := f.svc.Review(ctx, f.owner, f.job, session.ID, "", key)
	require.NoError(t, err)
	require.Equal(t, operation.ID, replay.ID)
	require.Equal(t, "failed", replay.Status)
	router := gin.New()
	NewHandler(f.svc).RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) { c.Set("user_id", f.owner) }, func(c *gin.Context) { c.Next() })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/jobs/"+f.job+"/interview-prep/operations/"+operation.ID, nil))
	require.Equal(t, 200, response.Code)
	var fetched Operation
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &fetched))
	require.Equal(t, operation.ID, fetched.ID)
	require.Equal(t, "failed", fetched.Status)
	var usageBefore, usageAfter int
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usageBefore))
	f.client.set("")
	_, err = f.svc.Retry(ctx, f.owner, f.job, operation.ID)
	require.NoError(t, err)
	require.NoError(t, f.svc.RunOnce(ctx))
	require.NoError(t, f.pool.QueryRow(ctx, `SELECT count(*) FROM ai_usage WHERE user_id=$1`, f.owner).Scan(&usageAfter))
	require.Equal(t, usageBefore, usageAfter)
	session, err = f.store.Session(ctx, f.owner, f.job, session.ID)
	require.NoError(t, err)
	require.NotNil(t, session.Summary)
}
