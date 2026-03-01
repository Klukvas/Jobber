import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { companiesService } from "@/services/companiesService";
import { showErrorNotification } from "@/shared/lib/notifications";
import type { CompanyDTO } from "@/shared/types/api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { AlertCircle, Loader2 } from "lucide-react";

interface DeleteCompanyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  company: CompanyDTO;
}

export function DeleteCompanyDialog({
  open,
  onOpenChange,
  company,
}: DeleteCompanyDialogProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  // Fetch related counts when dialog opens
  const { data: relatedCounts } = useQuery({
    queryKey: ["company-related-counts", company.id],
    queryFn: () => companiesService.getRelatedCounts(company.id),
    enabled: open,
  });

  const deleteMutation = useMutation({
    mutationFn: () => companiesService.delete(company.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("companies.deleteError"));
    },
  });

  const handleDelete = () => {
    deleteMutation.mutate();
  };

  const hasRelatedData =
    (relatedCounts?.jobs_count || 0) > 0 ||
    (relatedCounts?.applications_count || 0) > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("companies.delete")}</DialogTitle>
          <DialogDescription>
            {t("companies.deleteConfirm", { name: company.name })}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {hasRelatedData && (
            <div className="bg-amber-50 dark:bg-amber-950 border border-amber-200 dark:border-amber-800 rounded-md p-4">
              <div className="flex gap-3">
                <AlertCircle className="h-5 w-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                <div className="space-y-2">
                  <p className="text-sm font-medium text-amber-900 dark:text-amber-100">
                    {t("companies.deleteWarning")}
                  </p>
                  <ul className="text-sm text-amber-800 dark:text-amber-200 space-y-1 list-disc list-inside">
                    {relatedCounts && relatedCounts.jobs_count > 0 && (
                      <li>
                        {t("companies.deleteJobsWarning", {
                          count: relatedCounts.jobs_count,
                        })}
                      </li>
                    )}
                    {relatedCounts && relatedCounts.applications_count > 0 && (
                      <li>
                        {t("companies.deleteAppsWarning", {
                          count: relatedCounts.applications_count,
                        })}
                      </li>
                    )}
                  </ul>
                  <p className="text-sm text-amber-800 dark:text-amber-200 mt-2">
                    {t("companies.deleteRelatedNote")}
                  </p>
                </div>
              </div>
            </div>
          )}

          {!hasRelatedData && (
            <p className="text-sm text-muted-foreground">
              {t("companies.deleteSafe")}
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
              t("common.delete")
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
