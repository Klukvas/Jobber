import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { BoldTemplate } from "./BoldTemplate";
import { createMockSetup } from "./__tests__/templateTestHelpers";

let mockReturnValue: ReturnType<typeof createMockSetup> | null = createMockSetup();

vi.mock("./shared/useTemplateSetup", () => ({
  useTemplateSetup: () => mockReturnValue,
}));

vi.mock("./shared/TemplateSections", () => ({
  TemplateLayout: ({ config }: { config: { variant: string } }) => (
    <div data-testid="template-layout" data-variant={config.variant} />
  ),
}));

describe("BoldTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<BoldTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<BoldTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<BoldTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders colored banner header", () => {
    const { container } = render(<BoldTemplate />);
    const banner = container.querySelector(".rounded-lg");
    expect(banner).toBeTruthy();
    expect(banner!.getAttribute("style")).toContain("background-color");
  });

  it("renders contact icons as SVGs", () => {
    const { container } = render(<BoldTemplate />);
    const svgs = container.querySelectorAll("svg");
    expect(svgs.length).toBeGreaterThanOrEqual(6);
  });

  it("delegates to TemplateLayout with bold config", () => {
    render(<BoldTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("bold");
  });

  it("hides optional contact fields when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    render(<BoldTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
  });
});
