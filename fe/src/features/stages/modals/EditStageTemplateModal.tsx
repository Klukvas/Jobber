import { FIELD_LIMITS } from "@/shared/lib/fieldLimits";
import { useState, useRef, useEffect } from "react";
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
import { Select } from "@/shared/ui/Select";
import { Loader2 } from "lucide-react";
import { ApiError } from "@/services/api";
import {
  STAGE_KIND_OPTIONS,
  stageKindLabelKey,
  stageKindOf,
} from "@/features/stages/lib/stageKinds";
import type {
  StageKind,
  StageTemplateDTO,
  UpdateStageTemplateRequest,
} from "@/shared/types/api";

interface EditStageTemplateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: StageTemplateDTO | null;
}

function EditStageForm({ template, onOpenChange }: {
  template: StageTemplateDTO;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    return () => { active.current = false; };
  }, []);
  const [name, setName] = useState(template.name);
  const isNameTooLong = Array.from(name.trim()).length > FIELD_LIMITS.name;
  const [order, setOrder] = useState(String(template.order + 1));
  const [kind, setKind] = useState<StageKind>(stageKindOf(template));

  const updateMutation = useMutation({
    mutationFn: (data: UpdateStageTemplateRequest) =>
      stageTemplatesService.update(template.id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stage-templates"] });
      if (!active.current) return;
      showSuccessNotification(t("stages.updateSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      if (!active.current) return;
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
    if (name.trim() && !isNameTooLong && Number.isInteger(Number(order)) && Number(order) >= 1 && !updateMutation.isPending) {
      updateMutation.mutate({
        name: name.trim(),
        order: Number(order) - 1,
        // Only an explicit reclassification travels: a rename or a reorder
        // must leave what the column *means* exactly as it was.
        ...(kind !== stageKindOf(template) ? { kind } : {}),
      });
    }
  };

  return (
    <>
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
                maxLength={FIELD_LIMITS.name * 2}
                aria-invalid={isNameTooLong}
                aria-describedby={isNameTooLong ? "edit-name-error" : undefined}
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
              {isNameTooLong && <p id="edit-name-error" role="alert" className="text-sm text-destructive">{t("stages.nameTooLong", { max: FIELD_LIMITS.name })}</p>}
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-order">{t("stages.order")} *</Label>
              <Input
                id="edit-order"
                type="number"
                min="1"
                value={order}
                onChange={(e) => setOrder(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-kind">{t("stages.kind")}</Label>
              <Select
                id="edit-kind"
                value={kind}
                onChange={(e) => setKind(e.target.value as StageKind)}
              >
                {STAGE_KIND_OPTIONS.map((option) => (
                  <option key={option} value={option}>
                    {t(stageKindLabelKey(option))}
                  </option>
                ))}
              </Select>
              <p className="text-xs text-muted-foreground">
                {t("stages.kindDescription")}
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
              disabled={updateMutation.isPending || !name.trim() || isNameTooLong || !order || !Number.isInteger(Number(order)) || Number(order) < 1}
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
    </>
  );
}


export function EditStageTemplateModal({ open, onOpenChange, template }: EditStageTemplateModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        {template && <EditStageForm key={template.id} template={template} onOpenChange={onOpenChange} />}
      </DialogContent>
    </Dialog>
  );
}
