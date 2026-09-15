import { useState, useEffect } from "react";
import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Link, useSearchParams } from "react-router-dom";
import { companiesService } from "@/services/companiesService";
import { jobsService, type ListJobsParams } from "@/services/jobsService";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
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
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Plus,
  Briefcase,
  Calendar,
  Clock,
  MoreVertical,
  MessageSquare,
  GitBranch,
  ArrowUp,
  ArrowDown,
  LayoutGrid,
  Kanban,
  Chrome,
  Trash2,
  X,
  Search,
  Loader2,
  Link2,
  Filter,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { CreateJobModal } from "@/features/jobs/modals/CreateJobModal";
import { AddCommentModal } from "@/features/jobs/modals/AddCommentModal";
import { AddStageModal } from "@/features/jobs/modals/AddStageModal";
import {
  JobKanbanBoard,
  JOBS_KANBAN_QUERY_KEY,
} from "@/features/jobs/components/JobKanbanBoard";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useDebounce } from "@/shared/hooks/useDebounce";
import { CompanyAvatar } from "@/shared/ui/CompanyAvatar";
import { stageColor } from "@/features/jobs/lib/stageColors";
import type { JobDTO } from "@/shared/types/api";
import type { ArchivedFilter } from "@/services/jobsService";

type SortBy =
  "last_activity" | "applied_at" | "created_at" | "title" | "company_name";
type SortDir = "asc" | "desc";
type ViewMode = "list" | "kanban";

/**
 * The 44px floor a thumb needs, on the controls this page draws at pointer
 * density: the view toggle (32px tall), the banner's install link (28) and the
 * two icon-only buttons (24x24). Applied below `sm` only — the same page on a
 * pointer keeps every one of these exactly as it was drawn.
 */
const TOUCH_ROW_TARGET = "flex items-center justify-center max-sm:min-h-11";
const TOUCH_ICON_TARGET =
  "flex h-6 w-6 items-center justify-center max-sm:h-11 max-sm:w-11";

const EXTENSION_BANNER_KEY = "jobber-ext-banner-dismissed";
const VIEW_MODE_KEY = "apps-view-mode";
const PAGE_SIZE = 20;

const SORT_FIELDS: {
  field: SortBy;
  labelKey: string;
  icon?: "clock" | "calendar";
}[] = [
  { field: "last_activity", labelKey: "jobs.sortLastActivity", icon: "clock" },
  { field: "applied_at", labelKey: "jobs.sortAppliedDate", icon: "calendar" },
  { field: "created_at", labelKey: "jobs.sortCreatedDate", icon: "calendar" },
  { field: "title", labelKey: "jobs.sortJobTitle" },
  { field: "company_name", labelKey: "jobs.sortCompanyName" },
];

