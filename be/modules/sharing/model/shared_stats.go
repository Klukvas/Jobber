package model

import (
	"errors"
	"time"
)

// SnapshotSchemaVersion is stored inside every snapshot. Snapshots outlive
// code changes, so the version lets future readers render old payloads.
const SnapshotSchemaVersion = 1

// MaxActiveShares caps stored shares per user.
const MaxActiveShares = 20

// OverviewSnapshot mirrors analytics overview figures at share time.
// ResponseRate is a percentage (0-100).
type OverviewSnapshot struct {
	TotalApplications      int     `json:"total_applications"`
	ActiveApplications     int     `json:"active_applications"`
	ClosedApplications     int     `json:"closed_applications"`
	ResponseRate           float64 `json:"response_rate"`
	AvgDaysToFirstResponse float64 `json:"avg_days_to_first_response"`
}

// FunnelStageSnapshot mirrors one funnel stage at share time.
// Rates are percentages (0-100).
type FunnelStageSnapshot struct {
	StageName      string  `json:"stage_name"`
	StageOrder     int     `json:"stage_order"`
	Count          int     `json:"count"`
	ConversionRate float64 `json:"conversion_rate"`
	DropOffRate    float64 `json:"drop_off_rate"`
}

// StatsSnapshot is the frozen payload of a share. It is persisted as JSONB
// and served verbatim on the unauthenticated public endpoint, so it must
// only ever contain aggregates — no company names, resume titles or sources.
type StatsSnapshot struct {
	SchemaVersion int                   `json:"schema_version"`
	GeneratedAt   time.Time             `json:"generated_at"`
	Overview      OverviewSnapshot      `json:"overview"`
	Funnel        []FunnelStageSnapshot `json:"funnel"`
}

type SharedStats struct {
	ID        string
	UserID    string
	Token     string
	Snapshot  StatsSnapshot
	CreatedAt time.Time
}

type ShareDTO struct {
	ID        string        `json:"id"`
	Token     string        `json:"token"`
	Snapshot  StatsSnapshot `json:"snapshot"`
	CreatedAt time.Time     `json:"created_at"`
}

// PublicShareDTO omits the internal ID and owner — it is served without
// authentication.
type PublicShareDTO struct {
	Snapshot  StatsSnapshot `json:"snapshot"`
	CreatedAt time.Time     `json:"created_at"`
}

func (s *SharedStats) ToDTO() *ShareDTO {
	return &ShareDTO{
		ID:        s.ID,
		Token:     s.Token,
		Snapshot:  s.Snapshot,
		CreatedAt: s.CreatedAt,
	}
}

func (s *SharedStats) ToPublicDTO() *PublicShareDTO {
	return &PublicShareDTO{
		Snapshot:  s.Snapshot,
		CreatedAt: s.CreatedAt,
	}
}

var (
	ErrShareNotFound     = errors.New("share not found")
	ErrShareLimitReached = errors.New("share limit reached")
)

type ErrorCode string

const (
	CodeShareNotFound     ErrorCode = "SHARE_NOT_FOUND"
	CodeShareLimitReached ErrorCode = "SHARE_LIMIT_REACHED"
	CodeInternalError     ErrorCode = "INTERNAL_ERROR"
)
