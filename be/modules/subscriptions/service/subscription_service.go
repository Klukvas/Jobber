package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/ports"
)

// Provider names the billing provider exposed to the frontend.
const Provider = "fastspring"

// Billing environments. FastSpring marks every order and webhook event as live
// or test; the service refuses to act on events from the other mode.
const (
	EnvironmentLive = "live"
	EnvironmentTest = "test"
)

// Plan names.
const (
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Internal subscription statuses.
const (
	StatusFree      = "free"
	StatusActive    = "active"
	StatusPastDue   = "past_due"
	StatusCancelled = "cancelled"
	StatusPaused    = "paused"
)

// BillingConfig holds the FastSpring settings the service needs. Product paths
// and the checkout path are configurable because they are dashboard-owned
// identifiers, not code.
type BillingConfig struct {
	WebhookSecret string
	// CheckoutPath identifies the dashboard *popup* checkout the Sessions API
	// creates sessions against, e.g. "<storefront-id>/popup-<checkout-id>". The
	// popup storefront the browser loads is derived from it, so a single value
	// governs both halves of the flow.
	CheckoutPath          string
	Environment           string
	ProProductPath        string
	EnterpriseProductPath string
}

// IsLive reports whether the service is wired to the live FastSpring store.
func (c BillingConfig) IsLive() bool { return c.Environment == EnvironmentLive }

// SubscriptionService handles subscription business logic.
type SubscriptionService struct {
	repo    ports.SubscriptionRepository
	billing *fastspring.Client
	cfg     BillingConfig
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	repo ports.SubscriptionRepository,
	billing *fastspring.Client,
	cfg BillingConfig,
) *SubscriptionService {
	return &SubscriptionService{repo: repo, billing: billing, cfg: cfg}
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

// GetCheckoutConfig tells the frontend which provider and plans are live, and
// which popup storefront to load the provider's script against. It carries no
// credentials and no catalog product paths — checkout sessions are created
// server-side.
//
// The storefront is derived from the configured checkout path and the store
// mode rather than configured separately, so "which checkout the API creates a
// session against" and "which storefront the browser opens" can never name two
// different checkouts. A path that yields no storefront leaves the field empty,
// which the frontend reads as "checkout is not openable" — startup validation
// makes that unreachable while payments are on.
func (s *SubscriptionService) GetCheckoutConfig() *model.CheckoutConfigDTO {
	storefront, err := fastspring.PopupStorefront(s.cfg.CheckoutPath, s.cfg.IsLive())
	if err != nil {
		storefront = ""
	}
	return &model.CheckoutConfigDTO{
		Provider:    Provider,
		Environment: s.cfg.Environment,
		Storefront:  storefront,
		Plans:       s.purchasablePlans(),
	}
}

// purchasablePlans lists the plans with a configured product path, cheapest
// first. A plan without a path cannot be bought and is never advertised.
func (s *SubscriptionService) purchasablePlans() []string {
	plans := make([]string, 0, 2)
	if s.cfg.ProProductPath != "" {
		plans = append(plans, PlanPro)
	}
	if s.cfg.EnterpriseProductPath != "" {
		plans = append(plans, PlanEnterprise)
	}
	return plans
}

// productPathForPlan maps a plan name to its FastSpring catalog product path.
func (s *SubscriptionService) productPathForPlan(plan string) (string, error) {
	switch plan {
	case PlanPro:
		if s.cfg.ProProductPath == "" {
			return "", fmt.Errorf("%w: pro product path is not configured", model.ErrUnknownPlan)
		}
		return s.cfg.ProProductPath, nil
	case PlanEnterprise:
		if s.cfg.EnterpriseProductPath == "" {
			return "", fmt.Errorf("%w: enterprise product path is not configured", model.ErrUnknownPlan)
		}
		return s.cfg.EnterpriseProductPath, nil
	default:
		return "", fmt.Errorf("%w: %q", model.ErrUnknownPlan, plan)
	}
}

// planForProductPath is the reverse mapping, used when a webhook tells us what
// was actually bought. An unrecognised path never yields a paid plan.
func (s *SubscriptionService) planForProductPath(path string) (string, error) {
	switch {
	case path == "":
		return "", errors.New("subscription event carried no product path")
	case s.cfg.EnterpriseProductPath != "" && path == s.cfg.EnterpriseProductPath:
		return PlanEnterprise, nil
	case s.cfg.ProProductPath != "" && path == s.cfg.ProProductPath:
		return PlanPro, nil
	default:
		return "", fmt.Errorf("unrecognised product path %q", path)
	}
}

// effectivePlan resolves the plan whose limits actually apply right now for a
// user. Paid quotas only apply while the subscription is actually paying: a
// paused/cancelled paid plan falls back to free (past_due keeps a grace
// window, matching Subscription.IsActive semantics). A missing subscription
// row is treated as the free plan.
func (s *SubscriptionService) effectivePlan(ctx context.Context, userID string) (string, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrSubscriptionNotFound) {
			return PlanFree, nil
		}
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub.Plan != PlanFree && sub.Status != StatusActive && sub.Status != StatusPastDue {
		return PlanFree, nil
	}
	return sub.Plan, nil
}

// RequirePaidPlan returns ErrPaidFeature unless the user is on an effective
// paid plan. Gates paid-only features (e.g. AI extraction of an Autofill
// Profile from an Uploaded Resume — see docs/adr/0001-autofill-profile-economics.md).
func (s *SubscriptionService) RequirePaidPlan(ctx context.Context, userID string) error {
	plan, err := s.effectivePlan(ctx, userID)
	if err != nil {
		return err
	}
	if plan == PlanFree {
		return model.ErrPaidFeature
	}
	return nil
}

// CheckLimit checks if a user can create another resource of the given type.
func (s *SubscriptionService) CheckLimit(ctx context.Context, userID, resource string) error {
	plan, err := s.effectivePlan(ctx, userID)
	if err != nil {
		return err
	}
	limits := model.GetLimitsForPlan(plan)

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

// RecordResumeAutofillUsage records a resume autofill extraction usage event.
func (s *SubscriptionService) RecordResumeAutofillUsage(ctx context.Context, userID string) error {
	return s.repo.RecordResumeAutofillUsage(ctx, userID)
}

// EnsureFreeSubscription creates a free subscription for a user if one doesn't
// exist. An existing row is left untouched so it can never downgrade a payer.
func (s *SubscriptionService) EnsureFreeSubscription(ctx context.Context, userID string) error {
	return s.repo.EnsureFree(ctx, userID)
}

// getUsage returns current resource usage for a user in a single query.
func (s *SubscriptionService) getUsage(ctx context.Context, userID string) (model.Usage, error) {
	jobs, resumes, aiReqs, jobParses, resumeBuilders, coverLetters, err := s.repo.GetAllCounts(ctx, userID)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		Jobs:           jobs,
		Resumes:        resumes,
		AIRequests:     aiReqs,
		JobParses:      jobParses,
		ResumeBuilders: resumeBuilders,
		CoverLetters:   coverLetters,
	}, nil
}
