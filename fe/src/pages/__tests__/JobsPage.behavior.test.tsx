import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  render,
  screen,
  waitFor,
  within,
  fireEvent,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { JobDTO, PaginatedResponse } from "@/shared/types/api";

const mockList = vi.hoisted(() => vi.fn());
const mockDelete = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/jobsService", () => ({
  jobsService: {
    list: mockList,
    delete: mockDelete,
  },
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: mockShowSuccess,
  showErrorNotification: mockShowError,
}));

// Make debounce synchronous so search-driven queries fire immediately.
vi.mock("@/shared/hooks/useDebounce", () => ({
  useDebounce: (v: string) => v,
}));

// Stands in for the URL. The company filter lives there, so tests drive it by
// seeding these params rather than by rendering a real router.
const mockSearchParams = vi.hoisted(() => ({
  value: new URLSearchParams(),
  setSpy: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  useSearchParams: () => [
    mockSearchParams.value,
    (next: URLSearchParams) => {
      mockSearchParams.setSpy(next.toString());
      mockSearchParams.value = next;
    },
  ],
}));

const mockCompanyGetById = vi.hoisted(() => vi.fn());
vi.mock("@/services/companiesService", () => ({
  companiesService: { getById: mockCompanyGetById },
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
  CreateJobModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="create-job-modal" /> : null,
}));
vi.mock("@/features/jobs/modals/AddCommentModal", () => ({
  AddCommentModal: ({ jobId }: { jobId: string }) => (
    <div data-testid="add-comment-modal">comment:{jobId}</div>
  ),
}));
vi.mock("@/features/jobs/modals/AddStageModal", () => ({
  AddStageModal: ({ jobId }: { jobId: string }) => (
    <div data-testid="add-stage-modal">stage:{jobId}</div>
  ),
}));
vi.mock("@/features/jobs/components/JobKanbanBoard", () => ({
  JobKanbanBoard: () => <div data-testid="kanban-board" />,
  JOBS_KANBAN_QUERY_KEY: ["jobs", "kanban"],
}));

import JobsPage from "../Jobs";

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

function paginated(
  items: JobDTO[],
  total = items.length,
): PaginatedResponse<JobDTO> {
  return { items, pagination: { total, limit: 20, offset: 0 } };
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <JobsPage />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  mockSearchParams.value = new URLSearchParams();
  // default: banner already dismissed to keep the DOM small unless tested
  localStorage.setItem("jobber-ext-banner-dismissed", "true");
});

