import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { applicationsService } from "@/services/applicationsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { stageTemplatesService } from "@/services/stageTemplatesService";
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
import { Input } from "@/shared/ui/Input";

interface AddStageModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
}

export function AddStageModal({
  open,
  onOpenChange,
  applicationId,
}: AddStageModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [stageTemplateId, setStageTemplateId] = useState("");
  const [comment, setComment] = useState("");

  const { data: stageTemplates } = useQuery({
    queryKey: ["stage-templates"],
    queryFn: () => stageTemplatesService.list({ limit: 100, offset: 0 }),
    enabled: open,
  });

  const addStageMutation = useMutation({
    mutationFn: (data: { stage_template_id: string; comment?: string }) =>
      applicationsService.addStage(applicationId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["application-stages", applicationId],
      });
      // Application query now includes embedded comments, so this refreshes everything
      queryClient.invalidateQueries({
        queryKey: ["application", applicationId],
      });
      showSuccessNotification(t("applications.stageAddedSuccess"));
      onOpenChange(false);
      setStageTemplateId("");
      setComment("");
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("applications.stageAddedError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (stageTemplateId) {
      const data: { stage_template_id: string; comment?: string } = {
        stage_template_id: stageTemplateId,
      };
      if (comment.trim()) {
        data.comment = comment.trim();
      }
      addStageMutation.mutate(data);
    }
  };

  const sortedTemplates = [...(stageTemplates?.items || [])].sort(
    (a, b) => a.order - b.order,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("applications.addStageTitle")}</DialogTitle>
          <DialogDescription>
            {t("applications.addStageDescription")}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="stage">{`${t("applications.selectStage")} *`}</Label>
              <select
                id="stage"
                value={stageTemplateId}
                onChange={(e) => setStageTemplateId(e.target.value)}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                required
              >
                <option value="">{t("applications.selectStageOption")}</option>
                {sortedTemplates.map((template) => (
                  <option key={template.id} value={template.id}>
                    {template.order}. {template.name}
                  </option>
                ))}
              </select>
              {sortedTemplates.length === 0 && (
                <p className="text-xs text-muted-foreground">
                  {t("applications.noStageTemplates")}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="comment">
                {t("applications.commentOptional")}
              </Label>
              <Input
                id="comment"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder={t("applications.stagePlaceholder")}
                className="w-full"
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
              disabled={addStageMutation.isPending || !stageTemplateId}
            >
              {addStageMutation.isPending
                ? t("common.loading")
                : t("applications.addStage")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
