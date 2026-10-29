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
	return ledger.HoldCreditPayment(ctx, event.ID, event.Type, mode, id)
}
