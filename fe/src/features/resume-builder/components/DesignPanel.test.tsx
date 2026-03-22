import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { DesignPanel } from "./DesignPanel";
import { createMockStoreState } from "./__tests__/testHelpers";

const mockState = createMockStoreState();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

vi.mock("./SectionOrderPanel", () => ({
  SectionOrderPanel: () => (
    <div data-testid="section-order-panel">SectionOrderPanel</div>
  ),
}));

vi.mock("../lib/templateRegistry", () => ({
  TEMPLATE_LIST: [
    {
      id: "00000000-0000-0000-0000-000000000001",
      nameKey: "resumeBuilder.templates.professional",
    },
    {
      id: "00000000-0000-0000-0000-000000000002",
      nameKey: "resumeBuilder.templates.modern",
    },
  ],
}));

describe("DesignPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the design section heading", () => {
    render(<DesignPanel />);
    expect(
      screen.getByText("resumeBuilder.sections.design"),
    ).toBeInTheDocument();
  });

  it("renders title input with resume title value", () => {
    render(<DesignPanel />);
    const titleInput = screen.getByLabelText("resumeBuilder.design.title");
    expect(titleInput).toBeInTheDocument();
    expect(titleInput).toHaveValue("My Resume");
  });

  it("renders template selection buttons", () => {
    render(<DesignPanel />);
    expect(
      screen.getByText("resumeBuilder.templates.professional"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.templates.modern"),
    ).toBeInTheDocument();
  });

  it("renders color label", () => {
    render(<DesignPanel />);
    expect(screen.getByText("resumeBuilder.design.color")).toBeInTheDocument();
  });

  it("renders font selection", () => {
    render(<DesignPanel />);
    const fontSelect = screen.getByLabelText("resumeBuilder.design.font");
    expect(fontSelect).toBeInTheDocument();
  });

  it("renders spacing slider", () => {
    render(<DesignPanel />);
    expect(
      screen.getByText("resumeBuilder.design.spacing: 100%"),
    ).toBeInTheDocument();
  });

  it("renders margin inputs", () => {
    render(<DesignPanel />);
    expect(
      screen.getByText("resumeBuilder.design.margins"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.design.margin_top"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.design.margin_bottom"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.design.margin_left"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.design.margin_right"),
    ).toBeInTheDocument();
  });

  it("renders the SectionOrderPanel", () => {
    render(<DesignPanel />);
    expect(screen.getByTestId("section-order-panel")).toBeInTheDocument();
  });

  it("renders nothing when resume is null", () => {
    Object.assign(mockState, { resume: null });
    const { container } = render(<DesignPanel />);
    expect(container.innerHTML).toBe("");
  });
});
