import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { VividTemplate } from "./VividTemplate";
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

describe("VividTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<VividTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<VividTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<VividTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders two-tone header (colored top + white bottom)", () => {
    const { container } = render(<VividTemplate />);
    // Colored top section
    const coloredTop = container.querySelector("[style*='background-color']");
    expect(coloredTop).toBeTruthy();
    // White contact row with border
    const contactRow = container.querySelector(".border-t-0");
    expect(contactRow).toBeTruthy();
  });

  it("renders contact icon badges as colored circles", () => {
    const { container } = render(<VividTemplate />);
    const iconBadges = container.querySelectorAll(".rounded-full.text-white");
    expect(iconBadges.length).toBeGreaterThanOrEqual(6);
  });

  it("renders contact SVG icons", () => {
    const { container } = render(<VividTemplate />);
    const svgs = container.querySelectorAll("svg");
    expect(svgs.length).toBeGreaterThanOrEqual(6);
  });

  it("delegates to TemplateLayout with vivid config", () => {
    render(<VividTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout.getAttribute("data-variant")).toBe("vivid");
  });

  it("wraps header in rounded-lg overflow-hidden container", () => {
    const { container } = render(<VividTemplate />);
    const wrapper = container.querySelector(".rounded-lg.overflow-hidden");
    expect(wrapper).toBeTruthy();
  });
});
