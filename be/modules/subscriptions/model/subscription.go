package model

import (
	"errors"
	"time"
)

// Error sentinels
var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrLimitReached         = errors.New("plan limit reached")
)

// Subscription represents a user's subscription record.
type Subscription struct {
	ID                   string
	UserID               string
	PaddleSubscriptionID *string
	PaddleCustomerID     *string
	Status               string // free, active, past_due, cancelled, paused
	Plan                 string // free, pro, enterprise
	CurrentPeriodStart   *time.Time
	CurrentPeriodEnd     *time.Time
	CancelAt             *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SubscriptionDTO is the JSON response for a subscription.
type SubscriptionDTO struct {
	Plan             string    `json:"plan"`
	Status           string    `json:"status"`
	Limits           PlanLimits `json:"limits"`
	Usage            Usage     `json:"usage"`
	CurrentPeriodEnd *string   `json:"current_period_end,omitempty"`
	CancelAt         *string   `json:"cancel_at,omitempty"`
}

// Usage holds resource usage counts.
type Usage struct {
	Jobs         int `json:"jobs"`
	Resumes      int `json:"resumes"`
	Applications int `json:"applications"`
	AIRequests   int `json:"ai_requests"`
	JobParses    int `json:"job_parses"`
}

// PlanLimits defines resource limits for a plan. -1 means unlimited.
type PlanLimits struct {
	MaxJobs         int `json:"max_jobs"`
	MaxResumes      int `json:"max_resumes"`
	MaxApplications int `json:"max_applications"`
	MaxAIRequests   int `json:"max_ai_requests"`
	MaxJobParses    int `json:"max_job_parses"`
}

// FreePlanLimits defines limits for the free plan.
var FreePlanLimits = PlanLimits{
	MaxJobs:         5,
	MaxResumes:      1,
	MaxApplications: 5,
	MaxAIRequests:   1,
	MaxJobParses:    5,
}

// ProPlanLimits defines limits for the pro plan.
var ProPlanLimits = PlanLimits{
	MaxJobs:         50,
	MaxResumes:      10,
	MaxApplications: 50,
	MaxAIRequests:   -1,
	MaxJobParses:    -1,
}

// EnterprisePlanLimits defines limits for the enterprise plan (-1 = unlimited).
var EnterprisePlanLimits = PlanLimits{
	MaxJobs:         -1,
	MaxResumes:      -1,
	MaxApplications: -1,
	MaxAIRequests:   -1,
	MaxJobParses:    -1,
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

// CheckoutConfigDTO holds Paddle checkout config for the frontend.
type CheckoutConfigDTO struct {
	ClientToken string            `json:"client_token"`
	Prices      map[string]string `json:"prices"`
	Environment string            `json:"environment"`
}

// PortalSessionDTO holds the Paddle customer portal URL.
type PortalSessionDTO struct {
	URL string `json:"url"`
}
