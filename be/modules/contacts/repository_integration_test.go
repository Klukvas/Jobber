//go:build integration

package contacts

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCRUDOwnershipMigration(t *testing.T) {
	dsn := os.Getenv("CONTACTS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CONTACTS_TEST_DATABASE_URL required for isolated synthetic PostgreSQL")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "contacts_test", cfg.ConnConfig.Database)
	admin, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	schema := "contacts_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(t.Context(), `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
		_, err := admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
		require.NoError(t, err)
		admin.Close()
	})
	files, err := filepath.Glob("../../migrations/*.up.sql")
	require.NoError(t, err)
	sort.Strings(files)
	for _, file := range files {
		if strings.Contains(file, "000050") {
			continue
		}
		raw, err := os.ReadFile(file)
		require.NoError(t, err)
		_, err = pool.Exec(t.Context(), string(raw))
		require.NoError(t, err, file)
	}
	owner, other, job, secondJob, foreignJob := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(t.Context(), `INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Owner','no-login'),($3,$4,'Other','no-login')`, owner, owner+"@example.invalid", other, other+"@example.invalid")
	require.NoError(t, err)
	stage, foreignStage := uuid.NewString(), uuid.NewString()
	_, err = pool.Exec(t.Context(), `INSERT INTO stage_templates(id,user_id,name,"order") VALUES($1,$2,'Saved',0),($3,$4,'Saved',0)`, stage, owner, foreignStage, other)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO jobs(id,user_id,title,current_stage_template_id) VALUES($1,$2,'Existing job',$6),($3,$2,'Second job',$6),($4,$5,'Foreign job',$7)`, job, owner, secondJob, foreignJob, other, stage, foreignStage)
	require.NoError(t, err)
	var before string
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT row_to_json(j)::text FROM jobs j WHERE id=$1`, job).Scan(&before))
	raw, err := os.ReadFile("../../migrations/000050_job_contacts.up.sql")
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), string(raw))
	require.NoError(t, err)
	var after string
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT row_to_json(j)::text FROM jobs j WHERE id=$1`, job).Scan(&after))
	require.Equal(t, before, after, "migration must preserve existing data")
	svc := NewService(NewRepository(pool))
	router := gin.New()
	NewHandler(svc).RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		if user := c.GetHeader("X-Test-User"); user != "" {
			c.Set("user_id", user)
		}
		c.Next()
	})
	request := func(user, method, path string, body any, code int) []byte {
		t.Helper()
		data, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(method, "/api/v1/jobs/"+path, bytes.NewReader(data))
		req.Header.Set("X-Test-User", user)
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		require.Equal(t, code, out.Code, out.Body.String())
		return out.Body.Bytes()
	}
	request("", http.MethodGet, job+"/contacts", nil, 401)
	for _, id := range []string{foreignJob, uuid.NewString(), "invalid"} {
		request(owner, "GET", id+"/contacts", nil, 404)
		request(owner, "POST", id+"/contacts", Fields{Name: "No"}, 404)
	}
	request(owner, "POST", job+"/contacts", Fields{Name: " "}, 400)
	request(owner, "POST", job+"/contacts", Fields{Name: "N", Telegram: "javascript:alert(1)"}, 400)
	var contact Contact
	require.NoError(t, json.Unmarshal(request(owner, "POST", job+"/contacts", Fields{Name: " Ada ", Telegram: "@ada_hr", Note: "Plain <b>note</b>"}, 201), &contact))
	require.Equal(t, job, contact.JobID)
	require.Equal(t, "Ada", contact.Name)
	request(owner, "POST", job+"/contacts", Fields{Name: "Grace"}, 201)
	var list []Contact
	require.NoError(t, json.Unmarshal(request(owner, "GET", job+"/contacts", nil, 200), &list))
	require.Len(t, list, 2)
	for _, tc := range []struct{ user, job string }{{other, job}, {owner, secondJob}, {owner, foreignJob}} {
		request(tc.user, "PATCH", tc.job+"/contacts/"+contact.ID, Fields{Name: "Hijack"}, 404)
		request(tc.user, "DELETE", tc.job+"/contacts/"+contact.ID, nil, 404)
	}
	request(owner, "PATCH", job+"/contacts/not-a-uuid", Fields{Name: "N"}, 404)
	request(owner, "PATCH", job+"/contacts/"+contact.ID, Fields{Name: "Updated", Role: "Hiring manager", Email: "a@example.com", LinkedIn: "linkedin.com/in/ada"}, 200)
	reloaded, err := NewService(NewRepository(pool)).List(t.Context(), owner, job)
	require.NoError(t, err)
	require.Len(t, reloaded, 2)
	require.Equal(t, "Updated", reloaded[0].Name)
	require.Equal(t, "", reloaded[0].Note)
	request(owner, "DELETE", job+"/contacts/"+contact.ID, nil, 204)
	request(owner, "DELETE", job+"/contacts/"+contact.ID, nil, 404)
	var count int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM job_contacts WHERE job_id=$1`, job).Scan(&count))
	require.Equal(t, 1, count)
	_, err = pool.Exec(t.Context(), `DELETE FROM jobs WHERE id=$1`, job)
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM job_contacts`).Scan(&count))
	require.Zero(t, count)
	down, err := os.ReadFile("../../migrations/000050_job_contacts.down.sql")
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), string(down))
	require.NoError(t, err)
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM jobs`).Scan(&count))
	require.Equal(t, 2, count)
}
