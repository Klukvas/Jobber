import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { JOBS_BOARD_PAGE_SIZE } from "@/features/jobs/hooks/useJobBoardQuery";
import type { JobDTO, PaginatedResponse } from "@/shared/types/api";

const mockList = vi.hoisted(() => vi.fn());

vi.mock("@/services/jobsService", () => ({
  jobsService: { list: mockList, delete: vi.fn() },
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/services/companiesService", () => ({
  companiesService: { getById: vi.fn() },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (opts?.loaded !== undefined)
        return `${key}:${opts.loaded}/${opts.total}`;
      return key;
    },
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

vi.mock("@/features/jobs/modals/CreateJobModal", () => ({
  CreateJobModal: () => null,
}));
vi.mock("@/features/jobs/modals/AddCommentModal", () => ({
  AddCommentModal: () => null,
}));
vi.mock("@/features/jobs/modals/AddStageModal", () => ({
  AddStageModal: () => null,
}));

// The board itself is stubbed down to the cards it was handed: what this file
// is about is which cards reach it, not how they are laid out or dragged.
vi.mock("@/features/jobs/components/JobKanbanBoard", () => ({
  JobKanbanBoard: ({ jobs }: { jobs: JobDTO[] }) => (
    <div data-testid="kanban-board" data-count={jobs.length}>
      {jobs.map((job) => (
        <span key={job.id}>{job.id}</span>
      ))}
    </div>
  ),
}));

import JobsPage from "../Jobs";

function makeJob(id: string): JobDTO {
  return {
    id,
    title: `Job ${id}`,
    is_archived: false,
    is_favorite: false,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  } as JobDTO;
}

/** A server page: ids numbered by their real offset, so gaps are visible. */
function serverPage(
  offset: number,
  total: number,
  size = JOBS_BOARD_PAGE_SIZE,
): PaginatedResponse<JobDTO> {
  const count = Math.max(0, Math.min(size, total - offset));
  return {
    items: Array.from({ length: count }, (_, i) => makeJob(`j${offset + i}`)),
    pagination: { total, limit: size, offset },
  };
}

function boardCount(): number {
  return Number(screen.getByTestId("kanban-board").dataset.count);
}

function renderBoard(entry = "/app/jobs") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[entry]}>
        <JobsPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  localStorage.setItem("jobber-ext-banner-dismissed", "true");
  // The board is the view under test.
  localStorage.setItem("apps-view-mode", "kanban");
});

/**
 * The board used to fetch `limit: 500` once and render whatever came back. A
 * pipeline with 520 applications lost 20 of them with nothing on screen
 * admitting it — they existed in the list view and simply were not on the
 * board.
 */
