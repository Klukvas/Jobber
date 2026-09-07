package model

import (
	"errors"
	"strings"
	"time"
)

// Error sentinels
var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrLimitReached         = errors.New("plan limit reached")
	ErrPaidFeature          = errors.New("paid plan required")
	// ErrNoActiveSubscription is returned when an operation needs a provider
	// subscription (change plan, cancel, portal) but the user has none.
	ErrNoActiveSubscription = errors.New("no active provider subscription")
	// ErrUnknownPlan is returned for a plan name that is not purchasable.
	ErrUnknownPlan = errors.New("unknown plan")
	// ErrAlreadySubscribed is returned when a user who already holds a provider
	// subscription tries to start a *second* checkout. The row carries a single
	// external_subscription_id, so a second purchase would overwrite the first
	// and leave it billing invisibly — plan changes must go through the provider
	// subscription instead.
	ErrAlreadySubscribed = errors.New("user already has a provider subscription")
)

// Subscription represents a user's subscription record.
//
// The external IDs are provider-neutral: with FastSpring they hold the
// subscription ID and the customer account ID respectively.
type Subscription struct {
	ID                     string
	UserID                 string
	ExternalSubscriptionID *string
	ExternalAccountID      *string
	Status                 string // free, active, past_due, cancelled, paused
	Plan                   string // free, pro, enterprise
	CurrentPeriodStart     *time.Time
	CurrentPeriodEnd       *time.Time
	CancelAt               *time.Time
	// LastEventAt is the provider's own timestamp for the newest subscription
	// *state change* applied to this row — the payload's `data.changed`, not the
	// moment the webhook was created or delivered. (The charge events carry no
	// `changed`, so those fall back to the envelope's `created`.) The
	// distinction is the point: a manual resend arrives in a fresh envelope with
	// a fresh `created`, and using that would let it overwrite newer state.
	// An event is applied only when it is *strictly* newer, so neither a replay
	// nor a second event describing the same change can undo what already stands.
	LastEventAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SubscriptionDTO is the JSON response for a subscription.
type SubscriptionDTO struct {
	Plan             string     `json:"plan"`
	Status           string     `json:"status"`
	Limits           PlanLimits `json:"limits"`
	Usage            Usage      `json:"usage"`
	CurrentPeriodEnd *string    `json:"current_period_end,omitempty"`
	CancelAt         *string    `json:"cancel_at,omitempty"`
}

// Usage holds resource usage counts.
type Usage struct {
	Jobs           int `json:"jobs"`
	Resumes        int `json:"resumes"`
	AIRequests     int `json:"ai_requests"`
	JobParses      int `json:"job_parses"`
	ResumeBuilders int `json:"resume_builders"`
	CoverLetters   int `json:"cover_letters"`
}

// PlanLimits defines resource limits for a plan. -1 means unlimited.
type PlanLimits struct {
	MaxJobs           int `json:"max_jobs"`
	MaxResumes        int `json:"max_resumes"`
	MaxAIRequests     int `json:"max_ai_requests"`
	MaxJobParses      int `json:"max_job_parses"`
	MaxResumeBuilders int `json:"max_resume_builders"`
	MaxCoverLetters   int `json:"max_cover_letters"`
}

// FreePlanLimits defines limits for the free plan. These are fallback defaults
// for when config/plans.yaml is absent; plans.yaml is the source of truth and
// overrides them at startup via ApplyPlansConfig (prod ships max_cover_letters: 0).
var FreePlanLimits = PlanLimits{
	MaxJobs:           25,
	MaxResumes:        1,
	MaxAIRequests:     1,
	MaxJobParses:      5,
	MaxResumeBuilders: 1,
	MaxCoverLetters:   1,
}

// ProPlanLimits defines limits for the pro plan.
var ProPlanLimits = PlanLimits{
	MaxJobs:           100,
	MaxResumes:        10,
	MaxAIRequests:     -1,
	MaxJobParses:      -1,
	MaxResumeBuilders: 10,
	MaxCoverLetters:   10,
}

// EnterprisePlanLimits defines limits for the enterprise plan (-1 = unlimited).
var EnterprisePlanLimits = PlanLimits{
	MaxJobs:           -1,
	MaxResumes:        -1,
	MaxAIRequests:     -1,
	MaxJobParses:      -1,
	MaxResumeBuilders: -1,
	MaxCoverLetters:   -1,
}

// PlanLimitsConfig mirrors PlanLimits for external YAML config loading.
type PlanLimitsConfig struct {
	MaxJobs           int `yaml:"max_jobs"`
	MaxResumes        int `yaml:"max_resumes"`
	MaxAIRequests     int `yaml:"max_ai_requests"`
	MaxJobParses      int `yaml:"max_job_parses"`
	MaxResumeBuilders int `yaml:"max_resume_builders"`
	MaxCoverLetters   int `yaml:"max_cover_letters"`
}

// ApplyPlansConfig overwrites default plan limits from externally loaded config.
// Only plans present in the map are overwritten; missing plans keep hardcoded defaults.
func ApplyPlansConfig(plans map[string]PlanLimitsConfig) {
	if p, ok := plans["free"]; ok {
		FreePlanLimits = configToPlanLimits(p)
	}
	if p, ok := plans["pro"]; ok {
		ProPlanLimits = configToPlanLimits(p)
	}
	if p, ok := plans["enterprise"]; ok {
		EnterprisePlanLimits = configToPlanLimits(p)
	}
}

func configToPlanLimits(c PlanLimitsConfig) PlanLimits {
	return PlanLimits{
		MaxJobs:           c.MaxJobs,
		MaxResumes:        c.MaxResumes,
		MaxAIRequests:     c.MaxAIRequests,
		MaxJobParses:      c.MaxJobParses,
		MaxResumeBuilders: c.MaxResumeBuilders,
		MaxCoverLetters:   c.MaxCoverLetters,
	}
}

// GetLimitsForPlan returns plan limits for the given plan name.
func GetLimitsForPlan(plan string) PlanLimits {
	switch plan {
	case "pro":
		return ProPlanLimits
	case "enterprise":
		return EnterprisePlanLimits
	default:
		return FreePlanLimits
	}
}

// IsActive returns true if the subscription grants paid-plan access.
func (s *Subscription) IsActive() bool {
	return (s.Plan == "pro" || s.Plan == "enterprise") && (s.Status == "active" || s.Status == "past_due")
}

// ToDTO converts a Subscription to SubscriptionDTO with usage counts.
func (s *Subscription) ToDTO(usage Usage) *SubscriptionDTO {
	dto := &SubscriptionDTO{
		Plan:   s.Plan,
		Status: s.Status,
		Limits: GetLimitsForPlan(s.Plan),
		Usage:  usage,
	}

	if s.CurrentPeriodEnd != nil {
		formatted := s.CurrentPeriodEnd.Format(time.RFC3339)
		dto.CurrentPeriodEnd = &formatted
	}
	if s.CancelAt != nil {
		formatted := s.CancelAt.Format(time.RFC3339)
		dto.CancelAt = &formatted
	}

	return dto
}

// CheckoutConfigDTO tells the frontend which billing provider is wired up, which
// plans are purchasable, and which storefront the provider's popup script must
// be pointed at. It deliberately carries no credentials, no API keys and no
// catalog product paths: checkout sessions are created server-side.
type CheckoutConfigDTO struct {
	Provider    string `json:"provider"`
	Environment string `json:"environment"`
	// Storefront is the popup storefront the browser loads the provider's
	// Store Builder Library against, as "<host>/<popup-checkout-id>". It is
	// *derived* from the configured checkout path and the environment, never
	// configured separately, so the two cannot drift apart. Empty means the
	// checkout is not openable and the frontend must not offer it.
	Storefront string   `json:"storefront"`
	Plans      []string `json:"plans"`
}

// CheckoutSessionDTO holds a provider checkout session for the popup to open.
//
// There is deliberately no URL: the popup takes the opaque session id, so the
// browser is never handed a page to navigate to.
type CheckoutSessionDTO struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// PortalSessionDTO holds the authenticated customer account portal URL.
type PortalSessionDTO struct {
	URL string `json:"url"`
}

// ChangePlanRequest is the request body for changing a subscription plan.
type ChangePlanRequest struct {
	Plan string `json:"plan" binding:"required"`
}

// CheckoutSessionRequest is the request body for starting a checkout. Only the
// plan is accepted: the buyer is taken from the authenticated session.
type CheckoutSessionRequest struct {
	Plan string `json:"plan" binding:"required"`
}

// UserContact carries the buyer details handed to the billing provider when a
// checkout session is created. Sourced from the authenticated user record, never
// from the client.
type UserContact struct {
	Email  string
	Name   string
	Locale string
}

// FirstLast splits the stored display name into the first/last pair the billing
// provider expects. A single-word name becomes the first name only.
func (c UserContact) FirstLast() (first, last string) {
	fields := strings.Fields(c.Name)
	switch len(fields) {
	case 0:
		return "", ""
	case 1:
		return fields[0], ""
	default:
		return fields[0], strings.Join(fields[1:], " ")
	}
}
