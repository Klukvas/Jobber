import { useTranslation } from "react-i18next";
import { Check, Loader2, AlertCircle, CircleDot } from "lucide-react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";

interface SaveIndicatorProps {
  /** Retries the failed save; the failure text becomes the button. */
  readonly onRetry?: () => void;
}

export function SaveIndicator({ onRetry }: SaveIndicatorProps = {}) {
  const { t } = useTranslation();
  const saveStatus = useResumeBuilderStore((s) => s.saveStatus);
  const isDirty = useResumeBuilderStore((s) => s.isDirty);

  // Dirty wins over the last completed save. The indicator used to change only
  // when a save fired, so text typed during the debounce still read "Saved" —
  // and a reload in that window lost it with no warning.
  if (isDirty && saveStatus !== "saving" && saveStatus !== "error") {
    return (
      <span
        role="status"
        className="flex items-center gap-1.5 text-sm text-muted-foreground"
      >
        <CircleDot className="h-3.5 w-3.5" />
        {t("resumeBuilder.unsavedChanges")}
      </span>
    );
  }

  switch (saveStatus) {
    case "saving":
      return (
        <span
          role="status"
          className="flex items-center gap-1.5 text-sm text-muted-foreground"
        >
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {t("resumeBuilder.saving")}
        </span>
      );
    case "saved":
      return (
        <span
          role="status"
          className="flex items-center gap-1.5 text-sm text-green-600"
        >
          <Check className="h-3.5 w-3.5" />
          {t("resumeBuilder.saved")}
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
              {t("resumeBuilder.saveFailed")}
            </button>
          ) : (
            <>
              <AlertCircle className="h-3.5 w-3.5" />
              {t("resumeBuilder.saveFailed")}
            </>
          )}
        </span>
      );
    default:
      return null;
  }
}
