package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

// metadataUserIDKey names the checkout metadata entry that carries the local user
// ID. Creem echoes metadata back on the checkout's webhook events, which is how a
// first purchase is linked to the user who started it.
const metadataUserIDKey = "jobber_user_id"

// checkoutSuccessPath is where Creem sends the buyer after paying. The frontend
// reads the `subscription` parameter (Settings and AppLayout), so the two must
// agree; Creem appends its own query parameters, which are ignored.
const checkoutSuccessPath = "/settings?subscription=success"

// CheckoutSuccessURL builds the success_url for a public base URL.
func CheckoutSuccessURL(publicBaseURL string) string {
	return strings.TrimRight(publicBaseURL, "/") + checkoutSuccessPath
}

// CreateCheckoutSession creates a Creem hosted checkout for the authenticated user
// and returns the URL the browser is sent to.
//
// Purchase-to-user linking is entirely server-side:
//  1. this call sends the user's own email and name from their record, and their
//     UUID as checkout metadata. The metadata is written with the API key, and the
//     hosted checkout page offers the buyer no way to edit it, so unlike an
//     identifier chosen in the browser it needs no further proof;
//  2. the subscription.* and checkout.completed webhooks echo that metadata back,
//     and the webhook handler resolves the owner from it;
//  3. nothing is granted here — the plan only changes when a webhook lands, so an
//     abandoned checkout costs nothing.
//
// No user identifier is ever accepted from the browser.
//
// https://docs.creem.io/api-reference/endpoint/create-checkout
func (s *SubscriptionService) CreateCheckoutSession(ctx context.Context, userID, plan string) (*model.CheckoutSessionDTO, error) {
	productID, err := s.productIDForPlan(plan)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNotAlreadySubscribed(ctx, userID); err != nil {
		return nil, err
	}

	contact, err := s.repo.GetUserContact(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load buyer contact: %w", err)
	}
	// The webhook that grants the plan writes into this row, so it has to exist.
	if err := s.repo.EnsureFree(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to prepare subscription row: %w", err)
	}

	checkout, err := s.billing.CreateCheckout(ctx, creem.CheckoutRequest{
		ProductID:  productID,
		SuccessURL: s.cfg.SuccessURL,
		Customer:   &creem.CheckoutCustomer{Email: contact.Email, Name: contact.Name},
		Metadata:   map[string]string{metadataUserIDKey: userID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout: %w", err)
	}

	checkoutURL, err := validatedHTTPSURL(checkout.CheckoutURL)
	if err != nil {
		// The URL itself stays out of the message: it is a live session link.
		return nil, fmt.Errorf("checkout %q has an unusable URL: %w", checkout.ID, err)
	}
	return &model.CheckoutSessionDTO{CheckoutURL: checkoutURL}, nil
}

// ChangePlan switches an existing subscription to another plan, prorated.
//
// https://docs.creem.io/api-reference/endpoint/upgrade-subscription
func (s *SubscriptionService) ChangePlan(ctx context.Context, userID, newPlan string) error {
	productID, err := s.productIDForPlan(newPlan)
	if err != nil {
		return err
	}

	subscriptionID, err := s.externalSubscriptionID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.billing.UpgradeSubscription(ctx, subscriptionID, productID); err != nil {
		return fmt.Errorf("failed to change plan: %w", err)
	}
	// The resulting subscription.update webhook is what actually moves the local
	// plan, keeping provider state as the single source of truth.
	return nil
}

// CancelSubscription schedules cancellation at the end of the current billing
// period. Access continues until Creem ends the subscription.
//
// https://docs.creem.io/api-reference/endpoint/cancel-subscription
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID string) error {
	subscriptionID, err := s.externalSubscriptionID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.billing.CancelSubscriptionAtPeriodEnd(ctx, subscriptionID); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}
	return nil
}

// CreatePortalSession returns a login link to the Creem customer portal.
//
// https://docs.creem.io/features/customer-portal
func (s *SubscriptionService) CreatePortalSession(ctx context.Context, userID string) (string, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if sub.ExternalAccountID == nil || *sub.ExternalAccountID == "" {
		return "", model.ErrNoActiveSubscription
	}

	link, err := s.billing.CustomerPortalLink(ctx, *sub.ExternalAccountID)
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}
	portalURL, err := validatedHTTPSURL(link)
	if err != nil {
		return "", fmt.Errorf("portal session has an unusable URL: %w", err)
	}
	return portalURL, nil
}

// validatedHTTPSURL accepts only an absolute HTTPS URL with no credentials in it.
// The browser is sent straight to it, so a scheme like javascript: or a relative
// value must never reach window.location, and `https://creem.io@evil.example`
// must not pass for a creem.io address.
//
// The host is deliberately not pinned: the URL comes from Creem's API over TLS
// with our own key, and no documented contract fixes the host family.
func validatedHTTPSURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("no URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.New("unparseable URL")
	}
	if !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		return "", errors.New("not an absolute HTTPS URL")
	}
	if parsed.User != nil {
		return "", errors.New("URL carries credentials")
	}
	return trimmed, nil
}

// ensureNotAlreadySubscribed refuses a *second* checkout for a user who already
// holds a provider subscription.
//
// The row carries a single external_subscription_id, so a second purchase would
// overwrite the first: the original would keep billing at the provider with
// nothing in Jobber pointing at it, and neither the user nor the app could
// cancel it. A subscriber changing plan must go through ChangePlan, which
// updates the subscription they already pay for.
//
// The exceptions are deliberate:
//   - no row at all, or a row with no provider subscription ID — the user has
//     never completed a checkout, so this is their first;
//   - status cancelled — the provider has ended that subscription, so buying
//     again is the only way back to a paid plan.
//
// past_due, paused and a scheduled cancellation (active with cancel_at) all
// still bill, so they are refused: the subscription is alive at the provider.
//
// A database failure is never swallowed into "go ahead and buy" — an
// unreadable subscription state must fail the checkout, not bypass the guard.
func (s *SubscriptionService) ensureNotAlreadySubscribed(ctx context.Context, userID string) error {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrSubscriptionNotFound) {
			return nil
		}
		return fmt.Errorf("failed to load subscription before checkout: %w", err)
	}
	if !holdsLiveProviderSubscription(sub) {
		return nil
	}
	return fmt.Errorf("%w: plan %q is %q", model.ErrAlreadySubscribed, sub.Plan, sub.Status)
}

// holdsLiveProviderSubscription reports whether a row already points at a
// provider subscription that is still alive at Creem.
//
// `active`, `past_due`, `paused` and a scheduled cancellation (`active` with
// `cancel_at`) all still bill, so all four count as live. Only two rows have
// nothing to protect: one that never completed a checkout, and one the provider
// has already ended.
func holdsLiveProviderSubscription(sub *model.Subscription) bool {
	if sub.ExternalSubscriptionID == nil || *sub.ExternalSubscriptionID == "" {
		return false
	}
	return sub.Status != StatusCancelled
}

// externalSubscriptionID returns the provider subscription ID for a user, or
// ErrNoActiveSubscription when there is nothing to manage.
func (s *SubscriptionService) externalSubscriptionID(ctx context.Context, userID string) (string, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if sub.ExternalSubscriptionID == nil || *sub.ExternalSubscriptionID == "" {
		return "", model.ErrNoActiveSubscription
	}
	return *sub.ExternalSubscriptionID, nil
}
