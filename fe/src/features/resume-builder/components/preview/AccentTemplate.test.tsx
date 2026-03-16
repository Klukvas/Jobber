import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { AccentTemplate } from "./AccentTemplate";
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

describe("AccentTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<AccentTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<AccentTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<AccentTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
  });

  it("renders accent left border on name", () => {
    const { container } = render(<AccentTemplate />);
    const accentBar = container.querySelector('[class*="border-l-"]');
    expect(accentBar).toBeTruthy();
    expect(accentBar!.getAttribute("style")).toContain("border-color");
  });

  it("renders contact row with top border separator", () => {
    const { container } = render(<AccentTemplate />);
    const contactRow = container.querySelector(".border-t");
    expect(contactRow).toBeTruthy();
  });

  it("delegates to TemplateLayout with accent config", () => {
    render(<AccentTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout.getAttribute("data-variant")).toBe("accent");
  });

  it("does not render icons (clean design)", () => {
    const { container } = render(<AccentTemplate />);
    const svgs = container.querySelectorAll("svg");
    expect(svgs.length).toBe(0);
  });
});
