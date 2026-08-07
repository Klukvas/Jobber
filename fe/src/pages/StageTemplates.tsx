import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { stageTemplatesService } from "@/services/stageTemplatesService";
import { Button } from "@/shared/ui/Button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/Card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { StageTemplateListSkeleton } from "@/shared/ui/PageSkeleton";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  Plus,
  Trash2,
  Edit2,
  Check,
  Sparkles,
  Loader2,
} from "lucide-react";
import { CreateStageTemplateModal } from "@/features/stages/modals/CreateStageTemplateModal";
import { EditStageTemplateModal } from "@/features/stages/modals/EditStageTemplateModal";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { showErrorNotification } from "@/shared/lib/notifications";
import { ApiError } from "@/services/api";
import type { StagePhase, StageTemplateDTO } from "@/shared/types/api";
import {
  PHASE_LABEL_KEYS,
  groupTemplatesByPhase,
} from "@/features/stages/lib/phases";

// Interview steps live in in_progress; Negotiating details the Offer phase.
// No Applied/Offer/Rejected recommendations — the unified board's base
// columns already cover those zones.
const RECOMMENDED_STAGES: {
  nameKey: string;
  order: number;
  phase: StagePhase;
}[] = [
  { nameKey: "stages.phoneScreen", order: 1, phase: "in_progress" },
  { nameKey: "stages.technicalInterview", order: 2, phase: "in_progress" },
  { nameKey: "stages.onsiteInterview", order: 3, phase: "in_progress" },
  { nameKey: "stages.hrInterview", order: 4, phase: "in_progress" },
  { nameKey: "stages.negotiating", order: 1, phase: "offer" },
];

