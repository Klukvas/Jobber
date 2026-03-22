import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { EditStageTemplateModal } from "../EditStageTemplateModal";
import type { StageTemplateDTO } from "@/shared/types/api";

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

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: { update: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

const mockTemplate: StageTemplateDTO = {
  id: "st1",
  name: "Phone Screen",
  order: 1,
  created_at: "2025-01-01T00:00:00Z",
};

describe("EditStageTemplateModal", () => {
  it("renders when open with template", () => {
    render(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={mockTemplate}
      />,
    );
    expect(screen.getByText("stages.edit")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <EditStageTemplateModal
        open={false}
        onOpenChange={vi.fn()}
        template={mockTemplate}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders cancel and save buttons", () => {
    render(
      <EditStageTemplateModal
        open={true}
        onOpenChange={vi.fn()}
        template={mockTemplate}
      />,
    );
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
    expect(screen.getByText("common.save")).toBeInTheDocument();
  });
});
