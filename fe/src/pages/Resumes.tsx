import { useState, useEffect } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { resumesService } from "@/services/resumesService";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { SkeletonList } from "@/shared/ui/Skeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  Plus,
  FileText,
  ExternalLink,
  CheckCircle,
  XCircle,
  Calendar,
  MoreVertical,
  Edit,
  Trash2,
  ArrowUp,
  ArrowDown,
  Briefcase,
  Download,
  Cloud,
  Link as LinkIcon,
} from "lucide-react";
import { format } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { CreateResumeModal } from "@/features/resumes/modals/CreateResumeModal";
import { EditResumeModal } from "@/features/resumes/modals/EditResumeModal";
import { DeleteResumeModal } from "@/features/resumes/modals/DeleteResumeModal";
import { showErrorNotification } from "@/shared/lib/notifications";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import type { ResumeDTO } from "@/shared/types/api";

type SortBy = "created_at" | "title" | "is_active";
type SortDir = "asc" | "desc";

export default function Resumes() {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  usePageMeta({ titleKey: "resumes.title", noindex: true });
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [sortBy, setSortBy] = useState<SortBy>("created_at");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [editingResume, setEditingResume] = useState<ResumeDTO | null>(null);
  const [deletingResume, setDeletingResume] = useState<ResumeDTO | null>(null);

  // Close context menu when clicking outside
  useEffect(() => {
    if (!openMenuId) return;
    const handleClickOutside = () => setOpenMenuId(null);
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [openMenuId]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["resumes", sortBy, sortDir],
    queryFn: () =>
      resumesService.list({
        limit: 100,
        offset: 0,
        sort_by: sortBy,
        sort_dir: sortDir,
      }),
  });

  const toggleSort = (field: SortBy) => {
    if (sortBy === field) {
      setSortDir(sortDir === "desc" ? "asc" : "desc");
    } else {
      setSortBy(field);
      setSortDir("desc");
    }
  };

  const handleEdit = (resume: ResumeDTO) => {
    setEditingResume(resume);
    setOpenMenuId(null);
  };

  const handleDelete = (resume: ResumeDTO) => {
    setDeletingResume(resume);
    setOpenMenuId(null);
  };

  // Handle S3 resume download
  const downloadMutation = useMutation({
    mutationFn: resumesService.generateDownloadURL,
    onSuccess: (data) => {
      // Open download URL in new tab
      window.open(data.download_url, "_blank");
    },
    onError: (error: Error) => {
      showErrorNotification(error?.message || t("resumes.downloadError"));
    },
  });

  const handleDownload = (resumeId: string) => {
    downloadMutation.mutate(resumeId);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t("resumes.title")}</h1>
        </div>
        <SkeletonList count={3} />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t("resumes.title")}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const resumes = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t("resumes.title")}</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("resumes.create")}
        </Button>
      </div>

      {resumes.length === 0 ? (
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
          {/* Sorting Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">
              {t("common.sortBy")}
            </span>
            <Button
              variant={sortBy === "created_at" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("created_at")}
            >
              <Calendar className="h-3 w-3 mr-1" />
              {t("resumes.sortCreatedDate")}
              {sortBy === "created_at" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
            <Button
              variant={sortBy === "title" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("title")}
            >
              <FileText className="h-3 w-3 mr-1" />
              {t("resumes.sortTitle")}
              {sortBy === "title" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
            <Button
              variant={sortBy === "is_active" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("is_active")}
            >
              {t("resumes.sortActiveStatus")}
              {sortBy === "is_active" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
          </div>

          {/* Resume Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {resumes.map((resume) => (
              <Card
                key={resume.id}
                className={`transition-all hover:shadow-md h-full group relative ${
                  !resume.is_active ? "opacity-60" : ""
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-lg font-bold leading-tight flex-1">
                      {resume.title}
                    </CardTitle>
                    {/* Context Menu */}
                    <div
                      className="relative"
                      onClick={(e) => e.preventDefault()}
                    >
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          e.preventDefault();
                          setOpenMenuId(
                            openMenuId === resume.id ? null : resume.id,
                          );
                        }}
                        className="p-1 rounded-md hover:bg-accent transition-colors text-muted-foreground"
                        aria-label={t("resumes.actionsMenu")}
                      >
                        <MoreVertical className="h-4 w-4" />
                      </button>
                      {openMenuId === resume.id && (
                        <div className="absolute right-0 mt-1 w-48 bg-popover border rounded-md shadow-lg z-10">
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              handleEdit(resume);
                            }}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <Edit className="h-4 w-4" />
                            {t("common.edit")}
                          </button>
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              handleDelete(resume);
                            }}
                            disabled={resume.can_delete === false}
                            className={`flex items-center gap-2 w-full px-3 py-2 text-sm text-left ${
                              resume.can_delete !== false
                                ? "hover:bg-accent text-destructive"
                                : "opacity-50 cursor-not-allowed"
                            }`}
                            title={
                              resume.can_delete === false
                                ? t("resumes.cannotDeleteInUse")
                                : ""
                            }
                          >
                            <Trash2 className="h-4 w-4" />
                            {t("common.delete")}
                          </button>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Active/Inactive Badge */}
                  <div className="flex items-center gap-2 mt-2">
                    {resume.is_active ? (
                      <div className="flex items-center gap-1 text-xs font-medium text-green-600 bg-green-50 dark:bg-green-900/30 dark:text-green-400 px-2 py-1 rounded">
                        <CheckCircle className="h-3 w-3" />
                        {t("common.active")}
                      </div>
                    ) : (
                      <div className="flex items-center gap-1 text-xs font-medium text-muted-foreground bg-muted px-2 py-1 rounded">
                        <XCircle className="h-3 w-3" />
                        {t("common.inactive")}
                      </div>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  {/* File Access Section */}
                  <div className="space-y-2">
                    {resume.storage_type === "s3" ? (
                      <>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <Cloud className="h-3 w-3" />
                          <span>{t("resumes.cloudStorage")}</span>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDownload(resume.id)}
                          disabled={downloadMutation.isPending}
                          className="w-full"
                        >
                          <Download className="h-3 w-3 mr-2" />
                          {downloadMutation.isPending
                            ? t("resumes.generatingLink")
                            : t("resumes.downloadResume")}
                        </Button>
                      </>
                    ) : resume.file_url ? (
                      <>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <LinkIcon className="h-3 w-3" />
                          <span>{t("resumes.externalUrl")}</span>
                        </div>
                        <a
                          href={resume.file_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center justify-center gap-2 text-sm hover:underline w-full px-4 py-2 border rounded-md hover:bg-accent transition-colors"
                        >
                          <ExternalLink className="h-3 w-3" />
                          {t("resumes.viewResume")}
                        </a>
                      </>
                    ) : (
                      <div className="text-sm text-muted-foreground italic">
                        {t("resumes.noFileAttached")}
                      </div>
                    )}
                  </div>

                  {/* Usage Indicator */}
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Briefcase className="h-4 w-4" />
                    <span>
                      {(resume.applications_count ?? 0) === 0
                        ? t("resumes.notUsedYet")
                        : t("resumes.usedInApplications", {
                            count: resume.applications_count,
                          })}
                    </span>
                  </div>

                  <div className="text-sm text-muted-foreground">
                    {t("resumes.created")}{" "}
                    {format(new Date(resume.created_at), "PPP", {
                      locale: dateLocale,
                    })}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

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
    </div>
  );
}
