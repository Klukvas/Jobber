import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { formatDistanceToNow } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { jobsService } from "@/services/jobsService";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { ListPageSkeleton } from "@/shared/ui/PageSkeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import {
  Plus,
  Briefcase,
  ExternalLink,
  Building2,
  MoreVertical,
  Edit,
  Archive,
  Calendar,
  FileText,
  ArrowUp,
  ArrowDown,
  Chrome,
  X,
  Heart,
} from "lucide-react";
import { CreateJobModal } from "@/features/jobs/modals/CreateJobModal";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import type { JobDTO } from "@/shared/types/api";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";

type SortField = "created_at" | "title" | "company_name";
type SortDir = "asc" | "desc";

const EXTENSION_BANNER_KEY = "jobber-ext-banner-dismissed";

export default function Jobs() {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const navigate = useNavigate();
  usePageMeta({ titleKey: "jobs.title", noindex: true });
  const queryClient = useQueryClient();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState<JobDTO | undefined>(undefined);
  const [sortField, setSortField] = useState<SortField>("created_at");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [showExtBanner, setShowExtBanner] = useState(
    () => localStorage.getItem(EXTENSION_BANNER_KEY) !== "true",
  );

  // Close context menu when clicking outside
  useEffect(() => {
    if (!openMenuId) return;
    const handleClickOutside = () => setOpenMenuId(null);
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [openMenuId]);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["jobs", sortField, sortDir],
    queryFn: () =>
      jobsService.list({
        limit: 100,
        offset: 0,
        status: "active",
        sort: `${sortField}:${sortDir}`,
      }),
    staleTime: 30_000,
  });

  const archiveMutation = useMutation({
    mutationFn: jobsService.archive,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.archiveSuccess"));
    },
    onError: () => {
      showErrorNotification(t("jobs.archiveError"));
    },
  });

  const toggleFavoriteMutation = useMutation({
    mutationFn: jobsService.toggleFavorite,
    onMutate: async (jobId) => {
      const queryKey = ["jobs", sortField, sortDir];
      await queryClient.cancelQueries({ queryKey: ["jobs"] });
      const previous = queryClient.getQueryData(queryKey);
      queryClient.setQueryData(queryKey, (old: typeof data) => {
        if (!old) return old;
        return {
          ...old,
          items: old.items.map((j: JobDTO) =>
            j.id === jobId ? { ...j, is_favorite: !j.is_favorite } : j,
          ),
        };
      });
      return { previous, queryKey };
    },
    onError: (err, _jobId, context) => {
      if (context?.previous && context?.queryKey) {
        queryClient.setQueryData(context.queryKey, context.previous);
      }
      console.error("[Jobs] toggleFavorite failed:", err);
      showErrorNotification(t("common.error"));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
  });

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === "desc" ? "asc" : "desc");
    } else {
      setSortField(field);
      setSortDir("desc");
    }
  };

  const handleEdit = (job: JobDTO) => {
    setEditingJob(job);
    setIsCreateModalOpen(true);
    setOpenMenuId(null);
  };

  const handleArchive = (jobId: string) => {
    archiveMutation.mutate(jobId);
    setOpenMenuId(null);
  };

  const handleModalClose = () => {
    setIsCreateModalOpen(false);
    setEditingJob(undefined);
  };

  if (isLoading) {
    return <ListPageSkeleton cards={6} />;
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t("jobs.title")}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const jobs = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">{t("jobs.title")}</h1>
        <Button onClick={() => setIsCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          {t("jobs.create")}
        </Button>
      </div>

      {showExtBanner && (
        <div className="flex flex-wrap items-center gap-3 rounded-lg border border-cyan-200 bg-cyan-50 p-3 dark:border-cyan-800 dark:bg-cyan-950">
          <Chrome className="h-5 w-5 shrink-0 text-cyan-600 dark:text-cyan-400" />
          <p className="flex-1 text-sm text-cyan-800 dark:text-cyan-200">
            {t("jobs.extensionBanner")}
          </p>
          <a
            href="https://chromewebstore.google.com/detail/jobber-smart-job-saver/koegfmmcpedfgnjnohcaieecdoflmlab"
            target="_blank"
            rel="noopener noreferrer"
            className="shrink-0 rounded-md bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-700"
          >
            {t("jobs.installExtension")}
          </a>
          <button
            onClick={() => {
              setShowExtBanner(false);
              localStorage.setItem(EXTENSION_BANNER_KEY, "true");
            }}
            className="shrink-0 rounded-md p-1 text-cyan-600 transition-colors hover:bg-cyan-100 dark:text-cyan-400 dark:hover:bg-cyan-900"
            aria-label={t("common.close")}
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {jobs.length === 0 ? (
        <EmptyState
          icon={<Briefcase className="h-12 w-12" />}
          title={t("jobs.emptyTitle")}
          description={t("jobs.emptyDescription")}
          action={
            <Button onClick={() => setIsCreateModalOpen(true)}>
              <Plus className="h-4 w-4" />
              {t("jobs.createFirstJob")}
            </Button>
          }
        />
      ) : (
        <>
          {/* Sorting Controls */}
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm text-muted-foreground">
              {t("jobs.sortBy")}
            </span>
            <Button
              variant={sortField === "created_at" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("created_at")}
            >
              <Calendar className="h-3 w-3 mr-1" />
              {t("jobs.sortCreatedDate")}
              {sortField === "created_at" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
            <Button
              variant={sortField === "title" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("title")}
            >
              <FileText className="h-3 w-3 mr-1" />
              {t("jobs.sortJobTitle")}
              {sortField === "title" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
            <Button
              variant={sortField === "company_name" ? "default" : "outline"}
              size="sm"
              onClick={() => toggleSort("company_name")}
            >
              <Building2 className="h-3 w-3 mr-1" />
              {t("jobs.sortCompanyName")}
              {sortField === "company_name" &&
                (sortDir === "desc" ? (
                  <ArrowDown className="h-3 w-3 ml-1" />
                ) : (
                  <ArrowUp className="h-3 w-3 ml-1" />
                ))}
            </Button>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {jobs.map((job) => (
              <Card
                key={job.id}
                className="relative group cursor-pointer transition-all hover:shadow-md"
                onClick={() => navigate(`/app/jobs/${job.id}`)}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-lg flex-1">
                      {job.title}
                    </CardTitle>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleFavoriteMutation.mutate(job.id);
                        }}
                        disabled={toggleFavoriteMutation.isPending}
                        className="p-1 rounded-md hover:bg-accent transition-colors disabled:opacity-50"
                        aria-label={
                          job.is_favorite
                            ? t("common.removeFromFavorites")
                            : t("common.addToFavorites")
                        }
                      >
                        <Heart
                          className={`h-4 w-4 ${job.is_favorite ? "fill-red-500 text-red-500" : "text-muted-foreground"}`}
                        />
                      </button>
                      <div className="relative">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setOpenMenuId(
                              openMenuId === job.id ? null : job.id,
                            );
                          }}
                          className="p-1 rounded-md hover:bg-accent transition-colors text-muted-foreground"
                          aria-label={t("jobs.actionsMenu")}
                        >
                          <MoreVertical className="h-4 w-4" />
                        </button>
                        {openMenuId === job.id && (
                          <div className="absolute right-0 mt-1 w-40 bg-popover border rounded-md shadow-lg z-10">
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleEdit(job);
                              }}
                              className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                            >
                              <Edit className="h-4 w-4" />
                              {t("common.edit")}
                            </button>
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleArchive(job.id);
                              }}
                              className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                            >
                              <Archive className="h-4 w-4" />
                              {t("jobs.archive")}
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-2">
                  {job.company_name && (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <Building2 className="h-4 w-4" />
                      <span>{job.company_name}</span>
                    </div>
                  )}
                  {job.url &&
                    (() => {
                      try {
                        const protocol = new URL(job.url).protocol;
                        if (protocol !== "http:" && protocol !== "https:")
                          return null;
                      } catch {
                        return null;
                      }
                      return (
                        <div className="flex items-center gap-2 text-sm">
                          <a
                            href={job.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={(e) => e.stopPropagation()}
                            className="flex items-center gap-1 text-primary hover:underline"
                          >
                            {t("jobs.viewPosting")}
                            <ExternalLink className="h-3 w-3" />
                          </a>
                        </div>
                      );
                    })()}
                  {job.source && (
                    <div className="text-sm text-muted-foreground">
                      {t("jobs.source")}: {job.source}
                    </div>
                  )}
                  <div className="text-sm text-muted-foreground pt-2 border-t space-y-1">
                    <div className="flex items-center gap-2">
                      <Calendar className="h-3.5 w-3.5" />
                      <span>
                        {t("jobs.createdDate")}{" "}
                        {formatDistanceToNow(new Date(job.created_at), {
                          addSuffix: true,
                          locale: dateLocale,
                        })}
                      </span>
                    </div>
                    <div>
                      {job.applications_count > 0
                        ? t("jobs.applicationsCount", {
                            count: job.applications_count,
                          })
                        : t("jobs.noApplications")}
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

      <CreateJobModal
        open={isCreateModalOpen}
        onOpenChange={handleModalClose}
        job={editingJob}
      />
    </div>
  );
}
