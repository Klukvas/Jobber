package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/andreypavlenko/jobber/modules/calendar/ports"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// ---------------------------------------------------------------------------
// Mock repositories & clients
// ---------------------------------------------------------------------------

type MockCalendarTokenRepository struct {
	UpsertFunc    func(ctx context.Context, token *model.CalendarToken) error
	GetByUserIDFunc func(ctx context.Context, userID string) (*model.CalendarToken, error)
	DeleteFunc    func(ctx context.Context, userID string) error
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
	return nil, errors.New("not found")
}

func (m *MockCalendarTokenRepository) Delete(ctx context.Context, userID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID)
	}
	return nil
}

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
	return "", errors.New("not found")
}

func (m *MockCalendarStageRepository) GetStageUserID(ctx context.Context, stageID string) (string, error) {
	if m.GetStageUserIDFunc != nil {
		return m.GetStageUserIDFunc(ctx, stageID)
	}
	return "", errors.New("not found")
}

type MockGoogleCalendarClient struct {
	CreateEventFunc  func(ctx context.Context, token *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error)
	DeleteEventFunc  func(ctx context.Context, token *oauth2.Token, eventID string) error
	GetUserEmailFunc func(ctx context.Context, token *oauth2.Token) (string, error)
}

func (m *MockGoogleCalendarClient) CreateEvent(ctx context.Context, token *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, token, event)
	}
	return &ports.CreatedEvent{EventID: "evt_1", Link: "https://calendar.google.com/event/1"}, nil
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

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const (
	testUserID    = "user-123"
	testFrontendURL = "https://app.example.com"
)

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr
}

func newTestEncryptor(t *testing.T) *Encryptor {
	t.Helper()
	enc, err := NewEncryptor(generateTestKey())
	require.NoError(t, err)
	return enc
}

func newTestOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		RedirectURL: "https://api.example.com/callback",
		Scopes:      []string{"calendar"},
	}
}

func storeValidToken(t *testing.T, enc *Encryptor, tokenRepo *MockCalendarTokenRepository) {
	t.Helper()
	// Create a valid token and encrypt it
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	tokenJSON, err := json.Marshal(token)
	require.NoError(t, err)

	ciphertext, nonce, err := enc.Encrypt(tokenJSON)
	require.NoError(t, err)

	tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
		return &model.CalendarToken{
			ID:         "token-1",
			UserID:     testUserID,
			TokenBlob:  ciphertext,
			TokenNonce: nonce,
		}, nil
	}
}

func newCalendarService(
	tokenRepo *MockCalendarTokenRepository,
	stageRepo *MockCalendarStageRepository,
	gcalClient *MockGoogleCalendarClient,
	enc *Encryptor,
	redisClient *redis.Client,
) *CalendarService {
	return NewCalendarService(
		tokenRepo,
		stageRepo,
		gcalClient,
		enc,
		newTestOAuthConfig(),
		redisClient,
		testFrontendURL,
	)
}

// ---------------------------------------------------------------------------
// GetStatus tests
// ---------------------------------------------------------------------------

func TestCalendarService_GetStatus(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(tokenRepo *MockCalendarTokenRepository, gcalClient *MockGoogleCalendarClient)
		wantConn  bool
		wantEmail string
	}{
		{
			name: "returns connected with email",
			setup: func(tokenRepo *MockCalendarTokenRepository, gcalClient *MockGoogleCalendarClient) {
				gcalClient.GetUserEmailFunc = func(_ context.Context, _ *oauth2.Token) (string, error) {
					return "user@example.com", nil
				}
			},
			wantConn:  true,
			wantEmail: "user@example.com",
		},
		{
			name: "returns connected without email on API error",
			setup: func(tokenRepo *MockCalendarTokenRepository, gcalClient *MockGoogleCalendarClient) {
				gcalClient.GetUserEmailFunc = func(_ context.Context, _ *oauth2.Token) (string, error) {
					return "", errors.New("API error")
				}
			},
			wantConn:  true,
			wantEmail: "",
		},
		{
			name: "returns disconnected when no token",
			setup: func(tokenRepo *MockCalendarTokenRepository, gcalClient *MockGoogleCalendarClient) {
				tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
					return nil, errors.New("not found")
				}
			},
			wantConn: false,
		},
		{
			name: "returns disconnected when token decryption fails",
			setup: func(tokenRepo *MockCalendarTokenRepository, gcalClient *MockGoogleCalendarClient) {
				tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
					return &model.CalendarToken{
						UserID:     testUserID,
						TokenBlob:  "invalid-ciphertext",
						TokenNonce: "invalid-nonce",
					}, nil
				}
			},
			wantConn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisClient, _ := newTestRedis(t)
			enc := newTestEncryptor(t)
			tokenRepo := &MockCalendarTokenRepository{}
			gcalClient := &MockGoogleCalendarClient{}

			// Store a valid token unless the test overrides GetByUserID
			storeValidToken(t, enc, tokenRepo)

			if tt.setup != nil {
				tt.setup(tokenRepo, gcalClient)
			}

			svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, gcalClient, enc, redisClient)

			status, err := svc.GetStatus(context.Background(), testUserID)

			require.NoError(t, err)
			require.NotNil(t, status)
			assert.Equal(t, tt.wantConn, status.Connected)
			assert.Equal(t, tt.wantEmail, status.Email)
		})
	}
}

