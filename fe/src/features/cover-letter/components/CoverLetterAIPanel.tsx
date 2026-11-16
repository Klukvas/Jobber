import { isAIActionCancelled } from "@/shared/lib/aiCredits";
import { useAIAvailability, aiErrorKey } from "@/shared/hooks/useAIAvailability";
import { useTopmostEscape } from "@/shared/hooks/useTopmostEscape";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import { Select } from "@/shared/ui/Select";
import { flushCoverLetterDocument } from "../lib/coverLetterDocumentSync";
import { hasCandidateSource } from "../lib/candidateSource";
import { AIAvailabilityNotice } from "@/shared/ui/AIAvailabilityNotice";
import { useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Sparkles, Loader2, Check, X, Search } from "lucide-react";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import {
  useCoverLetterAIPanelState,
  type CoverLetterAIPanelState,
} from "../hooks/useCoverLetterAIPanelState";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { UpgradeBanner } from "@/features/subscription/components/UpgradeBanner";
import { jobsService } from "@/services/jobsService";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";

interface CoverLetterAIPanelProps {
  /**
   * Panel state owned by the caller, so it outlives this mount point. Without
   * it the panel keeps its own, which resets whenever the panel remounts.
   */
  readonly state?: CoverLetterAIPanelState;
}

export function CoverLetterAIPanel({ state }: CoverLetterAIPanelProps = {}) {
  const { t } = useTranslation();
  const { unavailable, isChecking, markUnavailable } = useAIAvailability();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateField = useCoverLetterStore((s) => s.updateField);
  const updateFields = useCoverLetterStore((s) => s.updateFields);
  const { canCreate } = useSubscription();
  const aiLimitReached = !canCreate("ai");

  const ownState = useCoverLetterAIPanelState(coverLetter?.id);
  const {
    jobDescription,
    setJobDescription,
    jobSearch,
    setJobSearch,
    generate,
  } = state ?? ownState;
  const prefix = useId();
  const [activeIndex, setActiveIndex] = useState(0);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [isPreparing, setIsPreparing] = useState(false);
  const [isApplying, setIsApplying] = useState(false);
  const applyLock = useRef(false);
  const prepareLock = useRef(false);
  const [isJobDropdownOpen, setIsJobDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useTopmostEscape(isJobDropdownOpen, dropdownRef, () => setIsJobDropdownOpen(false));

  const { data: resumeList } = useQuery({ queryKey: resumeQueryKeys.builderList(), queryFn: () => resumeBuilderService.list() });
  const { data: jobsData } = useQuery({
    // Status omitted => all non-archived jobs
    queryKey: ["jobs", "non-archived"],
    queryFn: () => jobsService.list({ limit: 100, offset: 0 }),
  });

  const jobs = jobsData?.items ?? [];

  const filteredJobs = jobSearch
    ? jobs.filter((job) => {
        const query = jobSearch.toLowerCase();
        return (
          job.title.toLowerCase().includes(query) ||
          (job.company_name?.toLowerCase().includes(query) ?? false)
        );
      })
    : jobs;

  const handleJobSelect = (jobId: string) => {
    const job = jobs.find((j) => j.id === jobId);
    if (!job) return;
    setJobSearch(
      `${job.title}${job.company_name ? ` (${job.company_name})` : ""}`,
    );
    setIsJobDropdownOpen(false);
    setJobDescription(job.description ?? "");
    if (coverLetter) {
      const updates: Parameters<typeof updateFields>[0] = { job_id: job.id };
      if (!coverLetter.company_name && job.company_name) {
        updates.company_name = job.company_name;
      }
      updateFields(updates);
    }
  };

  const handleGenerate = async () => {
    if (!coverLetter || unavailable || isChecking || generate.isPending || prepareLock.current) return;
    if (!jobDescription.trim()) { setValidationError("coverLetter.ai.jobDescriptionRequired"); return; }
    if (!coverLetter.resume_builder_id) { setValidationError("coverLetter.ai.resumeRequired"); return; }
    prepareLock.current = true;
    setIsPreparing(true);
    setValidationError(null);
    try {
      const source = await resumeBuilderService.getById(coverLetter.resume_builder_id);
      if (!hasCandidateSource(source)) { setValidationError("coverLetter.ai.resumeRequired"); return; }
      await flushCoverLetterDocument(coverLetter.id);
      if (useCoverLetterStore.getState().coverLetter?.id !== coverLetter.id) return;
      generate.mutate({ cover_letter_id: coverLetter.id, job_description: jobDescription.trim() }, { onError: markUnavailable });
    } catch {
      setValidationError("coverLetter.ai.sourceUnavailable");
    } finally {
      prepareLock.current = false;
      setIsPreparing(false);
    }
  };

  const handleApply = async () => {
    if (!generate.data || !coverLetter || applyLock.current) return;
    applyLock.current = true;
    setIsApplying(true);
    setValidationError(null);
    try {
      updateFields({ greeting: generate.data.greeting, paragraphs: generate.data.paragraphs, closing: generate.data.closing });
      await flushCoverLetterDocument(coverLetter.id);
      generate.reset();
    } catch {
      setValidationError("coverLetter.saveFailed");
    } finally {
      applyLock.current = false;
      setIsApplying(false);
    }
  };

  return (
    <div className="space-y-6">
      <AIAvailabilityNotice unavailable={unavailable} />
      {aiLimitReached && <UpgradeBanner resource="ai" />}

      <section className="space-y-3">
        <Label htmlFor={`${prefix}-resume`}>{t("coverLetter.ai.resume")}</Label>
        <Select id={`${prefix}-resume`} value={coverLetter?.resume_builder_id ?? ""} onChange={(event) => { updateField("resume_builder_id", event.target.value || null); setValidationError(null); }}>
          <option value="">{t("coverLetter.ai.selectResume")}</option>
          {(resumeList ?? []).map((resume) => <option key={resume.id} value={resume.id}>{resume.title}</option>)}
        </Select>
        <Label htmlFor={`${prefix}-description`}>{t("coverLetter.ai.jobDescription")}</Label>

        {jobs.length > 0 && (
          <div className="relative" ref={dropdownRef}>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                role="combobox"
                aria-label={t("coverLetter.ai.selectJobPlaceholder")}
                aria-autocomplete="list"
                aria-expanded={isJobDropdownOpen}
                aria-controls={`${prefix}-jobs`}
                aria-activedescendant={isJobDropdownOpen && filteredJobs[activeIndex] ? `${prefix}-job-${filteredJobs[activeIndex].id}` : undefined}
                value={jobSearch}
                onChange={(e) => {
                  setJobSearch(e.target.value);
                  setActiveIndex(0);
                  setIsJobDropdownOpen(true);
                }}
                onFocus={() => { setActiveIndex(0); setIsJobDropdownOpen(true); }}
                onKeyDown={(event) => {
                  if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
                    event.preventDefault();
                    setIsJobDropdownOpen(true);
                    setActiveIndex((current) => {
                      const last = filteredJobs.length - 1;
                      const next = event.key === "Home" ? 0 : event.key === "End" ? last : event.key === "ArrowDown" ? Math.min(current + (isJobDropdownOpen ? 1 : 0), last) : Math.max(current - 1, 0);
                      document.getElementById(`${prefix}-job-${filteredJobs[next]?.id}`)?.scrollIntoView?.({ block: "nearest" });
                      return Math.max(0, next);
                    });
                  }
                  if (event.key === "Enter" && isJobDropdownOpen) {
                    event.preventDefault();
                    const job = filteredJobs[activeIndex];
                    if (job) handleJobSelect(job.id);
                  }
                  if (event.key === "Escape" && isJobDropdownOpen) { event.preventDefault(); event.stopPropagation(); setIsJobDropdownOpen(false); }
                }}
                onBlur={() => setIsJobDropdownOpen(false)}
                placeholder={t("coverLetter.ai.selectJobPlaceholder")}
                className="w-full rounded-md border border-input bg-background py-2 pl-9 pr-3 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              />
            </div>
            {isJobDropdownOpen && (
              <ul id={`${prefix}-jobs`} role="listbox" aria-label={t("coverLetter.ai.selectJobPlaceholder")} className="absolute z-10 mt-1 max-h-48 w-full overflow-auto rounded-md border border-input bg-background py-1 shadow-md">
                {filteredJobs.map((job, index) => (
                  <li
                    key={job.id}
                    id={`${prefix}-job-${job.id}`}
                    role="option"
                    aria-selected={activeIndex === index}
                    onMouseDown={(event) => event.preventDefault()}
                    onClick={() => handleJobSelect(job.id)}
                    className={`min-h-11 cursor-pointer px-3 py-2 text-sm hover:bg-accent hover:text-accent-foreground ${activeIndex === index ? "bg-accent text-accent-foreground" : ""}`}
                  >
                    {job.title}
                    {job.company_name ? ` (${job.company_name})` : ""}
                  </li>
                ))}
              </ul>
            )}
            <p className="mt-1 text-xs text-muted-foreground">
              {t("coverLetter.ai.orEnterManually")}
            </p>
          </div>
        )}

        <Textarea
          id={`${prefix}-description`}
          aria-invalid={validationError === "coverLetter.ai.jobDescriptionRequired"}
          aria-describedby={validationError ? `${prefix}-error` : undefined}
          maxLength={10000}
          value={jobDescription}
          onChange={(e) => setJobDescription(e.target.value)}
          placeholder={t("coverLetter.ai.jobDescriptionPlaceholder")}
          rows={5}
        />

        {validationError && <p id={`${prefix}-error`} role="alert" className="text-sm text-destructive">{t(validationError)}</p>}
        <Button
          onClick={() => void handleGenerate()}
          disabled={isPreparing || unavailable || isChecking || generate.isPending || !coverLetter || aiLimitReached}
          variant="outline"
          className="w-full"
        >
          {generate.isPending ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("coverLetter.ai.generating")}
            </>
          ) : (
            <>
              <Sparkles className="h-4 w-4" />
              {t("coverLetter.ai.generate")}
            </>
          )}
        </Button>

        {generate.isError && !isAIActionCancelled(generate.error) && (
          <p role="alert" className="text-sm text-destructive">
            {t(aiErrorKey(generate.error))}
          </p>
        )}

        {generate.data && (
          <Card className="border-primary/20 bg-primary/5">
            <CardHeader className="p-4 pb-2">
              <CardTitle className="flex items-center gap-2 text-sm font-medium">
                <Sparkles className="h-3.5 w-3.5 text-primary-ink" />
                {t("coverLetter.ai.result")}
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4 pt-0">
              <div className="mb-3 space-y-2 text-sm text-foreground">
                <p className="font-medium">{generate.data.greeting}</p>
                {generate.data.paragraphs.map((paragraph, index) => (
                  <p key={`${index}-${paragraph.slice(0, 20)}`}>{paragraph}</p>
                ))}
                <p className="font-medium">{generate.data.closing}</p>
              </div>
              <div className="flex gap-2">
                <Button size="sm" onClick={() => void handleApply()} disabled={isApplying}>
                  <Check className="h-3.5 w-3.5" />
                  {t("coverLetter.ai.apply")}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={isApplying}
                  onClick={() => generate.reset()}
                >
                  <X className="h-3.5 w-3.5" />
                  {t("coverLetter.ai.dismiss")}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}
      </section>
    </div>
  );
}
