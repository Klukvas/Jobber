import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Timeline } from "../Timeline";
import type { JobStageDTO, CommentDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars?.stageName ? `${key}:${vars.stageName}` : key,
    i18n: { language: "en" },
  }),
}));

const mockMutationState = vi.hoisted(() => ({ isPending: false }));

vi.mock("@tanstack/react-query", () => ({
  useMutation: (opts: {
    mutationFn: (v: unknown) => Promise<unknown>;
    onSuccess?: () => void;
    onError?: (e: Error) => void;
  }) => ({
    mutate: (v: unknown) =>
      Promise.resolve(opts.mutationFn(v)).then(opts.onSuccess, opts.onError),
    isPending: mockMutationState.isPending,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

const mockJobsService = vi.hoisted(() => ({
  updateStage: vi.fn(),
  deleteStage: vi.fn(),
}));
vi.mock("@/services/jobsService", () => ({ jobsService: mockJobsService }));

const mockNotifications = vi.hoisted(() => ({
  showErrorNotification: vi.fn(),
  showSuccessNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

// Isolate from child modals (they have their own tests). Paths must match the
// specifiers Timeline.tsx imports, resolved via the @ alias.
vi.mock("@/features/jobs/modals/UpdateStageStatusModal", () => ({
  UpdateStageStatusModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="status-modal" /> : null,
}));
vi.mock("@/features/jobs/modals/AddCommentModal", () => ({
  AddCommentModal: ({
    open,
    stageName,
  }: {
    open: boolean;
    stageName?: string;
  }) => (open ? <div data-testid="comment-modal">{stageName}</div> : null),
}));
vi.mock("@/features/calendar/modals/ScheduleStageModal", () => ({
  ScheduleStageModal: () => <div data-testid="schedule-modal" />,
}));

function stage(overrides: Partial<JobStageDTO> = {}): JobStageDTO {
  return {
    id: "s1",
    job_id: "job-1",
    stage_template_id: "tpl-1",
    stage_name: "Screening",
    status: "active",
    order: 1,
    started_at: "2026-08-01T00:00:00Z",
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T00:00:00Z",
    ...overrides,
  } as JobStageDTO;
}

function comment(overrides: Partial<CommentDTO> = {}): CommentDTO {
  return {
    id: "c1",
    job_id: "job-1",
    content: "A note",
    created_at: "2026-08-02T00:00:00Z",
    updated_at: "2026-08-02T00:00:00Z",
    ...overrides,
  } as CommentDTO;
}

describe("Timeline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
  });

  it("shows an empty-state message when there are no items", () => {
    render(<Timeline stages={[]} jobId="job-1" stageComments={[]} />);
    expect(screen.getByText("jobs.noStagesYet")).toBeInTheDocument();
  });

  it("renders stages and comments merged into the timeline", () => {
    render(
      <Timeline stages={[stage()]} jobId="job-1" stageComments={[comment()]} />,
    );
    expect(screen.getByText("Screening")).toBeInTheDocument();
    expect(screen.getByText("A note")).toBeInTheDocument();
  });

  it("annotates a comment with its related stage name", () => {
    render(
      <Timeline
        stages={[stage({ id: "s1", stage_name: "Interview" })]}
        jobId="job-1"
        stageComments={[comment({ stage_id: "s1" })]}
      />,
    );
    expect(screen.getByText("jobs.commentOn:Interview")).toBeInTheDocument();
  });

  it("completes an active stage via the Complete button", async () => {
    const user = userEvent.setup();
    mockJobsService.updateStage.mockResolvedValue({});
    render(
      <Timeline
        stages={[stage({ status: "active" })]}
        jobId="job-1"
        stageComments={[]}
      />,
    );

    await user.click(screen.getByRole("button", { name: /jobs.complete/ }));
    await waitFor(() =>
      expect(mockJobsService.updateStage).toHaveBeenCalledWith(
        "job-1",
        "s1",
        expect.objectContaining({ status: "completed" }),
      ),
    );
  });

  it("does not offer a Complete button for a completed stage", () => {
    render(
      <Timeline
        stages={[
          stage({ status: "completed", completed_at: "2026-08-03T00:00:00Z" }),
        ]}
        jobId="job-1"
        stageComments={[]}
      />,
    );
    expect(
      screen.queryByRole("button", { name: /jobs.complete/ }),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/jobs\.completed/)).toBeInTheDocument();
  });

  it("opens the change-status modal from the options menu", async () => {
    const user = userEvent.setup();
    render(<Timeline stages={[stage()]} jobId="job-1" stageComments={[]} />);

    await user.click(screen.getByRole("button", { name: "jobs.stageOptions" }));
    await user.click(screen.getByText("jobs.changeStatus"));

    expect(screen.getByTestId("status-modal")).toBeInTheDocument();
  });

  it("opens the add-comment modal scoped to the stage", async () => {
    const user = userEvent.setup();
    render(
      <Timeline
        stages={[stage({ stage_name: "Offer" })]}
        jobId="job-1"
        stageComments={[]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "jobs.stageOptions" }));
    await user.click(screen.getByText("jobs.addComment"));

    const modal = screen.getByTestId("comment-modal");
    expect(modal).toHaveTextContent("Offer");
  });

  it("deletes a stage after confirming", async () => {
    const user = userEvent.setup();
    mockJobsService.deleteStage.mockResolvedValue(undefined);
    render(<Timeline stages={[stage()]} jobId="job-1" stageComments={[]} />);

    await user.click(screen.getByRole("button", { name: "jobs.stageOptions" }));
    await user.click(screen.getByText("common.delete"));
    // Confirmation dialog appears with its own delete button.
    expect(screen.getByText("jobs.deleteStage")).toBeInTheDocument();
    const confirm = screen
      .getAllByRole("button", { name: "common.delete" })
      .at(-1)!;
    await user.click(confirm);

    await waitFor(() =>
      expect(mockJobsService.deleteStage).toHaveBeenCalledWith("job-1", "s1"),
    );
  });

  it("shows an error toast when completing a stage fails", async () => {
    const user = userEvent.setup();
    mockJobsService.updateStage.mockRejectedValue(new Error("x"));
    render(
      <Timeline
        stages={[stage({ status: "active" })]}
        jobId="job-1"
        stageComments={[]}
      />,
    );

    await user.click(screen.getByRole("button", { name: /jobs.complete/ }));
    await waitFor(() =>
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "jobs.stageStatusUpdateError",
      ),
    );
  });
});