// ---------------------------------------------------------------------------
// Disconnect tests
// ---------------------------------------------------------------------------

func TestCalendarService_Disconnect(t *testing.T) {
	t.Run("deletes token successfully", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		var deletedUserID string
		tokenRepo := &MockCalendarTokenRepository{
			DeleteFunc: func(_ context.Context, uid string) error {
				deletedUserID = uid
				return nil
			},
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		err := svc.Disconnect(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, testUserID, deletedUserID)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		tokenRepo := &MockCalendarTokenRepository{
			DeleteFunc: func(_ context.Context, _ string) error {
				return errors.New("delete failed")
			},
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		err := svc.Disconnect(context.Background(), testUserID)

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// CreateEvent tests
// ---------------------------------------------------------------------------

func TestCalendarService_CreateEvent(t *testing.T) {
	tests := []struct {
		name        string
		req         *model.CreateEventRequest
		setupStage  func(stageRepo *MockCalendarStageRepository)
		setupGCal   func(gcalClient *MockGoogleCalendarClient)
		wantErr     bool
		errIs       error
		validate    func(t *testing.T, dto *model.CalendarEventDTO)
	}{
		{
			name: "creates event successfully",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Phone Interview",
				StartTime:   "2025-06-15T10:00:00Z",
				DurationMin: 60,
				Description: "Phone screen with hiring manager",
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not found")
				}
				stageRepo.SetCalendarEventIDFunc = func(_ context.Context, stageID, eventID string) error {
					assert.Equal(t, "stage-1", stageID)
					assert.Equal(t, "evt_1", eventID)
					return nil
				}
			},
			setupGCal: func(gcalClient *MockGoogleCalendarClient) {
				gcalClient.CreateEventFunc = func(_ context.Context, _ *oauth2.Token, event *ports.CalendarEvent) (*ports.CreatedEvent, error) {
					assert.Equal(t, "Phone Interview", event.Title)
					return &ports.CreatedEvent{EventID: "evt_1", Link: "https://calendar.google.com/event/1"}, nil
				}
			},
			validate: func(t *testing.T, dto *model.CalendarEventDTO) {
				assert.Equal(t, "evt_1", dto.EventID)
				assert.Equal(t, "stage-1", dto.StageID)
				assert.Equal(t, "Phone Interview", dto.Title)
				assert.Equal(t, "https://calendar.google.com/event/1", dto.Link)
			},
		},
		{
			name: "returns error when stage not owned by user",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Interview",
				StartTime:   "2025-06-15T10:00:00Z",
				DurationMin: 60,
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return "other-user", nil
				}
			},
			wantErr: true,
			errIs:   model.ErrStageNotFound,
		},
		{
			name: "returns error when event already exists for stage",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Interview",
				StartTime:   "2025-06-15T10:00:00Z",
				DurationMin: 60,
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "existing-evt", nil
				}
			},
			wantErr: true,
			errIs:   model.ErrEventAlreadyExists,
		},
		{
			name: "returns error for invalid time format",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Interview",
				StartTime:   "invalid-time",
				DurationMin: 60,
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not found")
				}
			},
			wantErr: true,
			errIs:   model.ErrInvalidTimeRange,
		},
		{
			name: "returns error when Google API fails",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Interview",
				StartTime:   "2025-06-15T10:00:00Z",
				DurationMin: 60,
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not found")
				}
			},
			setupGCal: func(gcalClient *MockGoogleCalendarClient) {
				gcalClient.CreateEventFunc = func(_ context.Context, _ *oauth2.Token, _ *ports.CalendarEvent) (*ports.CreatedEvent, error) {
					return nil, errors.New("Google API error")
				}
			},
			wantErr: true,
		},
		{
			name: "returns error when stage ownership check fails",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Interview",
				StartTime:   "2025-06-15T10:00:00Z",
				DurationMin: 60,
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return "", errors.New("stage not found")
				}
			},
			wantErr: true,
		},
		{
			name: "returns error when SetCalendarEventID fails",
			req: &model.CreateEventRequest{
				StageID:     "stage-1",
				Title:       "Interview",
				StartTime:   "2025-06-15T10:00:00Z",
				DurationMin: 60,
			},
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not found")
				}
				stageRepo.SetCalendarEventIDFunc = func(_ context.Context, _, _ string) error {
					return errors.New("set failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisClient, _ := newTestRedis(t)
			enc := newTestEncryptor(t)
			tokenRepo := &MockCalendarTokenRepository{}
			stageRepo := &MockCalendarStageRepository{}
			gcalClient := &MockGoogleCalendarClient{}

			storeValidToken(t, enc, tokenRepo)

			if tt.setupStage != nil {
				tt.setupStage(stageRepo)
			}
			if tt.setupGCal != nil {
				tt.setupGCal(gcalClient)
			}

			svc := newCalendarService(tokenRepo, stageRepo, gcalClient, enc, redisClient)

			result, err := svc.CreateEvent(context.Background(), testUserID, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestCalendarService_CreateEvent_GetUserTokenError(t *testing.T) {
	redisClient, _ := newTestRedis(t)
	enc := newTestEncryptor(t)

	// Token repo returns error (no token stored)
	tokenRepo := &MockCalendarTokenRepository{
		GetByUserIDFunc: func(_ context.Context, _ string) (*model.CalendarToken, error) {
			return nil, errors.New("no token found")
		},
	}

	stageRepo := &MockCalendarStageRepository{
		GetStageUserIDFunc: func(_ context.Context, _ string) (string, error) {
			return testUserID, nil
		},
		GetCalendarEventIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not found")
		},
	}

	svc := newCalendarService(tokenRepo, stageRepo, &MockGoogleCalendarClient{}, enc, redisClient)

	_, err := svc.CreateEvent(context.Background(), testUserID, &model.CreateEventRequest{
		StageID:     "stage-1",
		Title:       "Interview",
		StartTime:   "2025-06-15T10:00:00Z",
		DurationMin: 60,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token found")
}

// ---------------------------------------------------------------------------
// DeleteEvent tests
// ---------------------------------------------------------------------------

func TestCalendarService_DeleteEvent(t *testing.T) {
	tests := []struct {
		name       string
		setupStage func(stageRepo *MockCalendarStageRepository)
		setupGCal  func(gcalClient *MockGoogleCalendarClient)
		wantErr    bool
		errIs      error
	}{
		{
			name: "deletes event successfully",
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "evt_1", nil
				}
				stageRepo.ClearCalendarEventIDFunc = func(_ context.Context, _ string) error {
					return nil
				}
			},
		},
		{
			name: "returns error when stage not owned by user",
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return "other-user", nil
				}
			},
			wantErr: true,
			errIs:   model.ErrStageNotFound,
		},
		{
			name: "continues when Google delete fails (best effort)",
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "evt_1", nil
				}
				stageRepo.ClearCalendarEventIDFunc = func(_ context.Context, _ string) error {
					return nil
				}
			},
			setupGCal: func(gcalClient *MockGoogleCalendarClient) {
				gcalClient.DeleteEventFunc = func(_ context.Context, _ *oauth2.Token, _ string) error {
					return errors.New("Google API error")
				}
			},
			wantErr: false, // best effort
		},
		{
			name: "returns error when GetCalendarEventID fails",
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "", errors.New("not found")
				}
			},
			wantErr: true,
		},
		{
			name: "returns error when ClearCalendarEventID fails",
			setupStage: func(stageRepo *MockCalendarStageRepository) {
				stageRepo.GetStageUserIDFunc = func(_ context.Context, _ string) (string, error) {
					return testUserID, nil
				}
				stageRepo.GetCalendarEventIDFunc = func(_ context.Context, _ string) (string, error) {
					return "evt_1", nil
				}
				stageRepo.ClearCalendarEventIDFunc = func(_ context.Context, _ string) error {
					return errors.New("clear failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisClient, _ := newTestRedis(t)
			enc := newTestEncryptor(t)
			tokenRepo := &MockCalendarTokenRepository{}
			stageRepo := &MockCalendarStageRepository{}
			gcalClient := &MockGoogleCalendarClient{}

			storeValidToken(t, enc, tokenRepo)

			if tt.setupStage != nil {
				tt.setupStage(stageRepo)
			}
			if tt.setupGCal != nil {
				tt.setupGCal(gcalClient)
			}

			svc := newCalendarService(tokenRepo, stageRepo, gcalClient, enc, redisClient)

			err := svc.DeleteEvent(context.Background(), testUserID, "stage-1")
			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCalendarService_DeleteEvent_GetStageUserIDError(t *testing.T) {
	redisClient, _ := newTestRedis(t)
	enc := newTestEncryptor(t)
	tokenRepo := &MockCalendarTokenRepository{}
	storeValidToken(t, enc, tokenRepo)

	stageRepo := &MockCalendarStageRepository{
		GetStageUserIDFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("stage repo error")
		},
	}

	svc := newCalendarService(tokenRepo, stageRepo, &MockGoogleCalendarClient{}, enc, redisClient)

	err := svc.DeleteEvent(context.Background(), testUserID, "stage-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage repo error")
}

func TestCalendarService_DeleteEvent_GetUserTokenError(t *testing.T) {
	redisClient, _ := newTestRedis(t)
	enc := newTestEncryptor(t)

	tokenRepo := &MockCalendarTokenRepository{
		GetByUserIDFunc: func(_ context.Context, _ string) (*model.CalendarToken, error) {
			return nil, errors.New("no token found")
		},
	}

	stageRepo := &MockCalendarStageRepository{
		GetStageUserIDFunc: func(_ context.Context, _ string) (string, error) {
			return testUserID, nil
		},
		GetCalendarEventIDFunc: func(_ context.Context, _ string) (string, error) {
			return "evt_1", nil
		},
	}

	svc := newCalendarService(tokenRepo, stageRepo, &MockGoogleCalendarClient{}, enc, redisClient)

	err := svc.DeleteEvent(context.Background(), testUserID, "stage-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token found")
}

// ---------------------------------------------------------------------------
// FrontendURL tests
// ---------------------------------------------------------------------------

func TestCalendarService_FrontendURL(t *testing.T) {
	redisClient, _ := newTestRedis(t)
	enc := newTestEncryptor(t)
	svc := newCalendarService(
		&MockCalendarTokenRepository{},
		&MockCalendarStageRepository{},
		&MockGoogleCalendarClient{},
		enc,
		redisClient,
	)

	assert.Equal(t, testFrontendURL, svc.FrontendURL())
}

// ---------------------------------------------------------------------------
// GetAuthURL tests
// ---------------------------------------------------------------------------

func TestCalendarService_GetAuthURL(t *testing.T) {
	t.Run("generates auth URL with state", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		url, err := svc.GetAuthURL(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Contains(t, url, "accounts.google.com")
		assert.Contains(t, url, "state=")
	})

	t.Run("returns error for empty userID", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		_, err := svc.GetAuthURL(context.Background(), "")

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// validateState tests
// ---------------------------------------------------------------------------

func TestCalendarService_ValidateState(t *testing.T) {
	t.Run("returns ErrInvalidState for unknown state", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		// HandleCallback uses validateState internally
		_, err := svc.HandleCallback(context.Background(), "some-code", "invalid-state")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidState)
	})
}

// ---------------------------------------------------------------------------
// storeToken / decryptToken round-trip tests
// ---------------------------------------------------------------------------

func TestCalendarService_StoreAndDecryptToken(t *testing.T) {
	t.Run("round-trips token through encrypt/decrypt", func(t *testing.T) {
		enc := newTestEncryptor(t)

		token := &oauth2.Token{
			AccessToken:  "access-123",
			RefreshToken: "refresh-456",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(1 * time.Hour).Truncate(time.Second),
		}

		tokenJSON, err := json.Marshal(token)
		require.NoError(t, err)

		ciphertext, nonce, err := enc.Encrypt(tokenJSON)
		require.NoError(t, err)

		calToken := &model.CalendarToken{
			TokenBlob:  ciphertext,
			TokenNonce: nonce,
		}

		redisClient, _ := newTestRedis(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		decrypted, err := svc.decryptToken(calToken)

		require.NoError(t, err)
		assert.Equal(t, token.AccessToken, decrypted.AccessToken)
		assert.Equal(t, token.RefreshToken, decrypted.RefreshToken)
	})

	t.Run("returns error for invalid ciphertext", func(t *testing.T) {
		enc := newTestEncryptor(t)
		redisClient, _ := newTestRedis(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		calToken := &model.CalendarToken{
			TokenBlob:  "invalid",
			TokenNonce: "invalid",
		}

		_, err := svc.decryptToken(calToken)
		require.Error(t, err)
	})

	t.Run("returns error for valid ciphertext but invalid JSON", func(t *testing.T) {
		enc := newTestEncryptor(t)
		redisClient, _ := newTestRedis(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		// Encrypt non-JSON data
		ciphertext, nonce, err := enc.Encrypt([]byte("this is not valid JSON"))
		require.NoError(t, err)

		calToken := &model.CalendarToken{
			TokenBlob:  ciphertext,
			TokenNonce: nonce,
		}

		_, err = svc.decryptToken(calToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrDecryptionFailed)
	})
}

// ---------------------------------------------------------------------------
// getUserToken tests
// ---------------------------------------------------------------------------

func TestCalendarService_GetUserToken(t *testing.T) {
	t.Run("returns token when valid and not expired", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}
		storeValidToken(t, enc, tokenRepo)

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		token, err := svc.getUserToken(context.Background(), testUserID)

		require.NoError(t, err)
		assert.NotEmpty(t, token.AccessToken)
	})

	t.Run("returns error when no token stored", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{
			GetByUserIDFunc: func(_ context.Context, _ string) (*model.CalendarToken, error) {
				return nil, errors.New("not found")
			},
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		_, err := svc.getUserToken(context.Background(), testUserID)

		require.Error(t, err)
	})

	t.Run("returns error when initial token decryption fails", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{
			GetByUserIDFunc: func(_ context.Context, _ string) (*model.CalendarToken, error) {
				return &model.CalendarToken{
					UserID:     testUserID,
					TokenBlob:  "corrupted-blob",
					TokenNonce: "corrupted-nonce",
				}, nil
			},
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		_, err := svc.getUserToken(context.Background(), testUserID)
		require.Error(t, err)
	})

	t.Run("returns token as-is when expired but no refresh token", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}

		// Create an expired token WITHOUT a refresh token
		token := &oauth2.Token{
			AccessToken: "expired-access-token",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(-1 * time.Hour), // expired
			// No RefreshToken
		}
		tokenJSON, err := json.Marshal(token)
		require.NoError(t, err)

		ciphertext, nonce, err := enc.Encrypt(tokenJSON)
		require.NoError(t, err)

		tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
			return &model.CalendarToken{
				UserID:     testUserID,
				TokenBlob:  ciphertext,
				TokenNonce: nonce,
			}, nil
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		result, err := svc.getUserToken(context.Background(), testUserID)

		// Should return the expired token without attempting refresh (no refresh token)
		require.NoError(t, err)
		assert.Equal(t, "expired-access-token", result.AccessToken)
	})

	t.Run("returns current token when lock not acquired during refresh", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}

		// Create an expired token WITH a refresh token
		token := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour), // expired
		}
		tokenJSON, err := json.Marshal(token)
		require.NoError(t, err)

		ciphertext, nonce, err := enc.Encrypt(tokenJSON)
		require.NoError(t, err)

		tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
			return &model.CalendarToken{
				UserID:     testUserID,
				TokenBlob:  ciphertext,
				TokenNonce: nonce,
			}, nil
		}

		// Pre-acquire the lock to simulate another goroutine refreshing
		lockKey := "token_refresh_lock:" + testUserID
		err = redisClient.SetNX(context.Background(), lockKey, "1", 10*time.Second).Err()
		require.NoError(t, err)

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		result, err := svc.getUserToken(context.Background(), testUserID)

		// Should return the current (expired) token when lock is not acquired
		require.NoError(t, err)
		assert.Equal(t, "expired-access-token", result.AccessToken)
	})

	t.Run("returns re-read token when it became valid after lock acquired", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}

		// First call: expired token with refresh token
		expiredToken := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		expiredJSON, err := json.Marshal(expiredToken)
		require.NoError(t, err)
		expiredCipher, expiredNonce, err := enc.Encrypt(expiredJSON)
		require.NoError(t, err)

		// Second call (re-read): now-valid token (another goroutine refreshed it)
		validToken := &oauth2.Token{
			AccessToken:  "refreshed-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(1 * time.Hour),
		}
		validJSON, err := json.Marshal(validToken)
		require.NoError(t, err)
		validCipher, validNonce, err := enc.Encrypt(validJSON)
		require.NoError(t, err)

		callCount := 0
		tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
			callCount++
			if callCount <= 1 {
				return &model.CalendarToken{
					UserID:     testUserID,
					TokenBlob:  expiredCipher,
					TokenNonce: expiredNonce,
				}, nil
			}
			return &model.CalendarToken{
				UserID:     testUserID,
				TokenBlob:  validCipher,
				TokenNonce: validNonce,
			}, nil
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		result, err := svc.getUserToken(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, "refreshed-access-token", result.AccessToken)
	})

	t.Run("returns error when re-read from repo fails after lock", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}

		// First call: expired token with refresh token
		expiredToken := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		expiredJSON, err := json.Marshal(expiredToken)
		require.NoError(t, err)
		expiredCipher, expiredNonce, err := enc.Encrypt(expiredJSON)
		require.NoError(t, err)

		callCount := 0
		tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
			callCount++
			if callCount <= 1 {
				return &model.CalendarToken{
					UserID:     testUserID,
					TokenBlob:  expiredCipher,
					TokenNonce: expiredNonce,
				}, nil
			}
			return nil, errors.New("repo error on re-read")
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		_, err = svc.getUserToken(context.Background(), testUserID)

		require.Error(t, err)
	})

	t.Run("returns ErrTokenExpired when refresh fails", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}

		// Create an expired token with a refresh token
		expiredToken := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		expiredJSON, err := json.Marshal(expiredToken)
		require.NoError(t, err)
		expiredCipher, expiredNonce, err := enc.Encrypt(expiredJSON)
		require.NoError(t, err)

		tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
			return &model.CalendarToken{
				UserID:     testUserID,
				TokenBlob:  expiredCipher,
				TokenNonce: expiredNonce,
			}, nil
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		// The token refresh will fail because the OAuth config points at a fake URL
		_, err = svc.getUserToken(context.Background(), testUserID)

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrTokenExpired)
	})

	t.Run("returns error when decryption fails during re-read after lock", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{}

		// First call: return a valid expired token that needs refresh
		token := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		tokenJSON, err := json.Marshal(token)
		require.NoError(t, err)

		ciphertext, nonce, err := enc.Encrypt(tokenJSON)
		require.NoError(t, err)

		callCount := 0
		tokenRepo.GetByUserIDFunc = func(_ context.Context, _ string) (*model.CalendarToken, error) {
			callCount++
			if callCount <= 1 {
				return &model.CalendarToken{
					UserID:     testUserID,
					TokenBlob:  ciphertext,
					TokenNonce: nonce,
				}, nil
			}
			// Second call (re-read after lock): return corrupted token
			return &model.CalendarToken{
				UserID:     testUserID,
				TokenBlob:  "corrupted",
				TokenNonce: "corrupted",
			}, nil
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		_, err = svc.getUserToken(context.Background(), testUserID)

		require.Error(t, err)
	})

	t.Run("returns refreshed token on successful refresh", func(t *testing.T) {
		// Create a mock OAuth token refresh endpoint
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "refreshed-access-token",
				"refresh_token": "new-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
			})
		}))
		defer tokenServer.Close()

		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		expiredToken := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		expiredJSON, err := json.Marshal(expiredToken)
		require.NoError(t, err)
		expiredCipher, expiredNonce, err := enc.Encrypt(expiredJSON)
		require.NoError(t, err)

		tokenRepo := &MockCalendarTokenRepository{
			GetByUserIDFunc: func(_ context.Context, _ string) (*model.CalendarToken, error) {
				return &model.CalendarToken{
					UserID:     testUserID,
					TokenBlob:  expiredCipher,
					TokenNonce: expiredNonce,
				}, nil
			},
			UpsertFunc: func(_ context.Context, _ *model.CalendarToken) error {
				return nil // successful store
			},
		}

		oauthCfg := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		svc := NewCalendarService(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			oauthCfg,
			redisClient,
			testFrontendURL,
		)

		result, err := svc.getUserToken(context.Background(), testUserID)
		require.NoError(t, err)
		assert.Equal(t, "refreshed-access-token", result.AccessToken)
	})

	t.Run("returns error when storeToken fails after successful refresh", func(t *testing.T) {
		// Create a mock OAuth token refresh endpoint
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
			})
		}))
		defer tokenServer.Close()

		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		// Create expired token with refresh token
		expiredToken := &oauth2.Token{
			AccessToken:  "expired-access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(-1 * time.Hour),
		}
		expiredJSON, err := json.Marshal(expiredToken)
		require.NoError(t, err)
		expiredCipher, expiredNonce, err := enc.Encrypt(expiredJSON)
		require.NoError(t, err)

		tokenRepo := &MockCalendarTokenRepository{
			GetByUserIDFunc: func(_ context.Context, _ string) (*model.CalendarToken, error) {
				return &model.CalendarToken{
					UserID:     testUserID,
					TokenBlob:  expiredCipher,
					TokenNonce: expiredNonce,
				}, nil
			},
			UpsertFunc: func(_ context.Context, _ *model.CalendarToken) error {
				return fmt.Errorf("database write failed")
			},
		}

		oauthCfg := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		svc := NewCalendarService(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			oauthCfg,
			redisClient,
			testFrontendURL,
		)

		_, err = svc.getUserToken(context.Background(), testUserID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database write failed")
	})
}

