import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { jobsService, type ListJobsParams } from "@/services/jobsService";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { ListPageSkeleton } from "@/shared/ui/PageSkeleton";
import { EmptyState } from "@/shared/ui/EmptyState";
import { ErrorState } from "@/shared/ui/ErrorState";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Plus,
  Briefcase,
  Building2,
  Calendar,
  Clock,
  MoreVertical,
  MessageSquare,
  Archive,
  GitBranch,
  ArrowUp,
  ArrowDown,
  LayoutGrid,
  Kanban,
  Chrome,
  Trash2,
  X,
  CheckCircle,
  ChevronRight,
  Search,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { CreateJobModal } from "@/features/jobs/modals/CreateJobModal";
import { AddCommentModal } from "@/features/jobs/modals/AddCommentModal";
import { AddStageModal } from "@/features/jobs/modals/AddStageModal";
import { UpdateJobStatusModal } from "@/features/jobs/modals/UpdateJobStatusModal";
import { StatusQuickPick } from "@/features/jobs/components/StatusQuickPick";
import {
  JobKanbanBoard,
  JOBS_KANBAN_QUERY_KEY,
} from "@/features/jobs/components/JobKanbanBoard";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useDebounce } from "@/shared/hooks/useDebounce";
import type { JobDTO, JobStatus } from "@/shared/types/api";

type SortBy =
  | "last_activity"
  | "status"
  | "applied_at"
  | "created_at"
  | "title"
  | "company_name";
type SortDir = "asc" | "desc";
type ViewMode = "list" | "kanban";
/** "" => all active (non-archived), "all" => everything, otherwise a single status */
type StatusFilter = "" | "all" | JobStatus;

const EXTENSION_BANNER_KEY = "jobber-ext-banner-dismissed";
const VIEW_MODE_KEY = "apps-view-mode";
const PAGE_SIZE = 20;

const SORT_FIELDS: {
  field: SortBy;
  labelKey: string;
  icon?: "clock" | "calendar";
}[] = [
  { field: "last_activity", labelKey: "jobs.sortLastActivity", icon: "clock" },
  { field: "status", labelKey: "jobs.sortStatus" },
  { field: "applied_at", labelKey: "jobs.sortAppliedDate", icon: "calendar" },
  { field: "created_at", labelKey: "jobs.sortCreatedDate", icon: "calendar" },
  { field: "title", labelKey: "jobs.sortJobTitle" },
  { field: "company_name", labelKey: "jobs.sortCompanyName" },
];

const STATUS_FILTER_OPTIONS: { value: StatusFilter; labelKey: string }[] = [
  { value: "", labelKey: "jobs.filterActive" },
  { value: "saved", labelKey: "jobs.statusSaved" },
  { value: "applied", labelKey: "jobs.statusApplied" },
  { value: "on_hold", labelKey: "jobs.statusOnHold" },
  { value: "offer", labelKey: "jobs.statusOffer" },
  { value: "rejected", labelKey: "jobs.statusRejected" },
  { value: "archived", labelKey: "jobs.statusArchived" },
  { value: "all", labelKey: "jobs.filterAll" },
];

function getInitialViewMode(): ViewMode {
  const stored = localStorage.getItem(VIEW_MODE_KEY);
  return stored === "kanban" ? "kanban" : "list";
}

