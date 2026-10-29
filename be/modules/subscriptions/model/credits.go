package model

import (
	"errors"
	"time"
)

var (
	ErrCreditUnavailable   = errors.New("ai credits unavailable")
	ErrAIBudgetReached     = errors.New("ai provider budget reached")
	ErrAIOperationConflict = errors.New("ai operation already dispatched or input changed")
)

const (
	FreeCredits   = 30
	ProCredits    = 500
	ProPriceCents = 2400
)

// CreditBalance describes the allowance currently available, never a calendar-month reset.
type CreditBalance struct {
	SpendLimitNanodollars     int64      `json:"spend_limit_nanodollars"`
	SpendAllocatedNanodollars int64      `json:"spend_allocated_nanodollars"`
	Granted                   int        `json:"granted"`
	Used                      int        `json:"used"`
	Remaining                 int        `json:"remaining"`
	PeriodEnd                 *time.Time `json:"period_end,omitempty"`
	Kind                      string     `json:"kind"`
}

type CreditOperation struct {
	UserID      string
	Key         string
	Kind        string
	Fingerprint string
	Credits     int
}

// AIAttempt reserves a monetary liability before a provider request. Nanodollars
// keep all arithmetic integral (one dollar is 1,000,000,000 nanodollars).
type AIAttempt struct {
	Operation      CreditOperation
	Stage          string
	Model          string
	MaxNanodollars int64
}

type PaidCreditGrant struct {
	UserID         string
	Provider       string
	Mode           string
	TransactionID  string
	SubscriptionID string
	CustomerID     string
	ProductID      string
	PeriodStart    time.Time
	PeriodEnd      time.Time
}
