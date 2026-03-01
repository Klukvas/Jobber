import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { applicationsService } from "@/services/applicationsService";
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
import type { ApplicationStageDTO } from "@/shared/types/api";

interface UpdateStageStatusModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
  stage: ApplicationStageDTO;
}

const STAGE_STATUS_VALUES = [
  "pending",
  "active",
  "completed",
  "skipped",
  "cancelled",
] as const;

export function UpdateStageStatusModal({
  open,
  onOpenChange,
  applicationId,
  stage,
}: UpdateStageStatusModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [newStatus, setNewStatus] = useState(stage.status);

  const updateStatusMutation = useMutation({
    mutationFn: (status: string) =>
      applicationsService.updateStage(applicationId, stage.id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["application-stages", applicationId],
      });
      queryClient.invalidateQueries({
        queryKey: ["application", applicationId],
      });
      showSuccessNotification(t("applications.stageStatusUpdateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("applications.stageStatusUpdateError"),
      );
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
          <DialogTitle>{t("applications.changeStageStatus")}</DialogTitle>
          <DialogDescription>
            {t("applications.stageStatusDescription", {
              stageName: stage.stage_name,
            })}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("applications.currentStatus")}</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm">
                {stage.status.charAt(0).toUpperCase() + stage.status.slice(1)}
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="status">{`${t("applications.newStatus")} *`}</Label>
              <select
                id="status"
                value={newStatus}
                onChange={(e) => setNewStatus(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                {STAGE_STATUS_VALUES.map((value) => (
                  <option key={value} value={value}>
                    {t(
                      `applications.stageStatus${value.charAt(0).toUpperCase() + value.slice(1)}`,
                    )}{" "}
                    -{" "}
                    {t(
                      `applications.stageStatus${value.charAt(0).toUpperCase() + value.slice(1)}Desc`,
                    )}
                  </option>
                ))}
              </select>
            </div>
            {updateStatusMutation.isError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {t("applications.stageStatusUpdateFailed")}
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
                t("applications.updateStatus")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
