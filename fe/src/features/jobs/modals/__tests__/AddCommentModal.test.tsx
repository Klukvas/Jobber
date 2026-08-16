import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AddCommentModal } from "../AddCommentModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars?.stageName ? `${key}:${vars.stageName}` : key,
    i18n: { language: "en" },
  }),
}));

const mockMutate = vi.hoisted(() => vi.fn());
const mockMutationState = vi.hoisted(() => ({ isPending: false }));

vi.mock("@tanstack/react-query", () => ({
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

const mockCommentsService = vi.hoisted(() => ({ create: vi.fn() }));
vi.mock("@/services/commentsService", () => ({
  commentsService: mockCommentsService,
}));

const mockNotifications = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/shared/lib/notifications", () => mockNotifications);

describe("AddCommentModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutationState.isPending = false;
  });

  it("shows the general description when no stage is provided", () => {
    render(<AddCommentModal open onOpenChange={vi.fn()} jobId="job-1" />);
    expect(screen.getByText("jobs.addCommentGeneral")).toBeInTheDocument();
  });

  it("shows a stage-scoped description when a stage is provided", () => {
    render(
      <AddCommentModal
        open
        onOpenChange={vi.fn()}
        jobId="job-1"
        stageId="stage-1"
        stageName="Screening"
      />,
    );
    expect(
      screen.getByText("jobs.addCommentForStage:Screening"),
    ).toBeInTheDocument();
  });

  it("creates a trimmed comment with the stage id and closes on success", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockCommentsService.create.mockResolvedValue({ id: "cm-1" });

    render(
      <AddCommentModal
        open
        onOpenChange={onOpenChange}
        jobId="job-1"
        stageId="stage-9"
      />,
    );

    await user.type(screen.getByLabelText(/jobs.commentLabel/), "  hi there  ");
    await user.click(screen.getByRole("button", { name: "jobs.addComment" }));

    await waitFor(() =>
      expect(mockCommentsService.create).toHaveBeenCalledWith({
        job_id: "job-1",
        stage_id: "stage-9",
        content: "hi there",
      }),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(mockNotifications.showSuccessNotification).toHaveBeenCalledWith(
      "jobs.commentAddedSuccess",
    );
  });

  it("does not submit when the comment is only whitespace", async () => {
    const user = userEvent.setup();
    render(<AddCommentModal open onOpenChange={vi.fn()} jobId="job-1" />);

    await user.type(screen.getByLabelText(/jobs.commentLabel/), "   ");
    await user.click(screen.getByRole("button", { name: "jobs.addComment" }));

    expect(mockCommentsService.create).not.toHaveBeenCalled();
  });

  it("surfaces the error message on failure", async () => {
    const user = userEvent.setup();
    mockCommentsService.create.mockRejectedValue(new Error("server down"));

    render(<AddCommentModal open onOpenChange={vi.fn()} jobId="job-1" />);

    await user.type(screen.getByLabelText(/jobs.commentLabel/), "hello");
    await user.click(screen.getByRole("button", { name: "jobs.addComment" }));

    await waitFor(() =>
      expect(mockNotifications.showErrorNotification).toHaveBeenCalledWith(
        "server down",
      ),
    );
  });

  it("disables the submit button and shows a spinner while pending", () => {
    mockMutationState.isPending = true;
    render(<AddCommentModal open onOpenChange={vi.fn()} jobId="job-1" />);
    expect(screen.getByText("jobs.adding")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /jobs.adding/ })).toBeDisabled();
  });
});
