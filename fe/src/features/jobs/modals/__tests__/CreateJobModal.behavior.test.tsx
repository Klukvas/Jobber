import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CreateJobModal } from "../CreateJobModal";
import type { JobDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockMutationState = vi.hoisted(() => ({ isPending: false }));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: { items: [] } }),
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: (r: unknown) => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) =>
      Promise.resolve(opts.mutationFn(v)).then(opts.onSuccess, opts.onError),
    isPending: mockMutationState.isPending,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

const mockJobsService = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
}));
vi.mock("@/services/jobsService", () => ({ jobsService: mockJobsService }));
vi.mock("@/services/companiesService", () => ({
  companiesService: { list: vi.fn() },
}));
vi.mock("@/services/resumesService", () => ({
  resumesService: { list: vi.fn() },
}));
vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: { list: vi.fn() },
}));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

vi.mock("@/features/jobs/components/CompanySelectWithQuickAdd", () => ({
  CompanySelectWithQuickAdd: () => <div data-testid="company-select" />,
}));

const mockCanCreate = vi.hoisted(() => vi.fn(() => true));
vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ canCreate: mockCanCreate }),
}));

vi.mock("@/features/subscription/components/UpgradeBanner", () => ({
  UpgradeBanner: () => <div data-testid="upgrade-banner" />,
}));

function makeJob(overrides: Partial<JobDTO> = {}): JobDTO {
  return {
    id: "job-1",
    title: "Existing",
    company_id: "c1",
    url: "https://x.com",
    source: "LinkedIn",
    notes: "n",
    description: "d",
    is_archived: false,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  } as JobDTO;
}

describe("CreateJobModal submit behavior", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
    mockCanCreate.mockReturnValue(true);
  });

  it("creates a job with normalized fields and closes on success", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const onCreated = vi.fn();
    mockJobsService.create.mockResolvedValue({ id: "new-1" });

    render(
      <CreateJobModal
        open
        onOpenChange={onOpenChange}
        onCreated={onCreated}
      />,
    );

    await user.type(screen.getByLabelText(/jobs.title_field/), "New Role");
    await user.type(screen.getByLabelText("jobs.source"), "Referral");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockJobsService.create).toHaveBeenCalledWith({
        title: "New Role",
        company_id: undefined,
        url: undefined,
        source: "Referral",
        notes: undefined,
        description: undefined,
      }),
    );
    expect(onCreated).toHaveBeenCalledWith({ id: "new-1" });
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "jobs.createSuccess",
    );
  });

  it("includes applied_at when 'already applied' is toggled on", async () => {
    const user = userEvent.setup();
    mockJobsService.create.mockResolvedValue({ id: "new-2" });

    render(<CreateJobModal open onOpenChange={vi.fn()} />);

    await user.type(screen.getByLabelText(/jobs.title_field/), "Applied Role");
    await user.click(screen.getByRole("switch"));
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() => expect(mockJobsService.create).toHaveBeenCalled());
    const arg = mockJobsService.create.mock.calls[0][0];
    expect(arg.title).toBe("Applied Role");
    expect(arg.applied_at).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*Z$/);
  });

  it("edits an existing job via update when a job prop is passed", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockJobsService.update.mockResolvedValue(makeJob());

    render(
      <CreateJobModal open onOpenChange={onOpenChange} job={makeJob()} />,
    );

    // Edit mode uses the "save" label and prefills the title.
    const title = screen.getByLabelText(/jobs.title_field/);
    expect(title).toHaveValue("Existing");
    await user.clear(title);
    await user.type(title, "Renamed");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockJobsService.update).toHaveBeenCalledWith("job-1", {
        title: "Renamed",
        company_id: "c1",
        url: "https://x.com",
        source: "LinkedIn",
        notes: "n",
        description: "d",
      }),
    );
    expect(mockJobsService.create).not.toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not submit when the title is empty", async () => {
    const user = userEvent.setup();
    render(<CreateJobModal open onOpenChange={vi.fn()} />);

    const submit = screen.getByRole("button", { name: "common.create" });
    expect(submit).toBeDisabled();
    await user.click(submit);
    expect(mockJobsService.create).not.toHaveBeenCalled();
  });

  it("shows the upgrade banner and blocks submit when the plan limit is reached", () => {
    mockCanCreate.mockReturnValue(false);
    render(<CreateJobModal open onOpenChange={vi.fn()} />);

    expect(screen.getByTestId("upgrade-banner")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "common.create" }),
    ).toBeDisabled();
  });

  it("surfaces the error message when creation fails", async () => {
    const user = userEvent.setup();
    mockJobsService.create.mockRejectedValue(new Error("create failed"));

    render(<CreateJobModal open onOpenChange={vi.fn()} />);
    await user.type(screen.getByLabelText(/jobs.title_field/), "X");
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "create failed",
      ),
    );
  });
});
