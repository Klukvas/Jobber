import { useAIConsentSignal } from "@/shared/hooks/useAIConsentSignal";
import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";
import type {
  GenerateCoverLetterRequest,
  GenerateCoverLetterResponse,
} from "@/shared/types/cover-letter";

export function useCoverLetterAI() {
  const getConsentSignal = useAIConsentSignal();
  const generate = useMutation({
    mutationFn: async (data: GenerateCoverLetterRequest) =>
      apiClient.post<GenerateCoverLetterResponse>(
        "cover-letters/ai/generate",
        data,
        { consentSignal: getConsentSignal() },
      ),
  });

  return { generate };
}
