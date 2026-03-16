package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/ports"
)

// SubscriptionService handles subscription business logic.
type SubscriptionService struct {
	repo              ports.SubscriptionRepository
	webhookSecret     string
	paddleAPIKey      string
	proPriceID        string
	enterprisePriceID string
	clientToken       string
	environment       string
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	repo ports.SubscriptionRepository,
	webhookSecret string,
	paddleAPIKey string,
	proPriceID string,
	enterprisePriceID string,
	clientToken string,
	environment string,
) *SubscriptionService {
	return &SubscriptionService{
		repo:              repo,
		webhookSecret:     webhookSecret,
		paddleAPIKey:      paddleAPIKey,
		proPriceID:        proPriceID,
		enterprisePriceID: enterprisePriceID,
		clientToken:       clientToken,
		environment:       environment,
	}
}

// GetSubscription returns the current subscription with usage for a user.
func (s *SubscriptionService) GetSubscription(ctx context.Context, userID string) (*model.SubscriptionDTO, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	usage, err := s.getUsage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	return sub.ToDTO(usage), nil
}

// GetCheckoutConfig returns Paddle checkout configuration for the frontend.
func (s *SubscriptionService) GetCheckoutConfig() *model.CheckoutConfigDTO {
	prices := map[string]string{
		"pro": s.proPriceID,
	}
	if s.enterprisePriceID != "" {
		prices["enterprise"] = s.enterprisePriceID
	}
	return &model.CheckoutConfigDTO{
		ClientToken: s.clientToken,
		Prices:      prices,
		Environment: s.environment,
	}
}

// CheckLimit checks if a user can create another resource of the given type.
func (s *SubscriptionService) CheckLimit(ctx context.Context, userID, resource string) error {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		// If no subscription found, treat as free plan
		if errors.Is(err, model.ErrSubscriptionNotFound) {
			sub = &model.Subscription{Plan: "free", Status: "free"}
		} else {
			return fmt.Errorf("failed to get subscription: %w", err)
		}
	}

	limits := model.GetLimitsForPlan(sub.Plan)

	var current int
	var max int

	switch resource {
	case "jobs":
		max = limits.MaxJobs
		if max < 0 {
			return nil
		}
		current, err = s.repo.CountUserJobs(ctx, userID)
	case "resumes":
		max = limits.MaxResumes
		if max < 0 {
			return nil
		}
		current, err = s.repo.CountUserResumes(ctx, userID)
	case "applications":
		max = limits.MaxApplications
		if max < 0 {
			return nil
		}
		current, err = s.repo.CountUserApplications(ctx, userID)
	case "ai":
		max = limits.MaxAIRequests
		if max < 0 {
			return nil
		}
		if max == 0 {
			return model.ErrLimitReached
		}
		current, err = s.repo.CountUserAIRequestsThisMonth(ctx, userID)
	case "job_parses":
		max = limits.MaxJobParses
		if max < 0 {
			return nil
		}
		if max == 0 {
			return model.ErrLimitReached
		}
		current, err = s.repo.CountUserJobParsesThisMonth(ctx, userID)
	case "resume_builders":
		max = limits.MaxResumeBuilders
		if max < 0 {
			return nil
		}
		if max == 0 {
			return model.ErrLimitReached
		}
		current, err = s.repo.CountUserResumeBuilders(ctx, userID)
	case "cover_letters":
		max = limits.MaxCoverLetters
		if max < 0 {
			return nil
		}
		if max == 0 {
			return model.ErrLimitReached
		}
		current, err = s.repo.CountUserCoverLetters(ctx, userID)
	default:
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to count %s: %w", resource, err)
	}

	if current >= max {
		return model.ErrLimitReached
	}

	return nil
}

// RecordAIUsage records an AI usage event for the user.
func (s *SubscriptionService) RecordAIUsage(ctx context.Context, userID string) error {
	return s.repo.RecordAIUsage(ctx, userID)
}

// RecordJobParseUsage records a job parse usage event for the user.
func (s *SubscriptionService) RecordJobParseUsage(ctx context.Context, userID string) error {
	return s.repo.RecordJobParseUsage(ctx, userID)
}

// EnsureFreeSubscription creates a free subscription for a user if one doesn't exist.
func (s *SubscriptionService) EnsureFreeSubscription(ctx context.Context, userID string) error {
	sub := &model.Subscription{
		UserID: userID,
		Status: "free",
		Plan:   "free",
	}
	return s.repo.Upsert(ctx, sub)
}

