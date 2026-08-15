import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { StageSelector } from "../StageSelector";
import type { StageTemplateDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const tpl = (id: string, name: string, order: number): StageTemplateDTO => ({
  id,
  name,
  order,
  created_at: "2026-01-01T00:00:00Z",
});

// Out of order on purpose — the selector must present them in `order`.
const templates: StageTemplateDTO[] = [
  tpl("t-offer", "Offer", 3),
  tpl("t-wishlist", "Wishlist", 1),
  tpl("t-screen", "Screening", 2),
];

describe("StageSelector", () => {
  beforeEach(() => vi.clearAllMocks());

  it("lists the user's stages in `order`", () => {
    render(
      <StageSelector
        templates={templates}
        currentStageTemplateId="t-wishlist"
        onSelect={vi.fn()}
        isMoving={false}
      />,
    );
    const options = screen.getAllByRole("option") as HTMLOptionElement[];
    expect(options.map((o) => o.textContent)).toEqual([
      "Wishlist",
      "Screening",
      "Offer",
    ]);
  });

  it("reflects the current column as the selected value", () => {
    render(
      <StageSelector
        templates={templates}
        currentStageTemplateId="t-screen"
        onSelect={vi.fn()}
        isMoving={false}
      />,
    );
    expect(screen.getByRole("combobox")).toHaveValue("t-screen");
  });

  it("calls onSelect with the chosen stage-template id when moved", () => {
    const onSelect = vi.fn();
    render(
      <StageSelector
        templates={templates}
        currentStageTemplateId="t-wishlist"
        onSelect={onSelect}
        isMoving={false}
      />,
    );
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "t-offer" },
    });
    expect(onSelect).toHaveBeenCalledWith("t-offer");
  });

  it("does not fire onSelect when the value is unchanged", () => {
    const onSelect = vi.fn();
    render(
      <StageSelector
        templates={templates}
        currentStageTemplateId="t-screen"
        onSelect={onSelect}
        isMoving={false}
      />,
    );
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "t-screen" },
    });
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("shows a No-stage option and hint when the card has no column and no templates", () => {
    render(
      <StageSelector
        templates={[]}
        currentStageTemplateId={undefined}
        onSelect={vi.fn()}
        isMoving={false}
      />,
    );
    expect(screen.getByText("jobs.board.noStage")).toBeInTheDocument();
    expect(screen.getByText("jobs.noStageTemplates")).toBeInTheDocument();
  });

  it("disables the control while a move is in flight", () => {
    render(
      <StageSelector
        templates={templates}
        currentStageTemplateId="t-wishlist"
        onSelect={vi.fn()}
        isMoving={true}
      />,
    );
    expect(screen.getByRole("combobox")).toBeDisabled();
  });
});
