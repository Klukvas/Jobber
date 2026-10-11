import { useTranslation } from "react-i18next";
import { Check, Loader2, AlertCircle } from "lucide-react";
import { useCoverLetterStore } from "@/stores/coverLetterStore";

interface CoverLetterSaveIndicatorProps {
  /** Retries the failed save; the failure text becomes the button. */
  readonly onRetry?: () => void;
}

export function CoverLetterSaveIndicator({
  onRetry,
}: CoverLetterSaveIndicatorProps = {}) {
  const { t } = useTranslation();
  const saveStatus = useCoverLetterStore((s) => s.saveStatus);

  switch (saveStatus) {
    case "saving":
      return (
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {t("coverLetter.saving")}
        </span>
      );
    case "saved":
      return (
        <span className="flex items-center gap-1.5 text-sm text-green-600">
          <Check className="h-3.5 w-3.5" />
          {t("coverLetter.saved")}
        </span>
      );
    case "error":
      // A failed save stays on screen until it succeeds, and offers the retry
      // right there — the editor has no other way back from a lost request.
      return (
        <span
          role="alert"
          className="flex items-center gap-1.5 text-sm text-destructive"
        >
          {onRetry ? (
            <button
              type="button"
              onClick={onRetry}
              className="flex items-center gap-1.5 underline-offset-2 hover:underline"
            >
              <AlertCircle className="h-3.5 w-3.5" />
              {t("coverLetter.saveFailed")}
            </button>
          ) : (
            <>
              <AlertCircle className="h-3.5 w-3.5" />
              {t("coverLetter.saveFailed")}
            </>
          )}
        </span>
      );
    default:
      return null;
  }
}
