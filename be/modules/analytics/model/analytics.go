package model

// OverviewAnalytics contains high-level application statistics.
// RejectedApplications is a subset of ClosedApplications (closed = rejected +
// offer + archived) — broken out so rejections are visible on their own.
type OverviewAnalytics struct {
	TotalApplications      int     `json:"total_applications"`
	ActiveApplications     int     `json:"active_applications"`
	ClosedApplications     int     `json:"closed_applications"`
	RejectedApplications   int     `json:"rejected_applications"`
	ResponseRate           float64 `json:"response_rate"`
	AvgDaysToFirstResponse float64 `json:"avg_days_to_first_response"`
}

// FunnelStage represents one bucket of the application funnel.
//
// Top-level funnel buckets are the fixed pipeline PHASES — StageName carries the
// phase key ("applied", "in_progress", "offer") for the frontend to localize.
// SubStages is the optional drill-down: for the "in_progress" phase it lists the
// user's own in-progress stage templates (StageName = the template name).
type FunnelStage struct {
	StageName      string        `json:"stage_name"`
	StageOrder     int           `json:"stage_order"`
	Count          int           `json:"count"`
	ConversionRate float64       `json:"conversion_rate"`
	DropOffRate    float64       `json:"drop_off_rate"`
	SubStages      []FunnelStage `json:"sub_stages,omitempty"`
}

// RejectedStageCount is how many rejected applications ended at a given
// stage (the furthest stage they reached before the rejection).
type RejectedStageCount struct {
	StageName  string `json:"stage_name"`
	StageOrder int    `json:"stage_order"`
	Count      int    `json:"count"`
}

// RejectedSummary is the terminal "rejections" block next to the funnel.
// Rejection is a status, not a stage — it can happen at any pipeline depth,
// so it is reported alongside the stages, not as one of them.
type RejectedSummary struct {
	Total   int                  `json:"total"`
	ByStage []RejectedStageCount `json:"by_stage"`
}

// FunnelAnalytics contains the complete funnel analysis
type FunnelAnalytics struct {
	Stages []FunnelStage `json:"stages"`
	// Rejected is nil when the user has no rejected applications.
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
