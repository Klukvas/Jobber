import { useState, useEffect } from "react";
import {
  useQuery,
  useMutation,
  useQueryClient,
  keepPreviousData,
} from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { companiesService } from "@/services/companiesService";
import { jobsService, type ListJobsParams } from "@/services/jobsService";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
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
  ArrowUp,
  ArrowDown,
  LayoutGrid,
  Kanban,
  Chrome,
  X,
  Search,
  Loader2,
  Filter,
} from "lucide-react";
import { CreateJobModal } from "@/features/jobs/modals/CreateJobModal";
import { AddCommentModal } from "@/features/jobs/modals/AddCommentModal";
import { AddStageModal } from "@/features/jobs/modals/AddStageModal";
import { JobKanbanBoard } from "@/features/jobs/components/JobKanbanBoard";
import { JobBoardLoadMore } from "@/features/jobs/components/JobBoardLoadMore";
import { JobListGrid } from "@/features/jobs/components/JobListGrid";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useJobBoardQuery } from "@/features/jobs/hooks/useJobBoardQuery";
import {
  useJobsListUrlState,
  type JobsSortBy,
} from "@/features/jobs/hooks/useJobsListUrlState";
import { useJobsListReturnState } from "@/features/jobs/lib/jobsListReturn";
import {
  TOUCH_ICON_TARGET,
  TOUCH_ROW_TARGET,
} from "@/features/jobs/lib/touchTargets";
import type { JobDTO } from "@/shared/types/api";
import type { ArchivedFilter } from "@/services/jobsService";

type ViewMode = "list" | "kanban";

const EXTENSION_BANNER_KEY = "jobber-ext-banner-dismissed";
const VIEW_MODE_KEY = "apps-view-mode";
const PAGE_SIZE = 20;

