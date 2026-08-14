import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { stageTemplatesService } from "@/services/stageTemplatesService";
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
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Loader2 } from "lucide-react";
import { PhasePicker } from "../components/PhasePicker";
import { ApiError } from "@/services/api";
import type { StagePhase } from "@/shared/types/api";
import type { StageTemplateDTO } from "@/shared/types/api";

interface EditStageTemplateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: StageTemplateDTO | null;
}

export function EditStageTemplateModal({
  open,
  onOpenChange,
  template,
}: EditStageTemplateModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [order, setOrder] = useState("");
  const [phase, setPhase] = useState<StagePhase>("in_progress");
  const [prevTemplateId, setPrevTemplateId] = useState<string | null>(null);

  if (template && template.id !== prevTemplateId) {
    setPrevTemplateId(template.id);
    setName(template.name);
    setOrder(String(template.order));
    setPhase(template.phase);
  }

  const updateMutation = useMutation({
    mutationFn: (data: { name: string; order: number; phase: StagePhase }) =>
      stageTemplatesService.update(template!.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stage-templates"] });
      showSuccessNotification(t("stages.updateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      if (
        error instanceof ApiError &&
        error.code === "STAGE_TEMPLATE_NAME_EXISTS"
      ) {
        showErrorNotification(t("stages.nameExists"));
        return;
      }
      showErrorNotification(error.message || t("stages.updateError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name && order && template) {
      updateMutation.mutate({
        name,
        order: parseInt(order),
        phase,
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <DialogHeader>
          <DialogTitle>{t("stages.edit")}</DialogTitle>
          <DialogDescription>{t("stages.editDescription")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">{t("stages.name")} *</Label>
              <Input
                id="edit-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label>{t("stages.phase.label")}</Label>
              <PhasePicker value={phase} onChange={setPhase} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-order">{t("stages.order")} *</Label>
              <Input
                id="edit-order"
                type="number"
                min="0"
                value={order}
                onChange={(e) => setOrder(e.target.value)}
                required
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
              disabled={updateMutation.isPending || !name || !order}
            >
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t("common.loading")}
                </>
              ) : (
                t("common.save")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
