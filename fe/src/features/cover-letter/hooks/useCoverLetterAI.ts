import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";
import type {
  GenerateCoverLetterRequest,
  GenerateCoverLetterResponse,
} from "@/shared/types/cover-letter";

export function useCoverLetterAI() {
  const generate = useMutation({
    mutationFn: async (data: GenerateCoverLetterRequest) =>
      apiClient.post<GenerateCoverLetterResponse>(
        "cover-letters/ai/generate",
        data,
      ),
  });

  return { generate };
}
