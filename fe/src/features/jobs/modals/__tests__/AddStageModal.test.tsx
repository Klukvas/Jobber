import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AddStageModal } from "../AddStageModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockMutate = vi.hoisted(() => vi.fn());
const mockMutationState = vi.hoisted(() => ({ isPending: false }));
const mockQueryData = vi.hoisted(() => ({
  value: undefined as unknown,
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: mockQueryData.value }),
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: () => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) => {
      mockMutate(v);
      return Promise.resolve(opts.mutationFn(v)).then(
        opts.onSuccess,
        opts.onError,
      );
    },
    isPending: mockMutationState.isPending,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

const mockJobsService = vi.hoisted(() => ({ addStage: vi.fn() }));
vi.mock("@/services/jobsService", () => ({ jobsService: mockJobsService }));

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: { list: vi.fn() },
}));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

const templates = {
  items: [
    { id: "tpl-2", name: "Interview", order: 2 },
    { id: "tpl-1", name: "Screening", order: 1 },
  ],
  total: 2,
};

describe("AddStageModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
    mockQueryData.value = templates;
  });

  it("renders stage template options sorted by order", () => {
    render(<AddStageModal open onOpenChange={vi.fn()} jobId="job-1" />);
    const options = screen.getAllByRole("option");
    // First option is the placeholder, then sorted by order.
    expect(options[1]).toHaveTextContent("1. Screening");
    expect(options[2]).toHaveTextContent("2. Interview");
  });

  it("shows an empty-state hint when there are no templates", () => {
    mockQueryData.value = { items: [], total: 0 };
    render(<AddStageModal open onOpenChange={vi.fn()} jobId="job-1" />);
    expect(screen.getByText("jobs.noStageTemplates")).toBeInTheDocument();
  });

  it("keeps submit disabled until a stage template is chosen", async () => {
    const user = userEvent.setup();
    render(<AddStageModal open onOpenChange={vi.fn()} jobId="job-1" />);

    const submit = screen.getByRole("button", { name: "jobs.addStage" });
    expect(submit).toBeDisabled();

    await user.selectOptions(screen.getByLabelText(/jobs.selectStage/), "tpl-1");
    expect(submit).toBeEnabled();
  });

  it("adds a stage with a trimmed optional comment and closes on success", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockJobsService.addStage.mockResolvedValue({});

    render(<AddStageModal open onOpenChange={onOpenChange} jobId="job-1" />);

    await user.selectOptions(screen.getByLabelText(/jobs.selectStage/), "tpl-1");
    await user.type(screen.getByLabelText("jobs.commentOptional"), "  note  ");
    await user.click(screen.getByRole("button", { name: "jobs.addStage" }));

    await waitFor(() =>
      expect(mockJobsService.addStage).toHaveBeenCalledWith("job-1", {
        stage_template_id: "tpl-1",
        comment: "note",
      }),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "jobs.stageAddedSuccess",
    );
  });

  it("omits the comment field when it is blank", async () => {
    const user = userEvent.setup();
    mockJobsService.addStage.mockResolvedValue({});

    render(<AddStageModal open onOpenChange={vi.fn()} jobId="job-1" />);

    await user.selectOptions(screen.getByLabelText(/jobs.selectStage/), "tpl-2");
    await user.click(screen.getByRole("button", { name: "jobs.addStage" }));

    await waitFor(() =>
      expect(mockJobsService.addStage).toHaveBeenCalledWith("job-1", {
        stage_template_id: "tpl-2",
      }),
    );
  });

  it("surfaces the error message on failure", async () => {
    const user = userEvent.setup();
    mockJobsService.addStage.mockRejectedValue(new Error("cannot add"));

    render(<AddStageModal open onOpenChange={vi.fn()} jobId="job-1" />);

    await user.selectOptions(screen.getByLabelText(/jobs.selectStage/), "tpl-1");
    await user.click(screen.getByRole("button", { name: "jobs.addStage" }));

    await waitFor(() =>
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "cannot add",
      ),
    );
  });
});