export default function JobsPage() {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  usePageMeta({ titleKey: "jobs.title", noindex: true });
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [page, setPage] = useState(0);
  const [sortBy, setSortBy] = useState<SortBy>("last_activity");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("");
  const [searchInput, setSearchInput] = useState("");
  const debouncedSearch = useDebounce(searchInput.trim(), 300);
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [statusSubmenuId, setStatusSubmenuId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>(getInitialViewMode);
  const [activeQuickAction, setActiveQuickAction] = useState<{
    type: "comment" | "stage" | "status";
    job: JobDTO;
  } | null>(null);
  const [jobToDelete, setJobToDelete] = useState<JobDTO | null>(null);
  const queryClient = useQueryClient();
  const [showExtBanner, setShowExtBanner] = useState(
    () => localStorage.getItem(EXTENSION_BANNER_KEY) !== "true",
  );

  // Persist view mode
  useEffect(() => {
    localStorage.setItem(VIEW_MODE_KEY, viewMode);
  }, [viewMode]);

  // Close context menu when clicking outside
  useEffect(() => {
    if (!openMenuId) return;
    const handleClickOutside = () => {
      setOpenMenuId(null);
      setStatusSubmenuId(null);
    };
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [openMenuId]);

  // Kanban uses a shared constant key; list uses filter/sort-specific keys
  const queryKey =
    viewMode === "kanban"
      ? [...JOBS_KANBAN_QUERY_KEY]
      : ["jobs", "list", page, statusFilter, sortBy, sortDir, debouncedSearch];

  const listParams: ListJobsParams =
    viewMode === "kanban"
      ? { limit: 500, offset: 0, status: "all" }
      : {
          limit: PAGE_SIZE,
          offset: page * PAGE_SIZE,
          status: statusFilter || undefined,
          sort: `${sortBy}:${sortDir}`,
          search: debouncedSearch || undefined,
        };

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey,
    queryFn: () => jobsService.list(listParams),
    staleTime: 30_000,
  });

  const handleQuickAction = (
    type: "comment" | "stage" | "status",
    job: JobDTO,
  ) => {
    setActiveQuickAction({ type, job });
    setOpenMenuId(null);
  };

  const handleDelete = (job: JobDTO) => {
    setJobToDelete(job);
    setOpenMenuId(null);
  };

  const deleteMutation = useMutation({
    mutationFn: (id: string) => jobsService.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.deleteSuccess"));
      setJobToDelete(null);
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.deleteError"));
    },
  });

  // One-click status change from the card context menu (no modal)
  const quickStatusMutation = useMutation({
    mutationFn: ({ job, status }: { job: JobDTO; status: JobStatus }) =>
      jobsService.update(job.id, { status }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.statusUpdateSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.statusUpdateError"));
    },
  });

  // Completes the job's current stage straight from the context menu
  const completeStageMutation = useMutation({
    mutationFn: (job: JobDTO) =>
      jobsService.updateStage(job.id, job.current_stage_id!, {
        status: "completed",
        completed_at: new Date().toISOString(),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.stageCompletedSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.stageStatusUpdateError"));
    },
  });

  const handleStatusSelect = (job: JobDTO, status: JobStatus) => {
    setOpenMenuId(null);
    quickStatusMutation.mutate({ job, status });
  };

  const handleCompleteStage = (job: JobDTO) => {
    setOpenMenuId(null);
    if (job.current_stage_id) {
      completeStageMutation.mutate(job);
    }
  };

  const toggleSort = (field: SortBy) => {
    if (sortBy === field) {
      setSortDir(sortDir === "desc" ? "asc" : "desc");
    } else {
      setSortBy(field);
      setSortDir("desc");
    }
    setPage(0);
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
  const pagination = data?.pagination;
  // An active search is also a "filter": without this, a search with no matches
  // falls through to the first-run onboarding empty state (and hides the search
  // box entirely), instead of showing a "no results" message.
  const isFiltered = statusFilter !== "" || debouncedSearch !== "";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2 sm:space-y-0">
        {/* Row 1: title + create btn (visible on mobile, create btn hidden on sm+) */}
        <div className="flex items-center justify-between">
          <h1 className="text-3xl font-bold">{t("jobs.title")}</h1>
          <Button
            className="sm:hidden"
            size="sm"
            onClick={() => setIsCreateModalOpen(true)}
          >
            <Plus className="h-4 w-4" />
            {t("jobs.create")}
          </Button>
        </div>
        {/* Row 2 on mobile / inline on desktop: view toggle + create btn */}
        <div className="flex items-center justify-between sm:justify-end gap-2">
          {/* View Toggle */}
          <div className="flex items-center rounded-lg border bg-muted p-0.5">
            <button
              onClick={() => setViewMode("list")}
              className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                viewMode === "list"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <LayoutGrid className="h-4 w-4" />
              {t("jobs.viewList")}
            </button>
            <button
              onClick={() => setViewMode("kanban")}
              className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                viewMode === "kanban"
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              <Kanban className="h-4 w-4" />
              {t("jobs.viewBoard")}
            </button>
          </div>

          <Button
            className="hidden sm:flex"
            onClick={() => setIsCreateModalOpen(true)}
          >
            <Plus className="h-4 w-4" />
            {t("jobs.create")}
          </Button>
        </div>
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

      {jobs.length === 0 && !isFiltered && viewMode === "list" ? (
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
      ) : viewMode === "kanban" ? (
        <JobKanbanBoard
          jobs={jobs}
          onAddComment={(job) => handleQuickAction("comment", job)}
          onAddStage={(job) => handleQuickAction("stage", job)}
          onChangeStatus={(job) => handleQuickAction("status", job)}
          onStatusSelect={handleStatusSelect}
          onCompleteStage={handleCompleteStage}
          onDelete={(job) => handleDelete(job)}
        />
      ) : (
        <>
          {/* Filter + Sorting Controls */}
          <div className="flex items-center gap-x-4 gap-y-2 flex-wrap">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                value={searchInput}
                onChange={(e) => {
                  setSearchInput(e.target.value);
                  setPage(0);
                }}
                placeholder={t("jobs.searchPlaceholder")}
                aria-label={t("jobs.searchPlaceholder")}
                className="flex h-9 w-56 rounded-md border border-input bg-background pl-8 pr-3 py-1 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
            <div className="flex items-center gap-2">
              <label
                htmlFor="status-filter"
                className="text-sm text-muted-foreground"
              >
                {t("jobs.filterLabel")}
              </label>
              <select
                id="status-filter"
                value={statusFilter}
                onChange={(e) => {
                  setStatusFilter(e.target.value as StatusFilter);
                  setPage(0);
                }}
                className="flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {STATUS_FILTER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {t(opt.labelKey)}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-sm text-muted-foreground">
                {t("jobs.sortBy")}
              </span>
              {SORT_FIELDS.map(({ field, labelKey, icon }) => (
                <Button
                  key={field}
                  variant={sortBy === field ? "default" : "outline"}
                  size="sm"
                  onClick={() => toggleSort(field)}
                >
                  {icon === "clock" && <Clock className="h-3 w-3 mr-1" />}
                  {icon === "calendar" && <Calendar className="h-3 w-3 mr-1" />}
                  {t(labelKey)}
                  {sortBy === field &&
                    (sortDir === "desc" ? (
                      <ArrowDown className="h-3 w-3 ml-1" />
                    ) : (
                      <ArrowUp className="h-3 w-3 ml-1" />
                    ))}
                </Button>
              ))}
            </div>
          </div>

          {jobs.length === 0 ? (
            <EmptyState
              icon={
                debouncedSearch ? (
                  <Search className="h-12 w-12" />
                ) : (
                  <Briefcase className="h-12 w-12" />
                )
              }
              title={
                debouncedSearch
                  ? t("jobs.noSearchResults", { query: debouncedSearch })
                  : t("jobs.noJobsForFilter")
              }
              description={
                debouncedSearch
                  ? t("jobs.noSearchResultsHint")
                  : t("jobs.tryAnotherFilter")
              }
              action={
                debouncedSearch ? (
                  <Button
                    variant="outline"
                    onClick={() => {
                      setSearchInput("");
                      setPage(0);
                    }}
                  >
                    {t("jobs.clearSearch")}
                  </Button>
                ) : undefined
              }
            />
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {jobs.map((job) => (
                <div key={job.id} className="relative">
                  <Link to={`/app/jobs/${job.id}`}>
                    <Card className="transition-all hover:shadow-md h-full group">
                      <CardHeader className="pb-3">
                        <div className="flex items-start justify-between gap-2">
                          <CardTitle className="text-xl font-bold leading-tight mb-2 flex-1">
                            {job.title}
                          </CardTitle>
                          <div
                            className="relative"
                            onClick={(e) => e.preventDefault()}
                          >
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                e.preventDefault();
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
                              <div className="absolute right-0 mt-1 w-48 bg-popover border rounded-md shadow-lg z-10">
                                <button
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    handleQuickAction("comment", job);
                                  }}
                                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                                >
                                  <MessageSquare className="h-4 w-4" />
                                  {t("jobs.addComment")}
                                </button>
                                <button
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    handleQuickAction("stage", job);
                                  }}
                                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                                >
                                  <GitBranch className="h-4 w-4" />
                                  {t("jobs.addStage")}
                                </button>
                                {job.current_stage_id && (
                                  <button
                                    onClick={(e) => {
                                      e.preventDefault();
                                      e.stopPropagation();
                                      handleCompleteStage(job);
                                    }}
                                    className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                                  >
                                    <CheckCircle className="h-4 w-4 flex-shrink-0" />
                                    <span className="truncate">
                                      {t("jobs.completeCurrentStage")}
                                    </span>
                                  </button>
                                )}
                                <button
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    setStatusSubmenuId(
                                      statusSubmenuId === job.id
                                        ? null
                                        : job.id,
                                    );
                                  }}
                                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                                >
                                  <Archive className="h-4 w-4" />
                                  {t("jobs.changeStatus")}
                                  <ChevronRight
                                    className={`h-4 w-4 ml-auto transition-transform ${statusSubmenuId === job.id ? "rotate-90" : ""}`}
                                  />
                                </button>
                                {statusSubmenuId === job.id && (
                                  <div
                                    className="border-t bg-muted/40"
                                    onClick={(e) => {
                                      e.preventDefault();
                                      e.stopPropagation();
                                    }}
                                  >
                                    <StatusQuickPick
                                      currentStatus={job.status}
                                      onSelect={(status) => {
                                        setStatusSubmenuId(null);
                                        handleStatusSelect(job, status);
                                      }}
                                    />
                                  </div>
                                )}
                                <div
                                  className="my-1 border-t"
                                  role="separator"
                                />
                                <button
                                  onClick={(e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    handleDelete(job);
                                  }}
                                  className="flex items-center gap-2 w-full px-3 py-2 text-sm text-destructive hover:bg-destructive/10 text-left"
                                >
                                  <Trash2 className="h-4 w-4" />
                                  {t("jobs.delete")}
                                </button>
                              </div>
                            )}
                          </div>
                        </div>

                        {/* Secondary: Company and Stage */}
                        <div className="space-y-1">
                          {job.company_name && (
                            <div className="flex items-center gap-2 text-base font-medium text-foreground">
                              <Building2 className="h-4 w-4 text-muted-foreground" />
                              <span>{job.company_name}</span>
                            </div>
                          )}
                          {job.current_stage_name && (
                            <div className="flex items-center gap-2 text-base text-foreground">
                              <GitBranch className="h-4 w-4 text-muted-foreground" />
                              <span>{job.current_stage_name}</span>
                            </div>
                          )}
                        </div>
                      </CardHeader>

                      <CardContent className="space-y-3 pt-0">
                        {/* Status - Dominant Signal */}
                        <div className="flex items-center justify-between">
                          <StatusBadge status={job.status} size="lg" />
                        </div>

                        {/* Meta Information */}
                        <div className="space-y-2 text-sm text-muted-foreground border-t pt-3">
                          {job.resume?.name && (
                            <div className="flex items-center gap-2">
                              <span className="font-medium">
                                {t("jobs.resumeLabel")}:
                              </span>
                              <span>{job.resume.name}</span>
                            </div>
                          )}
                          <div className="flex items-center gap-2">
                            <Calendar className="h-3.5 w-3.5" />
                            <span>
                              {job.applied_at
                                ? `${t("jobs.applied")} ${formatDistanceToNow(
                                    new Date(job.applied_at),
                                    { addSuffix: true, locale: dateLocale },
                                  )}`
                                : `${t("jobs.createdDate")} ${formatDistanceToNow(
                                    new Date(job.created_at),
                                    { addSuffix: true, locale: dateLocale },
                                  )}`}
                            </span>
                          </div>
                          {job.last_activity_at && (
                            <div className="flex items-center gap-2">
                              <Clock className="h-3.5 w-3.5" />
                              <span>
                                {t("jobs.updated")}{" "}
                                {formatDistanceToNow(
                                  new Date(job.last_activity_at),
                                  {
                                    addSuffix: true,
                                    locale: dateLocale,
                                  },
                                )}
                              </span>
                            </div>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  </Link>
                </div>
              ))}
            </div>
          )}

          {/* Pagination */}
          {pagination && pagination.total > PAGE_SIZE && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                {t("common.previous")}
              </Button>
              <span className="flex items-center px-4 text-sm text-muted-foreground">
                {t("jobs.pageOf", {
                  page: page + 1,
                  total: Math.ceil(pagination.total / PAGE_SIZE),
                })}
              </span>
              <Button
                variant="outline"
                onClick={() => setPage((p) => p + 1)}
                disabled={(page + 1) * PAGE_SIZE >= pagination.total}
              >
                {t("common.next")}
              </Button>
            </div>
          )}
        </>
      )}

      {/* Modals */}
      <CreateJobModal
        open={isCreateModalOpen}
        onOpenChange={setIsCreateModalOpen}
      />

      {activeQuickAction?.type === "comment" && (
        <AddCommentModal
          open={true}
          onOpenChange={(open) => !open && setActiveQuickAction(null)}
          jobId={activeQuickAction.job.id}
        />
      )}

      {activeQuickAction?.type === "stage" && (
        <AddStageModal
          open={true}
          onOpenChange={(open) => !open && setActiveQuickAction(null)}
          jobId={activeQuickAction.job.id}
        />
      )}

      {activeQuickAction?.type === "status" && (
        <UpdateJobStatusModal
          open={true}
          onOpenChange={(open) => {
            if (!open) {
              setActiveQuickAction(null);
            }
          }}
          jobId={activeQuickAction.job.id}
          currentStatus={activeQuickAction.job.status}
        />
      )}

      {/* Delete confirmation */}
      <Dialog
        open={jobToDelete !== null}
        onOpenChange={(open) => !open && setJobToDelete(null)}
      >
        <DialogContent onClose={() => setJobToDelete(null)}>
          <DialogHeader>
            <DialogTitle>{t("jobs.delete")}</DialogTitle>
            <DialogDescription>{t("jobs.deleteConfirm")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setJobToDelete(null)}
              disabled={deleteMutation.isPending}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                jobToDelete && deleteMutation.mutate(jobToDelete.id)
              }
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending
                ? t("common.deleting")
                : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
