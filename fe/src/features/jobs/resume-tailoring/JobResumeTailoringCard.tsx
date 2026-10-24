import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Sparkles } from "lucide-react";
import { useSearchParams } from "react-router-dom";
import type { ResumeDTO, JobDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { ResumeTailoringModal } from "./ResumeTailoringModal";

type Props = {
  job: JobDTO;
  uploadedResumes?: readonly ResumeDTO[];
  builderResumes: readonly ResumeBuilderDTO[];
  isLoading: boolean;
  isError: boolean;
  isJobDirty: boolean;
  isJobSaving: boolean;
};

export function JobResumeTailoringCard({ job, builderResumes, uploadedResumes = [], isLoading, isError, isJobDirty, isJobSaving }: Props) {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const [open, setOpen] = useState(() => !!params.get("tailoringSource"));
  const hasDescription = !!job.description?.trim();
  const canOpen = hasDescription && !isJobDirty && !isJobSaving;
  return (
    <Card>
      <CardHeader><CardTitle className="text-lg">{t("jobs.tailoring.title")}</CardTitle></CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{t("jobs.tailoring.intro")}</p>
        {!hasDescription && <p className="text-sm text-muted-foreground">{t("jobs.tailoring.missingDescription")}</p>}
        {(isJobDirty || isJobSaving) && <p className="text-sm text-muted-foreground">{t("jobs.tailoring.saveJobFirst")}</p>}
        <Button variant="outline" disabled={!canOpen} onClick={() => setOpen(true)}><Sparkles className="h-4 w-4" />{t("jobs.tailoring.title")}</Button>
        {open && canOpen && <ResumeTailoringModal key={`${job.id}:${job.updated_at}`} job={job} builderResumes={builderResumes} uploadedResumes={uploadedResumes} isLoading={isLoading} isError={isError} onClose={() => { setOpen(false); if (params.has("tailoringSource")) { const next = new URLSearchParams(params); next.delete("tailoringSource"); setParams(next, { replace: true }); } }} />}
      </CardContent>
    </Card>
  );
}
