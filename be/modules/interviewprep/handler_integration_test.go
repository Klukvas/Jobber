//go:build integration

package interviewprep

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHTTPForeignObjectsAndStrictInput(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	v := f.generate(t)
	ss, err := f.store.StartSession(ctx, f.owner, f.job, v.ID, uuid.NewString())
	require.NoError(t, err)
	ss, err = f.store.SaveAnswer(ctx, f.owner, f.job, ss.ID, SaveAnswer{Key: uuid.NewString(), QuestionID: "q1", Text: "Actual saved synthetic answer", Revision: 0})
	require.NoError(t, err)
	op, err := f.svc.Review(ctx, f.owner, f.job, ss.ID, ss.Answers[0].ID, uuid.NewString())
	require.NoError(t, err)
	owner := f.other
	r := gin.New()
	NewHandler(f.svc).RegisterRoutes(r.Group("/api/v1"), func(c *gin.Context) {
		if owner != "" {
			c.Set("user_id", owner)
		}
		c.Next()
	}, func(c *gin.Context) { c.Next() })
	base := "/api/v1/jobs/" + f.job + "/interview-prep"
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{"GET", "", nil}, {"GET", "/versions/" + v.ID, nil}, {"GET", "/sessions/" + ss.ID, nil}, {"GET", "/operations/" + op.ID, nil},
		{"PUT", "/versions/" + v.ID + "/studied", map[string]any{"question_id": "q1", "studied": true}},
		{"POST", "/generate", map[string]any{"key": uuid.NewString(), "resume_builder_id": f.resume, "locale": "en"}},
		{"POST", "/operations/" + op.ID + "/retry", map[string]any{}},
		{"POST", "/sessions", map[string]any{"key": uuid.NewString(), "version_id": v.ID}},
		{"POST", "/sessions/" + ss.ID + "/answers", SaveAnswer{Key: uuid.NewString(), QuestionID: "q2", Text: "Foreign attempt", Revision: 1}},
		{"POST", "/sessions/" + ss.ID + "/action", SessionAction{Key: uuid.NewString(), Action: "finish", Revision: 1}},
		{"POST", "/sessions/" + ss.ID + "/answers/" + ss.Answers[0].ID + "/feedback", map[string]any{"key": uuid.NewString()}},
		{"POST", "/sessions/" + ss.ID + "/summary", map[string]any{"key": uuid.NewString()}},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			raw, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(tc.method, base+tc.path, bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, 404, w.Code)
			require.NotContains(t, w.Body.String(), "Actual saved")
		})
	}
	// Prove nested authorization independently from the top-level Job check.
	foreignJob, foreignStage := uuid.NewString(), uuid.NewString()
	_, err = f.pool.Exec(ctx, `INSERT INTO stage_templates(id,user_id,name,"order") VALUES($1,$2,'Other saved',0)`, foreignStage, f.other)
	require.NoError(t, err)
	_, err = f.pool.Exec(ctx, `INSERT INTO jobs(id,user_id,title,description,current_stage_template_id) VALUES($1,$2,'Other Job','Synthetic Job',$3)`, foreignJob, f.other, foreignStage)
	require.NoError(t, err)
	for _, entry := range []struct {
		method, path string
		body         any
	}{
		{"GET", "/versions/" + v.ID, nil}, {"GET", "/sessions/" + ss.ID, nil}, {"GET", "/operations/" + op.ID, nil},
		{"PUT", "/versions/" + v.ID + "/studied", map[string]any{"question_id": "q1", "studied": true}},
		{"POST", "/sessions", map[string]any{"key": uuid.NewString(), "version_id": v.ID}},
		{"POST", "/sessions/" + ss.ID + "/answers", SaveAnswer{Key: uuid.NewString(), QuestionID: "q2", Text: "foreign", Revision: 1}},
		{"POST", "/sessions/" + ss.ID + "/action", SessionAction{Key: uuid.NewString(), Action: "finish", Revision: 1}},
		{"POST", "/sessions/" + ss.ID + "/answers/" + ss.Answers[0].ID + "/feedback", map[string]any{"key": uuid.NewString()}},
		{"POST", "/sessions/" + ss.ID + "/summary", map[string]any{"key": uuid.NewString()}},
		{"POST", "/operations/" + op.ID + "/retry", map[string]any{}},
		{"POST", "/generate", map[string]any{"key": uuid.NewString(), "resume_builder_id": f.resume, "locale": "en"}},
	} {
		raw, _ := json.Marshal(entry.body)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(entry.method, "/api/v1/jobs/"+foreignJob+"/interview-prep"+entry.path, bytes.NewReader(raw)))
		require.Equal(t, 404, w.Code, entry.path)
	}
	// Same owner, different Job also cannot address a nested foreign parent.
	_, err = f.pool.Exec(ctx, `UPDATE jobs SET user_id=$1 WHERE id=$2`, f.owner, foreignJob)
	require.NoError(t, err)
	owner = f.owner
	for _, path := range []string{"/versions/" + v.ID, "/sessions/" + ss.ID, "/operations/" + op.ID} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/jobs/"+foreignJob+"/interview-prep"+path, nil))
		require.Equal(t, 404, w.Code)
	}
	owner = ""
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", base, nil))
	require.Equal(t, 401, w.Code)
	owner = f.owner
	for _, raw := range []string{`{"unknown":true}`, `{"key":"` + uuid.NewString() + `","resume_builder_id":"` + f.resume + `","locale":"en"} {}`, `{"key":"` + uuid.NewString() + `","resume_builder_id":"` + f.resume + `","locale":"` + string([]byte{255}) + `"}`} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", base+"/generate", bytes.NewBufferString(raw)))
		require.Equal(t, 400, w.Code)
	}
}
func TestRealRedisRateLimitAndDatabaseFallback(t *testing.T) {
	f := setup(t, 4)
	ctx := t.Context()
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:56381"})
	defer redisClient.Close()
	require.NoError(t, redisClient.Ping(ctx).Err())
	limiter := httpPlatform.UserRateLimitMiddleware(redisClient, httpPlatform.RateLimitConfig{MaxRequests: 1, Window: time.Second, KeyPrefix: "interview-integration-" + uuid.NewString()}, zap.NewNop())
	r := gin.New()
	NewHandler(f.svc).RegisterRoutes(r.Group("/api/v1"), func(c *gin.Context) { c.Set("user_id", f.owner); c.Next() }, limiter)
	base := "/api/v1/jobs/" + f.job + "/interview-prep/generate"
	for i, want := range []int{200, 429} {
		raw, _ := json.Marshal(map[string]string{"key": uuid.NewString(), "resume_builder_id": f.resume, "locale": "en"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, base, bytes.NewReader(raw)))
		require.Equal(t, want, w.Code, "request %d", i)
	}
	// Without Redis, the service still enforces pending cap and durable rate guard.
	for _, locale := range []string{"ru", "uk"} {
		_, err := f.svc.Generate(ctx, f.owner, f.job, f.resume, locale, uuid.NewString())
		require.NoError(t, err)
	}
	_, err := f.pool.Exec(ctx, `UPDATE resume_skills SET name='Changed content' WHERE resume_builder_id=$1`, f.resume)
	require.NoError(t, err)
	_, err = f.svc.Generate(ctx, f.owner, f.job, f.resume, "en", uuid.NewString())
	require.ErrorIs(t, err, ErrRateLimited)
}