// ---------------------------------------------------------------------------
// storeToken tests
// ---------------------------------------------------------------------------

func TestCalendarService_StoreToken(t *testing.T) {
	t.Run("stores encrypted token successfully", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		var storedToken *model.CalendarToken
		tokenRepo := &MockCalendarTokenRepository{
			UpsertFunc: func(_ context.Context, ct *model.CalendarToken) error {
				storedToken = ct
				return nil
			},
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		token := &oauth2.Token{
			AccessToken:  "access-123",
			RefreshToken: "refresh-456",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(1 * time.Hour),
		}

		err := svc.storeToken(context.Background(), testUserID, token)

		require.NoError(t, err)
		require.NotNil(t, storedToken)
		assert.Equal(t, testUserID, storedToken.UserID)
		assert.NotEmpty(t, storedToken.TokenBlob)
		assert.NotEmpty(t, storedToken.TokenNonce)

		// Verify it can be decrypted back
		decrypted, err := svc.decryptToken(storedToken)
		require.NoError(t, err)
		assert.Equal(t, "access-123", decrypted.AccessToken)
	})

	t.Run("returns error when upsert fails", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		tokenRepo := &MockCalendarTokenRepository{
			UpsertFunc: func(_ context.Context, _ *model.CalendarToken) error {
				return errors.New("upsert failed")
			},
		}

		svc := newCalendarService(tokenRepo, &MockCalendarStageRepository{}, &MockGoogleCalendarClient{}, enc, redisClient)

		token := &oauth2.Token{
			AccessToken: "access-123",
		}

		err := svc.storeToken(context.Background(), testUserID, token)

		require.Error(t, err)
	})

	t.Run("returns error when encryption fails", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		// Create an encryptor with an invalid key to trigger encryption failure
		brokenEnc := &Encryptor{key: []byte("bad-key")}

		svc := NewCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			brokenEnc,
			newTestOAuthConfig(),
			redisClient,
			testFrontendURL,
		)

		token := &oauth2.Token{
			AccessToken: "access-123",
		}

		err := svc.storeToken(context.Background(), testUserID, token)

		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// HandleCallback tests (with pre-seeded Redis state)