// HandleWebhook verifies and processes a Paddle webhook event.
func (s *SubscriptionService) HandleWebhook(ctx context.Context, body []byte, signature string) error {
	// Verify webhook signature
	if err := s.verifyWebhookSignature(body, signature); err != nil {
		return fmt.Errorf("invalid webhook signature: %w", err)
	}

	// Parse the event
	var event paddleEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("failed to parse webhook event: %w", err)
	}

	switch event.EventType {
	case "subscription.created", "subscription.activated":
		return s.handleSubscriptionActivated(ctx, &event)
	case "subscription.updated":
		return s.handleSubscriptionUpdated(ctx, &event)
	case "subscription.canceled":
		return s.handleSubscriptionCanceled(ctx, &event)
	case "subscription.past_due":
		return s.handleSubscriptionPastDue(ctx, &event)
	default:
		// Ignore unhandled events
		return nil
	}
}

// CreatePortalSession creates a Paddle customer portal session URL.
func (s *SubscriptionService) CreatePortalSession(ctx context.Context, userID string) (string, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}

	if sub.PaddleSubscriptionID == nil || *sub.PaddleSubscriptionID == "" {
		return "", fmt.Errorf("no paddle subscription found")
	}
	if sub.PaddleCustomerID == nil || *sub.PaddleCustomerID == "" {
		return "", fmt.Errorf("no paddle customer ID found")
	}

	// Determine API base URL
	baseURL := "https://api.paddle.com"
	if s.environment == "sandbox" {
		baseURL = "https://sandbox-api.paddle.com"
	}

	// Call Paddle API to create a portal session
	reqBody, err := json.Marshal(map[string][]string{
		"subscription_ids": {*sub.PaddleSubscriptionID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		baseURL+"/customers/"+*sub.PaddleCustomerID+"/portal-sessions",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.paddleAPIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call Paddle API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return "", fmt.Errorf("paddle API error (status %d), failed to read body: %w", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("paddle API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var portalResp struct {
		Data struct {
			URLs struct {
				General []struct {
					Overview string `json:"overview"`
				} `json:"general"`
			} `json:"urls"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&portalResp); err != nil {
		return "", fmt.Errorf("failed to decode portal response: %w", err)
	}

	if len(portalResp.Data.URLs.General) == 0 {
		return "", fmt.Errorf("no portal URL returned")
	}

	return portalResp.Data.URLs.General[0].Overview, nil
}

// getUsage returns current resource usage for a user in a single query.
func (s *SubscriptionService) getUsage(ctx context.Context, userID string) (model.Usage, error) {
	jobs, resumes, apps, aiReqs, jobParses, resumeBuilders, coverLetters, err := s.repo.GetAllCounts(ctx, userID)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		Jobs:           jobs,
		Resumes:        resumes,
		Applications:   apps,
		AIRequests:     aiReqs,
		JobParses:      jobParses,
		ResumeBuilders: resumeBuilders,
		CoverLetters:   coverLetters,
	}, nil
}

// verifyWebhookSignature verifies Paddle webhook signature using HMAC-SHA256.
func (s *SubscriptionService) verifyWebhookSignature(payload []byte, signature string) error {
	if s.webhookSecret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}

	// Paddle signature format: ts=<timestamp>;h1=<hash>
	parts := strings.Split(signature, ";")
	if len(parts) != 2 {
		return fmt.Errorf("invalid signature format")
	}

	var ts string
	var h1 string
	for _, part := range parts {
		if v, ok := strings.CutPrefix(part, "ts="); ok {
			ts = v
		} else if v, ok := strings.CutPrefix(part, "h1="); ok {
			h1 = v
		}
	}

	if ts == "" || h1 == "" {
		return fmt.Errorf("missing timestamp or hash in signature")
	}

	// Verify timestamp is not too old (5 minutes tolerance)
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	if time.Since(time.Unix(tsInt, 0)) > 5*time.Minute {
		return fmt.Errorf("webhook timestamp too old")
	}

	// Compute expected signature: HMAC-SHA256(ts:body)
	signedPayload := ts + ":" + string(payload)
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(h1), []byte(expectedMAC)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// Paddle webhook event types

type paddleEvent struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type paddleSubscriptionData struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	CustomerID       string  `json:"customer_id"`
	CurrentBillingPeriod *struct {
		StartsAt string `json:"starts_at"`
		EndsAt   string `json:"ends_at"`
	} `json:"current_billing_period"`
	ScheduledChange *struct {
		Action      string `json:"action"`
		EffectiveAt string `json:"effective_at"`
	} `json:"scheduled_change"`
	CustomData *struct {
		UserID string `json:"user_id"`
	} `json:"custom_data"`
	Items []struct {
		Price struct {
			ID string `json:"id"`
		} `json:"price"`
	} `json:"items"`
}

func (s *SubscriptionService) parseSubscriptionData(event *paddleEvent) (*paddleSubscriptionData, error) {
	var data paddleSubscriptionData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse subscription data: %w", err)
	}
	return &data, nil
}

func (s *SubscriptionService) handleSubscriptionActivated(ctx context.Context, event *paddleEvent) error {
	data, err := s.parseSubscriptionData(event)
	if err != nil {
		return err
	}

	if data.CustomData == nil || data.CustomData.UserID == "" {
		return fmt.Errorf("missing user_id in custom_data")
	}

	// Validate user_id is a valid UUID to prevent injection
	if _, err := parseUUID(data.CustomData.UserID); err != nil {
		return fmt.Errorf("invalid user_id in custom_data: %w", err)
	}

	plan := s.determinePlanFromEvent(data)

	sub := &model.Subscription{
		UserID:               data.CustomData.UserID,
		PaddleSubscriptionID: &data.ID,
		PaddleCustomerID:     &data.CustomerID,
		Status:               "active",
		Plan:                 plan,
	}

	if data.CurrentBillingPeriod != nil {
		if start, err := time.Parse(time.RFC3339, data.CurrentBillingPeriod.StartsAt); err == nil {
			sub.CurrentPeriodStart = &start
		}
		if end, err := time.Parse(time.RFC3339, data.CurrentBillingPeriod.EndsAt); err == nil {
			sub.CurrentPeriodEnd = &end
		}
	}

	return s.repo.Upsert(ctx, sub)
}

func (s *SubscriptionService) handleSubscriptionUpdated(ctx context.Context, event *paddleEvent) error {
	data, err := s.parseSubscriptionData(event)
	if err != nil {
		return err
	}

	// Try to find subscription by Paddle ID first
	existing, err := s.repo.GetByPaddleSubscriptionID(ctx, data.ID)
	if err != nil {
		return fmt.Errorf("subscription not found for paddle ID %s: %w", data.ID, err)
	}

	existing.Status = mapPaddleStatus(data.Status)
	existing.Plan = s.determinePlanFromEvent(data)

	if data.CurrentBillingPeriod != nil {
		if start, err := time.Parse(time.RFC3339, data.CurrentBillingPeriod.StartsAt); err == nil {
			existing.CurrentPeriodStart = &start
		}
		if end, err := time.Parse(time.RFC3339, data.CurrentBillingPeriod.EndsAt); err == nil {
			existing.CurrentPeriodEnd = &end
		}
	}

	if data.ScheduledChange != nil && data.ScheduledChange.Action == "cancel" {
		if cancelAt, err := time.Parse(time.RFC3339, data.ScheduledChange.EffectiveAt); err == nil {
			existing.CancelAt = &cancelAt
		}
	}

	return s.repo.Upsert(ctx, existing)
}

func (s *SubscriptionService) handleSubscriptionCanceled(ctx context.Context, event *paddleEvent) error {
	data, err := s.parseSubscriptionData(event)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetByPaddleSubscriptionID(ctx, data.ID)
	if err != nil {
		return fmt.Errorf("subscription not found for paddle ID %s: %w", data.ID, err)
	}

	existing.Status = "cancelled"
	existing.Plan = "free"
	existing.CancelAt = nil

	return s.repo.Upsert(ctx, existing)
}

func (s *SubscriptionService) handleSubscriptionPastDue(ctx context.Context, event *paddleEvent) error {
	data, err := s.parseSubscriptionData(event)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetByPaddleSubscriptionID(ctx, data.ID)
	if err != nil {
		return fmt.Errorf("subscription not found for paddle ID %s: %w", data.ID, err)
	}

	existing.Status = "past_due"

	return s.repo.Upsert(ctx, existing)
}

// determinePlanFromEvent determines the plan based on price IDs in the subscription items.
func (s *SubscriptionService) determinePlanFromEvent(data *paddleSubscriptionData) string {
	for _, item := range data.Items {
		if s.enterprisePriceID != "" && item.Price.ID == s.enterprisePriceID {
			return "enterprise"
		}
	}
	return "pro"
}

// parseUUID validates that a string is a valid UUID format.
func parseUUID(s string) (string, error) {
	if len(s) != 36 {
		return "", fmt.Errorf("invalid UUID length: %d", len(s))
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return "", fmt.Errorf("invalid UUID format")
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", fmt.Errorf("invalid UUID character at position %d", i)
		}
	}
	return s, nil
}

// mapPaddleStatus maps Paddle subscription status to our internal status.
func mapPaddleStatus(paddleStatus string) string {
	switch paddleStatus {
	case "active":
		return "active"
	case "past_due":
		return "past_due"
	case "canceled":
		return "cancelled"
	case "paused":
		return "paused"
	default:
		return paddleStatus
	}
}
