package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/andreypavlenko/jobber/modules/calendar/ports"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

const oauthStateTTL = 10 * time.Minute

// CalendarService handles calendar business logic
type CalendarService struct {
	tokenRepo   ports.CalendarTokenRepository
	stageRepo   ports.CalendarStageRepository
	gcalClient  ports.GoogleCalendarClient
	encryptor   *Encryptor
	oauthConfig *oauth2.Config
	redisClient *redis.Client
	frontendURL string
}

// NewCalendarService creates a new calendar service
func NewCalendarService(
	tokenRepo ports.CalendarTokenRepository,
	stageRepo ports.CalendarStageRepository,
	gcalClient ports.GoogleCalendarClient,
	encryptor *Encryptor,
	oauthConfig *oauth2.Config,
	redisClient *redis.Client,
	frontendURL string,
) *CalendarService {
	return &CalendarService{
		tokenRepo:   tokenRepo,
		stageRepo:   stageRepo,
		gcalClient:  gcalClient,
		encryptor:   encryptor,
		oauthConfig: oauthConfig,
		redisClient: redisClient,
		frontendURL: frontendURL,
	}
}

// GetAuthURL generates a Google OAuth URL with CSRF state
func (s *CalendarService) GetAuthURL(ctx context.Context, userID string) (string, error) {
	state, err := s.generateState(ctx, userID)
	if err != nil {
		return "", err
	}

	url := s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	return url, nil
}

// HandleCallback processes the OAuth callback and stores the encrypted token
func (s *CalendarService) HandleCallback(ctx context.Context, code, state string) (string, error) {
	userID, err := s.validateState(ctx, state)
	if err != nil {
		return "", err
	}

	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("%w: %v", model.ErrCalendarAPI, err)
	}

	if err := s.storeToken(ctx, userID, token); err != nil {
		return "", err
	}

	// Consume state only after successful exchange (one-time use)
	key := "oauth_state:" + state
	if delErr := s.redisClient.Del(ctx, key).Err(); delErr != nil {
		// Non-fatal: TTL will expire the key; log for observability
		fmt.Printf("warning: failed to delete OAuth state key: %v\n", delErr)
	}

	return s.frontendURL + "/settings?calendar=connected", nil
}

// GetStatus checks if the user has a connected calendar
func (s *CalendarService) GetStatus(ctx context.Context, userID string) (*model.CalendarStatusDTO, error) {
	tokenRow, err := s.tokenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return &model.CalendarStatusDTO{Connected: false}, nil
	}

	token, err := s.decryptToken(tokenRow)
	if err != nil {
		return &model.CalendarStatusDTO{Connected: false}, nil
	}

	email, err := s.gcalClient.GetUserEmail(ctx, token)
	if err != nil {
		return &model.CalendarStatusDTO{Connected: true}, nil
	}

	return &model.CalendarStatusDTO{Connected: true, Email: email}, nil
}

// Disconnect removes the stored calendar token
func (s *CalendarService) Disconnect(ctx context.Context, userID string) error {
	return s.tokenRepo.Delete(ctx, userID)
}

// CreateEvent creates a Google Calendar event for a stage
func (s *CalendarService) CreateEvent(ctx context.Context, userID string, req *model.CreateEventRequest) (*model.CalendarEventDTO, error) {
	// Verify stage ownership
	stageUserID, err := s.stageRepo.GetStageUserID(ctx, req.StageID)
	if err != nil {
		return nil, err
	}
	if stageUserID != userID {
		return nil, model.ErrStageNotFound
	}

	// Prevent duplicate events — check if stage already has a calendar event
	existingID, err := s.stageRepo.GetCalendarEventID(ctx, req.StageID)
	if err == nil && existingID != "" {
		return nil, model.ErrEventAlreadyExists
	}

	// Get user's OAuth token
	token, err := s.getUserToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Parse time
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, model.ErrInvalidTimeRange
	}
	endTime := startTime.Add(time.Duration(req.DurationMin) * time.Minute)

	// Create the Google Calendar event
	created, err := s.gcalClient.CreateEvent(ctx, token, &ports.CalendarEvent{
		Title:       req.Title,
		Description: req.Description,
		StartTime:   startTime.Format(time.RFC3339),
		EndTime:     endTime.Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}

	// Save event ID to stage
	if err := s.stageRepo.SetCalendarEventID(ctx, req.StageID, created.EventID); err != nil {
		// Best effort: if stage update fails, the event is already created in Google
		return nil, err
	}

	return &model.CalendarEventDTO{
		EventID:   created.EventID,
		StageID:   req.StageID,
		Title:     req.Title,
		StartTime: startTime,
		EndTime:   endTime,
		Link:      created.Link,
	}, nil
}

