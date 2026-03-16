import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ElegantTemplate } from "./ElegantTemplate";
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

describe("ElegantTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<ElegantTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<ElegantTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<ElegantTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders pipe separators between contact fields", () => {
    const { container } = render(<ElegantTemplate />);
    const pipes = container.querySelectorAll(".text-gray-300");
    // pipes between phone, location, website, linkedin, github (5 pipes after email)
    expect(pipes.length).toBe(5);
    expect(pipes[0].textContent).toBe("|");
  });

  it("renders a colored horizontal rule divider", () => {
    const { container } = render(<ElegantTemplate />);
    const hr = container.querySelector("hr");
    expect(hr).toBeTruthy();
    expect(hr!.getAttribute("style")).toContain("border-color");
  });

  it("renders name with colored styling", () => {
    const { container } = render(<ElegantTemplate />);
    const nameEl = container.querySelector("h1");
    expect(nameEl).toBeTruthy();
    expect(nameEl!.getAttribute("style")).toContain("color");
  });

  it("delegates to TemplateLayout with elegant config", () => {
    render(<ElegantTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("elegant");
  });

  it("hides optional contact fields and pipe separators when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    const { container } = render(<ElegantTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
    const pipes = container.querySelectorAll(".text-gray-300");
    expect(pipes.length).toBe(0);
  });
});