// ---------------------------------------------------------------------------

func TestCalendarService_HandleCallback(t *testing.T) {
	t.Run("returns error for invalid state", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		_, err := svc.HandleCallback(context.Background(), "auth-code", "bad-state")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidState)
	})

	t.Run("returns error for empty userID in state", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		// Seed state with empty userID
		err := redisClient.Set(context.Background(), "oauth_state:test-state", "", 5*time.Minute).Err()
		require.NoError(t, err)

		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		_, err = svc.HandleCallback(context.Background(), "auth-code", "test-state")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidState)
	})

	t.Run("returns ErrCalendarAPI when token exchange fails", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		// Seed a valid state in Redis with a real userID
		err := redisClient.Set(context.Background(), "oauth_state:valid-state", testUserID, 5*time.Minute).Err()
		require.NoError(t, err)

		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		// The OAuth exchange will fail because the config points to a fake token URL
		_, err = svc.HandleCallback(context.Background(), "fake-auth-code", "valid-state")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrCalendarAPI)
	})

	t.Run("returns redirect URL on successful exchange", func(t *testing.T) {
		// Create a mock OAuth token endpoint
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
			})
		}))
		defer tokenServer.Close()

		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		// Seed a valid state in Redis
		err := redisClient.Set(context.Background(), "oauth_state:valid-state", testUserID, 5*time.Minute).Err()
		require.NoError(t, err)

		var storedToken *model.CalendarToken
		tokenRepo := &MockCalendarTokenRepository{
			UpsertFunc: func(_ context.Context, ct *model.CalendarToken) error {
				storedToken = ct
				return nil
			},
		}

		// Create OAuth config pointing to mock token server
		oauthCfg := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/auth",
				TokenURL: tokenServer.URL,
			},
			RedirectURL: "https://api.example.com/callback",
			Scopes:      []string{"calendar"},
		}

		svc := NewCalendarService(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			oauthCfg,
			redisClient,
			testFrontendURL,
		)

		redirectURL, err := svc.HandleCallback(context.Background(), "test-code", "valid-state")

		require.NoError(t, err)
		assert.Equal(t, testFrontendURL+"/settings?calendar=connected", redirectURL)
		require.NotNil(t, storedToken)
		assert.Equal(t, testUserID, storedToken.UserID)
		assert.NotEmpty(t, storedToken.TokenBlob)
		assert.NotEmpty(t, storedToken.TokenNonce)

		// Verify state was consumed (deleted from Redis)
		_, stateErr := redisClient.Get(context.Background(), "oauth_state:valid-state").Result()
		assert.Error(t, stateErr) // should be gone
	})

	t.Run("handles Redis Del error gracefully after successful exchange", func(t *testing.T) {
		redisClient, mr := newTestRedis(t)
		enc := newTestEncryptor(t)

		// Seed a valid state
		err := redisClient.Set(context.Background(), "oauth_state:del-fail-state", testUserID, 5*time.Minute).Err()
		require.NoError(t, err)

		var storedToken *model.CalendarToken
		tokenRepo := &MockCalendarTokenRepository{
			UpsertFunc: func(_ context.Context, ct *model.CalendarToken) error {
				storedToken = ct
				// After token is stored, make Redis return errors for the Del call
				mr.SetError("READONLY You can't write against a read only replica")
				return nil
			},
		}

		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}))
		defer tokenServer.Close()

		oauthCfg := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		svc := NewCalendarService(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			oauthCfg,
			redisClient,
			testFrontendURL,
		)

		redirectURL, callbackErr := svc.HandleCallback(context.Background(), "test-code", "del-fail-state")

		// The Del failure is non-fatal; HandleCallback should still succeed
		require.NoError(t, callbackErr)
		assert.Equal(t, testFrontendURL+"/settings?calendar=connected", redirectURL)
		require.NotNil(t, storedToken)

		// Reset miniredis error for cleanup
		mr.SetError("")
	})

	t.Run("returns error when storeToken fails after exchange", func(t *testing.T) {
		tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}))
		defer tokenServer.Close()

		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)

		err := redisClient.Set(context.Background(), "oauth_state:valid-state-2", testUserID, 5*time.Minute).Err()
		require.NoError(t, err)

		tokenRepo := &MockCalendarTokenRepository{
			UpsertFunc: func(_ context.Context, _ *model.CalendarToken) error {
				return fmt.Errorf("database unavailable")
			},
		}

		oauthCfg := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenServer.URL,
			},
		}

		svc := NewCalendarService(
			tokenRepo,
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			oauthCfg,
			redisClient,
			testFrontendURL,
		)

		_, err = svc.HandleCallback(context.Background(), "test-code", "valid-state-2")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "database unavailable")
	})
}

