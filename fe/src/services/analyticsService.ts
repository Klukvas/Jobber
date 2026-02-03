import { apiClient } from './api';

// Analytics DTOs
export interface OverviewAnalytics {
  total_applications: number;
  active_applications: number;
  closed_applications: number;
  response_rate: number;
  avg_days_to_first_response: number;
}

export interface FunnelStage {
  stage_name: string;
  stage_order: number;
  count: number;
  conversion_rate: number;
  drop_off_rate: number;
}

export interface FunnelAnalytics {
  stages: FunnelStage[];
}

export interface StageTimeMetrics {
  stage_name: string;
  stage_order: number;
  avg_days: number;
  min_days: number;
  max_days: number;
  applications_count: number;
}

export interface StageTimeAnalytics {
  stages: StageTimeMetrics[];
}

export interface ResumeEffectiveness {
  resume_id: string;
  resume_title: string;
  applications_count: number;
  responses_count: number;
  interviews_count: number;
  response_rate: number;
}

export interface ResumeAnalytics {
  resumes: ResumeEffectiveness[];
}

export interface SourceMetrics {
  source_name: string;
  applications_count: number;
  responses_count: number;
  conversion_rate: number;
}

export interface SourceAnalytics {
  sources: SourceMetrics[];
}

export const analyticsService = {
  async getOverview(): Promise<OverviewAnalytics> {
    return apiClient.get<OverviewAnalytics>('analytics/overview');
  },

  async getFunnel(): Promise<FunnelAnalytics> {
    return apiClient.get<FunnelAnalytics>('analytics/funnel');
  },

  async getStageTime(): Promise<StageTimeAnalytics> {
    return apiClient.get<StageTimeAnalytics>('analytics/stages');
  },

  async getResumeEffectiveness(): Promise<ResumeAnalytics> {
    return apiClient.get<ResumeAnalytics>('analytics/resumes');
  },

  async getSourceAnalytics(): Promise<SourceAnalytics> {
    return apiClient.get<SourceAnalytics>('analytics/sources');
  },
};
