import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import type { JobDTO, PaginatedResponse } from "@/shared/types/api";

const mockList = vi.hoisted(() => vi.fn());
const mockDelete = vi.hoisted(() => vi.fn());
const mockGetById = vi.hoisted(() => vi.fn());

vi.mock("@/services/jobsService", () => ({
  jobsService: {
    list: mockList,
    delete: mockDelete,
    getById: mockGetById,
    listStages: vi.fn().mockResolvedValue([]),
  },
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
  companiesService: {
    getById: vi.fn(),
    list: vi.fn().mockResolvedValue({ items: [] }),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (opts?.query) return `${key}:${opts.query}`;
      if (opts?.page !== undefined) return `${key}:${opts.page}/${opts.total}`;
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
vi.mock("@/features/jobs/components/JobKanbanBoard", () => ({
  JobKanbanBoard: () => <div data-testid="kanban-board" />,
}));

// The detail page is only here to prove its Back button returns to the list
// URL the card was opened from, so everything below the header is stubbed out.
vi.mock("@/services/resumesService", () => ({
  resumesService: { list: vi.fn().mockResolvedValue({ items: [] }) },
}));
vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: { list: vi.fn().mockResolvedValue([]) },
}));
vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: vi.fn().mockResolvedValue({ items: [] }),
  },
}));
vi.mock("@/features/jobs/components/JobInfoCard", () => ({
  JobInfoCard: () => null,
}));
vi.mock("@/features/jobs/components/JobPipelineCard", () => ({
  JobPipelineCard: () => null,
}));
vi.mock("@/features/jobs/components/JobMatchScoreCard", () => ({
  JobMatchScoreCard: () => null,
}));
vi.mock("@/features/jobs/components/JobCommentsSection", () => ({
  JobCommentsSection: () => null,
}));
vi.mock("@/features/jobs/components/JobReminders", () => ({
  JobReminders: () => null,
}));
vi.mock("@/features/jobs/components/JobTags", () => ({ JobTags: () => null }));
vi.mock("@/features/jobs/components/Timeline", () => ({ Timeline: () => null }));
vi.mock("@/features/subscription/components/PricingModal", () => ({
  PricingModal: () => null,
}));

import JobsPage from "../Jobs";
import JobDetail from "../JobDetail";

function makeJob(overrides?: Partial<JobDTO>): JobDTO {
  return {
    id: "job-1",
    title: "Frontend Engineer",
    company_name: "Acme",
    is_archived: false,
    is_favorite: false,
    current_stage_name: "Applied",
    last_activity_at: "2026-01-10T00:00:00Z",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-10T00:00:00Z",
    ...overrides,
  };
}

function page(items: JobDTO[], total: number): PaginatedResponse<JobDTO> {
  return { items, pagination: { total, limit: 20, offset: 0 } };
}

/** Twenty distinct cards, so a page looks like a page. */
function fullPage(prefix: string, total: number): PaginatedResponse<JobDTO> {
  const items = Array.from({ length: 20 }, (_, i) =>
    makeJob({ id: `${prefix}-${i}`, title: `${prefix} job ${i}` }),
  );
  return page(items, total);
}

function UrlProbe() {
  const { pathname, search } = useLocation();
  return <div data-testid="url">{`${pathname}${search}`}</div>;
}

/** The browser's Back and Forward, driven through the router's own history. */
function HistoryControls() {
  const navigate = useNavigate();
  return (
    <div>
      <button onClick={() => navigate(-1)}>history-back</button>
      <button onClick={() => navigate(1)}>history-forward</button>
    </div>
  );
}

function currentUrl(): string {
  return screen.getByTestId("url").textContent ?? "";
}

function renderApp(entry = "/app/jobs") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/app/jobs" element={<JobsPage />} />
          <Route path="/app/jobs/:id" element={<JobDetail />} />
        </Routes>
        <UrlProbe />
        <HistoryControls />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  localStorage.setItem("jobber-ext-banner-dismissed", "true");
});

