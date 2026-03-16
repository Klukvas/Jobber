import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { TimelineTemplate } from "./TimelineTemplate";
import { createMockSetup } from "./__tests__/templateTestHelpers";

let mockReturnValue: ReturnType<typeof createMockSetup> | null =
  createMockSetup();

vi.mock("./shared/useTemplateSetup", () => ({
  useTemplateSetup: () => mockReturnValue,
}));

vi.mock("./shared/TemplateSections", () => ({
  TemplateLayout: ({ config }: { config: { variant: string } }) => (
    <div data-testid="template-layout" data-variant={config.variant} />
  ),
}));

describe("TimelineTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<TimelineTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<TimelineTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<TimelineTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
  });

  it("renders colored underline accent", () => {
    const { container } = render(<TimelineTemplate />);
    const underline = container.querySelector(".h-1.rounded-full");
    expect(underline).toBeTruthy();
    expect(underline!.getAttribute("style")).toContain("background-color");
  });

  it("does not render contact icons (clean header)", () => {
    const { container } = render(<TimelineTemplate />);
    const svgs = container.querySelectorAll("svg");
    expect(svgs.length).toBe(0);
  });

  it("delegates to TemplateLayout with timeline config", () => {
    render(<TimelineTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout.getAttribute("data-variant")).toBe("timeline");
  });
});
