import { cn } from "@/shared/lib/utils";
import { AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "./Button";

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  className?: string;
}

export function ErrorState({
  title,
  message,
  onRetry,
  className,
}: ErrorStateProps) {
  const { t } = useTranslation();

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-lg border border-destructive/20 bg-destructive/5 p-8 text-center",
        className,
      )}
      role="alert"
    >
      <AlertCircle className="mb-4 h-12 w-12 text-destructive" />
      <h3 className="mb-2 text-lg font-semibold">
        {title ?? t("errors.somethingWentWrong")}
      </h3>
      {/* An empty message is what the API client produces when the failure
          carried nothing quotable — show the generic line instead of a gap. */}
      <p className="mb-4 text-sm text-muted-foreground">
        {message?.trim() ? message : t("errors.unexpectedError")}
      </p>
      {onRetry && (
        <Button onClick={onRetry} variant="outline">
          {t("common.tryAgain")}
        </Button>
      )}
    </div>
  );
}