describe("Jobs board — paginated loading", () => {
  const TOTAL = 520;

  it("loads the first page and states how much of the pipeline is shown", async () => {
    mockList.mockImplementation((params) =>
      Promise.resolve(serverPage(params.offset, TOTAL)),
    );
    renderBoard();

    await screen.findByTestId("kanban-board");
    expect(boardCount()).toBe(JOBS_BOARD_PAGE_SIZE);
    expect(mockList.mock.calls[0][0]).toMatchObject({
      limit: JOBS_BOARD_PAGE_SIZE,
      offset: 0,
    });
    expect(
      screen.getByText(`jobs.board.loadedCount:100/${TOTAL}`),
    ).toBeInTheDocument();
  });

  it("brings in every remaining card page by page, past the old 500 ceiling", async () => {
    const user = userEvent.setup();
    mockList.mockImplementation((params) =>
      Promise.resolve(serverPage(params.offset, TOTAL)),
    );
    renderBoard();

    await screen.findByTestId("kanban-board");
    // Keep asking as long as the board offers more — six pages for 520 cards.
    for (let clicks = 0; clicks < 10; clicks++) {
      const more = screen.queryByRole("button", {
        name: "jobs.board.loadMore",
      });
      if (!more) break;
      const before = boardCount();
      await user.click(more);
      await waitFor(() => expect(boardCount()).toBeGreaterThan(before));
    }

    // All 520 are on the board — including the ones past 500 — and there is
    // nothing left to ask for.
    expect(boardCount()).toBe(TOTAL);
    expect(screen.getByText(`jobs.board.loadedCount:520/${TOTAL}`)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "jobs.board.loadMore" }),
    ).not.toBeInTheDocument();
  });

  it("shows no load-more control when one page is the whole pipeline", async () => {
    mockList.mockImplementation((params) =>
      Promise.resolve(serverPage(params.offset, 12)),
    );
    renderBoard();

    await screen.findByTestId("kanban-board");
    expect(boardCount()).toBe(12);
    expect(screen.getByText("jobs.board.loadedCount:12/12")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "jobs.board.loadMore" }),
    ).not.toBeInTheDocument();
  });

  // Offset paging re-reads a moving window: a card whose last activity changed
  // between two requests can come back twice. Two cards with one id would give
  // dnd-kit two draggables with the same id.
  it("keeps a card that the server repeats across pages exactly once", async () => {
    const user = userEvent.setup();
    mockList.mockImplementation((params) => {
      if (params.offset === 0) return Promise.resolve(serverPage(0, 150, 100));
      // Second page overlaps the first by one card.
      return Promise.resolve({
        items: [makeJob("j99"), makeJob("j100"), makeJob("j101")],
        pagination: { total: 150, limit: 100, offset: params.offset },
      });
    });
    renderBoard();

    await screen.findByTestId("kanban-board");
    await user.click(
      screen.getByRole("button", { name: "jobs.board.loadMore" }),
    );

    await waitFor(() => expect(boardCount()).toBe(102));
    expect(screen.getAllByText("j99")).toHaveLength(1);
  });

  it("says a load-more failed, keeps the cards already loaded, and retries", async () => {
    const user = userEvent.setup();
    mockList.mockImplementation((params) =>
      params.offset === 0
        ? Promise.resolve(serverPage(0, TOTAL))
        : Promise.reject(new Error("network down")),
    );
    renderBoard();

    await screen.findByTestId("kanban-board");
    await user.click(
      screen.getByRole("button", { name: "jobs.board.loadMore" }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "jobs.board.loadMoreError",
    );
    // The 100 cards that did load are still there — a failed page does not
    // empty the board.
    expect(boardCount()).toBe(JOBS_BOARD_PAGE_SIZE);

    mockList.mockImplementation((params) =>
      Promise.resolve(serverPage(params.offset, TOTAL)),
    );
    await user.click(screen.getByRole("button", { name: "common.tryAgain" }));

    await waitFor(() => expect(boardCount()).toBe(200));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("starts over at the first page when a filter changes", async () => {
    const user = userEvent.setup();
    mockList.mockImplementation((params) =>
      Promise.resolve(serverPage(params.offset, TOTAL)),
    );
    renderBoard();

    await screen.findByTestId("kanban-board");
    await user.click(
      screen.getByRole("button", { name: "jobs.board.loadMore" }),
    );
    await waitFor(() => expect(boardCount()).toBe(200));

    mockList.mockClear();
    await user.selectOptions(
      screen.getByLabelText("jobs.filterLabel"),
      "archived",
    );

    await waitFor(() => expect(boardCount()).toBe(JOBS_BOARD_PAGE_SIZE));
    expect(mockList.mock.calls[0][0]).toMatchObject({
      offset: 0,
      status: "archived",
    });
  });

  it("reports a first-page failure with a retry instead of an empty board", async () => {
    const user = userEvent.setup();
    mockList.mockRejectedValueOnce(new Error("boom"));
    renderBoard();

    expect(await screen.findByRole("alert")).toHaveTextContent("boom");
    expect(screen.queryByTestId("kanban-board")).not.toBeInTheDocument();

    mockList.mockImplementation((params) =>
      Promise.resolve(serverPage(params.offset, TOTAL)),
    );
    await user.click(screen.getByRole("button", { name: "common.tryAgain" }));

    await screen.findByTestId("kanban-board");
    expect(boardCount()).toBe(JOBS_BOARD_PAGE_SIZE);
  });
});
