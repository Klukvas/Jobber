package model

// OverviewAnalytics contains high-level application statistics
type OverviewAnalytics struct {
	TotalApplications      int     `json:"total_applications"`
	ActiveApplications     int     `json:"active_applications"`
	ClosedApplications     int     `json:"closed_applications"`
	ResponseRate           float64 `json:"response_rate"`
	AvgDaysToFirstResponse float64 `json:"avg_days_to_first_response"`
}

// FunnelStage represents a single stage in the application funnel
type FunnelStage struct {
	StageName      string  `json:"stage_name"`
	StageOrder     int     `json:"stage_order"`
	Count          int     `json:"count"`
	ConversionRate float64 `json:"conversion_rate"`
	DropOffRate    float64 `json:"drop_off_rate"`
}

// FunnelAnalytics contains the complete funnel analysis
type FunnelAnalytics struct {
	Stages []FunnelStage `json:"stages"`
}

// StageTimeMetrics contains timing metrics for a single stage
type StageTimeMetrics struct {
	StageName     string  `json:"stage_name"`
	StageOrder    int     `json:"stage_order"`
	AvgDays       float64 `json:"avg_days"`
	MinDays       float64 `json:"min_days"`
	MaxDays       float64 `json:"max_days"`
	ApplicationsCount int `json:"applications_count"`
}

// StageTimeAnalytics contains timing metrics for all stages
type StageTimeAnalytics struct {
	Stages []StageTimeMetrics `json:"stages"`
}

// ResumeEffectiveness contains effectiveness metrics for a resume
type ResumeEffectiveness struct {
	ResumeID          string `json:"resume_id"`
	ResumeTitle       string `json:"resume_title"`
	ApplicationsCount int    `json:"applications_count"`
	ResponsesCount    int    `json:"responses_count"`
	InterviewsCount   int    `json:"interviews_count"`
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
