import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";

export function useExportDOCX() {
  return useMutation({
    mutationFn: async (resumeId: string): Promise<Blob> => {
      return apiClient.postBlob(
        `resume-builder/${resumeId}/export-docx`,
        undefined,
        60000,
      );
    },
  });
}
