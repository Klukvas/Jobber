import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  Archive,
  Heart,
  Loader2,
  Save,
  Trash2,
} from "lucide-react";
import { Button } from "@/shared/ui/Button";
import type { JobDTO } from "@/shared/types/api";

interface JobDetailHeaderProps {
  readonly job: JobDTO;
  readonly isDirty: boolean;
  readonly canSave: boolean;
  readonly isSaving: boolean;
  readonly onBack: () => void;
  readonly onSave: () => void;
  readonly onDiscard: () => void;
  readonly onToggleFavorite: () => void;
  readonly isTogglingFavorite: boolean;
  readonly onArchive: () => void;
  readonly isArchiving: boolean;
  readonly onDelete: () => void;
  readonly isDeleting: boolean;
}

export function JobDetailHeader({
  job,
  isDirty,
  canSave,
  isSaving,
  onBack,
  onSave,
  onDiscard,
  onToggleFavorite,
  isTogglingFavorite,
  onArchive,
  isArchiving,
  onDelete,
  isDeleting,
}: JobDetailHeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <Button variant="ghost" onClick={onBack}>
        <ArrowLeft className="h-4 w-4" />
        {t("jobs.backToJobs")}
      </Button>
      <div className="flex flex-wrap gap-2">
        {isDirty && (
          <>
            <Button variant="outline" size="sm" onClick={onDiscard}>
              {t("common.cancel")}
            </Button>
            <Button size="sm" onClick={onSave} disabled={isSaving || !canSave}>
              {isSaving ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <Save className="h-4 w-4 mr-2" />
              )}
              {t("common.save")}
            </Button>
          </>
        )}
        <Button
          variant="outline"
          size="sm"
          onClick={onToggleFavorite}
          disabled={isTogglingFavorite}
          aria-label={
            job.is_favorite
              ? t("common.removeFromFavorites")
              : t("common.addToFavorites")
          }
        >
          <Heart
            className={`h-4 w-4 ${job.is_favorite ? "fill-red-500 text-red-500" : ""}`}
          />
        </Button>
        {job.status !== "archived" && (
          <Button
            variant="outline"
            size="sm"
            onClick={onArchive}
            disabled={isArchiving}
            aria-label={t("jobs.archive")}
          >
            <Archive className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">{t("jobs.archive")}</span>
          </Button>
        )}
        <Button
          variant="outline"
          size="sm"
          className="text-destructive hover:text-destructive"
          onClick={onDelete}
          disabled={isDeleting}
          aria-label={t("common.delete")}
        >
          <Trash2 className="h-4 w-4 sm:mr-2" />
          <span className="hidden sm:inline">{t("common.delete")}</span>
        </Button>
      </div>
    </div>
  );
}