describe("Jobs page — list state lives in the URL", () => {
  it("restores page, search, sort and the archived filter from the URL", async () => {
    mockList.mockResolvedValue(fullPage("p3", 200));
    renderApp("/app/jobs?page=3&q=react&sort=title&dir=asc&archived=archived");

    await screen.findByText("p3 job 0");
    expect(mockList.mock.calls[0][0]).toMatchObject({
      limit: 20,
      offset: 40,
      search: "react",
      sort: "title:asc",
      status: "archived",
    });

    // …and the controls show that state rather than their defaults.
    expect(screen.getByLabelText("jobs.searchPlaceholder")).toHaveValue(
      "react",
    );
    expect(screen.getByLabelText("jobs.filterLabel")).toHaveValue("archived");
    expect(await screen.findByText("jobs.pageOf:3/10")).toBeInTheDocument();
  });

  it("falls back to defaults for parameters that are not valid", async () => {
    mockList.mockResolvedValue(fullPage("p1", 60));
    renderApp("/app/jobs?page=-4&sort=DROP%20TABLE&dir=sideways&archived=nope");

    await waitFor(() => expect(mockList).toHaveBeenCalled());
    expect(mockList.mock.calls[0][0]).toMatchObject({
      offset: 0,
      sort: "last_activity:desc",
    });
    expect(mockList.mock.calls[0][0].status).toBeUndefined();
  });

  it("writes the page into the URL and restores it with Back", async () => {
    const user = userEvent.setup();
    mockList.mockImplementation((params) =>
      Promise.resolve(
        params.offset === 0 ? fullPage("first", 60) : fullPage("second", 60),
      ),
    );
    renderApp();

    await screen.findByText("first job 0");
    await user.click(screen.getByRole("button", { name: "common.next" }));

    await waitFor(() => expect(currentUrl()).toBe("/app/jobs?page=2"));
    expect(await screen.findByText("second job 0")).toBeInTheDocument();

    await user.click(screen.getByText("history-back"));

    await waitFor(() => expect(currentUrl()).toBe("/app/jobs"));
    expect(await screen.findByText("first job 0")).toBeInTheDocument();

    // Forward again, because a Back that cannot be undone is only half of it.
    await user.click(screen.getByText("history-forward"));
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs?page=2"));
  });

  it("resets to the first page when a filter changes", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(fullPage("p", 200));
    renderApp("/app/jobs?page=4");

    await screen.findByText("p job 0");
    await user.selectOptions(
      screen.getByLabelText("jobs.filterLabel"),
      "archived",
    );

    await waitFor(() => expect(currentUrl()).toBe("/app/jobs?archived=archived"));
  });

  it("commits the search once typing settles, then restores it with Back", async () => {
    mockList.mockImplementation((params) =>
      Promise.resolve(
        params.search ? page([makeJob({ title: "Searched" })], 1) : fullPage("p", 40),
      ),
    );
    renderApp();

    await screen.findByText("p job 0");
    const box = screen.getByLabelText("jobs.searchPlaceholder");
    fireEvent.change(box, { target: { value: "react" } });

    // Not in the URL yet — the debounce has not fired.
    expect(currentUrl()).toBe("/app/jobs");
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs?q=react"));
    expect(await screen.findByText("Searched")).toBeInTheDocument();

    fireEvent.click(screen.getByText("history-back"));
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs"));
    // The box follows the URL back, and so does the list.
    await waitFor(() => expect(box).toHaveValue(""));
    expect(await screen.findByText("p job 0")).toBeInTheDocument();
  });

  // The timer from a keystroke the user has already navigated away from must
  // not land on the URL afterwards and re-apply a search they abandoned.
  it("drops a debounce that fires after a Back navigation", async () => {
    mockList.mockImplementation((params) =>
      Promise.resolve(
        params.search
          ? page([makeJob({ title: `Found ${params.search}` })], 1)
          : fullPage("p", 40),
      ),
    );
    renderApp();

    await screen.findByText("p job 0");
    const box = screen.getByLabelText("jobs.searchPlaceholder");

    // One committed search, so there is a real history entry to go back from.
    fireEvent.change(box, { target: { value: "earlier" } });
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs?q=earlier"));
    await screen.findByText("Found earlier");

    // Now type again and leave before the debounce fires.
    fireEvent.change(box, { target: { value: "later" } });
    fireEvent.click(screen.getByText("history-back"));

    await waitFor(() => expect(currentUrl()).toBe("/app/jobs"));
    await waitFor(() => expect(box).toHaveValue(""));

    // Well past the 300ms debounce: the abandoned value never arrives.
    await new Promise((resolve) => setTimeout(resolve, 500));
    expect(currentUrl()).toBe("/app/jobs");
    expect(mockList).not.toHaveBeenCalledWith(
      expect.objectContaining({ search: "later" }),
      expect.anything(),
    );
  });
});

