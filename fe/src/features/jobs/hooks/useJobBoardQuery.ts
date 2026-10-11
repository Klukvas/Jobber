import { useMemo } from "react";
import { useInfiniteQuery, type InfiniteData } from "@tanstack/react-query";
import { jobsService, type ArchivedFilter } from "@/services/jobsService";
import type { JobDTO, PaginatedResponse } from "@/shared/types/api";

/**
 * How many cards one board request brings in.
 *
 * The board used to ask for `limit: 500` once and render whatever came back,
 * which silently dropped card 501 onwards: no count, no warning, and a card
 * the user could not find anywhere. Now the board pages like everything else
 * and says out loud how much of the pipeline is on screen.
 */
export const JOBS_BOARD_PAGE_SIZE = 100;

export const JOBS_BOARD_QUERY_KEY = ["jobs", "board"] as const;

export interface JobBoardFilters {
  archived: ArchivedFilter;
  /** `field:dir`, as the list endpoint takes it. */
  sort: string;
  search: string;
  companyId: string;
}

/** One cache entry per filter combination, so changing a filter starts over. */
export function jobBoardQueryKey(filters: JobBoardFilters) {
  return [
    ...JOBS_BOARD_QUERY_KEY,
    filters.archived,
    filters.sort,
    filters.search,
    filters.companyId,
  ] as const;
}

export type JobBoardData = InfiniteData<PaginatedResponse<JobDTO>, number>;

/**
 * Flattens the loaded pages into the card list the board renders.
 *
 * Offset paging can hand back a row twice — a card that moves column changes
 * its `last_activity` and with it its place in the sort order, shifting the
 * window under the next request. Two cards with the same id would give dnd-kit
 * two draggables with one id, so the first occurrence wins.
 */
function flattenPages(pages: PaginatedResponse<JobDTO>[]): JobDTO[] {
  const seen = new Set<string>();
  const jobs: JobDTO[] = [];
  for (const page of pages) {
    for (const job of page.items) {
      if (seen.has(job.id)) continue;
      seen.add(job.id);
      jobs.push(job);
    }
  }
  return jobs;
}

/** Rewrites one card wherever it sits in the loaded pages, immutably. */
export function mapBoardJob(
  data: JobBoardData | undefined,
  jobId: string,
  update: (job: JobDTO) => JobDTO,
): JobBoardData | undefined {
  if (!data) return data;
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      items: page.items.map((job) => (job.id === jobId ? update(job) : job)),
    })),
  };
}

/** The first card with this id across the loaded pages. */
export function findBoardJob(
  data: JobBoardData | undefined,
  jobId: string,
): JobDTO | undefined {
  return data?.pages
    .flatMap((page) => page.items)
    .find((job) => job.id === jobId);
}

export function useJobBoardQuery({
  filters,
  enabled,
}: {
  filters: JobBoardFilters;
  enabled: boolean;
}) {
  const queryKey = jobBoardQueryKey(filters);

  const query = useInfiniteQuery({
    queryKey,
    enabled,
    initialPageParam: 0,
    queryFn: ({ pageParam, signal }) =>
      jobsService.list(
        {
          limit: JOBS_BOARD_PAGE_SIZE,
          offset: pageParam,
          status: filters.archived || undefined,
          sort: filters.sort,
          search: filters.search || undefined,
          company_id: filters.companyId || undefined,
        },
        { signal },
      ),
    // The next offset is how much we already hold, so a page the server
    // returned short does not leave a gap. An empty page means the server has
    // run out regardless of what `total` claims — without that guard a stale
    // total would keep the "load more" button alive forever.
    getNextPageParam: (lastPage, allPages) => {
      if (lastPage.items.length === 0) return undefined;
      const loaded = allPages.reduce((count, page) => count + page.items.length, 0);
      return loaded < lastPage.pagination.total ? loaded : undefined;
    },
    staleTime: 30_000,
  });

  const pages = query.data?.pages;
  const jobs = useMemo(() => flattenPages(pages ?? []), [pages]);

  // The freshest total wins: it is the one the last request measured.
  const total = pages?.length
    ? pages[pages.length - 1].pagination.total
    : undefined;

  return {
    queryKey,
    jobs,
    /** Server-side size of the filtered pipeline, undefined until first load. */
    total,
    loaded: jobs.length,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    /** First load failed and there is nothing to show. */
    isLoadingError: query.isLoadingError,
    error: query.error,
    refetch: query.refetch,
    hasMore: query.hasNextPage,
    loadMore: query.fetchNextPage,
    isLoadingMore: query.isFetchingNextPage,
    /** A "load more" that came back empty-handed — the board keeps its cards. */
    loadMoreFailed: query.isFetchNextPageError,
  };
}
