package creem

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Product struct {
	ID            string `json:"id"`
	Mode          string `json:"mode"`
	Price         int64  `json:"price"`
	Currency      string `json:"currency"`
	BillingType   string `json:"billing_type"`
	BillingPeriod string `json:"billing_period"`
	Status        string `json:"status"`
	// TrialPeriod is in days; the API names it trial_period_days and sends null
	// for a product without a trial.
	TrialPeriod int64 `json:"trial_period_days"`
}

func (p Product) IsProMonthly(id string, live bool) bool {
	return p.ID == id && ModeMatches(p.Mode, live) && p.Price == 2400 && strings.EqualFold(p.Currency, "USD") && p.BillingType == "recurring" && p.BillingPeriod == "every-month" && p.Status == "active" && p.TrialPeriod == 0
}
func (c *Client) GetProduct(ctx context.Context, id string) (*Product, error) {
	var p Product
	if err := c.do(ctx, http.MethodGet, "/v1/products?product_id="+url.QueryEscape(id), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

type Transaction struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Mode           string          `json:"mode"`
	Status         string          `json:"status"`
	Currency       string          `json:"currency"`
	Amount         int64           `json:"amount"`
	AmountPaid     *int64          `json:"amount_paid"`
	DiscountAmount int64           `json:"discount_amount"`
	TaxAmount      int64           `json:"tax_amount"`
	RefundedAmount int64           `json:"refunded_amount"`
	Subscription   json.RawMessage `json:"subscription"`
	Customer       json.RawMessage `json:"customer"`
	PeriodStart    json.RawMessage `json:"period_start"`
	PeriodEnd      json.RawMessage `json:"period_end"`
}

func (c *Client) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	var t Transaction
	if err := c.do(ctx, http.MethodGet, "/v1/transactions?transaction_id="+url.QueryEscape(id), nil, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
func (t Transaction) ValidPaidPro(id, subscription, customer string, live bool, start, end time.Time) bool {
	return t.ID == id && ModeMatches(t.Mode, live) && t.Type == "invoice" && transactionTime(t.PeriodStart).Equal(start.Truncate(time.Second)) && transactionTime(t.PeriodEnd).Equal(end.Truncate(time.Second)) && t.Status == "paid" && strings.EqualFold(t.Currency, "USD") && t.Amount >= 2400 && t.AmountPaid != nil && *t.AmountPaid >= 2400 && t.DiscountAmount == 0 && t.RefundedAmount == 0 && decodeID(t.Subscription) == subscription && decodeID(t.Customer) == customer
}
func ParseProduct(raw json.RawMessage) (*Product, error) {
	var p Product
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.ID == "" {
		return nil, errors.New("missing product")
	}
	return &p, nil
}

func transactionTime(raw json.RawMessage) time.Time {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed.UTC().Truncate(time.Second)
		}
	}
	n, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n).UTC().Truncate(time.Second)
	}
	return time.Unix(n, 0).UTC()
}

// ModeMatches admits only the documented environments, never absent or guessed modes.
func ModeMatches(mode string, live bool) bool {
	if live {
		return mode == "prod"
	}
	return mode == "test" || mode == "sandbox"
}
