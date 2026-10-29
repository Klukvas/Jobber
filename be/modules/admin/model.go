package admin

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("user or manual grant not found")
var ErrConflict = errors.New("manual grant conflicts with existing grant")

type User struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	Locale         string     `json:"locale"`
	IsTest         bool       `json:"is_test"`
	EmailVerified  bool       `json:"email_verified"`
	CreatedAt      time.Time  `json:"created_at"`
	ProviderPlan   string     `json:"provider_plan"`
	ProviderStatus string     `json:"provider_status"`
	ManualUntil    *time.Time `json:"manual_until"`
}
type Grant struct {
	Active       bool       `json:"active"`
	ID           string     `json:"id"`
	Start        time.Time  `json:"period_start"`
	End          time.Time  `json:"period_end"`
	RevokedAt    *time.Time `json:"revoked_at"`
	ActorID      string     `json:"actor_id"`
	Reason       string     `json:"reason"`
	RevokedBy    *string    `json:"revoked_by"`
	RevokeReason *string    `json:"revoke_reason"`
	Credits      int        `json:"credits"`
	Used         int        `json:"used"`
}
type Detail struct {
	User                   User       `json:"user"`
	ProviderSubscriptionID *string    `json:"provider_subscription_id"`
	ProviderAccountID      *string    `json:"provider_account_id"`
	ProviderPeriodEnd      *time.Time `json:"provider_period_end"`
	ProviderCancelAt       *time.Time `json:"provider_cancel_at"`
	EffectivePlan          string     `json:"effective_plan"`
	ManualGrants           []Grant    `json:"manual_grants"`
}
type UserList struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
type ProfileUpdate struct {
	Name   string `json:"name"`
	Locale string `json:"locale"`
}
type GrantRequest struct {
	ID     string `json:"id"`
	Days   int    `json:"days"`
	Reason string `json:"reason"`
}
type RevokeRequest struct {
	Reason string `json:"reason"`
}
