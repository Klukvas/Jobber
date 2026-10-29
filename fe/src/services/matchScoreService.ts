import { apiClient, type RequestOptions } from "./api";
import type { MatchScoreResponse } from "@/shared/types/api";

export const matchScoreService = {
  async checkMatch(
    jobId: string,
    resumeId: string,
    options?: RequestOptions,
  ): Promise<MatchScoreResponse> {
    return apiClient.post<MatchScoreResponse>("match-score", {
      job_id: jobId,
      resume_id: resumeId,
    }, ...(options ? [options] as const : []));
  },
};
