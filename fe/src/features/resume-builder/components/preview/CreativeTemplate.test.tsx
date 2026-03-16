import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreativeTemplate } from "./CreativeTemplate";
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

describe("CreativeTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<CreativeTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<CreativeTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<CreativeTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders initials badge circle with colored background", () => {
    const { container } = render(<CreativeTemplate />);
    const badge = container.querySelector(".rounded-full.text-white");
    expect(badge).toBeTruthy();
    expect(badge!.getAttribute("style")).toContain("background-color");
    expect(badge!.textContent).toBe("JD");
  });

  it("renders initials from single-name contact", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Alice", email: "a@b.com", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    const { container } = render(<CreativeTemplate />);
    const badge = container.querySelector(".rounded-full.text-white");
    expect(badge!.textContent).toBe("A");
  });

  it("renders header with flex layout (badge + name side by side)", () => {
    const { container } = render(<CreativeTemplate />);
    const headerFlex = container.querySelector(".flex.items-center.gap-4");
    expect(headerFlex).toBeTruthy();
  });

  it("delegates to TemplateLayout with creative config", () => {
    render(<CreativeTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("creative");
  });

  it("hides optional contact fields when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    render(<CreativeTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
  });
});
