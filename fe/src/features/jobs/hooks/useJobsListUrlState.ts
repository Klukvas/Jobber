import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useDebounce } from "@/shared/hooks/useDebounce";
import type { ArchivedFilter } from "@/services/jobsService";

export type JobsSortBy =
  | "last_activity"
  | "applied_at"
  | "created_at"
  | "title"
  | "company_name";
export type JobsSortDir = "asc" | "desc";

const SORT_BY_VALUES: readonly JobsSortBy[] = [
  "last_activity",
  "applied_at",
  "created_at",
  "title",
  "company_name",
];

const DEFAULT_SORT_BY: JobsSortBy = "last_activity";
const DEFAULT_SORT_DIR: JobsSortDir = "desc";

const PARAM_PAGE = "page";
const PARAM_SEARCH = "q";
const PARAM_SORT = "sort";
const PARAM_DIR = "dir";
const PARAM_ARCHIVED = "archived";
const PARAM_COMPANY = "company_id";

/** Guards against `?page=1e9` turning into an absurd offset on the wire. */
const MAX_PAGE = 10_000;
/** The search box is a filter, not a payload. */
const MAX_SEARCH_LENGTH = 100;

const SEARCH_DEBOUNCE_MS = 300;

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** URL pages are 1-based for the person reading them; state is 0-based. */
function parsePage(raw: string | null): number {
  if (!raw) return 0;
  const parsed = Number(raw);
  if (!Number.isInteger(parsed) || parsed < 1 || parsed > MAX_PAGE) return 0;
  return parsed - 1;
}

function parseSearch(raw: string | null): string {
  return (raw ?? "").trim().slice(0, MAX_SEARCH_LENGTH);
}

function parseSortBy(raw: string | null): JobsSortBy {
  return SORT_BY_VALUES.includes(raw as JobsSortBy)
    ? (raw as JobsSortBy)
    : DEFAULT_SORT_BY;
}

function parseSortDir(raw: string | null): JobsSortDir {
  return raw === "asc" || raw === "desc" ? raw : DEFAULT_SORT_DIR;
}

/**
 * "active" is the same thing as the empty default — accepted from a URL
 * someone typed or an older link, normalized so the select still matches.
 */
function parseArchived(raw: string | null): ArchivedFilter {
  if (raw === "archived" || raw === "all") return raw;
  return "";
}

/** A company filter is a UUID or it is not a filter: junk is dropped. */
function parseCompanyId(raw: string | null): string {
  return raw && UUID_RE.test(raw) ? raw : "";
}

/** Defaults stay out of the URL so a plain `/app/jobs` link stays plain. */
function writeParam(params: URLSearchParams, key: string, value: string) {
  if (value) params.set(key, value);
  else params.delete(key);
}

function writePage(params: URLSearchParams, page: number) {
  writeParam(params, PARAM_PAGE, page > 0 ? String(page + 1) : "");
}

export interface JobsListUrlState {
  /** 0-based, already clamped to a sane range. */
  page: number;
  /** The committed search — what the list actually queries with. */
  search: string;
  /** What is in the box right now, which may be ahead of `search`. */
  searchInput: string;
  sortBy: JobsSortBy;
  sortDir: JobsSortDir;
  archivedFilter: ArchivedFilter;
  companyFilterId: string;
  /** True when anything narrows the list — including an active search. */
  isFiltered: boolean;
  setPage: (page: number, options?: { replace?: boolean }) => void;
  setSearchInput: (value: string) => void;
  clearSearch: () => void;
  toggleSort: (field: JobsSortBy) => void;
  setArchivedFilter: (value: ArchivedFilter) => void;
  clearCompanyFilter: () => void;
}

/**
 * The Jobs list state lives in the URL.
 *
 * Page, search, sort, the archived filter and the company filter were all
 * component state, so a card opened from page 3 of a filtered search came back
 * to an unfiltered page 1, a refresh lost the same context, and none of it
 * could be shared. Keeping them in query parameters makes Back, Forward,
 * refresh and copy-paste all restore the same list — and every parameter is
 * validated on the way in, because a URL is user input.
 *
 * Only the search box keeps a local copy: the box has to echo keystrokes
 * immediately while the URL is written once the typing settles.
 */
