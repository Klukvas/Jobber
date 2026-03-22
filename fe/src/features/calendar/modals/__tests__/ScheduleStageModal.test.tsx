import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ScheduleStageModal } from "../ScheduleStageModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQuery: () => ({ data: { connected: false } }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/services/calendarService", () => ({
  calendarService: {
    getStatus: vi.fn(),
    getOAuthURL: vi.fn(),
    createEvent: vi.fn(),
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

describe("ScheduleStageModal", () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    stageId: "stage-1",
    stageName: "Phone Screen",
    applicationId: "app-1",
  };

  it("renders when open", () => {
    render(<ScheduleStageModal {...defaultProps} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <ScheduleStageModal {...defaultProps} open={false} />,
    );
    expect(container.innerHTML).toBe("");
  });
});
