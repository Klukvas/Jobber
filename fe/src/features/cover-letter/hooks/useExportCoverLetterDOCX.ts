import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/services/api";

export function useExportCoverLetterDOCX() {
  return useMutation({
    mutationFn: async (coverLetterId: string): Promise<Blob> => {
      return apiClient.postBlob(
        `cover-letters/${coverLetterId}/export-docx`,
        undefined,
        60000,
      );
    },
  });
}
