import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Sparkles, Loader2, Check, X, Search } from "lucide-react";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { useCoverLetterAI } from "../hooks/useCoverLetterAI";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { UpgradeBanner } from "@/features/subscription/components/UpgradeBanner";
import { jobsService } from "@/services/jobsService";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";

export function CoverLetterAIPanel() {
  const { t } = useTranslation();
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const updateFields = useCoverLetterStore((s) => s.updateFields);
  const { generate } = useCoverLetterAI();
  const { canCreate } = useSubscription();
  const aiLimitReached = !canCreate("ai");

  const [jobDescription, setJobDescription] = useState("");
  const [jobSearch, setJobSearch] = useState("");
  const [isJobDropdownOpen, setIsJobDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const { data: jobsData } = useQuery({
    queryKey: ["jobs", "active"],
    queryFn: () =>
      jobsService.list({ status: "active", limit: 100, offset: 0 }),
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

  const handleGenerate = () => {
    if (!coverLetter) return;
    generate.mutate({
      cover_letter_id: coverLetter.id,
      job_description: jobDescription || undefined,
    });
  };

  const handleApply = () => {
    if (!generate.data) return;
    updateFields({
      greeting: generate.data.greeting,
      paragraphs: generate.data.paragraphs,
      closing: generate.data.closing,
    });
    generate.reset();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <Sparkles className="h-5 w-5 text-primary" />
        <h2 className="text-lg font-semibold">{t("coverLetter.ai.title")}</h2>
      </div>

      {aiLimitReached && <UpgradeBanner resource="ai" />}

      <section className="space-y-3">
        <Label>{t("coverLetter.ai.jobDescription")}</Label>

        {jobs.length > 0 && (
          <div className="relative" ref={dropdownRef}>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                value={jobSearch}
                onChange={(e) => {
                  setJobSearch(e.target.value);
                  setIsJobDropdownOpen(true);
                }}
                onFocus={() => setIsJobDropdownOpen(true)}
                onBlur={() => setIsJobDropdownOpen(false)}
                placeholder={t("coverLetter.ai.selectJobPlaceholder")}
                className="w-full rounded-md border border-input bg-background py-2 pl-9 pr-3 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              />
            </div>
            {isJobDropdownOpen && filteredJobs.length > 0 && (
              <ul className="absolute z-10 mt-1 max-h-48 w-full overflow-auto rounded-md border border-input bg-background py-1 shadow-md">
                {filteredJobs.map((job) => (
                  <li
                    key={job.id}
                    onMouseDown={() => handleJobSelect(job.id)}
                    className="cursor-pointer px-3 py-2 text-sm hover:bg-accent hover:text-accent-foreground"
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
          value={jobDescription}
          onChange={(e) => setJobDescription(e.target.value)}
          placeholder={t("coverLetter.ai.jobDescriptionPlaceholder")}
          rows={5}
        />

        <Button
          onClick={handleGenerate}
          disabled={generate.isPending || !coverLetter || aiLimitReached}
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

        {generate.isError && (
          <p className="text-sm text-destructive">
            {t("coverLetter.ai.error")}
          </p>
        )}

        {generate.data && (
          <Card className="border-primary/20 bg-primary/5">
            <CardHeader className="p-4 pb-2">
              <CardTitle className="flex items-center gap-2 text-sm font-medium">
                <Sparkles className="h-3.5 w-3.5 text-primary" />
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
                <Button size="sm" onClick={handleApply}>
                  <Check className="h-3.5 w-3.5" />
                  {t("coverLetter.ai.apply")}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
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
