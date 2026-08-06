import { apiClient } from "./api";
import type {
  OverviewAnalytics,
  FunnelStage,
} from "./analyticsService";

// Snapshot payloads reuse the analytics shapes — the backend freezes the
// same aggregates the Analytics page renders.
export interface StatsSnapshot {
  schema_version: number;
  generated_at: string;
  overview: OverviewAnalytics;
  funnel: FunnelStage[];
}

export interface ShareDTO {
  id: string;
  token: string;
  snapshot: StatsSnapshot;
  created_at: string;
}

export interface PublicShareDTO {
  snapshot: StatsSnapshot;
  created_at: string;
}

export function buildShareUrl(token: string): string {
  return `${window.location.origin}/s/${token}`;
}

export const sharingService = {
  async create(): Promise<ShareDTO> {
    return apiClient.post<ShareDTO>("shares");
  },

  async list(): Promise<ShareDTO[]> {
    return apiClient.get<ShareDTO[]>("shares");
  },

  async remove(id: string): Promise<void> {
    await apiClient.delete<{ message: string }>(`shares/${id}`);
  },

  async getPublic(token: string): Promise<PublicShareDTO> {
    return apiClient.get<PublicShareDTO>(
      `public/shares/${encodeURIComponent(token)}`,
    );
  },
};
