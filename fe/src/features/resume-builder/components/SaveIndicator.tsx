import { useTranslation } from "react-i18next";
import { Check, Loader2, AlertCircle } from "lucide-react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";

export function SaveIndicator() {
  const { t } = useTranslation();
  const saveStatus = useResumeBuilderStore((s) => s.saveStatus);

  switch (saveStatus) {
    case "saving":
      return (
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {t("resumeBuilder.saving")}
        </span>
      );
    case "saved":
      return (
        <span className="flex items-center gap-1.5 text-sm text-green-600">
          <Check className="h-3.5 w-3.5" />
          {t("resumeBuilder.saved")}
        </span>
      );
    case "error":
      return (
        <span className="flex items-center gap-1.5 text-sm text-destructive">
          <AlertCircle className="h-3.5 w-3.5" />
          {t("resumeBuilder.saveFailed")}
        </span>
      );
    default:
      return null;
  }
}
