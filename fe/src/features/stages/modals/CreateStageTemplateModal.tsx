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

interface CreateStageTemplateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Phase preselected when the modal opens (e.g. from an empty group CTA) */
  initialPhase?: StagePhase;
}

export function CreateStageTemplateModal({
  open,
  onOpenChange,
  initialPhase,
}: CreateStageTemplateModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [order, setOrder] = useState("");
  const [phase, setPhase] = useState<StagePhase>(initialPhase ?? "in_progress");
  const [prevOpen, setPrevOpen] = useState(open);

  // Sync the preselected phase on each open (render-time state adjustment —
  // the set-state-in-effect lint rule forbids the effect version)
  if (open !== prevOpen) {
    setPrevOpen(open);
    if (open) {
      setPhase(initialPhase ?? "in_progress");
    }
  }

  const createMutation = useMutation({
    mutationFn: stageTemplatesService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stage-templates"] });
      showSuccessNotification(t("stages.createSuccess"));
      onOpenChange(false);
      setName("");
      setOrder("");
      setPhase("in_progress");
    },
    onError: (error: Error) => {
      if (
        error instanceof ApiError &&
        error.code === "STAGE_TEMPLATE_NAME_EXISTS"
      ) {
        showErrorNotification(t("stages.nameExists"));
        return;
      }
      showErrorNotification(error.message || t("stages.createError"));
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name && order) {
      createMutation.mutate({
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
          <DialogTitle>{t("stages.createTitle")}</DialogTitle>
          <DialogDescription>{t("stages.createDescription")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">{`${t("stages.stageName")} *`}</Label>
              <Input
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t("stages.stageNamePlaceholder")}
                required
              />
            </div>
            <div className="space-y-2">
              <Label>{t("stages.phase.label")}</Label>
              <PhasePicker value={phase} onChange={setPhase} />
              <p className="text-xs text-muted-foreground">
                {t("stages.phase.description")}
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="order">{`${t("stages.order")} *`}</Label>
              <Input
                id="order"
                type="number"
                min="0"
                value={order}
                onChange={(e) => setOrder(e.target.value)}
                placeholder={t("stages.orderPlaceholder")}
                required
              />
              <p className="text-xs text-muted-foreground">
                {t("stages.orderDescription")}
              </p>
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
              disabled={createMutation.isPending || !name || !order}
            >
              {createMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  {t("common.loading")}
                </>
              ) : (
                t("common.create")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
