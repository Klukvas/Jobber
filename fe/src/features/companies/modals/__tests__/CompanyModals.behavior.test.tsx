import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { CompanyDTO } from "@/shared/types/api";
import { CreateCompanyModal } from "../CreateCompanyModal";
import { DeleteCompanyDialog } from "../DeleteCompanyDialog";

const mockCreate = vi.hoisted(() => vi.fn());
const mockUpdate = vi.hoisted(() => vi.fn());
const mockDelete = vi.hoisted(() => vi.fn());
const mockGetRelatedCounts = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/companiesService", () => ({
  companiesService: {
    create: mockCreate,
    update: mockUpdate,
    delete: mockDelete,
    getRelatedCounts: mockGetRelatedCounts,
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: mockShowSuccess,
  showErrorNotification: mockShowError,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) =>
      opts?.name ? `${key}:${opts.name}` : key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

const company: CompanyDTO = {
  id: "co-1",
  name: "Acme Corp",
  location: "Berlin",
  notes: "Great place",
  is_favorite: false,
  applications_count: 0,
  active_applications_count: 0,
  derived_status: "idle",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  mockGetRelatedCounts.mockResolvedValue({
    jobs_count: 0,
    applications_count: 0,
  });
});

describe("CreateCompanyModal — create mode", () => {
  it("shows the create title and disables submit with empty name", () => {
    renderWithClient(<CreateCompanyModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("companies.create")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "common.create" }),
    ).toBeDisabled();
  });

  it("submits create with optional fields collapsed to undefined", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockCreate.mockResolvedValue(company);
    renderWithClient(
      <CreateCompanyModal open={true} onOpenChange={onOpenChange} />,
    );

    await user.type(screen.getByLabelText(/companies\.name/), "NewCo");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      name: "NewCo",
      location: undefined,
      notes: undefined,
    });
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("companies.createSuccess"),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("passes location and notes when filled", async () => {
    const user = userEvent.setup();
    mockCreate.mockResolvedValue(company);
    renderWithClient(<CreateCompanyModal open={true} onOpenChange={vi.fn()} />);

    await user.type(screen.getByLabelText(/companies\.name/), "NewCo");
    await user.type(screen.getByLabelText(/companies\.location/), "Munich");
    await user.type(screen.getByLabelText(/companies\.notes/), "hello");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      name: "NewCo",
      location: "Munich",
      notes: "hello",
    });
  });

  it("shows a create error notification on failure", async () => {
    const user = userEvent.setup();
    mockCreate.mockRejectedValue(new Error("create boom"));
    renderWithClient(<CreateCompanyModal open={true} onOpenChange={vi.fn()} />);

    await user.type(screen.getByLabelText(/companies\.name/), "NewCo");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("create boom"),
    );
  });
});

describe("CreateCompanyModal — edit mode", () => {
  it("prefills the form and shows edit labels", () => {
    renderWithClient(
      <CreateCompanyModal
        open={true}
        onOpenChange={vi.fn()}
        company={company}
      />,
    );
    expect(screen.getByText("companies.edit")).toBeInTheDocument();
    expect(screen.getByLabelText(/companies\.name/)).toHaveValue("Acme Corp");
    expect(screen.getByLabelText(/companies\.location/)).toHaveValue("Berlin");
    expect(screen.getByRole("button", { name: "common.save" })).toBeEnabled();
  });

  it("submits an update with the company id and edited data", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockUpdate.mockResolvedValue(company);
    renderWithClient(
      <CreateCompanyModal
        open={true}
        onOpenChange={onOpenChange}
        company={company}
      />,
    );

    const nameInput = screen.getByLabelText(/companies\.name/);
    await user.clear(nameInput);
    await user.type(nameInput, "Acme Renamed");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("co-1", {
        name: "Acme Renamed",
        location: "Berlin",
        notes: "Great place",
      }),
    );
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("companies.updateSuccess"),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("shows an update error notification on failure", async () => {
    const user = userEvent.setup();
    mockUpdate.mockRejectedValue(new Error("update boom"));
    renderWithClient(
      <CreateCompanyModal
        open={true}
        onOpenChange={vi.fn()}
        company={company}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("update boom"),
    );
  });
});

describe("DeleteCompanyDialog", () => {
  it("shows the safe-delete message when there is no related data", async () => {
    mockGetRelatedCounts.mockResolvedValue({
      jobs_count: 0,
      applications_count: 0,
    });
    renderWithClient(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={vi.fn()}
        company={company}
      />,
    );
    expect(await screen.findByText("companies.deleteSafe")).toBeInTheDocument();
    expect(
      screen.queryByText("companies.deleteWarning"),
    ).not.toBeInTheDocument();
  });

  it("shows related-data warnings with counts", async () => {
    mockGetRelatedCounts.mockResolvedValue({
      jobs_count: 3,
      applications_count: 5,
    });
    renderWithClient(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={vi.fn()}
        company={company}
      />,
    );
    expect(
      await screen.findByText("companies.deleteWarning"),
    ).toBeInTheDocument();
    expect(screen.getByText("companies.deleteJobsWarning")).toBeInTheDocument();
    expect(screen.getByText("companies.deleteAppsWarning")).toBeInTheDocument();
  });

  it("calls delete and closes on confirm", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockDelete.mockResolvedValue(undefined);
    renderWithClient(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={onOpenChange}
        company={company}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.delete" }));

    await waitFor(() => expect(mockDelete).toHaveBeenCalledWith("co-1"));
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("surfaces a delete error and keeps the dialog open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockDelete.mockRejectedValue(new Error("delete failed"));
    renderWithClient(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={onOpenChange}
        company={company}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.delete" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("delete failed"),
    );
    // onOpenChange(false) should NOT be triggered by the error path
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });
});
