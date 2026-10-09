package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/ports"
)

// Provider names the billing provider exposed to the frontend.
const Provider = "creem"

// Billing environments. Creem marks every object it sends as live or test; the
// service refuses to act on events from the other mode.
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

// scheduledCancelGrace is how long past its end date a scheduled cancellation may
// still count as paid.
//
// The cancellation is a webhook, and Creem stops retrying one after 24 hours. A
// service that was down longer would otherwise keep a subscriber on a paid plan
// forever, because nothing would ever correct the row. Two days outlasts a late
// or retried delivery, so a subscriber whose period really was renewed in the
// meantime is not downgraded by it.
const scheduledCancelGrace = 48 * time.Hour

// BillingConfig holds the Creem settings the service needs. Product IDs are
// dashboard-owned identifiers, not code.
type BillingConfig struct {
	WebhookSecret string
	Environment   string
	// SuccessURL is where Creem sends the buyer after a completed checkout.
	SuccessURL          string
	ProProductID        string
	EnterpriseProductID string
}

// IsLive reports whether the service is wired to live Creem.
func (c BillingConfig) IsLive() bool { return c.Environment == EnvironmentLive }

// SubscriptionService handles subscription business logic.
type SubscriptionService struct {
	repo    ports.SubscriptionRepository
	billing *creem.Client
	cfg     BillingConfig
	now     func() time.Time
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	repo ports.SubscriptionRepository,
	billing *creem.Client,
	cfg BillingConfig,
) *SubscriptionService {
	return &SubscriptionService{repo: repo, billing: billing, cfg: cfg, now: time.Now}
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

	dto := sub.ToDTO(usage, s.effectivePlanOf(sub))
	if s.scheduledCancelLapsed(sub) {
		// The cancellation date passed and the webhook that should have ended the
		// subscription never arrived. Show what enforcement already applies, as the
		// cancellation itself would have: free, cancelled, no period.
		dto.Plan, dto.Status = PlanFree, StatusCancelled
		dto.CurrentPeriodEnd, dto.CancelAt = nil, nil
	}
	return dto, nil
}

// GetCheckoutConfig tells the frontend which provider and plans are live. It
// carries no credentials and no product IDs — checkouts are created server-side.
func (s *SubscriptionService) GetCheckoutConfig() *model.CheckoutConfigDTO {
	return &model.CheckoutConfigDTO{
		Provider:    Provider,
		Environment: s.cfg.Environment,
		Plans:       s.purchasablePlans(),
	}
}

// purchasablePlans lists the plans with a configured product, cheapest first. A
// plan without a product cannot be bought and is never advertised.
func (s *SubscriptionService) purchasablePlans() []string {
	plans := make([]string, 0, 2)
	if s.cfg.ProProductID != "" {
		plans = append(plans, PlanPro)
	}
	if s.cfg.EnterpriseProductID != "" {
		plans = append(plans, PlanEnterprise)
	}
	return plans
}

// productIDForPlan maps a plan name to its Creem product ID.
func (s *SubscriptionService) productIDForPlan(plan string) (string, error) {
	switch plan {
	case PlanPro:
		if s.cfg.ProProductID == "" {
			return "", fmt.Errorf("%w: pro product is not configured", model.ErrUnknownPlan)
		}
		return s.cfg.ProProductID, nil
	case PlanEnterprise:
		if s.cfg.EnterpriseProductID == "" {
			return "", fmt.Errorf("%w: enterprise product is not configured", model.ErrUnknownPlan)
		}
		return s.cfg.EnterpriseProductID, nil
	default:
		return "", fmt.Errorf("%w: %q", model.ErrUnknownPlan, plan)
	}
}

// planForProductID is the reverse mapping, used when a webhook tells us what was
// actually bought. An unrecognised ID never yields a paid plan.
func (s *SubscriptionService) planForProductID(productID string) (string, error) {
	switch {
	case productID == "":
		return "", errors.New("subscription event carried no product ID")
	case s.cfg.EnterpriseProductID != "" && productID == s.cfg.EnterpriseProductID:
		return PlanEnterprise, nil
	case s.cfg.ProProductID != "" && productID == s.cfg.ProProductID:
		return PlanPro, nil
	default:
		return "", fmt.Errorf("unrecognised product ID %q", productID)
	}
}

// effectivePlan resolves the plan whose limits actually apply right now for a
// user. A missing subscription row is treated as the free plan.
func (s *SubscriptionService) effectivePlan(ctx context.Context, userID string) (string, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrSubscriptionNotFound) {
			return PlanFree, nil
		}
		return "", fmt.Errorf("failed to get subscription: %w", err)
	}
	return s.effectivePlanOf(sub), nil
}

// effectivePlanOf is the plan whose limits apply to a stored subscription.
// Paid quotas only apply while the subscription is actually paying: a
// paused/cancelled paid plan falls back to free (past_due keeps a grace window),
// and so does a scheduled cancellation whose end date is long past.
//
// Every place that enforces or reports limits goes through here, so what the
// user is shown can never disagree with what the backend enforces.
func (s *SubscriptionService) effectivePlanOf(sub *model.Subscription) string {
	if sub.Plan != PlanFree && sub.Status != StatusActive && sub.Status != StatusPastDue {
		return PlanFree
	}
	if s.scheduledCancelLapsed(sub) {
		return PlanFree
	}
	return sub.Plan
}

// scheduledCancelLapsed reports whether a cancellation date passed long enough
// ago that the row can only be stale: the subscription.canceled that should have
// followed it never arrived.
func (s *SubscriptionService) scheduledCancelLapsed(sub *model.Subscription) bool {
	if sub.Status != StatusActive || sub.CancelAt == nil {
		return false
	}
	return s.now().After(sub.CancelAt.Add(scheduledCancelGrace))
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

// ResourceLimit reports how many of a countable resource the user's effective
// plan allows. -1 means unlimited; an unrecognised resource is also unlimited,
// matching CheckLimit's own default.
//
// Exists so a caller can enforce the ceiling where it can actually be enforced.
// CheckLimit answers "is there room right now?", which stops being true the
// moment it returns: two requests can both be told yes and both write. A caller
// that does its counting and its writing in one transaction needs the number,
// not the verdict.
func (s *SubscriptionService) ResourceLimit(ctx context.Context, userID, resource string) (int, error) {
	plan, err := s.effectivePlan(ctx, userID)
	if err != nil {
		return 0, err
	}
	limits := model.GetLimitsForPlan(plan)

	switch resource {
	case "jobs":
		return limits.MaxJobs, nil
	case "resumes":
		return limits.MaxResumes, nil
	case "ai":
		return limits.MaxAIRequests, nil
	case "job_parses":
		return limits.MaxJobParses, nil
	case "resume_builders":
		return limits.MaxResumeBuilders, nil
	case "cover_letters":
		return limits.MaxCoverLetters, nil
	default:
		return -1, nil
	}
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