describe("Jobs page", () => {
  it("shows the first-run empty state when there are no jobs and no filters", async () => {
    mockList.mockResolvedValue(paginated([]));
    renderPage();
    expect(await screen.findByText("jobs.emptyTitle")).toBeInTheDocument();
    expect(screen.getByText("jobs.createFirstJob")).toBeInTheDocument();
  });

  it("renders an error state with a working retry", async () => {
    const user = userEvent.setup();
    mockList
      .mockRejectedValueOnce(new Error("boom"))
      .mockResolvedValueOnce(paginated([makeJob({ title: "Recovered Job" })]));
    renderPage();

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("boom")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "common.tryAgain" }));
    expect(await screen.findByText("Recovered Job")).toBeInTheDocument();
  });

  it("renders job cards from the list response", async () => {
    mockList.mockResolvedValue(
      paginated([
        makeJob({ id: "j1", title: "Backend Dev", company_name: "Globex" }),
      ]),
    );
    renderPage();
    expect(await screen.findByText("Backend Dev")).toBeInTheDocument();
    expect(screen.getByText("Globex")).toBeInTheDocument();
  });

  it("queries with a sort param when a sort button is toggled", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();
    await screen.findByText("Frontend Engineer");

    mockList.mockClear();
    await user.click(
      screen.getByRole("button", { name: /jobs\.sortJobTitle/ }),
    );

    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({ sort: "title:desc" }),
        expect.anything(),
      ),
    );
  });

  it("queries with the archived filter when changed", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();
    await screen.findByText("Frontend Engineer");

    mockList.mockClear();
    await user.selectOptions(
      screen.getByLabelText("jobs.filterLabel"),
      "archived",
    );

    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({ status: "archived" }),
        expect.anything(),
      ),
    );
  });

  it("shows a no-results empty state and clears the search", async () => {
    const user = userEvent.setup();
    // first call returns a job; searching returns none
    mockList.mockImplementation((params) =>
      Promise.resolve(params.search ? paginated([]) : paginated([makeJob()])),
    );
    renderPage();
    await screen.findByText("Frontend Engineer");

    // Set the full query in one change so a single query fires with search="zzz"
    // (per-keystroke typing races the synchronous-debounce re-queries).
    fireEvent.change(screen.getByLabelText("jobs.searchPlaceholder"), {
      target: { value: "zzz" },
    });

    expect(
      await screen.findByText("jobs.noSearchResults:zzz"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "jobs.clearSearch" }));
    await waitFor(() =>
      expect(screen.getByText("Frontend Engineer")).toBeInTheDocument(),
    );
  });

  it("keeps the search box and current results while a search loads (no full-page skeleton)", async () => {
    let resolveSearch: (v: PaginatedResponse<JobDTO>) => void = () => {};
    const searchPending = new Promise<PaginatedResponse<JobDTO>>((res) => {
      resolveSearch = res;
    });
    // Initial load resolves immediately; the search request stays in flight
    // until we resolve it below.
    mockList.mockImplementation((params) =>
      params.search ? searchPending : Promise.resolve(paginated([makeJob()])),
    );
    renderPage();
    await screen.findByText("Frontend Engineer");

    fireEvent.change(screen.getByLabelText("jobs.searchPlaceholder"), {
      target: { value: "zzz" },
    });

    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({ search: "zzz" }),
        expect.anything(),
      ),
    );

    // While the search is pending the page must NOT fall back to the skeleton:
    // the search input and the previous results stay mounted, so only the list
    // updates once results arrive.
    expect(screen.getByLabelText("jobs.searchPlaceholder")).toBeInTheDocument();
    expect(screen.getByText("Frontend Engineer")).toBeInTheDocument();

    resolveSearch(paginated([]));
    expect(
      await screen.findByText("jobs.noSearchResults:zzz"),
    ).toBeInTheDocument();
  });

  it("shows pagination controls when total exceeds the page size", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob()], 45));
    renderPage();
    await screen.findByText("Frontend Engineer");

    const prev = screen.getByRole("button", { name: "common.previous" });
    const next = screen.getByRole("button", { name: "common.next" });
    expect(prev).toBeDisabled();
    expect(next).toBeEnabled();

    mockList.mockClear();
    await user.click(next);
    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({ offset: 20 }),
        expect.anything(),
      ),
    );
  });

  it("switches to the kanban board and persists the view mode", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();
    await screen.findByText("Frontend Engineer");

    await user.click(screen.getByRole("button", { name: /jobs\.viewBoard/ }));

    expect(await screen.findByTestId("kanban-board")).toBeInTheDocument();
    await waitFor(() =>
      expect(localStorage.getItem("apps-view-mode")).toBe("kanban"),
    );
  });

  it("keeps search and sorting on the board view and applies them", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();
    await screen.findByText("Frontend Engineer");

    await user.click(screen.getByRole("button", { name: /jobs\.viewBoard/ }));
    await screen.findByTestId("kanban-board");

    // Filter/sort/search controls render for the board too.
    expect(screen.getByLabelText("jobs.searchPlaceholder")).toBeInTheDocument();

    mockList.mockClear();
    await user.click(
      screen.getByRole("button", { name: /jobs\.sortJobTitle/ }),
    );

    // The board re-queries with the sort param (loading the full board).
    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({ sort: "title:desc", limit: 500 }),
        expect.anything(),
      ),
    );
  });

  it("dismisses the extension banner and remembers it", async () => {
    const user = userEvent.setup();
    localStorage.removeItem("jobber-ext-banner-dismissed");
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();
    await screen.findByText("Frontend Engineer");

    expect(screen.getByText("jobs.extensionBanner")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "common.close" }));

    expect(screen.queryByText("jobs.extensionBanner")).not.toBeInTheDocument();
    expect(localStorage.getItem("jobber-ext-banner-dismissed")).toBe("true");
  });

  it("opens the add-comment modal from the actions menu", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob({ id: "jc" })]));
    renderPage();
    await screen.findByText("Frontend Engineer");

    await user.click(screen.getByRole("button", { name: "jobs.actionsMenu" }));
    await user.click(screen.getByText("jobs.addComment"));

    expect(screen.getByTestId("add-comment-modal")).toHaveTextContent(
      "comment:jc",
    );
  });

  it("deletes a job through the confirmation dialog", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob({ id: "jd" })]));
    mockDelete.mockResolvedValue(undefined);
    renderPage();
    await screen.findByText("Frontend Engineer");

    await user.click(screen.getByRole("button", { name: "jobs.actionsMenu" }));
    await user.click(screen.getByText("jobs.delete"));

    // confirm dialog appears
    const dialog = await screen.findByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: "common.delete" }),
    );

    await waitFor(() => expect(mockDelete).toHaveBeenCalledWith("jd"));
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("jobs.deleteSuccess"),
    );
  });

  it("surfaces an error if deleting a job fails", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob({ id: "jd" })]));
    mockDelete.mockRejectedValue(new Error("delete failed"));
    renderPage();
    await screen.findByText("Frontend Engineer");

    await user.click(screen.getByRole("button", { name: "jobs.actionsMenu" }));
    await user.click(screen.getByText("jobs.delete"));
    const dialog = await screen.findByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: "common.delete" }),
    );

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("delete failed"),
    );
  });
});

