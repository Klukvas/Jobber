package model

import "time"

// ManualAccess is a separately audited allowance, never provider billing state.
type ManualAccess struct {
	ID          string    `json:"id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}