export default function StageTemplates() {
  usePageMeta({ titleKey: "nav.stages", noindex: true });
  const { t, i18n } = useTranslation();
  const queryClient = useQueryClient();
  const dateLocale = useDateLocale();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [createPhase, setCreatePhase] = useState<StagePhase | undefined>(
    undefined,
  );
  const [editingTemplate, setEditingTemplate] =
    useState<StageTemplateDTO | null>(null);
  const [deletingTemplate, setDeletingTemplate] =
    useState<StageTemplateDTO | null>(null);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["stage-templates"],
    queryFn: () => stageTemplatesService.list({ limit: 100, offset: 0 }),
  });

  const deleteMutation = useMutation({
    mutationFn: stageTemplatesService.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stage-templates"] });
    },
    onError: (error: Error) => {
      if (error instanceof ApiError && error.code === "STAGE_TEMPLATE_IN_USE") {
        showErrorNotification(t("stages.deleteInUseError"));
      } else {
        showErrorNotification(t("stages.deleteError"));
      }
    },
  });

  const createMutation = useMutation({
    mutationFn: stageTemplatesService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["stage-templates"] });
    },
    onError: () => {
      showErrorNotification(t("stages.createError"));
    },
  });

  const handleDelete = (template: StageTemplateDTO) => {
    setDeletingTemplate(template);
  };

  const confirmDelete = () => {
    if (deletingTemplate) {
      deleteMutation.mutate(deletingTemplate.id, {
        onSettled: () => setDeletingTemplate(null),
      });
    }
  };

  const handleAddRecommended = (
    nameKey: string,
    order: number,
    phase: StagePhase,
  ) => {
    createMutation.mutate({ name: t(nameKey), order, phase });
  };

  const handleAddAllRecommended = () => {
    const stages = data?.items || [];
    const existingNames = new Set(stages.map((s) => s.name.toLowerCase()));

    RECOMMENDED_STAGES.forEach((rec) => {
      const allNames = getAllTranslations(rec.nameKey);
      const alreadyExists = allNames.some((n) => existingNames.has(n));
      if (!alreadyExists) {
        const name = t(rec.nameKey);
        createMutation.mutate({ name, order: rec.order, phase: rec.phase });
      }
    });
  };

  const getAllTranslations = (nameKey: string) => {
    const languages = ["en", "ua", "ru"];
    return languages.map((lang) => i18n.getFixedT(lang)(nameKey).toLowerCase());
  };

  const isRecommendedAdded = (nameKey: string) => {
    const stages = data?.items || [];
    const allNames = getAllTranslations(nameKey);
    return stages.some((s) => allNames.includes(s.name.toLowerCase()));
  };

  if (isLoading) {
    return <StageTemplateListSkeleton />;
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t("stages.title")}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const stages = data?.items || [];
  const phaseGroups = groupTemplatesByPhase(stages);
  const allRecommendedAdded = RECOMMENDED_STAGES.every((rec) =>
    isRecommendedAdded(rec.nameKey),
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">{t("stages.title")}</h1>
          <p className="text-muted-foreground mt-1">
            {t("stages.description")}
          </p>
        </div>
        <Button
          onClick={() => {
            setCreatePhase(undefined);
            setIsCreateModalOpen(true);
          }}
        >
          <Plus className="h-4 w-4" />
          {t("stages.create")}
        </Button>
      </div>

      {/* Recommended Stages */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-primary" />
              <CardTitle>{t("stages.recommended")}</CardTitle>
            </div>
            {!allRecommendedAdded && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleAddAllRecommended}
                disabled={createMutation.isPending}
              >
                <Plus className="h-4 w-4" />
                {t("stages.addAll")}
              </Button>
            )}
          </div>
          <CardDescription>
            {t("stages.recommendedDescription")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {RECOMMENDED_STAGES.map((rec) => {
              const added = isRecommendedAdded(rec.nameKey);
              return (
                <Button
                  key={rec.nameKey}
                  variant={added ? "secondary" : "outline"}
                  size="sm"
                  disabled={added || createMutation.isPending}
                  onClick={() =>
                    handleAddRecommended(rec.nameKey, rec.order, rec.phase)
                  }
                >
                  {added ? (
                    <Check className="h-4 w-4" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  {t(rec.nameKey)}
                </Button>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* User's Stage Templates — all four phase groups, empty ones included */}
      {(
        <div className="space-y-6">
          {phaseGroups.map((group) => (
            <div key={group.phase} className="space-y-3">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
                {t(PHASE_LABEL_KEYS[group.phase])}
              </h2>
              {group.templates.length === 0 && (
                <div className="flex items-center justify-between rounded-lg border border-dashed px-4 py-3">
                  <p className="text-sm text-muted-foreground">
                    {t("stages.phase.emptyGroup")}
                  </p>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setCreatePhase(group.phase);
                      setIsCreateModalOpen(true);
                    }}
                  >
                    <Plus className="h-4 w-4" />
                    {t("stages.create")}
                  </Button>
                </div>
              )}
              {group.templates.map((stage) => (
            <Card key={stage.id}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary">
                    {stage.order}
                  </div>
                  <CardTitle className="text-lg">{stage.name}</CardTitle>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setEditingTemplate(stage)}
                    aria-label={t("common.edit")}
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleDelete(stage)}
                    disabled={deleteMutation.isPending}
                    aria-label={t("common.delete")}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="pt-0 pb-3">
                <p className="text-sm text-muted-foreground">
                  {t("stages.created", {
                    date: format(new Date(stage.created_at), "PP", {
                      locale: dateLocale,
                    }),
                  })}
                </p>
              </CardContent>
            </Card>
              ))}
            </div>
          ))}
        </div>
      )}

      <CreateStageTemplateModal
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
        initialPhase={createPhase}
      />

      <EditStageTemplateModal
        open={editingTemplate !== null}
        onOpenChange={(open) => {
          if (!open) setEditingTemplate(null);
        }}
        template={editingTemplate}
      />

      <Dialog
        open={deletingTemplate !== null}
        onOpenChange={(open) => {
          if (!open) setDeletingTemplate(null);
        }}
      >
        <DialogContent onClose={() => setDeletingTemplate(null)}>
          <DialogHeader>
            <DialogTitle>{t("stages.deleteStage")}</DialogTitle>
            <DialogDescription>
              {t("stages.deleteConfirm", { name: deletingTemplate?.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setDeletingTemplate(null)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={confirmDelete}
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
    </div>
  );
}