// DeleteEvent deletes a Google Calendar event for a stage
func (s *CalendarService) DeleteEvent(ctx context.Context, userID, stageID string) error {
	// Verify stage ownership
	stageUserID, err := s.stageRepo.GetStageUserID(ctx, stageID)
	if err != nil {
		return err
	}
	if stageUserID != userID {
		return model.ErrStageNotFound
	}

	// Get event ID
	eventID, err := s.stageRepo.GetCalendarEventID(ctx, stageID)
	if err != nil {
		return err
	}

	// Get user's OAuth token
	token, err := s.getUserToken(ctx, userID)
	if err != nil {
		return err
	}

	// Delete from Google Calendar — continue clearing local reference even on failure
	if delErr := s.gcalClient.DeleteEvent(ctx, token, eventID); delErr != nil {
		// Best effort: event may be orphaned in Google Calendar
		fmt.Printf("warning: failed to delete Google Calendar event %s: %v\n", eventID, delErr)
	}

	// Clear event ID from stage
	return s.stageRepo.ClearCalendarEventID(ctx, stageID)
}

// FrontendURL returns the configured frontend URL
func (s *CalendarService) FrontendURL() string {
	return s.frontendURL
}

func (s *CalendarService) getUserToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	tokenRow, err := s.tokenRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	token, err := s.decryptToken(tokenRow)
	if err != nil {
		return nil, err
	}

	// Check if token is expired and has a refresh token
	// Use token.Valid() to correctly handle zero-expiry tokens
	if !token.Valid() && token.RefreshToken != "" {
		// Distributed lock to prevent concurrent refresh race condition
		// (Google rotates refresh tokens on use — concurrent refreshes can invalidate each other)
		lockKey := "token_refresh_lock:" + userID
		acquired, lockErr := s.redisClient.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
		if lockErr != nil || !acquired {
			// Another goroutine is refreshing; return current token and let OAuth client handle 401
			return token, nil
		}
		defer s.redisClient.Del(ctx, lockKey)

		// Re-read token in case another goroutine already refreshed it
		tokenRow, err = s.tokenRepo.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		token, err = s.decryptToken(tokenRow)
		if err != nil {
			return nil, err
		}
		if token.Valid() {
			return token, nil
		}

		src := s.oauthConfig.TokenSource(ctx, token)
		newToken, err := src.Token()
		if err != nil {
			return nil, model.ErrTokenExpired
		}
		// Store refreshed token
		if err := s.storeToken(ctx, userID, newToken); err != nil {
			return nil, err
		}
		return newToken, nil
	}

	return token, nil
}

func (s *CalendarService) storeToken(ctx context.Context, userID string, token *oauth2.Token) error {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("%w: %v", model.ErrEncryptionFailed, err)
	}

	ciphertext, nonce, err := s.encryptor.Encrypt(tokenJSON)
	if err != nil {
		return err
	}

	return s.tokenRepo.Upsert(ctx, &model.CalendarToken{
		UserID:     userID,
		TokenBlob:  ciphertext,
		TokenNonce: nonce,
	})
}

func (s *CalendarService) decryptToken(tokenRow *model.CalendarToken) (*oauth2.Token, error) {
	plaintext, err := s.encryptor.Decrypt(tokenRow.TokenBlob, tokenRow.TokenNonce)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return nil, fmt.Errorf("%w: invalid token data", model.ErrDecryptionFailed)
	}

	return &token, nil
}

func (s *CalendarService) generateState(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID must not be empty")
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	state := hex.EncodeToString(randomBytes)

	// Store state -> userID in Redis
	key := "oauth_state:" + state
	if err := s.redisClient.Set(ctx, key, userID, oauthStateTTL).Err(); err != nil {
		return "", err
	}

	return state, nil
}

func (s *CalendarService) validateState(ctx context.Context, state string) (string, error) {
	key := "oauth_state:" + state
	userID, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return "", model.ErrInvalidState
	}
	if userID == "" {
		return "", model.ErrInvalidState
	}

	// Note: state is consumed AFTER successful exchange in HandleCallback
	return userID, nil
}
