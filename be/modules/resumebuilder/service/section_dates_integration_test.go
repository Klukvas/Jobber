//go:build integration

package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	rbHandler "github.com/andreypavlenko/jobber/modules/resumebuilder/handler"
	rbRepo "github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	rbSvc "github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegressionR3SectionDateHTTP(t *testing.T) {
	dsn := os.Getenv("R3_DATA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set R3_DATA_TEST_DATABASE_URL to a migrated jobber_r3_data_test database")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	var database string
	require.NoError(t, db.QueryRow(ctx, `SELECT current_database()`).Scan(&database))
	require.Equal(t, "jobber_r3_data_test", database, "refusing to modify a non-test database")
	userID, rbID := uuid.NewString(), uuid.NewString()
	_, err = db.Exec(ctx, `INSERT INTO users (id,email,name,password_hash) VALUES ($1,$2,'Regression','test-only')`, userID, userID+"@regression.test")
	require.NoError(t, err)
	_, err = db.Exec(ctx, `INSERT INTO resume_builders (id,user_id) VALUES ($1,$2)`, rbID, userID)
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("user_id", userID) })
	handler := rbHandler.NewResumeBuilderHandler(rbSvc.NewResumeBuilderService(rbRepo.NewResumeBuilderRepository(db), nil))
	router.POST("/:id/experiences", handler.CreateExperience)
	router.PATCH("/:id/experiences/:entryId", handler.UpdateExperience)
	router.POST("/:id/educations", handler.CreateEducation)
	router.PATCH("/:id/educations/:entryId", handler.UpdateEducation)
	request := func(t *testing.T, method, path string, payload map[string]any, status int) map[string]any {
		t.Helper()
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, response.Body.String())
		var data map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &data))
		if status == http.StatusBadRequest {
			assert.Contains(t, response.Body.String(), "INVALID_DATE_RANGE")
		}
		return data
	}
	for _, section := range []string{"experiences", "educations"} {
		t.Run(section, func(t *testing.T) {
			path := "/" + rbID + "/" + section
			payload := map[string]any{"company": "Test", "position": "Engineer", "institution": "School", "degree": "Degree", "start_date": "2025-01-01", "end_date": "2024-01-01"}
			request(t, http.MethodPost, path, payload, http.StatusBadRequest)
			payload["start_date"] = "2024-01-01"
			payload["end_date"] = "2024-06-01"
			created := request(t, http.MethodPost, path, payload, http.StatusCreated)
			// RespondWithData returns the DTO directly.
			entryID, ok := created["id"].(string)
			require.True(t, ok, "%#v", created)
			path += "/" + entryID
			request(t, http.MethodPatch, path, map[string]any{"start_date": "2025-01-01"}, http.StatusBadRequest)
			request(t, http.MethodPatch, path, map[string]any{"end_date": "2023-01-01"}, http.StatusBadRequest)
			got := request(t, http.MethodPatch, path, map[string]any{"end_date": "2024-01-01"}, http.StatusOK)
			assert.Equal(t, "2024-01-01", got["start_date"])
			assert.Equal(t, "2024-01-01", got["end_date"])
			got = request(t, http.MethodPatch, path, map[string]any{"end_date": ""}, http.StatusOK)
			assert.Empty(t, got["end_date"])
			got = request(t, http.MethodPatch, path, map[string]any{"is_current": true, "end_date": "2020-01-01"}, http.StatusOK)
			assert.Empty(t, got["end_date"])
			request(t, http.MethodPatch, path, map[string]any{"is_current": false}, http.StatusOK)
			// Historic invalid data must be repaired before unrelated edits can save it.
			table := "resume_experiences"
			if section == "educations" {
				table = "resume_educations"
			}
			_, err = db.Exec(ctx, `UPDATE `+table+` SET start_date='2025-01-01',end_date='2024-01-01' WHERE id=$1`, entryID)
			require.NoError(t, err)
			request(t, http.MethodPatch, path, map[string]any{"description": "unrelated"}, http.StatusBadRequest)
			got = request(t, http.MethodPatch, path, map[string]any{"end_date": "2025-01-01"}, http.StatusOK)
			assert.Equal(t, "2025-01-01", got["end_date"])
			request(t, http.MethodPatch, path, map[string]any{"start_date": "2024-02-30"}, http.StatusBadRequest)
			// Reload through the repository verifies failed PATCHes never persisted.
			var start, end string
			require.NoError(t, db.QueryRow(ctx, `SELECT start_date,end_date FROM `+table+` WHERE id=$1`, entryID).Scan(&start, &end))
			assert.Equal(t, "2025-01-01", start)
			assert.Equal(t, "2025-01-01", end)
		})
	}
}
