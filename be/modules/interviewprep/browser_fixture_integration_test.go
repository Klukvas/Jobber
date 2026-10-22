//go:build integration

package interviewprep

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type browserLimits struct {
	pool    *pgxpool.Pool
	maximum atomic.Int64
}

func (l *browserLimits) ResourceLimit(ctx context.Context, _ string, _ string) (int, error) {
	var n int
	err := l.pool.QueryRow(ctx, `SELECT $1::int`, l.maximum.Load()).Scan(&n)
	return n, err
}

// This fixture exists only in an integration test binary. Production has no
// switch, route, environment option or implementation selecting a fake AI.
func TestServeBrowserFixture(t *testing.T) {
	if os.Getenv("INTERVIEW_BROWSER_FIXTURE") != "1" {
		t.Skip("explicit synthetic browser fixture only")
	}
	f := setup(t, 8)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var unavailable atomic.Bool
	limits := &browserLimits{pool: f.pool}
	limits.maximum.Store(-1)
	f.svc.limits = limits
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:56381"})
	defer redisClient.Close()
	require.NoError(t, redisClient.Ping(ctx).Err())
	makeRouter := func(svc *Service) *gin.Engine {
		r := gin.New()
		r.Use(gin.Recovery(), func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "http://127.0.0.1:53081")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type,X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			c.Next()
		})
		auth := func(c *gin.Context) { c.Set("user_id", f.owner); c.Next() }
		limiter := httpPlatform.UserRateLimitMiddleware(redisClient, httpPlatform.RateLimitConfig{MaxRequests: 100, Window: time.Minute, KeyPrefix: "interview-fixture"}, zap.NewNop())
		NewHandler(svc).RegisterRoutes(r.Group("/api/v1"), auth, limiter)
		now := time.Now().UTC().Format(time.RFC3339)
		user := gin.H{"id": f.owner, "email": "synthetic@example.invalid", "name": "Synthetic Candidate", "locale": "en", "created_at": now}
		r.GET("/api/v1/session", func(c *gin.Context) { c.JSON(200, user) })
		r.GET("/api/v1/profile", func(c *gin.Context) { c.JSON(200, user) })
		r.GET("/api/v1/jobs/:id", func(c *gin.Context) {
			var title, description, stage string
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT title,description,current_stage_template_id::text FROM jobs WHERE id=$1`, f.job).Scan(&title, &description, &stage))
			c.JSON(200, gin.H{"id": f.job, "title": title, "description": description, "user_id": f.owner, "current_stage_template_id": stage, "current_stage_name": "Saved", "is_archived": false, "is_favorite": false, "created_at": now, "updated_at": now, "job_comments": []any{}, "resume": gin.H{"id": f.resume, "title": "Synthetic Builder Resume", "type": "builder"}})
		})
		r.GET("/api/v1/jobs/:id/stages", func(c *gin.Context) { c.JSON(200, []any{}) })
		r.GET("/api/v1/stage-templates", func(c *gin.Context) {
			var stage string
			require.NoError(t, f.pool.QueryRow(ctx, `SELECT current_stage_template_id::text FROM jobs WHERE id=$1`, f.job).Scan(&stage))
			c.JSON(200, gin.H{"items": []any{gin.H{"id": stage, "name": "Saved", "order": 0}}, "total": 1})
		})
		r.GET("/api/v1/companies", func(c *gin.Context) { c.JSON(200, gin.H{"items": []any{}, "total": 0}) })
		r.GET("/api/v1/resumes", func(c *gin.Context) { c.JSON(200, gin.H{"items": []any{}, "total": 0}) })
		r.GET("/api/v1/resume-builder", func(c *gin.Context) {
			c.JSON(200, []any{gin.H{"id": f.resume, "title": "Synthetic Builder Resume", "created_at": now, "updated_at": now}})
		})
		r.GET("/api/v1/jobs/:id/reminders", func(c *gin.Context) { c.JSON(200, []any{}) })
		r.GET("/api/v1/tags", func(c *gin.Context) { c.JSON(200, []any{}) })
		r.GET("/api/v1/jobs/:id/tags", func(c *gin.Context) { c.JSON(200, []any{}) })
		r.GET("/api/v1/subscription", func(c *gin.Context) {
			c.JSON(200, gin.H{"plan": "pro", "status": "active", "limits": gin.H{"max_ai_requests": -1}, "usage": gin.H{"ai_requests": 0}})
		})
		return r
	}
	live := makeRouter(f.svc)
	offline := makeRouter(NewService(f.store, nil, f.svc.limits))
	controls := http.NewServeMux()
	controls.HandleFunc("/fixture", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"owner": f.owner, "job": f.job, "resume": f.resume, "schema": f.schema})
	})
	controls.HandleFunc("/fixture/mode", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode string `json:"mode"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			w.WriteHeader(400)
			return
		}
		unavailable.Store(req.Mode == "unavailable")
		f.client.set(req.Mode)
		w.WriteHeader(204)
	})
	controls.HandleFunc("/fixture/quota", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Limit int64 `json:"limit"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Limit < -1 || req.Limit > 100 {
			w.WriteHeader(400)
			return
		}
		limits.maximum.Store(req.Limit)
		w.WriteHeader(204)
	})
	controls.HandleFunc("/fixture/interrupt", func(w http.ResponseWriter, r *http.Request) {
		_, err := f.pool.Exec(r.Context(), `UPDATE interview_operations SET lease_until=now()-interval '1 second' WHERE user_id=$1 AND status='running'`, f.owner)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
	})

	controls.HandleFunc("/fixture/history-noise", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OperationID string `json:"operation_id"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || !uuidValid(body.OperationID) {
			w.WriteHeader(400)
			return
		}
		if _, err := f.pool.Exec(r.Context(), `UPDATE interview_operations SET created_at=now()-interval '3 minutes' WHERE user_id=$1 AND job_id=$2 AND id=$3`, f.owner, f.job, body.OperationID); err != nil {
			w.WriteHeader(500)
			return
		}
		_, err := f.pool.Exec(r.Context(), `INSERT INTO interview_operations(id,user_id,job_id,kind,business_key,fingerprint,input,status,created_at) SELECT gen_random_uuid(),$1,$2,'plan',gen_random_uuid()::text,'synthetic','{}','succeeded',now()-interval '2 minutes' FROM generate_series(1,31)`, f.owner, f.job)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
	})
	controls.HandleFunc("/fixture/stale", func(w http.ResponseWriter, r *http.Request) {
		_, err := f.pool.Exec(r.Context(), `UPDATE resume_skills SET name=name||' changed' WHERE resume_builder_id=$1`, f.resume)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(204)
	})
	controls.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if unavailable.Load() {
			offline.ServeHTTP(w, r)
		} else {
			live.ServeHTTP(w, r)
		}
	})
	server := http.Server{Addr: "127.0.0.1:58081", Handler: controls, ReadHeaderTimeout: 5 * time.Second}
	go f.svc.Run(ctx)
	t.Logf("synthetic fixture listening 127.0.0.1:58081 job=%s owner=%s", f.job, f.owner)
	require.NoError(t, server.ListenAndServe())
}
