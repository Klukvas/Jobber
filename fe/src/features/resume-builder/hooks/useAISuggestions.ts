import { useAIConsentSignal } from "@/shared/hooks/useAIConsentSignal";
import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";

interface BulletSuggestions {
  bullets: string[];
}

interface SuggestBulletsRequest {
  job_title: string;
  company: string;
  current_description: string;
}

interface SuggestSummaryRequest {
  resume_id: string;
}

interface ImproveTextRequest {
  text: string;
  instruction: string;
}

interface ImproveTextResponse {
  improved: string;
}

interface SuggestSummaryResponse {
  summary: string;
}

export function useAISuggestions() {
  const getConsentSignal = useAIConsentSignal();
  const suggestBullets = useMutation({
    mutationFn: async (data: SuggestBulletsRequest) =>
      apiClient.post<BulletSuggestions>(
        "resume-builder/ai/suggest-bullets",
        data,
        { consentSignal: getConsentSignal() },
      ),
  });

  const suggestSummary = useMutation({
    mutationFn: async (data: SuggestSummaryRequest) =>
      apiClient.post<SuggestSummaryResponse>(
        "resume-builder/ai/suggest-summary",
        data,
        { consentSignal: getConsentSignal() },
      ),
  });

  const improveText = useMutation({
    mutationFn: async (data: ImproveTextRequest) =>
      apiClient.post<ImproveTextResponse>(
        "resume-builder/ai/improve-text",
        data,
        { consentSignal: getConsentSignal() },
      ),
  });

  return { suggestBullets, suggestSummary, improveText };
}
