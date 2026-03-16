import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { ProfessionalTemplate } from "./ProfessionalTemplate";
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

describe("ProfessionalTemplate", () => {
  beforeEach(() => {
    mockReturnValue = createMockSetup();
  });

  it("renders null when setup returns null", () => {
    mockReturnValue = null;
    const { container } = render(<ProfessionalTemplate />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the full name", () => {
    render(<ProfessionalTemplate />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
  });

  it("renders all contact fields", () => {
    render(<ProfessionalTemplate />);
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
    expect(screen.getByText("+1234567890")).toBeInTheDocument();
    expect(screen.getByText("NYC")).toBeInTheDocument();
    expect(screen.getByText("jane.dev")).toBeInTheDocument();
    expect(screen.getByText("linkedin.com/in/jane")).toBeInTheDocument();
    expect(screen.getByText("github.com/jane")).toBeInTheDocument();
  });

  it("renders center-aligned header with colored name", () => {
    const { container } = render(<ProfessionalTemplate />);
    const header = container.querySelector(".text-center");
    expect(header).toBeTruthy();
    const nameEl = container.querySelector("h1");
    expect(nameEl).toBeTruthy();
    expect(nameEl!.getAttribute("style")).toContain("color");
  });

  it("delegates to TemplateLayout with professional config", () => {
    render(<ProfessionalTemplate />);
    const layout = screen.getByTestId("template-layout");
    expect(layout).toBeInTheDocument();
    expect(layout.getAttribute("data-variant")).toBe("professional");
  });

  it("hides optional contact fields when empty and not editable", () => {
    mockReturnValue = createMockSetup({
      contact: { full_name: "Jane Doe", email: "", phone: "", location: "", website: "", linkedin: "", github: "" },
    });
    render(<ProfessionalTemplate editable={false} />);
    expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    expect(screen.queryByText("jane@example.com")).not.toBeInTheDocument();
  });
});
