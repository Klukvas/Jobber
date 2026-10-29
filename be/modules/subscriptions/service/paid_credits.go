package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

func (s *SubscriptionService) validateProCatalog(ctx context.Context) error {
	if s.billing == nil {
		return creem.ErrNotConfigured
	}
	p, err := s.billing.GetProduct(ctx, s.cfg.ProProductID)
	if err != nil {
		return fmt.Errorf("verify pro product: %w", err)
	}
	if !p.IsProMonthly(s.cfg.ProProductID, s.cfg.IsLive()) {
		return model.ErrUnknownPlan
	}
	return nil
}

// applyPaidEvent writes a subscription.paid together with the period's credit
// grant when the payment qualifies for one.
//
// A payment that does not qualify — a prorated plan-change charge, a discount, a
// product other than Pro — still carries the subscription's state, and Creem sends
// no subscription.active, so this event is what grants a first purchase. Its state
// is written without a grant and errPaidWithoutCredits says why. Only a failure to
// check the payment (Creem unreachable, a malformed object) is returned as err and
// retried.
func (s *SubscriptionService) applyPaidEvent(
	ctx context.Context, event creem.Event, incoming *creem.Subscription, updated *model.Subscription,
) (outcome model.WebhookApplyOutcome, withheldCredits, err error) {
	grant, grantErr := s.paidCreditGrant(ctx, event, incoming, updated.UserID)
	if isIneligiblePayment(grantErr) {
		outcome, err = s.repo.ApplySubscriptionEvent(ctx, event.ID, event.Type, updated)
		return outcome, fmt.Errorf("%w: %w", errPaidWithoutCredits, grantErr), err
	}
	if grantErr != nil {
		return "", nil, grantErr
	}
	ledger, ok := s.repo.(interface {
		ApplyPaidCreditEvent(context.Context, string, string, *model.Subscription, model.PaidCreditGrant) (model.WebhookApplyOutcome, error)
	})
	if !ok {
		return "", nil, model.ErrCreditUnavailable
	}
	outcome, err = ledger.ApplyPaidCreditEvent(ctx, event.ID, event.Type, updated, grant)
	return outcome, nil, err
}

// isIneligiblePayment reports whether paidCreditGrant judged the payment itself,
// as opposed to failing to look at it: no retry can make such a payment qualify.
func isIneligiblePayment(err error) bool {
	return errors.Is(err, model.ErrCreditUnavailable) || errors.Is(err, model.ErrUnknownPlan)
}

func (s *SubscriptionService) paidCreditGrant(ctx context.Context, event creem.Event, incoming *creem.Subscription, owner string) (model.PaidCreditGrant, error) {
	var payment struct {
		Mode        string          `json:"mode"`
		Transaction string          `json:"last_transaction_id"`
		Product     json.RawMessage `json:"product"`
	}
	if err := json.Unmarshal(event.Object, &payment); err != nil {
		return model.PaidCreditGrant{}, err
	}
	mode := "test"
	if s.cfg.IsLive() {
		mode = "live"
	}
	if !creem.ModeMatches(payment.Mode, s.cfg.IsLive()) || incoming.CustomerID == "" || incoming.ProductID != s.cfg.ProProductID || payment.Transaction == "" || incoming.CurrentPeriodStart == nil || incoming.CurrentPeriodEnd == nil || incoming.Status != "active" {
		return model.PaidCreditGrant{}, model.ErrCreditUnavailable
	}
	duration := incoming.CurrentPeriodEnd.Sub(*incoming.CurrentPeriodStart)
	if duration < 27*24*time.Hour || duration > 32*24*time.Hour || incoming.CurrentPeriodStart.After(s.now().Add(5*time.Minute)) {
		return model.PaidCreditGrant{}, model.ErrCreditUnavailable
	}
	product, err := creem.ParseProduct(payment.Product)
	if err != nil {
		if err := s.validateProCatalog(ctx); err != nil {
			return model.PaidCreditGrant{}, err
		}
	} else if !product.IsProMonthly(s.cfg.ProProductID, s.cfg.IsLive()) {
		return model.PaidCreditGrant{}, model.ErrUnknownPlan
	}
	// A catalog price is not payment evidence. Fetch the settled transaction even
	// when a signed event embeds the product; zero/trial/discount purchases grant nothing.
	if s.billing == nil {
		return model.PaidCreditGrant{}, creem.ErrNotConfigured
	}
	transaction, err := s.billing.GetTransaction(ctx, payment.Transaction)
	if err != nil {
		return model.PaidCreditGrant{}, err
	}
	if !transaction.ValidPaidPro(payment.Transaction, incoming.ID, incoming.CustomerID, s.cfg.IsLive(), *incoming.CurrentPeriodStart, *incoming.CurrentPeriodEnd) {
		return model.PaidCreditGrant{}, model.ErrCreditUnavailable
	}
	return model.PaidCreditGrant{UserID: owner, Provider: "creem", Mode: mode, TransactionID: payment.Transaction, SubscriptionID: incoming.ID, CustomerID: incoming.CustomerID, ProductID: incoming.ProductID, PeriodStart: incoming.CurrentPeriodStart.UTC(), PeriodEnd: incoming.CurrentPeriodEnd.UTC()}, nil
}

func (s *SubscriptionService) holdCreditPayment(ctx context.Context, event creem.Event) error {
	var object struct {
		Mode          string          `json:"mode"`
		Transaction   json.RawMessage `json:"transaction"`
		TransactionID string          `json:"transaction_id"`
	}
	if err := json.Unmarshal(event.Object, &object); err != nil {
		return err
	}
	mode := "test"
	if s.cfg.IsLive() {
		mode = "live"
	}
	if object.Mode == "" || (object.Mode != "test" && object.Mode != "sandbox" && object.Mode != "prod") {
		return model.ErrCreditUnavailable
	}
	if !creem.ModeMatches(object.Mode, s.cfg.IsLive()) {
		return ErrEnvironmentMismatch
	}
	id := object.TransactionID
	if id == "" {
		var text string
		if json.Unmarshal(object.Transaction, &text) == nil {
			id = text
		} else {
			var expanded struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(object.Transaction, &expanded); err != nil {
				return err
			}
			id = expanded.ID
		}
	}
	if id == "" {
		return errors.New("refund or dispute missing transaction identity")
	}
	ledger, ok := s.repo.(interface {
		HoldCreditPayment(context.Context, string, string, string, string) error
	})
	if !ok {
		return model.ErrCreditUnavailable
	}
	if err := ledger.HoldCreditPayment(ctx, event.ID, event.Type, mode, id); err != nil {
		return err
	}
	return fmt.Errorf("%w: transaction %q", ErrRefundNeedsReview, id)
}
