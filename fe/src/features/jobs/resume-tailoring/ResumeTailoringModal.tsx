import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowRight, Loader2, Sparkles } from "lucide-react";
import type { JobDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";
import { Button } from "@/shared/ui/Button";
import { Checkbox } from "@/shared/ui/Checkbox";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/shared/ui/Dialog";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Select } from "@/shared/ui/Select";
import { showSuccessNotification } from "@/shared/lib/notifications";
import { useResumeTailoring } from "./useResumeTailoring";
import { TailoringChangeCard } from "./TailoringChangeCard";

type Props = {
  job: JobDTO;
  builderResumes: readonly ResumeBuilderDTO[];
  isLoading: boolean;
  isError: boolean;
  onClose: () => void;
};

export function ResumeTailoringModal({ job, builderResumes, isLoading, isError, onClose }: Props) {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const initialSource = job.resume?.type === "builder" && builderResumes.some((resume) => resume.id === job.resume?.id) ? job.resume.id : builderResumes[0]?.id ?? "";
  const [sourceId, setSourceId] = useState(initialSource);
  const [title, setTitle] = useState(() => t("jobs.tailoring.defaultTitle", { title: job.title }).slice(0, 255));
  const tailoring = useResumeTailoring(job.id);
  const source = builderResumes.find((resume) => resume.id === sourceId);
  const review = tailoring.review;
  const sourceChanged = !!review && source?.id !== review.proposal.source_resume.id;
  const mustRegenerate = tailoring.mustRegenerate || sourceChanged;
  const changes = review?.proposal.changes ?? [];
  const allSelected = changes.length > 0 && tailoring.acceptedIds.length === changes.length;
  const canSave = !!review && !mustRegenerate && tailoring.acceptedIds.length > 0 && !!title.trim() && title.trim().length <= 255;
  const close = () => { if (!tailoring.isSaving) onClose(); };
  const locale = i18n.language.startsWith("ru") ? "ru" : i18n.language.startsWith("uk") ? "uk" : "en";
  const generate = () => { if (source) void tailoring.generate(source.id, locale); };
  const save = async () => {
    if (!source || !canSave) return;
    const result = await tailoring.apply(title, source.id);
    if (!result) return;
    showSuccessNotification(t("jobs.tailoring.saved"));
    navigate(`/app/resume-builder/${result.resume_builder.id}`);
  };

  return (
    <Dialog open onOpenChange={(open) => { if (!open) close(); }} className="max-w-4xl">
      <DialogContent onClose={tailoring.isSaving ? undefined : close} className="flex max-h-[90dvh] max-w-none flex-col overflow-hidden p-0">
        <DialogHeader className="shrink-0 border-b p-5 pr-12 text-left sm:p-6 sm:pr-12">
          <DialogTitle className="leading-snug">{t("jobs.tailoring.title")}</DialogTitle>
          <DialogDescription className="break-words">{job.title}{job.company_name ? ` — ${job.company_name}` : ""}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 space-y-5 overflow-y-auto overscroll-contain p-5 sm:p-6">
          <p className="text-sm text-muted-foreground">{t("jobs.tailoring.description")}</p>
          {isLoading ? <p role="status" className="text-sm text-muted-foreground">{t("common.loading")}</p>
            : isError ? <p role="alert" className="text-sm text-destructive">{t("jobs.tailoring.errors.resumesFailed")}</p>
              : builderResumes.length === 0 ? (
                <div className="space-y-3 rounded-lg border border-dashed p-4 text-sm">
                  <p>{t("jobs.tailoring.empty")}</p>
                  <Link className="inline-flex min-h-11 items-center gap-2 text-foreground underline underline-offset-4" to="/app/resumes">{t("jobs.tailoring.createResume")}<ArrowRight className="h-4 w-4" /></Link>
                </div>
              ) : (
                <div className="space-y-2">
                  <Label htmlFor="tailoring-source">{t("jobs.tailoring.source")}</Label>
                  <Select id="tailoring-source" value={sourceId} disabled={tailoring.isSaving} onChange={(event) => { tailoring.reset(); setSourceId(event.target.value); }}>
                    <option value="">{t("jobs.tailoring.chooseSource")}</option>
                    {builderResumes.map((resume) => <option key={resume.id} value={resume.id}>{resume.title}</option>)}
                  </Select>
                </div>
              )}
          {tailoring.isGenerating && <p role="status" className="flex items-center gap-2 text-sm"><Loader2 className="h-4 w-4 animate-spin" />{t("jobs.tailoring.generating")}</p>}
          {(tailoring.errorKey || sourceChanged) && (
            <div role="alert" className="space-y-2 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm">
              <p>{t(`jobs.tailoring.errors.${sourceChanged ? "stale" : tailoring.errorKey}`)}</p>
              {tailoring.errorKey === "sourceUnsaved" && source && <Link className="inline-flex min-h-11 items-center text-foreground underline" to={`/app/resume-builder/${source.id}`}>{t("jobs.tailoring.openSource")}</Link>}
            </div>
          )}
          {review && (
            <div className="space-y-4">
              <div className="space-y-2">
                <h3 className="font-semibold">{t("jobs.tailoring.review")}</h3>
                <p className="text-sm text-muted-foreground">{t("jobs.tailoring.reviewHint")}</p>
              </div>
              {changes.length === 0 ? <p role="status" className="rounded-md bg-muted p-4 text-sm">{t("jobs.tailoring.noChanges")}</p> : (
                <>
                  <label className="flex min-h-11 cursor-pointer items-center gap-3 text-sm" htmlFor="tailoring-all">
                    <Checkbox id="tailoring-all" checked={allSelected} disabled={tailoring.isSaving || mustRegenerate} onCheckedChange={tailoring.selectAll} />{t("jobs.tailoring.selectAll")}
                  </label>
                  {changes.map((change) => <TailoringChangeCard key={change.id} change={change} checked={tailoring.acceptedIds.includes(change.id)} disabled={tailoring.isSaving || mustRegenerate} onCheckedChange={(checked) => tailoring.toggle(change.id, checked)} />)}
                  <div className="space-y-2">
                    <Label htmlFor="tailoring-title">{t("jobs.tailoring.copyTitle")}</Label>
                    <Input id="tailoring-title" value={title} maxLength={255} disabled={tailoring.isSaving} onChange={(event) => setTitle(event.target.value)} />
                    <p className="text-xs text-muted-foreground">{t("jobs.tailoring.copyHint")}</p>
                  </div>
                </>
              )}
            </div>
          )}
        </div>
        <DialogFooter className="shrink-0 flex-col border-t bg-background p-4 sm:flex-row sm:items-center sm:p-5">
          {review && changes.length > 0 && <p aria-live="polite" className="text-sm text-muted-foreground sm:mr-auto">{t("jobs.tailoring.selected", { count: tailoring.acceptedIds.length, total: changes.length })}</p>}
          <Button variant="outline" onClick={close} disabled={tailoring.isSaving}>{t("common.cancel")}</Button>
          {review && !mustRegenerate && changes.length > 0 ? (
            <Button onClick={() => void save()} disabled={!canSave || tailoring.isSaving}>
              {tailoring.isSaving && <Loader2 className="h-4 w-4 animate-spin" />}{t("jobs.tailoring.saveCopy")}
            </Button>
          ) : (
            <Button onClick={generate} disabled={!source || isLoading || isError || tailoring.isGenerating || tailoring.isSaving}>
              {tailoring.isGenerating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
              {t(review ? "jobs.tailoring.regenerate" : "jobs.tailoring.generate")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
