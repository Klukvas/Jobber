package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

// accountLookupPrefix namespaces the merchant-owned account key Jobber sets on
// every FastSpring account. It is sent as the session's
// `customer.externalAccountId` and read back as the account's `lookup.custom`,
// which is what makes the webhook fallback work. FastSpring requires an
// alphanumeric key of at least four characters, so the user UUID travels
// without its hyphens.
const accountLookupPrefix = "jobber-"

// CreateCheckoutSession creates a FastSpring checkout session for the
// authenticated user and returns the session id the Store Builder Library popup
// opens.
//
// Purchase-to-user linking is entirely server-side:
//  1. this call sends the user's own contact details, a merchant-owned external
//     account ID derived from their UUID, and the same UUID as an order tag —
//     carrying the proof that this server wrote it (see ordertag.go);
//  2. FastSpring answers with the account ID the session is bound to — but only
//     for a buyer who already has one. A first-time buyer's account is created
//     during checkout, so this field comes back empty;
//  3. whatever account ID did come back is stored on the user's subscription row
//     (plan untouched), so an abandoned checkout grants nothing;
//  4. the subscription.activated webhook then resolves the buyer through that
//     account ID, or — for the first purchase, where there was none to store —
//     through the order tag, whose proof is what separates the tag this call
//     wrote from one a storefront visitor wrote for themselves.
//
// No user identifier is ever accepted from the browser, and no checkout URL
// goes back to it: the browser receives an opaque session id and hands it to the
// popup, so there is nothing to navigate to and nothing to tamper with.
//
// https://developer.fastspring.com/reference/sessions-overview
// https://developer.fastspring.com/reference/createsession
func (s *SubscriptionService) CreateCheckoutSession(ctx context.Context, userID, plan string) (*model.CheckoutSessionDTO, error) {
	productPath, err := s.productPathForPlan(plan)
	if err != nil {
		return nil, err
	}
	if s.cfg.CheckoutPath == "" {
		return nil, errors.New("checkout path is not configured")
	}
	if err := s.ensureNotAlreadySubscribed(ctx, userID); err != nil {
		return nil, err
	}

	contact, err := s.repo.GetUserContact(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load buyer contact: %w", err)
	}

	tags, err := orderTags(s.cfg.WebhookSecret, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare checkout session: %w", err)
	}

	first, last := contact.FirstLast()
	session, err := s.billing.CreateSession(ctx, s.cfg.CheckoutPath, fastspring.SessionRequest{
		// Sent explicitly rather than inherited from the store, so a test
		// deployment can never open a live checkout.
		Live:   s.cfg.IsLive(),
		Locale: checkoutLocale(contact.Locale),
		Customer: &fastspring.SessionCustomer{
			ExternalAccountID: accountLookupKey(userID),
			BillToContact: &fastspring.SessionContact{
				FirstName: first,
				LastName:  last,
				Email:     contact.Email,
			},
		},
		OrderTags: tags,
		Cart: fastspring.SessionCart{
			LineItems: []fastspring.SessionLineItem{{ProductPath: productPath, Quantity: 1}},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	if !session.IsReady() {
		return nil, fmt.Errorf("checkout session %q is not ready for checkout: statuses [%s]",
			session.ID, session.CheckoutStatusString())
	}
	// Link before returning the session: the webhook can arrive before the popup
	// even closes, and it resolves the user through this account ID. A first-time
	// buyer has no account yet, so there is simply nothing to link — the order
	// tag covers that case on the webhook side.
	if session.Customer.AccountID != "" {
		if err := s.repo.LinkExternalAccount(ctx, userID, session.Customer.AccountID); err != nil {
			return nil, fmt.Errorf("failed to link billing account: %w", err)
		}
	}

	dto := &model.CheckoutSessionDTO{SessionID: session.ID}
	if expires, ok := session.ExpiresAt(); ok {
		dto.ExpiresAt = expires.Format(time.RFC3339)
	}
	return dto, nil
}

// ChangePlan switches an existing subscription to another plan, prorated.
//
// https://developer.fastspring.com/reference/update-a-subscription
func (s *SubscriptionService) ChangePlan(ctx context.Context, userID, newPlan string) error {
	productPath, err := s.productPathForPlan(newPlan)
	if err != nil {
		return err
	}

	subscriptionID, err := s.externalSubscriptionID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.billing.ChangeSubscriptionProduct(ctx, subscriptionID, productPath); err != nil {
		return fmt.Errorf("failed to change plan: %w", err)
	}
	// The resulting subscription.updated webhook is what actually moves the
	// local plan, keeping provider state as the single source of truth.
	return nil
}

// CancelSubscription schedules cancellation at the end of the current billing
// period. Access continues until FastSpring deactivates the subscription.
//
// https://developer.fastspring.com/reference/cancel-a-subscription
func (s *SubscriptionService) CancelSubscription(ctx context.Context, userID string) error {
	subscriptionID, err := s.externalSubscriptionID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.billing.CancelSubscription(ctx, subscriptionID, true); err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}
	return nil
}

// CreatePortalSession returns a pre-authenticated Account Management Portal URL
// for the user, landing on the Subscriptions tab.
//
// https://developer.fastspring.com/reference/retrieve-authenticated-account-management-url
func (s *SubscriptionService) CreatePortalSession(ctx context.Context, userID string) (string, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return "", err
	}
	if sub.ExternalAccountID == nil || *sub.ExternalAccountID == "" {
		return "", model.ErrNoActiveSubscription
	}

	portalURL, err := s.billing.AuthenticateAccount(ctx, *sub.ExternalAccountID)
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}
	portalURL, err = validatedPortalURL(portalURL)
	if err != nil {
		return "", err
	}
	parsedPortalURL, err := url.Parse(portalURL)
	if err != nil {
		return "", fmt.Errorf("validated portal session URL became unparseable: %w", err)
	}
	if parsedPortalURL.Fragment != "" {
		return portalURL, nil
	}
	// "#/subscriptions" is the portal's own client-side route, so the buyer lands
	// on their subscriptions rather than the account overview. Preserve any route
	// FastSpring already supplied instead of producing a second fragment marker.
	return portalURL + "#/subscriptions", nil
}

