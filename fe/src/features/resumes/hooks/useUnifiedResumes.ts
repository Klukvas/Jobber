import { useCallback, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { resumesService } from "@/services/resumesService";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import type { ResumeDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

export type ResumeKindFilter = "all" | "uploaded" | "built";
export type UnifiedSortBy = "updated_at" | "title" | "created_at";
export type SortDir = "asc" | "desc";

export type UnifiedResumeItem =
  | { kind: "uploaded"; data: ResumeDTO }
  | { kind: "built"; data: ResumeBuilderDTO };

// Large limit to fetch all user resumes in one page.
// Resumes are user-scoped — in practice users have <100.
const RESUME_FETCH_LIMIT = 500;

function getSortDate(
  item: UnifiedResumeItem,
  field: "updated_at" | "created_at",
): number {
  const raw = item.data[field];
  if (!raw) return 0;
  const ts = new Date(raw).getTime();
  return Number.isNaN(ts) ? 0 : ts;
}

function compareItems(
  a: UnifiedResumeItem,
  b: UnifiedResumeItem,
  sortBy: UnifiedSortBy,
  sortDir: SortDir,
): number {
  const multiplier = sortDir === "asc" ? 1 : -1;

  if (sortBy === "title") {
    return (
      multiplier *
      a.data.title.localeCompare(b.data.title, undefined, {
        sensitivity: "base",
      })
    );
  }

  return multiplier * (getSortDate(a, sortBy) - getSortDate(b, sortBy));
}

export function useUnifiedResumes() {
  const [kindFilter, setKindFilter] = useState<ResumeKindFilter>("all");
  const [sortBy, setSortBy] = useState<UnifiedSortBy>("updated_at");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const uploadedQuery = useQuery({
    queryKey: ["resumes", "all"],
    queryFn: () =>
      resumesService.list({ limit: RESUME_FETCH_LIMIT, offset: 0 }),
  });

  const builderQuery = useQuery({
    queryKey: ["resume-builders"],
    queryFn: () => resumeBuilderService.list(),
  });

  const isLoading = uploadedQuery.isLoading || builderQuery.isLoading;
  const isError = uploadedQuery.isError || builderQuery.isError;
  const error = uploadedQuery.error || builderQuery.error;

  const items = useMemo(() => {
    const uploaded: UnifiedResumeItem[] = (
      uploadedQuery.data?.items ?? []
    ).map((data) => ({ kind: "uploaded" as const, data }));
    const built: UnifiedResumeItem[] = (builderQuery.data ?? []).map(
      (data) => ({ kind: "built" as const, data }),
    );

    const merged =
      kindFilter === "uploaded"
        ? uploaded
        : kindFilter === "built"
          ? built
          : [...uploaded, ...built];

    return [...merged].sort((a, b) => compareItems(a, b, sortBy, sortDir));
  }, [uploadedQuery.data, builderQuery.data, kindFilter, sortBy, sortDir]);

  const toggleSort = useCallback(
    (field: UnifiedSortBy) => {
      if (sortBy === field) {
        setSortDir((prev) => (prev === "desc" ? "asc" : "desc"));
      } else {
        setSortBy(field);
        setSortDir("desc");
      }
    },
    [sortBy],
  );

  const refetch = useCallback(
    () => Promise.all([uploadedQuery.refetch(), builderQuery.refetch()]),
    [uploadedQuery, builderQuery],
  );

  return {
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
  };
}
