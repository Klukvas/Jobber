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
import type { JobStageDTO } from "@/shared/types/api";

interface UpdateStageStatusModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  jobId: string;
  stage: JobStageDTO;
}

const STAGE_STATUS_VALUES = [
  "pending",
  "active",
  "completed",
  "skipped",
  "cancelled",
] as const;

// Static i18n keys — a missing translation for a new stage status surfaces at
// build/lint time instead of silently rendering a broken runtime-built key.
const STAGE_STATUS_LABEL_KEYS: Record<JobStageDTO["status"], string> = {
  pending: "jobs.stageStatusPending",
  active: "jobs.stageStatusActive",
  completed: "jobs.stageStatusCompleted",
  skipped: "jobs.stageStatusSkipped",
  cancelled: "jobs.stageStatusCancelled",
};

const STAGE_STATUS_DESC_KEYS: Record<JobStageDTO["status"], string> = {
  pending: "jobs.stageStatusPendingDesc",
  active: "jobs.stageStatusActiveDesc",
  completed: "jobs.stageStatusCompletedDesc",
  skipped: "jobs.stageStatusSkippedDesc",
  cancelled: "jobs.stageStatusCancelledDesc",
};

export function UpdateStageStatusModal({
  open,
  onOpenChange,
  jobId,
  stage,
}: UpdateStageStatusModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [newStatus, setNewStatus] = useState<JobStageDTO["status"]>(
    stage.status,
  );

  const updateStatusMutation = useMutation({
    mutationFn: (status: JobStageDTO["status"]) =>
      jobsService.updateStage(jobId, stage.id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job-stages", jobId] });
      queryClient.invalidateQueries({ queryKey: ["job", jobId] });
      showSuccessNotification(t("jobs.stageStatusUpdateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.stageStatusUpdateError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newStatus !== stage.status) {
      updateStatusMutation.mutate(newStatus);
    } else {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("jobs.changeStageStatus")}</DialogTitle>
          <DialogDescription>
            {t("jobs.stageStatusDescription", {
              stageName: stage.stage_name,
            })}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("jobs.currentStatus")}</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm">
                {stage.status.charAt(0).toUpperCase() + stage.status.slice(1)}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="status">{`${t("jobs.newStatus")} *`}</Label>
              <select
                id="status"
                value={newStatus}
                onChange={(e) =>
                  setNewStatus(e.target.value as JobStageDTO["status"])
                }
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                {STAGE_STATUS_VALUES.map((value) => (
                  <option key={value} value={value}>
                    {t(STAGE_STATUS_LABEL_KEYS[value])} -{" "}
                    {t(STAGE_STATUS_DESC_KEYS[value])}
                  </option>
                ))}
              </select>
            </div>
            {updateStatusMutation.isError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {t("jobs.stageStatusUpdateFailed")}
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
                updateStatusMutation.isPending || newStatus === stage.status
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
