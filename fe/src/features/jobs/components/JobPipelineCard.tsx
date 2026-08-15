import { useTranslation } from "react-i18next";
import { format, formatDistanceToNow } from "date-fns";
import { Calendar, CheckCircle, Loader2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Label } from "@/shared/ui/Label";
import { StageSelector } from "@/features/jobs/components/StageSelector";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { resumeSelectValue } from "@/features/jobs/hooks/useJobDetailMutations";
import type { JobDTO, ResumeDTO, StageTemplateDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

interface JobPipelineCardProps {
  readonly job: JobDTO;
  readonly templates: readonly StageTemplateDTO[];
  readonly uploadedResumes: readonly ResumeDTO[];
  readonly builderResumes: readonly ResumeBuilderDTO[];
  readonly onMoveToStage: (stageTemplateId: string) => void;
  readonly isMoving: boolean;
  readonly onCompleteStage: (stageId: string) => void;
  readonly isCompletingStage: boolean;
  readonly onChangeResume: (value: string) => void;
  readonly isChangingResume: boolean;
}

export function JobPipelineCard({
  job,
  templates,
  uploadedResumes,
  builderResumes,
  onMoveToStage,
  isMoving,
  onCompleteStage,
  isCompletingStage,
  onChangeResume,
  isChangingResume,
}: JobPipelineCardProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{t("jobs.pipelineTitle")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Primary control: the pipeline column selector (the card's state) */}
        <StageSelector
          templates={templates}
          currentStageTemplateId={job.current_stage_template_id}
          onSelect={onMoveToStage}
          isMoving={isMoving}
        />

        {job.current_stage_id && job.current_stage_name && (
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">
              {t("jobs.currentStage")}
            </p>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">
                {job.current_stage_name}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  if (job.current_stage_id) {
                    onCompleteStage(job.current_stage_id);
                  }
                }}
                disabled={isCompletingStage}
              >
                {isCompletingStage ? (
                  <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
                ) : (
                  <CheckCircle className="h-4 w-4 mr-1.5" />
                )}
                {t("jobs.complete")}
              </Button>
            </div>
          </div>
        )}

        {job.applied_at && (
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">
              {t("jobs.applied")}{" "}
              {format(new Date(job.applied_at), "PP", {
                locale: dateLocale,
              })}{" "}
              (
              {formatDistanceToNow(new Date(job.applied_at), {
                addSuffix: true,
                locale: dateLocale,
              })}
              )
            </span>
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="attached-resume">{t("jobs.resumeLabel")}</Label>
          <select
            id="attached-resume"
            value={resumeSelectValue(job)}
            onChange={(e) => onChangeResume(e.target.value)}
            disabled={isChangingResume}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          >
            <option value="">{t("jobs.selectResume")}</option>
            {uploadedResumes.length > 0 && (
              <optgroup label={t("jobs.uploadedResumes")}>
                {uploadedResumes.map((resume) => (
                  <option
                    key={`uploaded:${resume.id}`}
                    value={`uploaded:${resume.id}`}
                  >
                    {resume.title}
                  </option>
                ))}
              </optgroup>
            )}
            {builderResumes.length > 0 && (
              <optgroup label={t("jobs.builderResumes")}>
                {builderResumes.map((rb) => (
                  <option key={`builder:${rb.id}`} value={`builder:${rb.id}`}>
                    {rb.title}
                  </option>
                ))}
              </optgroup>
            )}
          </select>
        </div>
      </CardContent>
    </Card>
  );
}