// ---------------------------------------------------------------------------
// generateState tests
// ---------------------------------------------------------------------------

func TestCalendarService_GenerateState(t *testing.T) {
	t.Run("generates unique states", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		state1, err := svc.generateState(context.Background(), testUserID)
		require.NoError(t, err)

		state2, err := svc.generateState(context.Background(), testUserID)
		require.NoError(t, err)

		assert.NotEqual(t, state1, state2)
		assert.Len(t, state1, 64) // 32 bytes hex-encoded
	})

	t.Run("returns error when Redis Set fails", func(t *testing.T) {
		redisClient, mr := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		// Close miniredis to make Set fail
		mr.Close()

		_, err := svc.generateState(context.Background(), testUserID)
		require.Error(t, err)
	})

	t.Run("stores state in Redis", func(t *testing.T) {
		redisClient, _ := newTestRedis(t)
		enc := newTestEncryptor(t)
		svc := newCalendarService(
			&MockCalendarTokenRepository{},
			&MockCalendarStageRepository{},
			&MockGoogleCalendarClient{},
			enc,
			redisClient,
		)

		state, err := svc.generateState(context.Background(), testUserID)
		require.NoError(t, err)

		// Verify state is stored in Redis
		stored, err := redisClient.Get(context.Background(), "oauth_state:"+state).Result()
		require.NoError(t, err)
		assert.Equal(t, testUserID, stored)
	})
}