export function useJobsListUrlState(): JobsListUrlState {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = parsePage(searchParams.get(PARAM_PAGE));
  const search = parseSearch(searchParams.get(PARAM_SEARCH));
  const sortBy = parseSortBy(searchParams.get(PARAM_SORT));
  const sortDir = parseSortDir(searchParams.get(PARAM_DIR));
  const archivedFilter = parseArchived(searchParams.get(PARAM_ARCHIVED));
  const companyFilterId = parseCompanyId(searchParams.get(PARAM_COMPANY));

  const [searchInput, setSearchInput] = useState(search);
  const debouncedSearch = useDebounce(searchInput.trim(), SEARCH_DEBOUNCE_MS);
  const [adoptedSearch, setAdoptedSearch] = useState(search);

  // The URL's search changed without us asking — Back, Forward or a fresh
  // link. Adopt it into the box, unless it is simply our own pending
  // keystrokes landing, in which case the box is already ahead and must not
  // be rewound. Render-time adjustment: the set-state-in-effect lint rule
  // forbids the effect version.
  if (search !== adoptedSearch) {
    setAdoptedSearch(search);
    if (search !== debouncedSearch) setSearchInput(search);
  }

  const updateParams = useCallback(
    (
      mutate: (params: URLSearchParams) => void,
      options?: { replace?: boolean },
    ) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          mutate(next);
          return next;
        },
        { replace: options?.replace ?? false },
      );
    },
    [setSearchParams],
  );

  const setPage = useCallback(
    (nextPage: number, options?: { replace?: boolean }) => {
      updateParams(
        (params) => writePage(params, Math.max(0, nextPage)),
        options,
      );
    },
    [updateParams],
  );

  /** Any filter change starts the list again from its first page. */
  const commitSearch = useCallback(
    (value: string) => {
      updateParams((params) => {
        writeParam(params, PARAM_SEARCH, value);
        writePage(params, 0);
      });
    },
    [updateParams],
  );

  // Read through a ref so a debounce that fires *after* a Back navigation
  // compares itself against the URL as it is now, not as it was when the
  // timer started. Without it the stale timer would re-apply the search the
  // user just navigated away from.
  const commitRef = useRef({ search, commitSearch });
  useEffect(() => {
    commitRef.current = { search, commitSearch };
  });

  useEffect(() => {
    const { search: current, commitSearch: commit } = commitRef.current;
    if (debouncedSearch === current) return;
    commit(debouncedSearch);
  }, [debouncedSearch]);

  const clearSearch = useCallback(() => {
    setSearchInput("");
    commitSearch("");
  }, [commitSearch]);

  const toggleSort = useCallback(
    (field: JobsSortBy) => {
      updateParams((params) => {
        const currentBy = parseSortBy(params.get(PARAM_SORT));
        const currentDir = parseSortDir(params.get(PARAM_DIR));
        const nextDir: JobsSortDir =
          currentBy === field
            ? currentDir === "desc"
              ? "asc"
              : "desc"
            : DEFAULT_SORT_DIR;
        writeParam(params, PARAM_SORT, field === DEFAULT_SORT_BY ? "" : field);
        writeParam(params, PARAM_DIR, nextDir === DEFAULT_SORT_DIR ? "" : nextDir);
        writePage(params, 0);
      });
    },
    [updateParams],
  );

  const setArchivedFilter = useCallback(
    (value: ArchivedFilter) => {
      updateParams((params) => {
        writeParam(params, PARAM_ARCHIVED, parseArchived(value));
        writePage(params, 0);
      });
    },
    [updateParams],
  );

  const clearCompanyFilter = useCallback(() => {
    updateParams((params) => {
      params.delete(PARAM_COMPANY);
      writePage(params, 0);
    });
  }, [updateParams]);

  return {
    page,
    search,
    searchInput,
    sortBy,
    sortDir,
    archivedFilter,
    companyFilterId,
    isFiltered:
      archivedFilter !== "" || search !== "" || companyFilterId !== "",
    setPage,
    setSearchInput,
    clearSearch,
    toggleSort,
    setArchivedFilter,
    clearCompanyFilter,
  };
}
