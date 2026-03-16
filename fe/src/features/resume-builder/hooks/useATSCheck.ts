import { useMutation } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { apiClient } from "@/services/api";

interface ATSIssue {
  severity: "critical" | "warning" | "info";
  description: string;
}

interface ATSCheckResult {
  score: number;
  issues: ATSIssue[];
  suggestions: string[];
  keywords_found: string[];
}

export function useATSCheck() {
  const { i18n } = useTranslation();

  return useMutation({
    mutationFn: async (resumeId: string) =>
      apiClient.post<ATSCheckResult>(`resume-builder/${resumeId}/ats-check`, {
        locale: i18n.language,
      }),
    onError: (err) => console.error("[ATS] check failed:", err),
  });
}

export type { ATSIssue, ATSCheckResult };
