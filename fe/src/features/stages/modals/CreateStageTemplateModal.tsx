import { useCurrentFormSession } from "@/shared/hooks/useCurrentFormSession";
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
import { Select } from "@/shared/ui/Select";
import { Loader2 } from "lucide-react";
import { ApiError } from "@/services/api";
import {
  DEFAULT_STAGE_KIND,
  STAGE_KIND_OPTIONS,
  stageKindLabelKey,
} from "@/features/stages/lib/stageKinds";
import type { StageKind } from "@/shared/types/api";

interface CreateStageTemplateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Order preselected when the modal opens (e.g. append to the end) */
  initialOrder?: number;
}

function CreateStageForm({
  onOpenChange,
  initialOrder,
}: Omit<CreateStageTemplateModalProps, "open">) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isCurrentSession = useCurrentFormSession();
  const [name, setName] = useState("");
  const [order, setOrder] = useState(
    initialOrder !== undefined ? String(initialOrder + 1) : "",
  );
  // A stage the user invents means nothing to analytics until they say so.
  const [kind, setKind] = useState<StageKind>(DEFAULT_STAGE_KIND);
  const createMutation = useMutation({
    mutationFn: stageTemplatesService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stage-templates"] });
      if (!isCurrentSession()) return;
      showSuccessNotification(t("stages.createSuccess"));
      onOpenChange(false);
      setName("");
      setOrder("");
    },
    onError: (error: Error) => {
      if (!isCurrentSession()) return;
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
    if (name.trim() && Number(order) >= 1 && !createMutation.isPending) {
      createMutation.mutate({
        name: name.trim(),
        order: Number(order) - 1,
        // An omitted kind already means "other" on the server, so the default
        // selection travels as an omission rather than as redundant noise.
        ...(kind !== DEFAULT_STAGE_KIND ? { kind } : {}),
      });
    }
  };

  return (
    <>
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
              <Label htmlFor="order">{`${t("stages.order")} *`}</Label>
              <Input
                id="order"
                type="number"
                min="1"
                value={order}
                onChange={(e) => setOrder(e.target.value)}
                placeholder={t("stages.orderPlaceholder")}
                required
              />
              <p className="text-xs text-muted-foreground">
                {t("stages.orderDescription")}
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="kind">{t("stages.kind")}</Label>
              <Select
                id="kind"
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
              disabled={createMutation.isPending || !name.trim() || !order || Number(order) < 1}
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
    </>
  );
}


export function CreateStageTemplateModal({ open, onOpenChange, initialOrder }: CreateStageTemplateModalProps) {
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent onClose={() => onOpenChange(false)}>
      <CreateStageForm onOpenChange={onOpenChange} initialOrder={initialOrder} />
    </DialogContent>
  </Dialog>;
}
