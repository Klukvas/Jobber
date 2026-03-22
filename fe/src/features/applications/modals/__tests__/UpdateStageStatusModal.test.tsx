import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { UpdateStageStatusModal } from "../UpdateStageStatusModal";
import type { ApplicationStageDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/services/applicationsService", () => ({
  applicationsService: { updateStage: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

const mockStage: ApplicationStageDTO = {
  id: "stage-1",
  application_id: "app-1",
  stage_template_id: "tpl-1",
  stage_name: "Phone Screen",
  order: 1,
  status: "active",
  started_at: "2025-01-01T00:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
};

describe("UpdateStageStatusModal", () => {
  it("renders when open", () => {
    render(
      <UpdateStageStatusModal
        open={true}
        onOpenChange={vi.fn()}
        applicationId="app-1"
        stage={mockStage}
      />,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <UpdateStageStatusModal
        open={false}
        onOpenChange={vi.fn()}
        applicationId="app-1"
        stage={mockStage}
      />,
    );
    expect(container.innerHTML).toBe("");
  });
});
