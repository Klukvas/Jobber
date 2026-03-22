package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/andreypavlenko/jobber/modules/calendar/ports"
	"github.com/andreypavlenko/jobber/modules/calendar/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockCalendarTokenRepository implements ports.CalendarTokenRepository
type MockCalendarTokenRepository struct {
	UpsertFunc      func(ctx context.Context, token *model.CalendarToken) error
	GetByUserIDFunc func(ctx context.Context, userID string) (*model.CalendarToken, error)
	DeleteFunc      func(ctx context.Context, userID string) error
}

func (m *MockCalendarTokenRepository) Upsert(ctx context.Context, token *model.CalendarToken) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, token)
	}
	return nil
}

func (m *MockCalendarTokenRepository) GetByUserID(ctx context.Context, userID string) (*model.CalendarToken, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, model.ErrNotConnected
}

func (m *MockCalendarTokenRepository) Delete(ctx context.Context, userID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID)
	}
	return nil
}

// MockCalendarStageRepository implements ports.CalendarStageRepository
type MockCalendarStageRepository struct {
	SetCalendarEventIDFunc   func(ctx context.Context, stageID, eventID string) error
	ClearCalendarEventIDFunc func(ctx context.Context, stageID string) error
	GetCalendarEventIDFunc   func(ctx context.Context, stageID string) (string, error)
	GetStageUserIDFunc       func(ctx context.Context, stageID string) (string, error)
}

func (m *MockCalendarStageRepository) SetCalendarEventID(ctx context.Context, stageID, eventID string) error {
	if m.SetCalendarEventIDFunc != nil {
		return m.SetCalendarEventIDFunc(ctx, stageID, eventID)
	}
	return nil
}

func (m *MockCalendarStageRepository) ClearCalendarEventID(ctx context.Context, stageID string) error {
	if m.ClearCalendarEventIDFunc != nil {
		return m.ClearCalendarEventIDFunc(ctx, stageID)
	}
	return nil
}

func (m *MockCalendarStageRepository) GetCalendarEventID(ctx context.Context, stageID string) (string, error) {
	if m.GetCalendarEventIDFunc != nil {
		return m.GetCalendarEventIDFunc(ctx, stageID)
	}
	return "", model.ErrEventNotFound
}

func (m *MockCalendarStageRepository) GetStageUserID(ctx context.Context, stageID string) (string, error) {
	if m.GetStageUserIDFunc != nil {
		return m.GetStageUserIDFunc(ctx, stageID)
	}
	return "", model.ErrStageNotFound
}

// MockGoogleCalendarClient implements ports.GoogleCalendarClient
type MockGoogleCalendarClient struct {
	CreateEventFunc  func(ctx context.Context, token *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error)
	DeleteEventFunc  func(ctx context.Context, token *oauth2.Token, eventID string) error
	GetUserEmailFunc func(ctx context.Context, token *oauth2.Token) (string, error)
}

func (m *MockGoogleCalendarClient) CreateEvent(ctx context.Context, token *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, token, event)
	}
	return &ports.CreatedEvent{EventID: "event-123", Link: "https://calendar.google.com/event/123"}, nil
}

func (m *MockGoogleCalendarClient) DeleteEvent(ctx context.Context, token *oauth2.Token, eventID string) error {
	if m.DeleteEventFunc != nil {
		return m.DeleteEventFunc(ctx, token, eventID)
	}
	return nil
}

func (m *MockGoogleCalendarClient) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	if m.GetUserEmailFunc != nil {
		return m.GetUserEmailFunc(ctx, token)
	}
	return "user@example.com", nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func mockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// testEncryptionKey is a valid 64-hex-char key for testing
const testEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func newTestCalendarHandler(
	tokenRepo *MockCalendarTokenRepository,
	stageRepo *MockCalendarStageRepository,
	gcalClient *MockGoogleCalendarClient,
	redisClient *redis.Client,
) *CalendarHandler {
	encryptor, _ := service.NewEncryptor(testEncryptionKey)

	oauthConfig := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: "http://localhost:8080/api/v1/calendar/callback",
		Scopes:      []string{"https://www.googleapis.com/auth/calendar"},
	}

	svc := service.NewCalendarService(
		tokenRepo,
		stageRepo,
		gcalClient,
		encryptor,
		oauthConfig,
		redisClient,
		"http://localhost:3000",
	)

	return NewCalendarHandler(svc)
}

func newMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rc
}

