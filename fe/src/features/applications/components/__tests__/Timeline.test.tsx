import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Timeline } from "../Timeline";
import type { ApplicationStageDTO, CommentDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (params) {
        return Object.entries(params).reduce(
          (acc, [k, v]) => acc.replace(`{{${k}}}`, v),
          key,
        );
      }
      return key;
    },
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));

vi.mock("@/shared/lib/notifications", () => ({
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/lib/features", () => ({
  FEATURES: { GOOGLE_CALENDAR: false },
}));

vi.mock("../../modals/UpdateStageStatusModal", () => ({
  UpdateStageStatusModal: () => null,
}));

vi.mock("../../modals/AddCommentModal", () => ({
  AddCommentModal: () => null,
}));

vi.mock("@/features/calendar/modals/ScheduleStageModal", () => ({
  ScheduleStageModal: () => null,
}));

const makeStage = (
  overrides: Partial<ApplicationStageDTO> = {},
): ApplicationStageDTO => ({
  id: "stage-1",
  application_id: "app-1",
  stage_template_id: "tpl-1",
  stage_name: "Phone Screen",
  order: 1,
  status: "active",
  started_at: "2025-01-01T00:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  ...overrides,
});

const makeComment = (
  overrides: Partial<CommentDTO> = {},
): CommentDTO => ({
  id: "comment-1",
  application_id: "app-1",
  content: "Great interview!",
  created_at: "2025-01-02T00:00:00Z",
  ...overrides,
});

describe("Timeline", () => {
  const defaultProps = {
    stages: [] as ApplicationStageDTO[],
    applicationId: "app-1",
    stageComments: [] as CommentDTO[],
  };

  it("shows empty message when no stages or comments", () => {
    render(<Timeline {...defaultProps} />);
    expect(screen.getByText("applications.noStagesYet")).toBeInTheDocument();
  });

  it("renders stage name", () => {
    render(
      <Timeline
        {...defaultProps}
        stages={[makeStage({ stage_name: "Technical Interview" })]}
      />,
    );
    expect(screen.getByText("Technical Interview")).toBeInTheDocument();
  });

  it("renders stage status badge", () => {
    render(
      <Timeline
        {...defaultProps}
        stages={[makeStage({ status: "completed" })]}
      />,
    );
    expect(
      screen.getByText("applications.stageStatusCompleted"),
    ).toBeInTheDocument();
  });

  it("renders started time", () => {
    render(
      <Timeline {...defaultProps} stages={[makeStage()]} />,
    );
    expect(screen.getByText(/applications.started/)).toBeInTheDocument();
  });

  it("renders completed time when completed_at exists", () => {
    render(
      <Timeline
        {...defaultProps}
        stages={[
          makeStage({
            status: "completed",
            completed_at: "2025-01-05T00:00:00Z",
          }),
        ]}
      />,
    );
    expect(screen.getByText(/applications.completed/)).toBeInTheDocument();
  });

  it("renders comment content", () => {
    render(
      <Timeline
        {...defaultProps}
        stageComments={[makeComment({ content: "Went well" })]}
      />,
    );
    expect(screen.getByText("Went well")).toBeInTheDocument();
  });

  it("renders multiple stages in order", () => {
    const stages = [
      makeStage({
        id: "s1",
        stage_name: "Applied",
        started_at: "2025-01-01T00:00:00Z",
      }),
      makeStage({
        id: "s2",
        stage_name: "Phone Screen",
        started_at: "2025-01-05T00:00:00Z",
      }),
    ];
    render(<Timeline {...defaultProps} stages={stages} />);
    expect(screen.getByText("Applied")).toBeInTheDocument();
    expect(screen.getByText("Phone Screen")).toBeInTheDocument();
  });

  it("shows complete button for active stages", () => {
    render(
      <Timeline
        {...defaultProps}
        stages={[makeStage({ status: "active" })]}
      />,
    );
    expect(screen.getByText("applications.complete")).toBeInTheDocument();
  });

  it("does not show complete button for completed stages", () => {
    render(
      <Timeline
        {...defaultProps}
        stages={[
          makeStage({
            status: "completed",
            completed_at: "2025-01-05T00:00:00Z",
          }),
        ]}
      />,
    );
    expect(
      screen.queryByText("applications.complete"),
    ).not.toBeInTheDocument();
  });
});