// The Companies page links here with ?company_id=…; the list has to actually
// narrow, say so, and offer a way back to everything.
describe("Jobs page — company filter", () => {
  const COMPANY_ID = "0149b69a-0000-4000-8000-000000000001";

  beforeEach(() => {
    mockSearchParams.value = new URLSearchParams(`company_id=${COMPANY_ID}`);
    mockCompanyGetById.mockResolvedValue({ id: COMPANY_ID, name: "TechNova" });
  });

  it("passes the company filter to the API", async () => {
    mockList.mockResolvedValue(paginated([makeJob({ title: "Backend Dev" })]));
    renderPage();

    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({ company_id: COMPANY_ID }),
        expect.anything(),
      ),
    );
  });

  it("shows a visible indicator naming the company", async () => {
    mockList.mockResolvedValue(paginated([makeJob({ title: "Backend Dev" })]));
    renderPage();

    expect(
      await screen.findByText("jobs.filteredByCompany"),
    ).toBeInTheDocument();
  });

  it("clears the filter and drops the parameter from the URL", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeJob({ title: "Backend Dev" })]));
    renderPage();

    await user.click(await screen.findByText("jobs.clearCompanyFilter"));

    expect(mockSearchParams.setSpy).toHaveBeenCalledWith("");
  });

  it("treats an empty filtered result as 'no matches', not a first-run account", async () => {
    mockList.mockResolvedValue(paginated([]));
    renderPage();

    expect(await screen.findByText("jobs.noJobsForFilter")).toBeInTheDocument();
    expect(screen.queryByText("jobs.emptyTitle")).not.toBeInTheDocument();
  });

  it("sends no company filter when the URL has none", async () => {
    mockSearchParams.value = new URLSearchParams();
    mockList.mockResolvedValue(paginated([makeJob({ title: "Backend Dev" })]));
    renderPage();

    await waitFor(() => expect(mockList).toHaveBeenCalled());
    expect(mockList.mock.calls[0][0].company_id).toBeUndefined();
    expect(
      screen.queryByText("jobs.filteredByCompany"),
    ).not.toBeInTheDocument();
  });
});

/**
 * Every one of these measured under 44x44 on a 375/390 viewport: the banner's
 * close button at 24x24 and its install link at 115x28, the List/Board toggle
 * at 32px tall, the search field at 36 and the status select at 36, and the
 * card's own actions button at 24x24.
 *
 * Asserted on classes rather than geometry — jsdom does no layout, and the
 * classes are what decide the size. Each floor is scoped to `max-sm` so the
 * same page on a pointer keeps the density it was drawn with.
 */
describe("Jobs page — touch targets", () => {
  const PHONE_ROW = /max-sm:min-h-11/;

  it("gives the extension banner's close button a 44x44 target on phones", async () => {
    localStorage.removeItem("jobber-ext-banner-dismissed");
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();

    const close = await screen.findByLabelText("common.close");
    expect(close.className).toMatch(/max-sm:h-11/);
    expect(close.className).toMatch(/max-sm:w-11/);
    // …and stays the 24x24 it was drawn as on a pointer.
    expect(close.className).toMatch(/(^|\s)h-6(\s|$)/);
  });

  it("gives the install-extension link a 44px height on phones", async () => {
    localStorage.removeItem("jobber-ext-banner-dismissed");
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();

    const install = await screen.findByText("jobs.installExtension");
    expect(install.className).toMatch(PHONE_ROW);
  });

  it.each(["jobs.viewList", "jobs.viewBoard"])(
    "gives the %s toggle a 44px height on phones",
    async (label) => {
      mockList.mockResolvedValue(paginated([makeJob()]));
      renderPage();

      const toggle = await screen.findByText(label);
      expect(toggle.className).toMatch(PHONE_ROW);
    },
  );

  // The field used to hard-code `h-9` and bypass the shared input entirely.
  it("gives the search field the shared input's phone height", async () => {
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();

    const search = await screen.findByLabelText("jobs.searchPlaceholder");
    expect(search.className).toMatch(/max-sm:h-11/);
    expect(search.className).not.toMatch(/(^|\s)h-9(\s|$)/);
  });

  it("gives the status filter the same phone height", async () => {
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();

    await screen.findByText("Frontend Engineer");
    const select = screen.getByLabelText("jobs.filterLabel");

    expect(select.className).toMatch(/max-sm:h-11/);
    expect(select.className).not.toMatch(/(^|\s)h-9(\s|$)/);
  });

  it("gives the card's actions button a 44x44 target on phones", async () => {
    mockList.mockResolvedValue(paginated([makeJob()]));
    renderPage();

    const actions = await screen.findByLabelText("jobs.actionsMenu");
    expect(actions.className).toMatch(/max-sm:h-11/);
    expect(actions.className).toMatch(/max-sm:w-11/);
  });
});
