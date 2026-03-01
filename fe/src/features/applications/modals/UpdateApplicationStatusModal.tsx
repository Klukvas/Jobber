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

interface UpdateApplicationStatusModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
  currentStatus: string;
}

const APPLICATION_STATUS_VALUES = [
  "active",
  "on_hold",
  "rejected",
  "offer",
  "archived",
] as const;

function statusToTranslationKey(status: string): string {
  return status
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

export function UpdateApplicationStatusModal({
  open,
  onOpenChange,
  applicationId,
  currentStatus,
}: UpdateApplicationStatusModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [newStatus, setNewStatus] = useState(currentStatus);

  const updateStatusMutation = useMutation({
    mutationFn: (status: string) =>
      applicationsService.update(applicationId, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["application", applicationId],
      });
      queryClient.invalidateQueries({ queryKey: ["applications"] });
      showSuccessNotification(t("applications.statusUpdateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("applications.statusUpdateError"),
      );
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
          <DialogTitle>{t("applications.changeStatusTitle")}</DialogTitle>
          <DialogDescription>
            {t("applications.changeStatusDescription")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("applications.currentStatus")}</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm">
                {t(
                  `applications.status${statusToTranslationKey(currentStatus)}`,
                )}
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
                {APPLICATION_STATUS_VALUES.map((value) => {
                  const key = statusToTranslationKey(value);
                  return (
                    <option key={value} value={value}>
                      {t(`applications.status${key}`)} -{" "}
                      {t(`applications.status${key}Desc`)}
                    </option>
                  );
                })}
              </select>
            </div>
            {updateStatusMutation.isError && (
              <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                {t("applications.statusUpdateFailed")}
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
                t("applications.updateStatus")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
