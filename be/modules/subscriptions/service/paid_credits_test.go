package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/require"
)

type paidLedgerFixture struct {
	*MockSubscriptionRepository
	grants []model.PaidCreditGrant
	holds  []string
}

func (p *paidLedgerFixture) ApplyPaidCreditEvent(_ context.Context, _, _ string, _ *model.Subscription, g model.PaidCreditGrant) (model.WebhookApplyOutcome, error) {
	p.grants = append(p.grants, g)
	return model.WebhookApplied, nil
}
func (p *paidLedgerFixture) HoldCreditPayment(_ context.Context, _, _, mode, transaction string) error {
	p.holds = append(p.holds, mode+":"+transaction)
	return nil
}
func paidProductFixture() creem.Product {
	return creem.Product{ID: testProProductID, Mode: "test", Price: 2400, Currency: "USD", BillingType: "recurring", BillingPeriod: "every-month", Status: "active"}
}
func paidTransactionFixture() map[string]any {
	start, _ := time.Parse(time.RFC3339, periodStartText)
	end, _ := time.Parse(time.RFC3339, periodEndText)
	return map[string]any{"id": "tran_paid", "type": "invoice", "mode": "test", "status": "paid", "currency": "USD", "amount": 2400, "amount_paid": 2400, "discount_amount": 0, "tax_amount": 0, "refunded_amount": 0, "subscription": "sub_1", "customer": "cust_1", "period_start": start.UnixMilli(), "period_end": end.UnixMilli()}
}
func TestPaidCreditsRequireSettledUndiscountedUSDInvoiceAndMatchingPeriod(t *testing.T) {
	cases := []struct {
		name    string
		change  func(map[string]any)
		allowed bool
	}{
		{name: "valid monthly paid invoice", allowed: true},
		{name: "tax may increase payment", change: func(v map[string]any) { v["amount_paid"] = 2640; v["tax_amount"] = 240 }, allowed: true},
		{name: "zero payment", change: func(v map[string]any) { v["amount_paid"] = 0 }},
		{name: "missing settlement", change: func(v map[string]any) { delete(v, "amount_paid") }},
		{name: "discount not approved", change: func(v map[string]any) { v["discount_amount"] = 1 }},
		{name: "different currency", change: func(v map[string]any) { v["currency"] = "EUR" }},
		{name: "pending payment", change: func(v map[string]any) { v["status"] = "pending" }},
		{name: "refunded payment", change: func(v map[string]any) { v["refunded_amount"] = 1 }},
		{name: "different customer", change: func(v map[string]any) { v["customer"] = "cust_foreign" }},
		{name: "different subscription", change: func(v map[string]any) { v["subscription"] = "sub_foreign" }},
		{name: "different environment", change: func(v map[string]any) { v["mode"] = "prod" }},
		{name: "different transaction", change: func(v map[string]any) { v["id"] = "tran_other" }},
		{name: "missing period", change: func(v map[string]any) { delete(v, "period_start") }},
		{name: "different period", change: func(v map[string]any) { v["period_end"] = 1_800_000_000_000 }},
		{name: "one time purchase", change: func(v map[string]any) { v["type"] = "payment" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			transaction := paidTransactionFixture()
			if tc.change != nil {
				tc.change(transaction)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v1/transactions", r.URL.Path)
				require.Equal(t, "tran_paid", r.URL.Query().Get("transaction_id"))
				require.NoError(t, json.NewEncoder(w).Encode(transaction))
			}))
			defer server.Close()
			ledger := &paidLedgerFixture{MockSubscriptionRepository: &MockSubscriptionRepository{GetByUserIDFunc: func(_ context.Context, owner string) (*model.Subscription, error) {
				return &model.Subscription{UserID: owner, Plan: "free", Status: "free"}, nil
			}}}
			svc := NewSubscriptionService(ledger, creem.NewClient(creem.Config{APIKey: "fake-only", BaseURL: server.URL}), testBillingConfig())
			body, signature := signed(t, "paid", creem.EventSubscriptionPaid, subscriptionObject(map[string]any{"product": paidProductFixture(), "last_transaction_id": "tran_paid"}))
			_, err := svc.HandleWebhook(context.Background(), body, signature)
			if tc.allowed {
				require.NoError(t, err)
				require.Len(t, ledger.grants, 1)
				require.Equal(t, "tran_paid", ledger.grants[0].TransactionID)
			} else {
				require.Error(t, err)
				require.Empty(t, ledger.grants)
			}
		})
	}
}
func TestCheckoutValidatesCurrentCatalogEveryTimeAndPinsOneUnit(t *testing.T) {
	price := int64(2400)
	checkouts := 0
	products := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/products":
			products++
			p := paidProductFixture()
			p.Price = price
			require.NoError(t, json.NewEncoder(w).Encode(p))
		case "/v1/checkouts":
			checkouts++
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.EqualValues(t, 1, req["units"])
			require.NotContains(t, req, "discount_code")
			require.NotContains(t, req, "custom_price")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"id": "checkout", "checkout_url": "https://creem.test/fixture"}))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	svc := newTestServiceWithAPI(&MockSubscriptionRepository{}, server.URL)
	_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)
	require.NoError(t, err)
	price = 700
	_, err = svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)
	require.ErrorIs(t, err, model.ErrUnknownPlan)
	require.Equal(t, 2, products)
	require.Equal(t, 1, checkouts)
}
func TestRefundAndDisputeHoldTransactionBeforeAnyPaidGrant(t *testing.T) {
	for _, kind := range []string{creem.EventRefundCreated, creem.EventDisputeCreated} {
		t.Run(kind, func(t *testing.T) {
			ledger := &paidLedgerFixture{MockSubscriptionRepository: &MockSubscriptionRepository{}}
			svc := NewSubscriptionService(ledger, creem.NewClient(creem.Config{}), testBillingConfig())
			body, signature := signed(t, "hold", kind, map[string]any{"mode": "test", "transaction": map[string]string{"id": "tran_hold"}})
			_, err := svc.HandleWebhook(context.Background(), body, signature)
			require.NoError(t, err)
			require.Equal(t, []string{"test:tran_hold"}, ledger.holds)
			require.Empty(t, ledger.grants)
		})
	}
}

