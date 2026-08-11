package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/andreypavlenko/jobber/modules/reminders/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockReminderRepository implements ports.ReminderRepository.
type MockReminderRepository struct {
	CreateFunc     func(ctx context.Context, r *model.Reminder) error
	ListByUserFunc func(ctx context.Context, userID string) ([]*model.Reminder, error)
	ListByJobFunc  func(ctx context.Context, userID, jobID string) ([]*model.Reminder, error)
	GetByIDFunc    func(ctx context.Context, userID, reminderID string) (*model.Reminder, error)
	UpdateFunc     func(ctx context.Context, r *model.Reminder) error
	DeleteFunc     func(ctx context.Context, userID, reminderID string) error
}

func (m *MockReminderRepository) Create(ctx context.Context, r *model.Reminder) error {
	return m.CreateFunc(ctx, r)
}
func (m *MockReminderRepository) ListByUser(ctx context.Context, userID string) ([]*model.Reminder, error) {
	return m.ListByUserFunc(ctx, userID)
}
func (m *MockReminderRepository) ListByJob(ctx context.Context, userID, jobID string) ([]*model.Reminder, error) {
	return m.ListByJobFunc(ctx, userID, jobID)
}
func (m *MockReminderRepository) GetByID(ctx context.Context, userID, reminderID string) (*model.Reminder, error) {
	return m.GetByIDFunc(ctx, userID, reminderID)
}
func (m *MockReminderRepository) Update(ctx context.Context, r *model.Reminder) error {
	return m.UpdateFunc(ctx, r)
}
func (m *MockReminderRepository) Delete(ctx context.Context, userID, reminderID string) error {
	return m.DeleteFunc(ctx, userID, reminderID)
}

// newTestRouter wires the handler through the real RegisterRoutes so the test
// also asserts the route registration itself does not panic.
func newTestRouter(repo *MockReminderRepository, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := service.NewReminderService(repo)
	h := NewReminderHandler(svc)

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	h.RegisterRoutes(v1, func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	return engine
}

func do(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestReminderHandler_Create(t *testing.T) {
	remindAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	t.Run("201 on success", func(t *testing.T) {
		repo := &MockReminderRepository{
			CreateFunc: func(_ context.Context, r *model.Reminder) error {
				r.ID = "rem-1"
				return nil
			},
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodPost, "/api/v1/reminders",
			`{"job_id":"job-1","remind_at":"`+remindAt+`","message":"ping"}`)

		require.Equal(t, http.StatusCreated, w.Code)
		var dto model.ReminderDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "rem-1", dto.ID)
		assert.Equal(t, "ping", dto.Message)
	})

	t.Run("404 when the job is not owned by the user", func(t *testing.T) {
		repo := &MockReminderRepository{
			CreateFunc: func(_ context.Context, _ *model.Reminder) error {
				return model.ErrJobNotFound
			},
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodPost, "/api/v1/reminders",
			`{"job_id":"not-mine","remind_at":"`+remindAt+`","message":"ping"}`)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("400 on malformed body", func(t *testing.T) {
		repo := &MockReminderRepository{}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodPost, "/api/v1/reminders", `not json`)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestReminderHandler_ListUpdateDelete(t *testing.T) {
	t.Run("GET /reminders returns the list", func(t *testing.T) {
		repo := &MockReminderRepository{
			ListByUserFunc: func(_ context.Context, _ string) ([]*model.Reminder, error) {
				return []*model.Reminder{{ID: "a"}, {ID: "b"}}, nil
			},
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodGet, "/api/v1/reminders", "")

		require.Equal(t, http.StatusOK, w.Code)
		var dtos []model.ReminderDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dtos))
		assert.Len(t, dtos, 2)
	})

	t.Run("GET /jobs/:id/reminders returns the job's reminders", func(t *testing.T) {
		var gotJob string
		repo := &MockReminderRepository{
			ListByJobFunc: func(_ context.Context, _, jobID string) ([]*model.Reminder, error) {
				gotJob = jobID
				return []*model.Reminder{{ID: "a"}}, nil
			},
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodGet, "/api/v1/jobs/job-42/reminders", "")

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "job-42", gotJob)
	})

	t.Run("PATCH toggles is_done", func(t *testing.T) {
		repo := &MockReminderRepository{
			GetByIDFunc: func(_ context.Context, _, id string) (*model.Reminder, error) {
				return &model.Reminder{ID: id, UserID: "user-1", Message: "m"}, nil
			},
			UpdateFunc: func(_ context.Context, _ *model.Reminder) error { return nil },
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodPatch, "/api/v1/reminders/rem-1", `{"is_done":true}`)

		require.Equal(t, http.StatusOK, w.Code)
		var dto model.ReminderDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.True(t, dto.IsDone)
	})

	t.Run("PATCH returns 404 for a missing reminder", func(t *testing.T) {
		repo := &MockReminderRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Reminder, error) {
				return nil, model.ErrReminderNotFound
			},
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodPatch, "/api/v1/reminders/missing", `{"is_done":true}`)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DELETE returns 200", func(t *testing.T) {
		repo := &MockReminderRepository{
			DeleteFunc: func(_ context.Context, _, _ string) error { return nil },
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodDelete, "/api/v1/reminders/rem-1", "")

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("DELETE returns 404 when not found", func(t *testing.T) {
		repo := &MockReminderRepository{
			DeleteFunc: func(_ context.Context, _, _ string) error { return model.ErrReminderNotFound },
		}
		engine := newTestRouter(repo, "user-1")

		w := do(engine, http.MethodDelete, "/api/v1/reminders/missing", "")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
