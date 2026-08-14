import { useTranslation } from "react-i18next";
import { Loader2, Sparkles } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Label } from "@/shared/ui/Label";
import { MatchScoreCard } from "@/features/jobs/components/MatchScoreCard";
import type { MatchScoreResponse, ResumeDTO } from "@/shared/types/api";

interface JobMatchScoreCardProps {
  readonly uploadedResumes: readonly ResumeDTO[];
  readonly effectiveMatchResumeId: string;
  readonly onSelectResume: (value: string) => void;
  readonly onCheckMatch: () => void;
  readonly isChecking: boolean;
  readonly matchScore: MatchScoreResponse | null;
  readonly matchScoreError: string | null;
}

export function JobMatchScoreCard({
  uploadedResumes,
  effectiveMatchResumeId,
  onSelectResume,
  onCheckMatch,
  isChecking,
  matchScore,
  matchScoreError,
}: JobMatchScoreCardProps) {
  const { t } = useTranslation();

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">
            {t("jobs.matchScore.checkMatch")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="flex-1 space-y-2">
              <Label htmlFor="resume-select">{t("jobs.selectResume")}</Label>
              <select
                id="resume-select"
                value={effectiveMatchResumeId}
                onChange={(e) => onSelectResume(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="">{t("jobs.selectResume")}</option>
                {uploadedResumes.map((resume) => (
                  <option key={resume.id} value={resume.id}>
                    {resume.title}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={onCheckMatch}
              disabled={!effectiveMatchResumeId || isChecking}
            >
              {isChecking ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Sparkles className="h-4 w-4 mr-2" />
              )}
              {isChecking
                ? t("jobs.matchScore.checking")
                : t("jobs.matchScore.checkMatch")}
            </Button>
          </div>

          {matchScoreError && (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
              {matchScoreError}
            </div>
          )}
        </CardContent>
      </Card>

      {matchScore && <MatchScoreCard data={matchScore} />}
    </>
  );
}
