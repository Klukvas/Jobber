package main

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/internal/config"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// newBillingWebhookRateLimiter bounds the one public route with no auth
// middleware in front of it.
//
// The HMAC is what authenticates a delivery, but verifying it means reading and
// hashing the whole body first, so an unsigned flood still buys real work. The
// limit sits far above anything FastSpring produces and the limiter fails open
// on a Redis error, deliberately: a throttled delivery is a lifecycle event that
// has to come back through the retry schedule, and one that never comes back is
// a subscriber left on the wrong plan.
func newBillingWebhookRateLimiter(rdb *redis.Client, log *zap.Logger) gin.HandlerFunc {
	return httpPlatform.RateLimitMiddleware(rdb, httpPlatform.RateLimitConfig{
		MaxRequests: 600,
		Window:      1 * time.Minute,
		KeyPrefix:   "billing-webhook",
	}, log)
}

// startBillingReconciliation launches the sweep, or explains why it did not.
//
// The API credentials are checked here and not only inside the sweep: payments
// off with webhook ingestion on is a supported combination, and in it the
// webhook flag guarantees the secret while nothing requires the credentials. A
// sweep started there would fail on every tick forever instead of saying so once
// at boot.
func startBillingReconciliation(
	features config.FeaturesConfig, client *fastspring.Client,
	svc *subService.SubscriptionService, log *zap.Logger,
) {
	switch {
	case !features.BillingWebhookEnabled:
		log.Warn("Billing reconciliation disabled with webhook ingestion, missed provider events will not be recovered")
	case !client.IsConfigured():
		log.Warn("Billing reconciliation needs FastSpring API credentials, missed provider events will not be recovered")
	default:
		go reconcileBillingEvents(svc, log)
	}
}

// Recovery of provider events the webhook endpoint never received.
//
// FastSpring stops retrying a failed delivery after 7 days, so a deployment
// that was down, unreachable or holding a stale secret for longer than that
// keeps subscription rows nothing will ever correct. The provider's documented
// remedy is to pull the events it still has no acknowledgement for, which is
// what this runs — through the same pipeline, with the same guards, and
// idempotently against anything the push channel already applied. The design is
// in ADR-0002; how far back a sweep reaches is the service's own
// ReconcileWindowDays.
const (
	billingReconcileInterval = 6 * time.Hour
	billingReconcileTimeout  = 5 * time.Minute
	// The first sweep waits for the rest of the process to settle. It runs at
	// all because the case this exists for — a deployment that was down or
	// misconfigured — is the case where a restart has just happened.
	billingReconcileStartDelay = 2 * time.Minute
)

// reconcileBillingEvents sweeps for provider events the webhook endpoint never
// received, on a timer, for the life of the process.
func reconcileBillingEvents(svc *subService.SubscriptionService, log *zap.Logger) {
	time.Sleep(billingReconcileStartDelay)
	sweepBillingEvents(svc, log)

	ticker := time.NewTicker(billingReconcileInterval)
	defer ticker.Stop()
	for range ticker.C {
		sweepBillingEvents(svc, log)
	}
}

func sweepBillingEvents(svc *subService.SubscriptionService, log *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), billingReconcileTimeout)
	defer cancel()

	result, err := svc.ReconcileMissedEvents(ctx)
	if err != nil {
		// Billing is degraded, not the app: log it and let the next sweep try.
		log.Error("Billing reconciliation sweep failed", zap.Error(err))
		return
	}
	// A sweep that found nothing is the normal case and says nothing worth
	// reading, so only a sweep that actually did something is reported.
	if result.Total() == 0 && len(result.Unacknowledged) == 0 {
		return
	}

	log.Warn("Billing reconciliation found events the webhook endpoint never received",
		zap.Int("events", result.Total()),
		zap.Int("listings", result.Listings),
		// Everything the sweep is done with — applied, duplicate, superseded or
		// deliberately skipped — not only the ones that changed a row.
		zap.Int("acknowledged", len(result.Processed)),
		zap.Int("skipped", len(result.Skipped)),
		zap.Int("failed", len(result.Failed)),
		zap.Int("unacknowledged", len(result.Unacknowledged)),
	)
	logSweptOutcomes(result, log)
}

// logSweptOutcomes reports each recovered event the way the webhook endpoint
// would have.
//
// Without this a refund, a forged order tag or a subscriber paying twice is
// visible only when it arrives by webhook, and invisible when it arrives
// through the very sweep that exists to recover what the webhook lost. The
// verdict comes from the service so both channels cannot drift apart.
func logSweptOutcomes(result subService.ReconcileResult, log *zap.Logger) {
	for _, skipped := range result.Skipped {
		message, needsAttention := subService.SkipReport(skipped.Err)
		if !needsAttention {
			continue
		}
		log.Warn(message+" (recovered by reconciliation)",
			zap.String("event_id", skipped.EventID),
			zap.String("event_type", skipped.EventType),
			zap.Error(skipped.Err),
		)
	}
	for _, failed := range result.Failed {
		// Not acknowledged, so it stays on the provider's unprocessed list and
		// the next sweep tries again — until it ages out of the window.
		log.Error("Billing event failed during reconciliation, left for the next sweep",
			zap.String("event_id", failed.EventID),
			zap.String("event_type", failed.EventType),
			zap.Error(failed.Err),
		)
	}
	for _, unacknowledged := range result.Unacknowledged {
		log.Warn("Billing event applied but not acknowledged to the provider",
			zap.String("event_id", unacknowledged.EventID),
			zap.Error(unacknowledged.Err),
		)
	}
}
