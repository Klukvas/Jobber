import { createElement, type ReactNode } from "react";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useJobBoardQuery, mapBoardJob, findBoardJob, type JobBoardData } from "../useJobBoardQuery";
import type { JobDTO, PaginatedResponse } from "@/shared/types/api";
const list = vi.hoisted(() => vi.fn());
vi.mock("@/services/jobsService", () => ({ jobsService: { list } }));
function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return createElement(QueryClientProvider, { client }, children);
}
const filters = { archived: "" as const, sort: "last_activity" as const, search: "", companyId: "" };
const page = (items: JobDTO[] | null, total = 0): PaginatedResponse<JobDTO> =>
  ({ items, pagination: { total, offset: 0, limit: 100 } }) as PaginatedResponse<JobDTO>;
beforeEach(() => list.mockReset());
describe("empty board responses", () => {
  it.each([null, []])("renders %j collection without error or another page", async (items) => {
    list.mockResolvedValue(page(items));
    const { result } = renderHook(() => useJobBoardQuery({ filters, enabled: true }), { wrapper });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.jobs).toEqual([]);
    expect(result.current.hasMore).toBe(false);
    expect(list).toHaveBeenCalledTimes(1);
  });
  it("stops on a legacy null next page with a stale nonzero total", async () => {
    const job = { id: "one" } as JobDTO;
    list.mockResolvedValueOnce(page([job], 9)).mockResolvedValueOnce(page(null, 9));
    const { result } = renderHook(() => useJobBoardQuery({ filters, enabled: true }), { wrapper });
    await waitFor(() => expect(result.current.hasMore).toBe(true));
    await act(async () => { await result.current.loadMore(); });
    await waitFor(() => expect(result.current.hasMore).toBe(false));
    expect(result.current.jobs).toEqual([job]);
  });
  it("cache helpers tolerate legacy null pages", () => {
    const job = { id: "one", title: "One" } as JobDTO;
    const data: JobBoardData = { pages: [page(null), page([job])], pageParams: [0, 1] };
    expect(findBoardJob(data, "one")).toEqual(job);
    expect(mapBoardJob(data, "one", (entry) => ({ ...entry, title: "Changed" }))?.pages[1].items[0].title).toBe("Changed");
  });
});
