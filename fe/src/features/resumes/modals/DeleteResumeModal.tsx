import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { resumesService } from "@/services/resumesService";
import { showErrorNotification } from "@/shared/lib/notifications";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { AlertTriangle, Loader2 } from "lucide-react";
import type { ResumeDTO } from "@/shared/types/api";

interface DeleteResumeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resume: ResumeDTO;
}

export function DeleteResumeModal({
  open,
  onOpenChange,
  resume,
}: DeleteResumeModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: () => resumesService.delete(resume.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resumes"] });
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("resumes.deleteError"));
    },
  });

  const handleDelete = () => {
    deleteMutation.mutate();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-destructive" />
            <DialogTitle>{t("resumes.delete")}</DialogTitle>
          </div>
          <DialogDescription>
            {t("resumes.deleteConfirm", { name: resume.title })}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {(resume.applications_count ?? 0) > 0 ? (
            <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-800">
              <p className="font-medium">
                {t("resumes.inUseWarning", {
                  count: resume.applications_count,
                })}
              </p>
              <p className="mt-1">{t("resumes.inUseNote")}</p>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              {t("resumes.deleteWarning")}
            </p>
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
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t("common.loading")}
              </>
            ) : (
              t("resumes.delete")
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
