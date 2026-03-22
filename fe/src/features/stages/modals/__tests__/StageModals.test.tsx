import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreateStageTemplateModal } from "../CreateStageTemplateModal";

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
  stageTemplatesService: { create: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

describe("CreateStageTemplateModal", () => {
  it("renders when open", () => {
    render(<CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("stages.createTitle")).toBeInTheDocument();
    expect(screen.getByText("stages.createDescription")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <CreateStageTemplateModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(<CreateStageTemplateModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText(/stages.stageName/)).toBeInTheDocument();
    expect(screen.getByText(/stages\.order \*/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
    expect(screen.getByText("common.create")).toBeInTheDocument();
  });
});
