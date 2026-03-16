import { useTranslation } from "react-i18next";
import {
  ShieldCheck,
  Loader2,
  AlertCircle,
  AlertTriangle,
  Info,
} from "lucide-react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useATSCheck } from "../hooks/useATSCheck";
import type { ATSIssue } from "../hooks/useATSCheck";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { cn } from "@/shared/lib/utils";

const SEVERITY_STYLES: Record<ATSIssue["severity"], string> = {
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  warning:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  info: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

const SEVERITY_ICONS: Record<ATSIssue["severity"], React.ElementType> = {
  critical: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

function getScoreColor(score: number): string {
  if (score >= 80) return "text-green-600 dark:text-green-400";
  if (score >= 60) return "text-yellow-600 dark:text-yellow-400";
  return "text-red-600 dark:text-red-400";
}

function getScoreBorderColor(score: number): string {
  if (score >= 80) return "border-green-500";
  if (score >= 60) return "border-yellow-500";
  return "border-red-500";
}

export function ATSCheckerPanel() {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const atsCheck = useATSCheck();

  const handleCheck = () => {
    if (!resume) return;
    atsCheck.mutate(resume.id);
  };

  const result = atsCheck.data;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <ShieldCheck className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.ats.title")}
        </h2>
      </div>

      <Button
        onClick={handleCheck}
        disabled={atsCheck.isPending || !resume}
        variant="outline"
        className="w-full"
      >
        {atsCheck.isPending ? (
          <>
            <Loader2 className="h-4 w-4 animate-spin" />
            {t("resumeBuilder.ats.checking")}
          </>
        ) : (
          <>
            <ShieldCheck className="h-4 w-4" />
            {t("resumeBuilder.ats.check")}
          </>
        )}
      </Button>

      {atsCheck.isError && (
        <p className="text-sm text-destructive">{t("common.error")}</p>
      )}

      {result && (
        <div className="space-y-4">
          {/* Score Display */}
          <div className="flex justify-center">
            <div
              className={cn(
                "flex h-28 w-28 flex-col items-center justify-center rounded-full border-4",
                getScoreBorderColor(result.score),
              )}
            >
              <span
                className={cn(
                  "text-3xl font-bold",
                  getScoreColor(result.score),
                )}
              >
                {result.score}
              </span>
              <span className="text-xs text-muted-foreground">
                {t("resumeBuilder.ats.score")}
              </span>
            </div>
          </div>

          {/* Issues */}
          <Card>
            <CardHeader className="p-4 pb-2">
              <CardTitle className="text-sm font-medium">
                {t("resumeBuilder.ats.issues")}
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-0">
              {result.issues.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {t("resumeBuilder.ats.noIssues")}
                </p>
              ) : (
                <ul className="space-y-2">
                  {result.issues.map((issue, index) => {
                    const SeverityIcon = SEVERITY_ICONS[issue.severity];
                    return (
                      <li key={index} className="flex items-start gap-2">
                        <span
                          className={cn(
                            "mt-0.5 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                            SEVERITY_STYLES[issue.severity],
                          )}
                        >
                          <SeverityIcon className="h-3 w-3" />
                          {t(`resumeBuilder.ats.${issue.severity}`)}
                        </span>
                        <span className="text-sm">{issue.description}</span>
                      </li>
                    );
                  })}
                </ul>
              )}
            </CardContent>
          </Card>

          {/* Suggestions */}
          {result.suggestions.length > 0 && (
            <Card>
              <CardHeader className="p-4 pb-2">
                <CardTitle className="text-sm font-medium">
                  {t("resumeBuilder.ats.suggestions")}
                </CardTitle>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <ul className="space-y-1.5">
                  {result.suggestions.map((suggestion, index) => (
                    <li
                      key={index}
                      className="flex items-start gap-2 text-sm text-muted-foreground"
                    >
                      <span className="mt-1 block h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
                      {suggestion}
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
          )}

          {/* Keywords Found */}
          {result.keywords_found.length > 0 && (
            <Card>
              <CardHeader className="p-4 pb-2">
                <CardTitle className="text-sm font-medium">
                  {t("resumeBuilder.ats.keywords")}
                </CardTitle>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <div className="flex flex-wrap gap-1.5">
                  {result.keywords_found.map((keyword) => (
                    <span
                      key={keyword}
                      className="inline-flex items-center rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-medium text-primary"
                    >
                      {keyword}
                    </span>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}
