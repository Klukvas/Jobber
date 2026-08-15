package model

// OverviewAnalytics contains high-level pipeline statistics.
//
// In the single-axis stage model there is no status, so the buckets are:
//   - ActiveApplications  = cards currently sitting in a column
//   - ClosedApplications  = archived cards
//   - RejectedApplications = always 0 (rejection was a status; it no longer
//     exists). The field is retained for DTO/JSON stability.
type OverviewAnalytics struct {
	TotalApplications      int     `json:"total_applications"`
	ActiveApplications     int     `json:"active_applications"`
	ClosedApplications     int     `json:"closed_applications"`
	RejectedApplications   int     `json:"rejected_applications"`
	ResponseRate           float64 `json:"response_rate"`
	AvgDaysToFirstResponse float64 `json:"avg_days_to_first_response"`
}

// FunnelStage represents one bucket of the stage funnel — one of the user's own
// pipeline columns (stage_templates), in "order". StageName is the column name,
// Count the number of non-archived cards that reached it.
//
// SubStages is retained for JSON/DTO stability but is no longer populated: the
// single-axis model has no phase drill-down.
type FunnelStage struct {
	StageName      string        `json:"stage_name"`
	StageOrder     int           `json:"stage_order"`
	Count          int           `json:"count"`
	ConversionRate float64       `json:"conversion_rate"`
	DropOffRate    float64       `json:"drop_off_rate"`
	SubStages      []FunnelStage `json:"sub_stages,omitempty"`
}

// RejectedStageCount is retained for JSON/DTO stability. Rejection was a status
// and no longer exists in the single-axis model, so this is never populated.
type RejectedStageCount struct {
	StageName  string `json:"stage_name"`
	StageOrder int    `json:"stage_order"`
	Count      int    `json:"count"`
}

// RejectedSummary is retained for JSON/DTO stability. Rejection was a status and
// no longer exists in the single-axis model, so it is never populated.
type RejectedSummary struct {
	Total   int                  `json:"total"`
	ByStage []RejectedStageCount `json:"by_stage"`
}

// FunnelAnalytics contains the complete funnel analysis.
type FunnelAnalytics struct {
	Stages []FunnelStage `json:"stages"`
	// Rejected is retained for JSON/DTO stability but is always nil in the
	// single-axis model.
	Rejected *RejectedSummary `json:"rejected,omitempty"`
}

// StageTimeMetrics contains timing metrics for a single stage
type StageTimeMetrics struct {
	StageName         string  `json:"stage_name"`
	StageOrder        int     `json:"stage_order"`
	AvgDays           float64 `json:"avg_days"`
	MinDays           float64 `json:"min_days"`
	MaxDays           float64 `json:"max_days"`
	ApplicationsCount int     `json:"applications_count"`
}

// StageTimeAnalytics contains timing metrics for all stages
type StageTimeAnalytics struct {
	Stages []StageTimeMetrics `json:"stages"`
}

// ResumeEffectiveness contains effectiveness metrics for a resume
type ResumeEffectiveness struct {
	ResumeID          string  `json:"resume_id"`
	ResumeTitle       string  `json:"resume_title"`
	ApplicationsCount int     `json:"applications_count"`
	ResponsesCount    int     `json:"responses_count"`
	InterviewsCount   int     `json:"interviews_count"`
	ResponseRate      float64 `json:"response_rate"`
}

// ResumeAnalytics contains effectiveness metrics for all resumes
type ResumeAnalytics struct {
	Resumes []ResumeEffectiveness `json:"resumes"`
}

// SourceMetrics contains metrics for a single job source
type SourceMetrics struct {
	SourceName        string  `json:"source_name"`
	ApplicationsCount int     `json:"applications_count"`
	ResponsesCount    int     `json:"responses_count"`
	ConversionRate    float64 `json:"conversion_rate"`
}

// SourceAnalytics contains metrics for all job sources
type SourceAnalytics struct {
	Sources []SourceMetrics `json:"sources"`
}
