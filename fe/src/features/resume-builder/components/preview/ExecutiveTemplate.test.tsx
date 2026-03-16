import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExecutiveTemplate } from "./ExecutiveTemplate";
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

describe("ExecutiveTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<ExecutiveTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<ExecutiveTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<ExecutiveTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders colored background header bar", () => {
    const { container } = render(<ExecutiveTemplate />);
    const header = container.querySelector(".rounded.px-6.py-4");
    expect(header).toBeTruthy();
    expect(header!.getAttribute("style")).toContain("background-color");
  });

  it("renders name with white text on colored bar", () => {
    const { container } = render(<ExecutiveTemplate />);
    const nameEl = container.querySelector("h1");
    expect(nameEl).toBeTruthy();
    expect(nameEl!.className).toContain("text-white");
  });

  it("renders contact row with white translucent text", () => {
    const { container } = render(<ExecutiveTemplate />);
    const contactRow = container.querySelector(".text-white\\/80");
    expect(contactRow).toBeTruthy();
  });

  it("delegates to TemplateLayout with executive config", () => {
    render(<ExecutiveTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("executive");
  });

  it("hides optional contact fields when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    render(<ExecutiveTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
  });
});
