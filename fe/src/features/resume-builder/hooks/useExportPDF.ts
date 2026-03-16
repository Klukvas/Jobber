import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";

export function useExportPDF() {
  return useMutation({
    mutationFn: async (resumeId: string): Promise<Blob> => {
      return apiClient.postBlob(
        `resume-builder/${resumeId}/export-pdf`,
        undefined,
        60000,
      );
    },
  });
}