describe("Jobs page — a page that ran out of cards", () => {
  it("clamps a page past the end back onto the last real one", async () => {
    mockList.mockImplementation((params) =>
      Promise.resolve(params.offset === 0 ? fullPage("p", 15) : page([], 15)),
    );
    renderApp("/app/jobs?page=9");

    // It does not sit there showing "no matches" for a page that cannot exist.
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs"));
    expect(await screen.findByText("p job 0")).toBeInTheDocument();
    expect(screen.queryByText("jobs.noJobsForFilter")).not.toBeInTheDocument();
  });

  it("steps back a page when the last card on it is deleted", async () => {
    const user = userEvent.setup();
    const lonely = makeJob({ id: "lonely", title: "Lonely Job" });
    // 21 cards: page 2 holds exactly one. Deleting it leaves page 2 empty.
    mockList.mockImplementation((params) =>
      Promise.resolve(
        params.offset === 0
          ? fullPage("first", 21)
          : page(mockDelete.mock.calls.length ? [] : [lonely], 21),
      ),
    );
    mockDelete.mockResolvedValue(undefined);
    renderApp("/app/jobs?page=2");

    await screen.findByText("Lonely Job");

    await user.click(screen.getByRole("button", { name: "jobs.actionsMenu" }));
    await user.click(screen.getByText("jobs.delete"));
    const dialog = await screen.findByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: "common.delete" }),
    );

    await waitFor(() => expect(mockDelete).toHaveBeenCalledWith("lonely"));
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs"));
    expect(await screen.findByText("first job 0")).toBeInTheDocument();
  });

  it("keeps the pagination controls on an empty page so there is a way back", async () => {
    mockList.mockImplementation((params) =>
      Promise.resolve(params.offset === 0 ? fullPage("p", 60) : page([], 60)),
    );
    renderApp("/app/jobs?page=2");

    // Even mid-clamp the controls are there: the total, not the current
    // page's contents, decides whether they render.
    expect(
      await screen.findByRole("button", { name: "common.previous" }),
    ).toBeInTheDocument();
  });
});

describe("Job detail — coming back to the list", () => {
  it("returns to the exact list URL the card was opened from", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(page([makeJob({ id: "j1" })], 60));
    mockGetById.mockResolvedValue(makeJob({ id: "j1" }));

    const listUrl = "/app/jobs?page=2&q=react&sort=title&dir=asc";
    renderApp(listUrl);

    await user.click(await screen.findByText("Frontend Engineer"));
    await waitFor(() => expect(currentUrl()).toBe("/app/jobs/j1"));
    // Wait for the loaded detail page: its skeleton draws a Back button too,
    // and a click on that discarded node would prove nothing.
    await screen.findByText("jobs.timeline");

    await user.click(
      screen.getByRole("button", { name: /jobs\.backToJobs/ }),
    );

    expect(currentUrl()).toBe(listUrl);
  });

  it("falls back to the bare list when the card was opened directly", async () => {
    const user = userEvent.setup();
    mockGetById.mockResolvedValue(makeJob({ id: "j1" }));
    mockList.mockResolvedValue(page([], 0));

    renderApp("/app/jobs/j1");

    await screen.findByText("jobs.timeline");
    await user.click(
      screen.getByRole("button", { name: /jobs\.backToJobs/ }),
    );

    expect(currentUrl()).toBe("/app/jobs");
  });
});
