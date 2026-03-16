import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ModernTemplate } from "./ModernTemplate";
import { createMockSetup } from "./__tests__/templateTestHelpers";

let mockReturnValue: ReturnType<typeof createMockSetup> | null = createMockSetup();

vi.mock("./shared/useTemplateSetup", () => ({
  useTemplateSetup: () => mockReturnValue,
}));

vi.mock("./shared/TemplateSections", () => ({
  SectionRenderer: () => <div data-testid="section-renderer" />,
  TemplateLayout: ({ config }: { config: { variant: string } }) => (
    <div data-testid="template-layout" data-variant={config.variant} />
  ),
}));

vi.mock("./TwoColumnLayout", () => ({
  TwoColumnLayout: ({ sidebarContent, mainContent }: { sidebarContent: React.ReactNode; mainContent: React.ReactNode }) => (
    <div data-testid="two-column-layout">
      <div data-testid="sidebar-content">{sidebarContent}</div>
      <div data-testid="main-content">{mainContent}</div>
    </div>
  ),
}));

vi.mock("../inline/SectionDivider", () => ({
  SectionDivider: () => <div data-testid="section-divider" />,
}));

describe("ModernTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<ModernTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name in single-column mode", () => {
    render(<ModernTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders the colored name heading in single-column mode", () => {
    const { container } = render(<ModernTemplate />);
    const nameEl = container.querySelector("h1");
    expect(nameEl).toBeTruthy();
    expect(nameEl!.getAttribute("style")).toContain("color");
  });

  it("renders Contact heading in single-column mode", () => {
    mockReturnValue = createMockSetup({
      visibleSections: [{ section_key: "contact", sort_order: 1 }],
    });
    render(<ModernTemplate />);
    expect(screen.getByText("Contact")).toBeInTheDocument();
  });

  it("renders contact fields in single-column mode", () => {
    mockReturnValue = createMockSetup({
      visibleSections: [{ section_key: "contact", sort_order: 1 }],
    });
    render(<ModernTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
  });

  it("renders two-column layout when isTwoColumn is true", () => {
    mockReturnValue = createMockSetup({
      isTwoColumn: true,
      layoutMode: "double-left",
      sidebarSections: [{ section_key: "contact", sort_order: 1 }],
      mainSections: [],
    });
    render(<ModernTemplate />);
    expect(screen.getByTestId("two-column-layout")).toBeInTheDocument();
  });

  it("renders name and contact in sidebar in two-column mode", () => {
    mockReturnValue = createMockSetup({
      isTwoColumn: true,
      layoutMode: "double-left",
      sidebarSections: [{ section_key: "contact", sort_order: 1 }],
      mainSections: [],
    });
    render(<ModernTemplate />);
    const sidebar = screen.getByTestId("sidebar-content");
    expect(sidebar).toHaveTextContent("Jane Doe");
    expect(sidebar).toHaveTextContent("jane@example.com");
  });

  it("hides optional contact fields when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    render(<ModernTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
  });
});
