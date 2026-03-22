import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  Plus,
  FileText,
  Calendar,
  ArrowUp,
  ArrowDown,
  Import,
  ChevronDown,
  Upload,
  PenTool,
} from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { ListPageSkeleton } from "@/shared/ui/PageSkeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import { CreateResumeModal } from "@/features/resumes/modals/CreateResumeModal";
import { EditResumeModal } from "@/features/resumes/modals/EditResumeModal";
import { RenameBuilderResumeModal } from "@/features/resumes/modals/RenameBuilderResumeModal";
import { DeleteResumeModal } from "@/features/resumes/modals/DeleteResumeModal";
import { ImportResumeModal } from "@/features/resume-builder/components/ImportResumeModal";
import { UploadedResumeCard } from "@/features/resumes/components/UploadedResumeCard";
import { BuilderResumeCard } from "@/features/resumes/components/BuilderResumeCard";
import {
  useUnifiedResumes,
  type ResumeKindFilter,
  type UnifiedSortBy,
} from "@/features/resumes/hooks/useUnifiedResumes";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { UpgradeBanner } from "@/features/subscription/components/UpgradeBanner";
import { ApiError } from "@/services/api";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/shared/ui/Dialog";
import type { ResumeDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

const FILTER_OPTIONS: { value: ResumeKindFilter; labelKey: string }[] = [
  { value: "all", labelKey: "resumes.filterAll" },
  { value: "uploaded", labelKey: "resumes.filterUploaded" },
  { value: "built", labelKey: "resumes.filterBuilt" },
];

const SORT_OPTIONS: { value: UnifiedSortBy; labelKey: string }[] = [
  { value: "updated_at", labelKey: "resumes.sortLastModified" },
  { value: "created_at", labelKey: "resumes.sortCreatedDate" },
  { value: "title", labelKey: "resumes.sortTitle" },
];

export default function Resumes() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  usePageMeta({ titleKey: "resumes.title", noindex: true });

  const { canCreate } = useSubscription();
  const uploadLimitReached = !canCreate("resumes");
  const builderLimitReached = !canCreate("resume_builders");

  const {
    items,
    isLoading,
    isError,
    error,
    refetch,
    kindFilter,
    setKindFilter,
    sortBy,
    sortDir,
    toggleSort,
  } = useUnifiedResumes();

  // Modals state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);
  const [isCreateDropdownOpen, setIsCreateDropdownOpen] = useState(false);
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [editingResume, setEditingResume] = useState<ResumeDTO | null>(null);
  const [deletingResume, setDeletingResume] = useState<ResumeDTO | null>(null);
  const [deleteBuilderTarget, setDeleteBuilderTarget] =
    useState<ResumeBuilderDTO | null>(null);
  const [renamingBuilder, setRenamingBuilder] =
    useState<ResumeBuilderDTO | null>(null);

  // Close dropdowns on outside click
  useEffect(() => {
    if (!openMenuId && !isCreateDropdownOpen) return;
    const handleClick = () => {
      setOpenMenuId(null);
      setIsCreateDropdownOpen(false);
    };
    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, [openMenuId, isCreateDropdownOpen]);

  // Builder mutations
  const createBuilderMutation = useMutation({
    mutationFn: () => resumeBuilderService.create({}),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["resume-builders"] });
      navigate(`/app/resume-builder/${data.id}`);
    },
    onError: (err) => {
      if (err instanceof ApiError && err.code === "PLAN_LIMIT_REACHED") {
        queryClient.invalidateQueries({ queryKey: ["subscription"] });
        return;
      }
      showErrorNotification(
        err instanceof Error ? err.message : t("common.error"),
      );
    },
  });

  const duplicateBuilderMutation = useMutation({
    mutationFn: (id: string) => resumeBuilderService.duplicate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resume-builders"] });
      showSuccessNotification(t("resumeBuilder.duplicated"));
    },
    onError: (err) => {
      if (err instanceof ApiError && err.code === "PLAN_LIMIT_REACHED") {
        queryClient.invalidateQueries({ queryKey: ["subscription"] });
        return;
      }
      showErrorNotification(
        err instanceof Error ? err.message : t("common.error"),
      );
    },
  });

  const deleteBuilderMutation = useMutation({
    mutationFn: (id: string) => resumeBuilderService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["resume-builders"] });
      setDeleteBuilderTarget(null);
      showSuccessNotification(t("resumeBuilder.deleted"));
    },
    onError: (err) => {
      showErrorNotification(
        err instanceof Error ? err.message : t("common.error"),
      );
    },
  });

  if (isLoading) {
    return <ListPageSkeleton cards={6} />;
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t("resumes.title")}</h1>
        <ErrorState
          message={error instanceof Error ? error.message : t("common.error")}
          onRetry={refetch}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-3xl font-bold">{t("resumes.title")}</h1>

        {/* Create dropdown */}
        <div className="relative" onClick={(e) => e.stopPropagation()}>
          <Button
            onClick={() => {
              setOpenMenuId(null);
              setIsCreateDropdownOpen((prev) => !prev);
            }}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("resumes.create")}
            <ChevronDown className="h-4 w-4 ml-1" />
          </Button>
          {isCreateDropdownOpen && (
            <div className="absolute right-0 mt-1 w-56 bg-popover border rounded-md shadow-lg z-10">
              <button
                onClick={() => {
                  setIsCreateDropdownOpen(false);
                  setIsCreateModalOpen(true);
                }}
                disabled={uploadLimitReached}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Upload className="h-4 w-4" />
                {t("resumes.uploadResume")}
              </button>
              <button
                onClick={() => {
                  setIsCreateDropdownOpen(false);
                  createBuilderMutation.mutate();
                }}
                disabled={
                  builderLimitReached || createBuilderMutation.isPending
                }
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <PenTool className="h-4 w-4" />
                {t("resumes.buildResume")}
              </button>
              <button
                onClick={() => {
                  setIsCreateDropdownOpen(false);
                  setShowImportModal(true);
                }}
                disabled={builderLimitReached}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Import className="h-4 w-4" />
                {t("resumes.importResume")}
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Upgrade banners */}
      {uploadLimitReached && <UpgradeBanner resource="resumes" />}
      {builderLimitReached && <UpgradeBanner resource="resume_builders" />}

      {items.length === 0 && kindFilter === "all" ? (
        <EmptyState
          icon={<FileText className="h-12 w-12" />}
          title={t("resumes.noResumes")}
          description={t("resumes.createFirst")}
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t("resumes.create")}
            </Button>
          }
        />
      ) : (
        <>
          {/* Filter & Sort Controls */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* Filter tabs */}
            <div className="flex items-center gap-1">
              {FILTER_OPTIONS.map((opt) => (
                <Button
                  key={opt.value}
                  variant={kindFilter === opt.value ? "default" : "outline"}
                  size="sm"
                  onClick={() => setKindFilter(opt.value)}
                >
                  {t(opt.labelKey)}
                </Button>
              ))}
            </div>

            {/* Sort buttons */}
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm text-muted-foreground">
                {t("common.sortBy")}
              </span>
              {SORT_OPTIONS.map((opt) => (
                <Button
                  key={opt.value}
                  variant={sortBy === opt.value ? "default" : "outline"}
                  size="sm"
                  onClick={() => toggleSort(opt.value)}
                >
                  {opt.value === "created_at" && (
                    <Calendar className="h-3 w-3 mr-1" />
                  )}
                  {opt.value === "title" && (
                    <FileText className="h-3 w-3 mr-1" />
                  )}
                  {t(opt.labelKey)}
                  {sortBy === opt.value &&
                    (sortDir === "desc" ? (
                      <ArrowDown className="h-3 w-3 ml-1" />
                    ) : (
                      <ArrowUp className="h-3 w-3 ml-1" />
                    ))}
                </Button>
              ))}
            </div>
          </div>

          {/* Cards grid */}
          {items.length === 0 ? (
            <EmptyState
              icon={<FileText className="h-12 w-12" />}
              title={t("resumes.noResumesForFilter")}
              description={t("resumes.tryOtherFilter")}
            />
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {items.map((item) =>
                item.kind === "uploaded" ? (
                  <UploadedResumeCard
                    key={`uploaded-${item.data.id}`}
                    resume={item.data}
                    isMenuOpen={openMenuId === item.data.id}
                    onToggleMenu={() => {
                      setIsCreateDropdownOpen(false);
                      setOpenMenuId(
                        openMenuId === item.data.id ? null : item.data.id,
                      );
                    }}
                    onEdit={(resume) => {
                      setEditingResume(resume);
                      setOpenMenuId(null);
                    }}
                    onDelete={(resume) => {
                      setDeletingResume(resume);
                      setOpenMenuId(null);
                    }}
                  />
                ) : (
                  <BuilderResumeCard
                    key={`built-${item.data.id}`}
                    resume={item.data}
                    limitReached={builderLimitReached}
                    onDuplicate={(id) => duplicateBuilderMutation.mutate(id)}
                    onDelete={(resume) => setDeleteBuilderTarget(resume)}
                    onRename={(resume) => setRenamingBuilder(resume)}
                  />
                ),
              )}
            </div>
          )}
        </>
      )}

      {/* Uploaded resume modals */}
      <CreateResumeModal
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
      />

      {editingResume && (
        <EditResumeModal
          open={!!editingResume}
          onOpenChange={(open) => !open && setEditingResume(null)}
          resume={editingResume}
        />
      )}

      {deletingResume && (
        <DeleteResumeModal
          open={!!deletingResume}
          onOpenChange={(open) => !open && setDeletingResume(null)}
          resume={deletingResume}
        />
      )}

      {renamingBuilder && (
        <RenameBuilderResumeModal
          open={!!renamingBuilder}
          onOpenChange={(open) => !open && setRenamingBuilder(null)}
          resume={renamingBuilder}
        />
      )}

      {/* Builder delete dialog */}
      <Dialog
        open={!!deleteBuilderTarget}
        onOpenChange={(open) => !open && setDeleteBuilderTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("resumeBuilder.deleteConfirmTitle")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("resumeBuilder.deleteConfirmDescription", {
              title: deleteBuilderTarget?.title,
            })}
          </p>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteBuilderTarget(null)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                deleteBuilderTarget &&
                deleteBuilderMutation.mutate(deleteBuilderTarget.id)
              }
              disabled={deleteBuilderMutation.isPending}
            >
              {t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Import modal */}
      <ImportResumeModal
        open={showImportModal}
        onOpenChange={setShowImportModal}
      />
    </div>
  );
}