const SORT_FIELDS: {
  field: JobsSortBy;
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
  usePageMeta({ titleKey: "jobs.title", noindex: true });
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  // Page, search, sort and the two filters live in the URL: Back, Forward, a
  // refresh and a pasted link all have to restore the same list.
  const {
    page,
    search,
    searchInput,
    sortBy,
    sortDir,
    archivedFilter,
    companyFilterId,
    isFiltered,
    setPage,
    setSearchInput,
    clearSearch,
    toggleSort,
    setArchivedFilter,
    clearCompanyFilter,
  } = useJobsListUrlState();
  // Every card carries the list URL, so the detail page's Back comes back here
  // rather than to a fresh, unfiltered page 1.
  const returnState = useJobsListReturnState();
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

  const isListView = viewMode === "list";
  const sort = `${sortBy}:${sortDir}`;

  const listParams: ListJobsParams = {
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
    status: archivedFilter || undefined,
    sort,
    search: search || undefined,
    company_id: companyFilterId || undefined,
  };

  const listQuery = useQuery({
    queryKey: [
      "jobs",
      "list",
      page,
      archivedFilter,
      sortBy,
      sortDir,
      search,
      companyFilterId,
    ],
    // React Query aborts this signal when the key changes, so a search that
    // has been superseded by the next keystroke is cancelled on the wire
    // instead of running to completion behind the one that matters.
    queryFn: ({ signal }) => jobsService.list(listParams, { signal }),
    enabled: isListView,
    staleTime: 30_000,
    // Keep showing the current results while a new query (search/filter/sort/
    // page) loads, so only the list updates instead of the whole page falling
    // back to the skeleton and the search input losing focus.
    placeholderData: keepPreviousData,
  });

  // The board loads the same list in pages and says how many of them it holds.
  // It used to ask for a flat 500 and render whatever came back, so card 501
  // existed in the list view and nowhere on the board.
  const boardQuery = useJobBoardQuery({
    filters: {
      archived: archivedFilter,
      sort,
      search,
      companyId: companyFilterId,
    },
    enabled: !isListView,
  });

  const { data, isError, error, refetch } = listQuery;
  // The unfiltered active list is empty. That is the first-run state only if
  // the account has no jobs at all: archived ones count, and hiding the filter
  // bar for them made the Archived filter unreachable.
  const activeListIsEmpty =
    isListView &&
    !isFiltered &&
    page === 0 &&
    !listQuery.isPlaceholderData &&
    // Only once the list has actually answered: a loading or failed list is
    // not an empty one, and must not trigger the account-wide lookup.
    !!data &&
    // The API serialises an empty page as `items: null`, not `[]`.
    (data.items ?? []).length === 0;
  const anyJobsQuery = useQuery({
    queryKey: ["jobs", "any"],
    queryFn: ({ signal }) =>
      jobsService.list({ limit: 1, status: "all" }, { signal }),
    enabled: activeListIsEmpty,
    staleTime: 30_000,
  });
  const isFetching = isListView ? listQuery.isFetching : boardQuery.isFetching;
  const pagination = data?.pagination;
  const totalPages = pagination
    ? Math.max(1, Math.ceil(pagination.total / PAGE_SIZE))
    : undefined;
  // Deleting the last card on the last page — or a `?page=` someone typed —
  // leaves the list pointing past the end. The data for that page is empty,
  // which must not be read as "no matches".
  const isPageOutOfRange =
    isListView &&
    totalPages !== undefined &&
    !listQuery.isPlaceholderData &&
    page > totalPages - 1;

  useEffect(() => {
    if (!isPageOutOfRange || totalPages === undefined) return;
    // Replace, not push: the out-of-range page is not a place to go Back to.
    setPage(totalPages - 1, { replace: true });
  }, [isPageOutOfRange, totalPages, setPage]);

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
      // Deleting the only card on this page would leave an empty page behind.
      // Stepping back here (rather than waiting for the refetch) means the
      // user never sees the gap.
      if (isListView && page > 0 && (data?.items?.length ?? 0) <= 1) {
        setPage(page - 1, { replace: true });
      }
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.deleteError"));
    },
  });

  if (isListView && listQuery.isLoading) {
    return <ListPageSkeleton cards={6} />;
  }

  if (isListView && isError) {
    return (
      <div className="space-y-4">
        <h1 className="text-3xl font-bold">{t("jobs.title")}</h1>
        <ErrorState message={error.message} onRetry={() => refetch()} />
      </div>
    );
  }

  const jobs = isListView ? data?.items || [] : boardQuery.jobs;
  // The first-run onboarding state belongs to an account with no applications
  // at all — not to a page that simply ran past the end of the list, which
  // would otherwise hide the pagination that is the only way back.
  // Still finding out whether archived jobs exist: neither state is right yet.
  const isCheckingArchived = activeListIsEmpty && anyJobsQuery.isLoading;
  const hasOnlyArchivedJobs =
    activeListIsEmpty && (anyJobsQuery.data?.pagination.total ?? 0) > 0;
  const isFirstRun =
    isListView &&
    jobs.length === 0 &&
    !isFiltered &&
    page === 0 &&
    !isPageOutOfRange &&
    !isCheckingArchived &&
    !hasOnlyArchivedJobs;

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

      {isFirstRun ? (
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
      ) : isCheckingArchived ? (
        <ListPageSkeleton cards={3} />
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
                // The box echoes the keystroke; the URL (and with it the page
                // reset and the query) follows once the typing settles.
                onChange={(e) => setSearchInput(e.target.value)}
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
                onChange={(e) =>
                  setArchivedFilter(e.target.value as ArchivedFilter)
                }
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

          {!isListView ? (
            boardQuery.isLoading ? (
              <ListPageSkeleton cards={3} />
            ) : boardQuery.isLoadingError ? (
              <ErrorState
                message={boardQuery.error?.message ?? ""}
                onRetry={() => boardQuery.refetch()}
              />
            ) : (
              <>
                <JobKanbanBoard
                  jobs={jobs}
                  queryKey={boardQuery.queryKey}
                  onAddComment={(job) => handleQuickAction("comment", job)}
                  onAddStage={(job) => handleQuickAction("stage", job)}
                  onDelete={(job) => handleDelete(job)}
                />
                <JobBoardLoadMore
                  loaded={boardQuery.loaded}
                  total={boardQuery.total}
                  hasMore={boardQuery.hasMore}
                  isLoadingMore={boardQuery.isLoadingMore}
                  failed={boardQuery.loadMoreFailed}
                  onLoadMore={() => boardQuery.loadMore()}
                />
              </>
            )
          ) : // An empty page past the end of the list is a page we are already
          // leaving — showing "no matches" there would be a lie.
          isPageOutOfRange ? (
            <ListPageSkeleton cards={3} />
          ) : hasOnlyArchivedJobs ? (
            // Everything the account has is archived: say so, and point at the
            // filter that shows it instead of a bare "nothing here".
            <EmptyState
              icon={<Briefcase className="h-12 w-12" />}
              title={t("jobs.noActiveJobs")}
              description={t("jobs.noActiveJobsHint")}
              action={
                <Button
                  variant="outline"
                  onClick={() => setArchivedFilter("archived")}
                >
                  {t("jobs.viewArchived")}
                </Button>
              }
            />
          ) : jobs.length === 0 ? (
            <EmptyState
              icon={
                search ? (
                  <Search className="h-12 w-12" />
                ) : (
                  <Briefcase className="h-12 w-12" />
                )
              }
              title={
                search
                  ? t("jobs.noSearchResults", { query: search })
                  : t("jobs.noJobsForFilter")
              }
              description={
                search
                  ? t("jobs.noSearchResultsHint")
                  : t("jobs.tryAnotherFilter")
              }
              action={
                search ? (
                  <Button variant="outline" onClick={clearSearch}>
                    {t("jobs.clearSearch")}
                  </Button>
                ) : undefined
              }
            />
          ) : (
            <JobListGrid
              jobs={jobs}
              isFetching={isFetching}
              openMenuId={openMenuId}
              onToggleMenu={setOpenMenuId}
              onAddComment={(job) => handleQuickAction("comment", job)}
              onAddStage={(job) => handleQuickAction("stage", job)}
              onDelete={(job) => handleDelete(job)}
              returnState={returnState}
            />
          )}

          {/* Pagination — list view only. Driven by the total, not by what the
              current page happens to hold, so an empty out-of-range page keeps
              its way back. */}
          {isListView && pagination && pagination.total > PAGE_SIZE && (
            <div className="flex justify-center gap-2">
              <Button
                variant="outline"
                onClick={() => setPage(page - 1)}
                disabled={page === 0}
              >
                {t("common.previous")}
              </Button>
              <span className="flex items-center px-4 text-sm text-muted-foreground">
                {t("jobs.pageOf", {
                  page: Math.min(page + 1, totalPages ?? page + 1),
                  total: totalPages,
                })}
              </span>
              <Button
                variant="outline"
                onClick={() => setPage(page + 1)}
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
