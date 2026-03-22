import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { AddCommentModal } from "../AddCommentModal";
import { AddStageModal } from "../AddStageModal";
import { UpdateApplicationStatusModal } from "../UpdateApplicationStatusModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params) {
        return Object.entries(params).reduce(
          (acc, [k, v]) => acc.replace(`{{${k}}}`, String(v)),
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
    isError: false,
  }),
  useQuery: () => ({
    data: { items: [] },
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/services/commentsService", () => ({
  commentsService: { create: vi.fn() },
}));

vi.mock("@/services/applicationsService", () => ({
  applicationsService: {
    addStage: vi.fn(),
    update: vi.fn(),
  },
}));

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: { list: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

// ---------- AddCommentModal ----------
describe("AddCommentModal", () => {
  it("renders when open", () => {
    render(
      <AddCommentModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    // Title and submit button both show "applications.addComment"
    expect(
      screen.getAllByText("applications.addComment").length,
    ).toBeGreaterThanOrEqual(1);
    expect(
      screen.getByText("applications.addCommentGeneral"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <AddCommentModal
        open={false}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders stage-specific description when stageId is provided", () => {
    render(
      <AddCommentModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
        stageId="stage-1"
        stageName="Phone Screen"
      />,
    );
    expect(
      screen.getByText(/applications.addCommentForStage/),
    ).toBeInTheDocument();
  });

  it("renders comment textarea and buttons", () => {
    render(
      <AddCommentModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    expect(screen.getByText(/applications.commentLabel/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});

// ---------- AddStageModal ----------
describe("AddStageModal", () => {
  it("renders when open", () => {
    render(
      <AddStageModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    expect(screen.getByText("applications.addStageTitle")).toBeInTheDocument();
    expect(
      screen.getByText("applications.addStageDescription"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <AddStageModal
        open={false}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders stage select and comment input", () => {
    render(
      <AddStageModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    // "applications.selectStage" appears in both label and option text
    expect(
      screen.getAllByText(/applications\.selectStage/).length,
    ).toBeGreaterThanOrEqual(1);
    expect(
      screen.getByText("applications.commentOptional"),
    ).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });

  it("shows no templates message when empty", () => {
    render(
      <AddStageModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
      />,
    );
    expect(
      screen.getByText("applications.noStageTemplates"),
    ).toBeInTheDocument();
  });
});

// ---------- UpdateApplicationStatusModal ----------
describe("UpdateApplicationStatusModal", () => {
  it("renders when open", () => {
    render(
      <UpdateApplicationStatusModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
        currentStatus="active"
      />,
    );
    expect(
      screen.getByText("applications.changeStatusTitle"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("applications.changeStatusDescription"),
    ).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <UpdateApplicationStatusModal
        open={false}
        onOpenChange={vi.fn()}
        applicationId="app-1"
        currentStatus="active"
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders current status and new status select", () => {
    render(
      <UpdateApplicationStatusModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
        currentStatus="active"
      />,
    );
    expect(screen.getByText("applications.currentStatus")).toBeInTheDocument();
    expect(screen.getByText(/applications.newStatus/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
    expect(screen.getByText("applications.updateStatus")).toBeInTheDocument();
  });
});
