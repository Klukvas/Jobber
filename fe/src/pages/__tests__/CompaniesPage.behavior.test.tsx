import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { CompanyDTO, PaginatedResponse } from "@/shared/types/api";

const mockList = vi.hoisted(() => vi.fn());
const mockToggleFavorite = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());
const mockNavigate = vi.hoisted(() => vi.fn());

vi.mock("@/services/companiesService", () => ({
  companiesService: {
    list: mockList,
    toggleFavorite: mockToggleFavorite,
  },
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));
vi.mock("@/shared/lib/notifications", () => ({
  showErrorNotification: mockShowError,
  showSuccessNotification: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) =>
      opts?.count !== undefined ? `${key}:${opts.count}` : key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

// Lightweight modal stubs so the page's action state changes are observable.
vi.mock("@/features/companies/modals/CreateCompanyModal", () => ({
  CreateCompanyModal: ({
    open,
    company,
  }: {
    open: boolean;
    company?: CompanyDTO;
  }) =>
    open ? (
      <div data-testid="create-company-modal">
        {company ? `editing:${company.id}` : "creating"}
      </div>
    ) : null,
}));

vi.mock("@/features/companies/modals/DeleteCompanyDialog", () => ({
  DeleteCompanyDialog: ({ company }: { company: CompanyDTO }) => (
    <div data-testid="delete-company-dialog">deleting:{company.id}</div>
  ),
}));

import Companies from "../Companies";

function makeCompany(overrides?: Partial<CompanyDTO>): CompanyDTO {
  return {
    id: "co-1",
    name: "Acme Corp",
    location: "Berlin",
    notes: "",
    is_favorite: false,
    applications_count: 0,
    active_applications_count: 0,
    derived_status: "idle",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function paginated(items: CompanyDTO[]): PaginatedResponse<CompanyDTO> {
  return {
    items,
    pagination: { total: items.length, limit: 100, offset: 0 },
  };
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <Companies />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("Companies page", () => {
  it("shows a loading skeleton before data resolves", () => {
    let resolve!: (v: PaginatedResponse<CompanyDTO>) => void;
    mockList.mockReturnValue(
      new Promise<PaginatedResponse<CompanyDTO>>((r) => {
        resolve = r;
      }),
    );
    renderPage();
    // title is not rendered during the skeleton state
    expect(screen.queryByText("companies.title")).not.toBeInTheDocument();
    resolve(paginated([]));
  });

  it("shows the empty state when no companies exist", async () => {
    mockList.mockResolvedValue(paginated([]));
    renderPage();
    expect(
      await screen.findByText("companies.noCompanies"),
    ).toBeInTheDocument();
    expect(screen.getByText("companies.createFirst")).toBeInTheDocument();
  });

  it("shows an error state with a retry that refetches", async () => {
    const user = userEvent.setup();
    mockList
      .mockRejectedValueOnce(new Error("network down"))
      .mockResolvedValueOnce(paginated([makeCompany({ name: "Recovered" })]));
    renderPage();

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("network down")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "common.tryAgain" }));
    expect(await screen.findByText("Recovered")).toBeInTheDocument();
  });

  it("renders company cards with status and stats", async () => {
    mockList.mockResolvedValue(
      paginated([
        makeCompany({
          id: "c1",
          name: "Globex",
          derived_status: "active",
          applications_count: 4,
          active_applications_count: 2,
        }),
      ]),
    );
    renderPage();

    expect(await screen.findByText("Globex")).toBeInTheDocument();
    expect(screen.getByText("companies.statusActive")).toBeInTheDocument();
    expect(screen.getByText("companies.totalApplications")).toBeInTheDocument();
    // view-applications quick action only shows when there are applications
    expect(
      screen.getByText("companies.viewApplications:4"),
    ).toBeInTheDocument();
  });

  it("sorts by applications when the sort button is clicked", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(paginated([makeCompany()]));
    renderPage();
    await screen.findByText("Acme Corp");

    mockList.mockClear();
    await user.click(
      screen.getByRole("button", { name: /companies\.sortApplications/ }),
    );

    await waitFor(() =>
      expect(mockList).toHaveBeenCalledWith(
        expect.objectContaining({
          sort_by: "applications_count",
          sort_dir: "desc",
        }),
      ),
    );
  });

  it("navigates to filtered jobs when viewing applications", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(
      paginated([makeCompany({ id: "c9", applications_count: 3 })]),
    );
    renderPage();
    await screen.findByText("Acme Corp");

    await user.click(screen.getByText("companies.viewApplications:3"));
    expect(mockNavigate).toHaveBeenCalledWith("/app/jobs?company_id=c9");
  });

  it("opens the edit modal from the actions menu", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(
      paginated([makeCompany({ id: "c-edit", name: "EditMe" })]),
    );
    renderPage();
    await screen.findByText("EditMe");

    await user.click(
      screen.getByRole("button", { name: "companies.actionsMenu" }),
    );
    await user.click(screen.getByText("common.edit"));

    expect(screen.getByTestId("create-company-modal")).toHaveTextContent(
      "editing:c-edit",
    );
  });

  it("opens the delete dialog from the actions menu", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(
      paginated([makeCompany({ id: "c-del", name: "DeleteMe" })]),
    );
    renderPage();
    await screen.findByText("DeleteMe");

    await user.click(
      screen.getByRole("button", { name: "companies.actionsMenu" }),
    );
    await user.click(screen.getByText("common.delete"));

    expect(screen.getByTestId("delete-company-dialog")).toHaveTextContent(
      "deleting:c-del",
    );
  });

  it("optimistically toggles favorite and rolls back on error", async () => {
    const user = userEvent.setup();
    mockList.mockResolvedValue(
      paginated([makeCompany({ id: "cf", is_favorite: false })]),
    );
    mockToggleFavorite.mockRejectedValue(new Error("fav failed"));
    renderPage();
    await screen.findByText("Acme Corp");

    const favButton = screen.getByRole("button", {
      name: "common.addToFavorites",
    });
    await user.click(favButton);

    // service called with the company id (RQ appends a mutation context arg)
    await waitFor(() => expect(mockToggleFavorite).toHaveBeenCalledTimes(1));
    expect(mockToggleFavorite.mock.calls[0][0]).toBe("cf");
    // error path surfaces a notification and rolls back to non-favorite
    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("common.error"),
    );
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: "common.addToFavorites" }),
      ).toBeInTheDocument(),
    );
  });
});