func TestPaidCreditsAcceptDocumentedSandboxAndRefundModes(t *testing.T) {
	for _, mode := range []string{"test", "sandbox"} {
		t.Run(mode, func(t *testing.T) {
			transaction := paidTransactionFixture()
			transaction["mode"] = mode
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.NoError(t, json.NewEncoder(w).Encode(transaction))
			}))
			defer server.Close()
			ledger := &paidLedgerFixture{MockSubscriptionRepository: &MockSubscriptionRepository{GetByUserIDFunc: func(_ context.Context, owner string) (*model.Subscription, error) {
				return &model.Subscription{UserID: owner, Plan: "free", Status: "free"}, nil
			}}}
			svc := NewSubscriptionService(ledger, creem.NewClient(creem.Config{APIKey: "fake-only", BaseURL: server.URL}), testBillingConfig())
			product := paidProductFixture()
			product.Mode = mode
			body, signature := signed(t, "paid", creem.EventSubscriptionPaid, subscriptionObject(map[string]any{"mode": mode, "product": product, "last_transaction_id": "tran_paid"}))
			_, err := svc.HandleWebhook(context.Background(), body, signature)
			require.NoError(t, err)
			require.Len(t, ledger.grants, 1)
			body, signature = signed(t, "refund", creem.EventRefundCreated, map[string]any{"mode": mode, "transaction_id": "tran_paid"})
			_, err = svc.HandleWebhook(context.Background(), body, signature)
			require.NoError(t, err)
			require.Equal(t, []string{"test:tran_paid"}, ledger.holds)
		})
	}
}
