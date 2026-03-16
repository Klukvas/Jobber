import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MinimalTemplate } from "./MinimalTemplate";
import { createMockSetup } from "./__tests__/templateTestHelpers";

let mockReturnValue: ReturnType<typeof createMockSetup> | null = createMockSetup();

vi.mock("./shared/useTemplateSetup", () => ({
  useTemplateSetup: () => mockReturnValue,
}));

vi.mock("./shared/TemplateSections", () => ({
  TemplateLayout: ({ config }: { config: { variant: string } }) => (
    <div data-testid="template-layout" data-variant={config.variant} />
  ),
  SectionRenderer: () => <div data-testid="section-renderer" />,
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

describe("MinimalTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<MinimalTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<MinimalTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders name with light font weight", () => {
    const { container } = render(<MinimalTemplate />);
    const nameEl = container.querySelector(".font-light");
    expect(nameEl).toBeTruthy();
    expect(nameEl!.textContent).toBe("Jane Doe");
  });

  it("renders all contact fields in single-column mode", () => {
    render(<MinimalTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders a colored horizontal rule divider", () => {
    const { container } = render(<MinimalTemplate />);
    const hr = container.querySelector("hr");
    expect(hr).toBeTruthy();
    expect(hr!.getAttribute("style")).toContain("border-color");
  });

  it("delegates to TemplateLayout with minimal config in single-column mode", () => {
    render(<MinimalTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("minimal");
  });

  it("renders two-column layout when isTwoColumn is true", () => {
    mockReturnValue = createMockSetup({
      isTwoColumn: true,
      layoutMode: "double-left",
      sidebarSections: [{ section_key: "contact", sort_order: 1 }],
      mainSections: [],
    });
    render(<MinimalTemplate />);
    expect(screen.getByTestId("two-column-layout")).toBeInTheDocument();
  });

  it("hides contact fields inline but shows them in sidebar in two-column mode", () => {
    mockReturnValue = createMockSetup({
      isTwoColumn: true,
      layoutMode: "double-left",
      sidebarSections: [{ section_key: "contact", sort_order: 1 }],
      mainSections: [],
    });
    render(<MinimalTemplate />);
    const sidebar = screen.getByTestId("sidebar-content");
    expect(sidebar).toHaveTextContent("jane@example.com");
  });

  it("hides optional contact fields when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    render(<MinimalTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
  });
});