// validatedPortalURL accepts only an absolute HTTPS URL. The browser is sent
// straight here, so a scheme like javascript: or a relative value must never
// reach window.location.
//
// The host is deliberately *not* pinned. The Account Management Portal is the
// one flow that still navigates the current tab (checkout itself is a popup, so
// it navigates nothing). It is served from the storefront, and a FastSpring storefront
// can run on a merchant's own custom domain — no documented host contract fixes
// it to *.onfastspring.com. Inventing a whitelist here would break the moment a
// custom domain is configured in the dashboard, so the actual host is a
// verification item on the go-live checklist (ADR-0002) rather than a guess
// hardcoded in the service.
func validatedPortalURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("portal session carried no URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("portal session URL is unparseable: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		return "", fmt.Errorf("portal session URL is not absolute HTTPS: %q", trimmed)
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
// provider subscription that is still alive at FastSpring.
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

// accountLookupKey derives the merchant-owned FastSpring account key from a
// user UUID.
func accountLookupKey(userID string) string {
	return accountLookupPrefix + strings.ReplaceAll(userID, "-", "")
}

// userIDFromLookupKey reverses accountLookupKey, restoring the UUID hyphens.
// It returns false for any key Jobber did not create.
func userIDFromLookupKey(key string) (string, bool) {
	const hexLen = 32
	hex, ok := strings.CutPrefix(key, accountLookupPrefix)
	if !ok || len(hex) != hexLen {
		return "", false
	}
	for _, c := range hex {
		if !isHexDigit(c) {
			return "", false
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex[0:8], hex[8:12], hex[12:16], hex[16:20], hex[20:32]), true
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// FastSpring checkout languages, as two-letter codes. The session endpoint
// documents `locale` as a standard 2-letter language code and its language map
// lists the codes a storefront can render; a regional tag like "en_US" is not
// one of them.
//
// https://developer.fastspring.com/reference/createsession
const (
	localeEnglish = "en"
	localeRussian = "ru"
)

// checkoutLocale maps a Jobber UI locale onto the FastSpring checkout language.
// An unknown locale falls back to English rather than forwarding a code the
// provider does not render.
//
// Ukrainian has no FastSpring checkout language of its own — "uk" is absent from
// the documented language set — so a Ukrainian buyer gets Russian, FastSpring's
// own default language for Ukraine. That default is a store-level setting and is
// the one thing here that must be confirmed in the dashboard's test mode before
// go-live (see ADR-0002); if the store's Ukraine default is English instead,
// this arm becomes localeEnglish.
func checkoutLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "ru":
		return localeRussian
	case "ua", "uk":
		return localeRussian
	default:
		return localeEnglish
	}
}
