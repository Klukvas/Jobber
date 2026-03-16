import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { IconicTemplate } from "./IconicTemplate";
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

describe("IconicTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<IconicTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<IconicTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<IconicTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders colored icon badges as small circles", () => {
    const { container } = render(<IconicTemplate />);
    const iconBadges = container.querySelectorAll(".rounded-full.text-white");
    // 6 contact fields = 6 icon badges
    expect(iconBadges.length).toBe(6);
    expect(iconBadges[0].getAttribute("style")).toContain("background-color");
  });

  it("renders lucide SVG icons inside badges", () => {
    const { container } = render(<IconicTemplate />);
    const svgs = container.querySelectorAll("svg");
    expect(svgs.length).toBeGreaterThanOrEqual(6);
  });

  it("renders name with colored styling", () => {
    const { container } = render(<IconicTemplate />);
    const nameEl = container.querySelector("h1");
    expect(nameEl).toBeTruthy();
    expect(nameEl!.getAttribute("style")).toContain("color");
  });

  it("delegates to TemplateLayout with iconic config", () => {
    render(<IconicTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("iconic");
  });

  it("hides optional contact fields and icon badges when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    const { container } = render(<IconicTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
    const iconBadges = container.querySelectorAll(".rounded-full.text-white");
    expect(iconBadges.length).toBe(0);
  });
});
