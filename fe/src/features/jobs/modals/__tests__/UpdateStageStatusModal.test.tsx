import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UpdateStageStatusModal } from "../UpdateStageStatusModal";
import type { JobStageDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockMutate = vi.hoisted(() => vi.fn());
const mockMutationState = vi.hoisted(() => ({
  isPending: false,
  isError: false,
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: () => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) => {
      mockMutate(v);
      return opts.mutationFn(v).then(opts.onSuccess, opts.onError);
    },
    isPending: mockMutationState.isPending,
    isError: mockMutationState.isError,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

const mockJobsService = vi.hoisted(() => ({ updateStage: vi.fn() }));
vi.mock("@/services/jobsService", () => ({ jobsService: mockJobsService }));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

function makeStage(overrides: Partial<JobStageDTO> = {}): JobStageDTO {
  return {
    id: "stage-1",
    job_id: "job-1",
    stage_template_id: "tpl-1",
    stage_name: "Screening",
    status: "active",
    order: 1,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
    ...overrides,
  } as JobStageDTO;
}

describe("UpdateStageStatusModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
    mockMutationState.isError = false;
  });

  it("renders the current status and stage name", () => {
    render(
      <UpdateStageStatusModal
        open
        onOpenChange={vi.fn()}
        jobId="job-1"
        stage={makeStage()}
      />,
    );
    expect(screen.getByText("jobs.changeStageStatus")).toBeInTheDocument();
    // "active" capitalised in the read-only current status box
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("keeps the submit button disabled while the status is unchanged", () => {
    render(
      <UpdateStageStatusModal
        open
        onOpenChange={vi.fn()}
        jobId="job-1"
        stage={makeStage({ status: "active" })}
      />,
    );
    const submit = screen.getByRole("button", { name: "jobs.updateStatus" });
    expect(submit).toBeDisabled();
  });

  it("submits the new status and closes on success", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockJobsService.updateStage.mockResolvedValue({});

    render(
      <UpdateStageStatusModal
        open
        onOpenChange={onOpenChange}
        jobId="job-1"
        stage={makeStage({ status: "active" })}
      />,
    );

    await user.selectOptions(
      screen.getByLabelText(/jobs.newStatus/),
      "completed",
    );
    const submit = screen.getByRole("button", { name: "jobs.updateStatus" });
    expect(submit).toBeEnabled();
    await user.click(submit);

    await waitFor(() =>
      expect(mockJobsService.updateStage).toHaveBeenCalledWith(
        "job-1",
        "stage-1",
        { status: "completed" },
      ),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "jobs.stageStatusUpdateSuccess",
    );
  });

  it("just closes without a service call when submitting the same status", async () => {
    const onOpenChange = vi.fn();

    render(
      <UpdateStageStatusModal
        open
        onOpenChange={onOpenChange}
        jobId="job-1"
        stage={makeStage({ status: "active" })}
      />,
    );

    // Force submit even though the button is disabled by submitting the form.
    const form = screen
      .getByRole("button", { name: "jobs.updateStatus" })
      .closest("form")!;
    form.requestSubmit();

    expect(mockJobsService.updateStage).not.toHaveBeenCalled();
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("shows the error banner when the mutation errored", () => {
    mockMutationState.isError = true;
    render(
      <UpdateStageStatusModal
        open
        onOpenChange={vi.fn()}
        jobId="job-1"
        stage={makeStage()}
      />,
    );
    expect(
      screen.getByText("jobs.stageStatusUpdateFailed"),
    ).toBeInTheDocument();
  });

  it("shows a spinner and disables submit while pending", () => {
    mockMutationState.isPending = true;
    render(
      <UpdateStageStatusModal
        open
        onOpenChange={vi.fn()}
        jobId="job-1"
        stage={makeStage({ status: "pending" })}
      />,
    );
    expect(screen.getByText("common.loading")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /common.loading/ }),
    ).toBeDisabled();
  });
});