func TestCalendarHandler_GetAuthURL(t *testing.T) {
	userID := "user-123"

	t.Run("returns auth URL successfully", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/auth", mockAuthMiddleware(userID), handler.GetAuthURL)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/auth", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.OAuthURLResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response.URL, "accounts.google.com")
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/auth", handler.GetAuthURL)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/auth", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCalendarHandler_HandleCallback(t *testing.T) {
	t.Run("redirects to error page when code is empty", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/callback", handler.HandleCallback)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/callback?state=some-state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "calendar=error")
	})

	t.Run("redirects to error page when state is empty", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/callback", handler.HandleCallback)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/callback?code=some-code", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "calendar=error")
	})

	t.Run("redirects to error page when both code and state are empty", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/callback", handler.HandleCallback)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/callback", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "calendar=error")
	})

	t.Run("redirects to error page when state is invalid", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/callback", handler.HandleCallback)

		// State not in Redis -> invalid state error -> redirect to error
		req, _ := http.NewRequest(http.MethodGet, "/calendar/callback?code=some-code&state=invalid-state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "calendar=error")
	})
}

func TestCalendarHandler_GetStatus(t *testing.T) {
	userID := "user-123"

	t.Run("returns connected status with email", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		encryptor, _ := service.NewEncryptor(testEncryptionKey)
		tokenJSON := `{"access_token":"test-token","token_type":"Bearer","expiry":"2099-01-01T00:00:00Z"}`
		ciphertext, nonce, _ := encryptor.Encrypt([]byte(tokenJSON))

		tokenRepo := &MockCalendarTokenRepository{
			GetByUserIDFunc: func(ctx context.Context, uid string) (*model.CalendarToken, error) {
				return &model.CalendarToken{
					ID:         "token-1",
					UserID:     uid,
					TokenBlob:  ciphertext,
					TokenNonce: nonce,
				}, nil
			},
		}

		handler := newTestCalendarHandler(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/status", mockAuthMiddleware(userID), handler.GetStatus)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.CalendarStatusDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Connected)
		assert.Equal(t, "user@example.com", response.Email)
	})

	t.Run("returns not connected when no token", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/status", mockAuthMiddleware(userID), handler.GetStatus)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.CalendarStatusDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.False(t, response.Connected)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.GET("/calendar/status", handler.GetStatus)

		req, _ := http.NewRequest(http.MethodGet, "/calendar/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestCalendarHandler_Disconnect(t *testing.T) {
	userID := "user-123"

	t.Run("disconnects successfully", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		tokenRepo := &MockCalendarTokenRepository{
			DeleteFunc: func(ctx context.Context, uid string) error {
				return nil
			},
		}

		handler := newTestCalendarHandler(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar", mockAuthMiddleware(userID), handler.Disconnect)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar", handler.Disconnect)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 404 when not connected", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		tokenRepo := &MockCalendarTokenRepository{
			DeleteFunc: func(ctx context.Context, uid string) error {
				return model.ErrNotConnected
			},
		}

		handler := newTestCalendarHandler(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar", mockAuthMiddleware(userID), handler.Disconnect)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCalendarHandler_CreateEvent(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", handler.CreateEvent)

		body := `{"stage_id":"stage-1","title":"Interview","start_time":"2025-01-01T10:00:00Z","duration_min":60}`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing required fields", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		body := `{"title":"Interview"}`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid duration", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		// duration_min < 15 is invalid per binding:"min=15"
		body := `{"stage_id":"stage-1","title":"Interview","start_time":"2025-01-01T10:00:00Z","duration_min":5}`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 404 when stage not found", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "", model.ErrStageNotFound
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		body := `{"stage_id":"nonexistent","title":"Interview","start_time":"2025-01-01T10:00:00Z","duration_min":60}`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 409 when event already exists", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return userID, nil
				},
				GetCalendarEventIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "existing-event-id", nil
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		body := `{"stage_id":"stage-1","title":"Interview","start_time":"2025-01-01T10:00:00Z","duration_min":60}`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("creates event successfully", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		encryptor, _ := service.NewEncryptor(testEncryptionKey)
		tokenJSON := `{"access_token":"test-token","token_type":"Bearer","expiry":"2099-01-01T00:00:00Z"}`
		ciphertext, nonce, _ := encryptor.Encrypt([]byte(tokenJSON))

		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{
				GetByUserIDFunc: func(ctx context.Context, uid string) (*model.CalendarToken, error) {
					return &model.CalendarToken{
						ID: "token-1", UserID: uid,
						TokenBlob: ciphertext, TokenNonce: nonce,
					}, nil
				},
			},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return userID, nil
				},
				GetCalendarEventIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "", model.ErrEventNotFound
				},
				SetCalendarEventIDFunc: func(ctx context.Context, stageID, eventID string) error {
					return nil
				},
			},
			&MockGoogleCalendarClient{
				CreateEventFunc: func(ctx context.Context, token *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error) {
					return &ports.CreatedEvent{EventID: "event-123", Link: "https://calendar.google.com/event/123"}, nil
				},
			},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		startTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		body, _ := json.Marshal(map[string]interface{}{
			"stage_id":     "stage-1",
			"title":        "Interview",
			"start_time":   startTime,
			"duration_min": 60,
		})
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.CalendarEventDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "event-123", response.EventID)
		assert.Equal(t, "stage-1", response.StageID)
	})

	t.Run("returns 422 when calendar not connected", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{
				GetByUserIDFunc: func(ctx context.Context, uid string) (*model.CalendarToken, error) {
					return nil, model.ErrNotConnected
				},
			},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return userID, nil
				},
				GetCalendarEventIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "", model.ErrEventNotFound
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.POST("/calendar/events", mockAuthMiddleware(userID), handler.CreateEvent)

		body := `{"stage_id":"stage-1","title":"Interview","start_time":"2025-01-01T10:00:00Z","duration_min":60}`
		req, _ := http.NewRequest(http.MethodPost, "/calendar/events", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestCalendarHandler_DeleteEvent(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar/events/:stageId", handler.DeleteEvent)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar/events/stage-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 404 when stage not found", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "", model.ErrStageNotFound
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar/events/:stageId", mockAuthMiddleware(userID), handler.DeleteEvent)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar/events/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 404 when event not found", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return userID, nil
				},
				GetCalendarEventIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "", model.ErrEventNotFound
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar/events/:stageId", mockAuthMiddleware(userID), handler.DeleteEvent)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar/events/stage-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("deletes event successfully", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		encryptor, _ := service.NewEncryptor(testEncryptionKey)
		tokenJSON := `{"access_token":"test-token","token_type":"Bearer","expiry":"2099-01-01T00:00:00Z"}`
		ciphertext, nonce, _ := encryptor.Encrypt([]byte(tokenJSON))

		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{
				GetByUserIDFunc: func(ctx context.Context, uid string) (*model.CalendarToken, error) {
					return &model.CalendarToken{
						ID: "token-1", UserID: uid,
						TokenBlob: ciphertext, TokenNonce: nonce,
					}, nil
				},
			},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return userID, nil
				},
				GetCalendarEventIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "event-123", nil
				},
				ClearCalendarEventIDFunc: func(ctx context.Context, stageID string) error {
					return nil
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar/events/:stageId", mockAuthMiddleware(userID), handler.DeleteEvent)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar/events/stage-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 422 when calendar not connected", func(t *testing.T) {
		_, rc := newMiniRedis(t)
		handler := newTestCalendarHandler(
			&MockCalendarTokenRepository{
				GetByUserIDFunc: func(ctx context.Context, uid string) (*model.CalendarToken, error) {
					return nil, model.ErrNotConnected
				},
			},
			&MockCalendarStageRepository{
				GetStageUserIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return userID, nil
				},
				GetCalendarEventIDFunc: func(ctx context.Context, stageID string) (string, error) {
					return "event-123", nil
				},
			},
			&MockGoogleCalendarClient{},
			rc,
		)

		router := setupTestRouter()
		router.DELETE("/calendar/events/:stageId", mockAuthMiddleware(userID), handler.DeleteEvent)

		req, _ := http.NewRequest(http.MethodDelete, "/calendar/events/stage-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestCalendarHandler_RegisterRoutes(t *testing.T) {
	_, rc := newMiniRedis(t)
	handler := newTestCalendarHandler(
		&MockCalendarTokenRepository{},
		&MockCalendarStageRepository{},
		&MockGoogleCalendarClient{},
		rc,
	)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	// Verify all expected routes are in gin's route table
	registeredRoutes := router.Routes()
	expectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/calendar/auth"},
		{http.MethodGet, "/api/v1/calendar/callback"},
		{http.MethodGet, "/api/v1/calendar/status"},
		{http.MethodDelete, "/api/v1/calendar"},
		{http.MethodPost, "/api/v1/calendar/events"},
		{http.MethodDelete, "/api/v1/calendar/events/:stageId"},
	}

	for _, expected := range expectedRoutes {
		t.Run(expected.method+" "+expected.path, func(t *testing.T) {
			found := false
			for _, r := range registeredRoutes {
				if r.Method == expected.method && r.Path == expected.path {
					found = true
					break
				}
			}
			assert.True(t, found, "Route %s %s should be registered", expected.method, expected.path)
		})
	}

	// Smoke-test routes that don't conflict with gin routing tree
	smokeTests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/calendar/auth"},
		{http.MethodGet, "/api/v1/calendar/callback"},
		{http.MethodGet, "/api/v1/calendar/status"},
		{http.MethodDelete, "/api/v1/calendar"},
	}

	for _, route := range smokeTests {
		t.Run("smoke "+route.method+" "+route.path, func(t *testing.T) {
			req, _ := http.NewRequest(route.method, route.path, bytes.NewBuffer(nil))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be reachable", route.method, route.path)
		})
	}
}
