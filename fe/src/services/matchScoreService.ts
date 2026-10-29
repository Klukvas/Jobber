import { z } from "zod";
import { apiClient, ApiError, type RequestOptions } from "./api";
import type { MatchScoreResponse } from "@/shared/types/api";

const cachedMatchSchema = z.object({
  overall_score: z.number().int().min(0).max(100),
  categories: z.array(z.object({ name: z.string(), score: z.number().int().min(0).max(100), details: z.string() })).nullable(),
  missing_keywords: z.array(z.string()).nullable(),
  strengths: z.array(z.string()).nullable(),
  summary: z.string(),
  from_cache: z.literal(true),
});

export const matchScoreService = {
  async checkMatch(
    jobId: string,
    resumeId: string,
    options?: RequestOptions,
  ): Promise<MatchScoreResponse> {
    const origin = `${window.location.href}:${window.history.state?.key ?? ""}`;
    const signals = [options?.signal, options?.consentSignal].filter((signal): signal is AbortSignal => !!signal);
    const signal = signals.length ? AbortSignal.any(signals) : undefined;
    try {
      const cached = await apiClient.get<unknown>(
        `match-score?${new URLSearchParams({ job_id: jobId, resume_id: resumeId })}`,
        { signal },
      );
      signal?.throwIfAborted();
      const result = cachedMatchSchema.safeParse(cached);
      if (!result.success) throw new ApiError("", "INVALID_RESPONSE", 0);
      return { ...result.data, categories: result.data.categories ?? [], missing_keywords: result.data.missing_keywords ?? [], strengths: result.data.strengths ?? [] };
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 404 && error.code === "MATCH_SCORE_NOT_READY")) throw error;
    }
    signal?.throwIfAborted();
    // The lookup must not open a new confirmation after its originating route disappeared.
    if (origin !== `${window.location.href}:${window.history.state?.key ?? ""}`) throw new DOMException("AI action cancelled", "AbortError");
    return apiClient.post<MatchScoreResponse>("match-score", {
      job_id: jobId,
      resume_id: resumeId,
    }, ...(options ? [options] as const : []));
  },
};
