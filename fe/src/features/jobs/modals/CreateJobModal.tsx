import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { companiesService } from "@/services/companiesService";
import { resumesService } from "@/services/resumesService";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import type {
  JobDTO,
  CreateJobRequest,
  UpdateJobRequest,
} from "@/shared/types/api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";
import { Switch } from "@/shared/ui/Switch";
import { CompanySelectWithQuickAdd } from "@/features/jobs/components/CompanySelectWithQuickAdd";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { UpgradeBanner } from "@/features/subscription/components/UpgradeBanner";
import { Loader2 } from "lucide-react";

interface CreateJobModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  job?: JobDTO; // If provided, the modal is in edit mode
  onCreated?: (job: JobDTO) => void;
}

function todayISODate(): string {
  return new Date().toISOString().slice(0, 10);
}

// Inner content component that resets state when key changes
function ModalContent({
  job,
  onOpenChange,
  open,
  onCreated,
}: CreateJobModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isEditMode = !!job;
  const { canCreate } = useSubscription();
  const showLimitBanner = !isEditMode && !canCreate("jobs");

  const [title, setTitle] = useState(job?.title || "");
  const [companyId, setCompanyId] = useState(job?.company_id || "");
  const [url, setUrl] = useState(job?.url || "");
  const [source, setSource] = useState(job?.source || "");
  const [notes, setNotes] = useState(job?.notes || "");
  const [description, setDescription] = useState(job?.description || "");
  const [alreadyApplied, setAlreadyApplied] = useState(false);
  const [selectedResume, setSelectedResume] = useState("");
  const [appliedDate, setAppliedDate] = useState(todayISODate());

  const { data: companiesData } = useQuery({
    queryKey: ["companies"],
    queryFn: () => companiesService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const { data: resumesData } = useQuery({
    queryKey: ["resumes"],
    queryFn: () => resumesService.list({ limit: 100, offset: 0 }),
    enabled: open && !isEditMode,
  });

  const { data: builderResumes } = useQuery({
    queryKey: ["resume-builder"],
    queryFn: () => resumeBuilderService.list(),
    enabled: open && !isEditMode,
  });

  const createMutation = useMutation({
    mutationFn: jobsService.create,
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.createSuccess"));
      onCreated?.(data);
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.createError"));
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateJobRequest }) =>
      jobsService.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.updateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.updateError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title) return;

    if (isEditMode && job) {
      updateMutation.mutate({
        id: job.id,
        data: {
          title,
          company_id: companyId || undefined,
          url: url || undefined,
          source: source || undefined,
          notes: notes || undefined,
          description: description || undefined,
        },
      });
      return;
    }

    const request: CreateJobRequest = {
      title,
      company_id: companyId || undefined,
      url: url || undefined,
      source: source || undefined,
      notes: notes || undefined,
      description: description || undefined,
    };

    if (alreadyApplied) {
      // Marking a job as already-applied just records the applied date; the
      // backend places the card in the appropriate column.
      request.applied_at = appliedDate
        ? new Date(`${appliedDate}T00:00:00`).toISOString()
        : new Date().toISOString();
      if (selectedResume.startsWith("uploaded:")) {
        request.resume_id = selectedResume.slice("uploaded:".length);
      } else if (selectedResume.startsWith("builder:")) {
        request.resume_builder_id = selectedResume.slice("builder:".length);
      }
    }

    createMutation.mutate(request);
  };

  const isPending = createMutation.isPending || updateMutation.isPending;
  const uploadedResumes = resumesData?.items ?? [];
  const builderResumesList = builderResumes ?? [];

  return (
    <>
      <DialogHeader>
        <DialogTitle>
          {isEditMode ? t("jobs.edit") : t("jobs.create")}
        </DialogTitle>
        <DialogDescription>
          {isEditMode ? t("jobs.editDescription") : t("jobs.createDescription")}
        </DialogDescription>
      </DialogHeader>
      {showLimitBanner && (
        <div className="py-2">
          <UpgradeBanner resource="jobs" />
        </div>
      )}
      <form onSubmit={handleSubmit}>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="title">{t("jobs.title_field")} *</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("jobs.titlePlaceholder")}
              required
            />
          </div>
          <CompanySelectWithQuickAdd
            companies={companiesData?.items || []}
            value={companyId}
            onChange={setCompanyId}
          />
          <div className="space-y-2">
            <Label htmlFor="url">{t("jobs.url")}</Label>
            <Input
              id="url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com/jobs/123"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="source">{t("jobs.source")}</Label>
            <Input
              id="source"
              value={source}
              onChange={(e) => setSource(e.target.value)}
              placeholder={t("jobs.sourcePlaceholder")}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">{t("jobs.description")}</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("jobs.descriptionPlaceholder")}
              className="min-h-[100px]"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="notes">{t("jobs.notes")}</Label>
            <Textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t("jobs.notesPlaceholder")}
            />
          </div>

          {!isEditMode && (
            <div className="space-y-4 rounded-lg border p-3">
              <div className="flex items-center justify-between gap-3">
                <button
                  type="button"
                  onClick={() => setAlreadyApplied(!alreadyApplied)}
                  className="flex-1 text-left"
                >
                  <span
                    id="already-applied-label"
                    className="block text-sm font-medium"
                  >
                    {t("jobs.alreadyApplied")}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {t("jobs.alreadyAppliedHint")}
                  </span>
                </button>
                <Switch
                  id="already-applied"
                  checked={alreadyApplied}
                  onCheckedChange={setAlreadyApplied}
                  aria-labelledby="already-applied-label"
                />
              </div>

              {alreadyApplied && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="resume">{t("jobs.resumeLabel")}</Label>
                    <select
                      id="resume"
                      value={selectedResume}
                      onChange={(e) => setSelectedResume(e.target.value)}
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
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
                      {builderResumesList.length > 0 && (
                        <optgroup label={t("jobs.builderResumes")}>
                          {builderResumesList.map((rb) => (
                            <option
                              key={`builder:${rb.id}`}
                              value={`builder:${rb.id}`}
                            >
                              {rb.title}
                            </option>
                          ))}
                        </optgroup>
                      )}
                    </select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="applied-date">
                      {t("jobs.appliedDateLabel")}
                    </Label>
                    <Input
                      id="applied-date"
                      type="date"
                      value={appliedDate}
                      onChange={(e) => setAppliedDate(e.target.value)}
                    />
                  </div>
                </>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            type="submit"
            disabled={isPending || !title || showLimitBanner}
          >
            {isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t("common.loading")}
              </>
            ) : isEditMode ? (
              t("common.save")
            ) : (
              t("common.create")
            )}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

export function CreateJobModal({
  open,
  onOpenChange,
  job,
  onCreated,
}: CreateJobModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        {/* Key prop resets the form state when job changes or modal reopens */}
        <ModalContent
          key={`${job?.id || "new"}-${open}`}
          job={job}
          onOpenChange={onOpenChange}
          open={open}
          onCreated={onCreated}
        />
      </DialogContent>
    </Dialog>
  );
}
