import { apiClient } from "./api";
import type { MatchScoreResponse } from "@/shared/types/api";

export const matchScoreService = {
  async checkMatch(
    jobId: string,
    resumeId: string,
  ): Promise<MatchScoreResponse> {
    return apiClient.post<MatchScoreResponse>("match-score", {
      job_id: jobId,
      resume_id: resumeId,
    });
  },
};
