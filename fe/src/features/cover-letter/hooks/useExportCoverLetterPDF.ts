import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";

export function useExportCoverLetterPDF() {
  return useMutation({
    mutationFn: async (coverLetterId: string): Promise<Blob> => {
      return apiClient.postBlob(
        `cover-letters/${coverLetterId}/export-pdf`,
        undefined,
        60000,
      );
    },
  });
}
