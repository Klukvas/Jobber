import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { companiesService } from "@/services/companiesService";
import type { JobDTO, UpdateJobRequest } from "@/shared/types/api";
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

  const { data: companiesData } = useQuery({
    queryKey: ["companies"],
    queryFn: () => companiesService.list({ limit: 100, offset: 0 }),
    enabled: open,
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
    if (title) {
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
      } else {
        createMutation.mutate({
          title,
          company_id: companyId || undefined,
          url: url || undefined,
          source: source || undefined,
          notes: notes || undefined,
          description: description || undefined,
        });
      }
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;

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
