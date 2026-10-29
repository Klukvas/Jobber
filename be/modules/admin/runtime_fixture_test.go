package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	userRepo "github.com/andreypavlenko/jobber/modules/users/repository"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// This opt-in local fixture runs production admin handlers against a disposable
// database. It never reads app .env or connects to an existing application DB.
func TestAdminRuntimeFixture(t *testing.T) {
	if os.Getenv("ADMIN_RUNTIME_FIXTURE") != "1" {
		t.Skip("runtime fixture not requested")
	}
	pool := isolatedPool(t)
	fixtureUsers(t, pool)
	_, err := pool.Exec(t.Context(), `INSERT INTO users(email,name,email_verified) SELECT 'fixture-'||n||'@example.com','Fixture User '||n,true FROM generate_series(1,23) n`)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO subscriptions(user_id,status,plan) VALUES($1,'free','free'),($2,'free','free')`, testUser, testGrant)
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jwt := auth.NewJWTManager("local-fixture-access-signing", "local-fixture-refresh-signing", time.Hour, time.Hour)
	identities := userRepo.NewUserRepository(pool)
	router.GET("/api/v1/fixture/login", func(c *gin.Context) {
		id := testUser
		if c.Query("role") == "user" {
			id = testGrant
		}
		token, err := jwt.GenerateAccessToken(id)
		require.NoError(t, err)
		c.SetCookie(auth.AccessTokenCookie, token, 3600, "/", "", false, true)
		user, err := identities.GetByID(c.Request.Context(), id)
		require.NoError(t, err)
		data, err := json.Marshal(map[string]any{"state": map[string]any{"user": user.ToDTO(), "isAuthenticated": true}, "version": 0})
		require.NoError(t, err)
		c.Data(200, "text/html; charset=utf-8", []byte(`<script>localStorage.setItem("jobber-auth",`+string(mustJSON(string(data)))+`);localStorage.setItem("language","en");localStorage.setItem("jobber-onboarding-completed","true");location.href="/app/admin";</script>`))
	})
	v1 := router.Group("/api/v1")
	authenticate := auth.AuthMiddleware(jwt)
	NewHandler(NewRepository(pool), identities, "admin@example.com", "http://localhost:3000,http://127.0.0.1:3000").RegisterRoutes(v1, authenticate)
	v1.GET("/session", authenticate, func(c *gin.Context) {
		id, _ := auth.GetUserID(c)
		user, err := identities.GetByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(401, gin.H{})
			return
		}
		c.JSON(200, user.ToDTO())
	})
	v1.GET("/subscription", authenticate, func(c *gin.Context) {
		c.JSON(200, gin.H{"plan": "free", "status": "free", "limits": gin.H{"max_jobs": 25, "max_resumes": 1, "max_ai_requests": 30, "max_job_parses": 0, "max_resume_builders": 2, "max_cover_letters": 1}, "usage": gin.H{"jobs": 0, "resumes": 0, "ai_requests": 0, "job_parses": 0, "resume_builders": 0, "cover_letters": 0}})
	})
	v1.GET("/features/ai", authenticate, func(c *gin.Context) { c.JSON(200, gin.H{"available": false}) })
	fmt.Println("ADMIN FIXTURE READY http://127.0.0.1:8080; production admin handlers, isolated disposable schema")
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: router, ReadHeaderTimeout: 5 * time.Second}
	t.Cleanup(func() { require.NoError(t, server.Close()) })
	require.NoError(t, server.ListenAndServe())
}

func mustJSON(value string) []byte { data, _ := json.Marshal(value); return data }
