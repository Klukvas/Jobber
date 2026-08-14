import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Label } from "@/shared/ui/Label";
import { Loader2 } from "lucide-react";
import type { JobStatus } from "@/shared/types/api";
import {
  JOB_STATUS_VALUES,
  JOB_STATUS_LABEL_KEYS,
  JOB_STATUS_DESC_KEYS,
} from "@/shared/lib/jobStatus";
import { JOBS_KANBAN_QUERY_KEY } from "@/features/jobs/components/JobKanbanBoard";

interface UpdateJobStatusModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  jobId: string;
  currentStatus: JobStatus;
}

export function UpdateJobStatusModal({
  open,
  onOpenChange,
  jobId,
  currentStatus,
}: UpdateJobStatusModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [newStatus, setNewStatus] = useState<JobStatus>(currentStatus);

  const updateStatusMutation = useMutation({
    mutationFn: (status: JobStatus) => jobsService.update(jobId, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", jobId] });
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      queryClient.invalidateQueries({ queryKey: JOBS_KANBAN_QUERY_KEY });
      showSuccessNotification(t("jobs.statusUpdateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.statusUpdateError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newStatus !== currentStatus) {
      updateStatusMutation.mutate(newStatus);
    } else {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("jobs.changeStatusTitle")}</DialogTitle>
          <DialogDescription>
            {t("jobs.changeStatusDescription")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("jobs.currentStatus")}</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm">
                {t(JOB_STATUS_LABEL_KEYS[currentStatus])}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="status">{`${t("jobs.newStatus")} *`}</Label>
              <select
                id="status"
                value={newStatus}
                onChange={(e) => setNewStatus(e.target.value as JobStatus)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                {JOB_STATUS_VALUES.map((value) => (
                  <option key={value} value={value}>
                    {t(JOB_STATUS_LABEL_KEYS[value])} -{" "}
                    {t(JOB_STATUS_DESC_KEYS[value])}
                  </option>
                ))}
              </select>
            </div>
            {updateStatusMutation.isError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {t("jobs.statusUpdateFailed")}
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
              disabled={
                updateStatusMutation.isPending || newStatus === currentStatus
              }
            >
              {updateStatusMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t("common.loading")}
                </>
              ) : (
                t("jobs.updateStatus")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