// The list "status" filter now means the archived filter.
const ARCHIVED_FILTER_OPTIONS: { value: ArchivedFilter; labelKey: string }[] = [
  { value: "", labelKey: "jobs.filterActive" },
  { value: "archived", labelKey: "jobs.filterArchived" },
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
  const [archivedFilter, setArchivedFilter] = useState<ArchivedFilter>("");
  const [searchInput, setSearchInput] = useState("");
  const debouncedSearch = useDebounce(searchInput.trim(), 300);
  const [openMenuId, setOpenMenuId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>(getInitialViewMode);
  const [activeQuickAction, setActiveQuickAction] = useState<{
    type: "comment" | "stage";
    job: JobDTO;
  } | null>(null);
  const [jobToDelete, setJobToDelete] = useState<JobDTO | null>(null);
  const queryClient = useQueryClient();
  const [showExtBanner, setShowExtBanner] = useState(
    () => localStorage.getItem(EXTENSION_BANNER_KEY) !== "true",
  );

  // The company filter lives in the URL, not in component state: it arrives as
  // a deep link from the Companies page, and it has to survive a refresh and
  // be shareable. Clearing it drops the parameter again.
  const [searchParams, setSearchParams] = useSearchParams();
  const companyFilterId = searchParams.get("company_id") ?? "";

  const clearCompanyFilter = () => {
    const next = new URLSearchParams(searchParams);
    next.delete("company_id");
    setSearchParams(next, { replace: true });
    setPage(0);
  };

  // Only to name the filter in the UI — the list itself is filtered server-side.
  const { data: filteredCompany } = useQuery({
    queryKey: ["company", companyFilterId],
    queryFn: () => companiesService.getById(companyFilterId),
    enabled: !!companyFilterId,
    staleTime: 5 * 60_000,
  });

  // Persist view mode
  useEffect(() => {
    localStorage.setItem(VIEW_MODE_KEY, viewMode);
  }, [viewMode]);

  // Close context menu when clicking outside
  useEffect(() => {
    if (!openMenuId) return;
    const handleClickOutside = () => {
      setOpenMenuId(null);
    };
    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, [openMenuId]);

  // Kanban uses a shared constant key (varying only by its archived toggle);
  // list uses filter/sort-specific keys.
  const queryKey =
    viewMode === "kanban"
      ? [
          ...JOBS_KANBAN_QUERY_KEY,
          archivedFilter,
          sortBy,
          sortDir,
          debouncedSearch,
          companyFilterId,
        ]
      : [
          "jobs",
          "list",
          page,
          archivedFilter,
          sortBy,
          sortDir,
          debouncedSearch,
          companyFilterId,
        ];

  const listParams: ListJobsParams =
    viewMode === "kanban"
      ? {
          limit: 500,
          offset: 0,
          status: archivedFilter || undefined,
          sort: `${sortBy}:${sortDir}`,
          search: debouncedSearch || undefined,
          company_id: companyFilterId || undefined,
        }
      : {
          limit: PAGE_SIZE,
          offset: page * PAGE_SIZE,
          status: archivedFilter || undefined,
          sort: `${sortBy}:${sortDir}`,
          search: debouncedSearch || undefined,
          company_id: companyFilterId || undefined,
        };

  const { data, isLoading, isFetching, isError, error, refetch } = useQuery({
    queryKey,
    // React Query aborts this signal when the key changes, so a search that
    // has been superseded by the next keystroke is cancelled on the wire
    // instead of running to completion behind the one that matters.
    queryFn: ({ signal }) => jobsService.list(listParams, { signal }),
    staleTime: 30_000,
    // Keep showing the current results while a new query (search/filter/sort/
    // page) loads, so only the list updates instead of the whole page falling
    // back to the skeleton and the search input losing focus.
    placeholderData: keepPreviousData,
  });

  const handleQuickAction = (type: "comment" | "stage", job: JobDTO) => {
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
  const isFiltered =
    archivedFilter !== "" || debouncedSearch !== "" || companyFilterId !== "";

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
              className={`${TOUCH_ROW_TARGET} gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
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
              className={`${TOUCH_ROW_TARGET} gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
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
            className={`${TOUCH_ROW_TARGET} shrink-0 rounded-md bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-700`}
          >
            {t("jobs.installExtension")}
          </a>
          <button
            onClick={() => {
              setShowExtBanner(false);
              localStorage.setItem(EXTENSION_BANNER_KEY, "true");
            }}
            className={`${TOUCH_ICON_TARGET} shrink-0 rounded-md text-cyan-600 transition-colors hover:bg-cyan-100 dark:text-cyan-400 dark:hover:bg-cyan-900`}
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
      ) : (
        <>
          {/* An active company filter has to be visible and reversible — a
              silently filtered list looks like missing data. */}
          {companyFilterId && (
            <div
              role="status"
              className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/50 px-3 py-2 text-sm"
            >
              <Filter className="h-4 w-4 shrink-0 text-muted-foreground" />
              <span>
                {t("jobs.filteredByCompany", {
                  company:
                    filteredCompany?.name ?? t("jobs.filterCompanyFallback"),
                })}
              </span>
              {/* The only way out of a filtered list, and it was 28px tall on
                  a phone — the same 44px floor as every other row control
                  here, bought from the shared constant. */}
              <button
                type="button"
                onClick={clearCompanyFilter}
                className={`${TOUCH_ROW_TARGET} gap-1 rounded-md px-2 py-1 text-sm font-medium text-primary transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring`}
              >
                <X className="h-3.5 w-3.5" />
                {t("jobs.clearCompanyFilter")}
              </button>
            </div>
          )}

          {/* Filter + Sorting Controls — shared by list and board */}
          <div className="flex items-center gap-x-4 gap-y-2 flex-wrap">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              {/* The shared field, not a hand-rolled one: it carries the
                  44px phone height the rest of the app's inputs have. The
                  local copy hard-coded `h-9` and was 36px on a 375px screen. */}
              <Input
                type="text"
                value={searchInput}
                onChange={(e) => {
                  setSearchInput(e.target.value);
                  setPage(0);
                }}
                placeholder={t("jobs.searchPlaceholder")}
                aria-label={t("jobs.searchPlaceholder")}
                className="w-56 pl-8 pr-8"
              />
              {isFetching && (
                <Loader2
                  aria-hidden
                  className="pointer-events-none absolute right-2.5 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-muted-foreground"
                />
              )}
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
                value={archivedFilter}
                onChange={(e) => {
                  setArchivedFilter(e.target.value as ArchivedFilter);
                  setPage(0);
                }}
                // Same height scale as the search field beside it: 40px as
                // drawn, 44 below `sm` where a thumb has to hit it.
                className="flex h-10 max-sm:h-11 rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {ARCHIVED_FILTER_OPTIONS.map((opt) => (
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

          {viewMode === "kanban" ? (
            <JobKanbanBoard
              jobs={jobs}
              queryKey={queryKey}
              onAddComment={(job) => handleQuickAction("comment", job)}
              onAddStage={(job) => handleQuickAction("stage", job)}
              onDelete={(job) => handleDelete(job)}
            />
          ) : jobs.length === 0 ? (
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
            <div
              className={`grid gap-4 md:grid-cols-2 lg:grid-cols-3 transition-opacity ${
                isFetching ? "opacity-60" : ""
              }`}
            >
              {jobs.map((job) => {
                const stage = stageColor(
                  job.current_stage_name,
                  job.is_archived,
                );
                return (
                  <div key={job.id} className="relative">
                    <Link to={`/app/jobs/${job.id}`}>
                      <Card
                        className={`transition-all hover:shadow-md motion-safe:hover:-translate-y-0.5 h-full group border-l-4 ${stage.border}`}
                      >
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
                                className={`${TOUCH_ICON_TARGET} rounded-md hover:bg-accent transition-colors text-muted-foreground`}
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

                          {/* Secondary: company (with monogram) and source.
                            The stage lives in the coloured pill below, so it is
                            not repeated here. */}
                          <div className="space-y-1">
                            {job.company_name && (
                              <div className="flex items-center gap-2 text-base font-medium text-foreground">
                                <CompanyAvatar
                                  name={job.company_name}
                                  size="sm"
                                />
                                <span>{job.company_name}</span>
                              </div>
                            )}
                            {job.source && (
                              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                <Link2 className="h-4 w-4" />
                                <span>{job.source}</span>
                              </div>
                            )}
                          </div>
                        </CardHeader>

                        <CardContent className="space-y-3 pt-0">
                          {/* Pipeline column — the card's state */}
                          <div className="flex items-center justify-between">
                            <span
                              className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-medium ${stage.pill}`}
                            >
                              <span
                                className={`h-1.5 w-1.5 rounded-full ${stage.dot}`}
                                aria-hidden
                              />
                              {job.is_archived
                                ? t("jobs.archived")
                                : (job.current_stage_name ??
                                  t("jobs.board.noStage"))}
                            </span>
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
                );
              })}
            </div>
          )}

          {/* Pagination — list view only */}
          {viewMode === "list" &&
            pagination &&
            pagination.total > PAGE_SIZE && (
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

      {/* Delete confirmation */}
      <Dialog
        open={jobToDelete !== null}
        onOpenChange={(open) => !open && setJobToDelete(null)}
      >
        <DialogContent onClose={() => setJobToDelete(null)}>
          <DialogHeader>
            <DialogTitle>{t("jobs.delete")}</DialogTitle>
            {/* Naming the card is the difference between a confirmation and a
                coin toss when several are open. */}
            <DialogDescription>
              {t("jobs.deleteConfirm", {
                title: jobToDelete?.title || t("common.untitled"),
              })}
            </DialogDescription>
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
