import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TypographyPopover } from "./TypographyPopover";
import { createMockStoreState } from "../__tests__/testHelpers";

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

vi.mock("./SidebarPopover", () => ({
  SidebarPopover: ({
    isOpen,
    title,
    children,
  }: {
    isOpen: boolean;
    onClose: () => void;
    title: string;
    children: React.ReactNode;
  }) =>
    isOpen ? (
      <div data-testid="sidebar-popover">
        <h3>{title}</h3>
        {children}
      </div>
    ) : null,
}));

describe("TypographyPopover", () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <TypographyPopover isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the popover when open", () => {
    render(<TypographyPopover {...defaultProps} />);
    expect(screen.getByTestId("sidebar-popover")).toBeInTheDocument();
  });

  it("renders font family selector with current value", () => {
    render(<TypographyPopover {...defaultProps} />);
    const select = screen.getByDisplayValue("Georgia");
    expect(select).toBeInTheDocument();
  });

  it("renders free and premium font optgroups", () => {
    const { container } = render(<TypographyPopover {...defaultProps} />);
    const optgroups = container.querySelectorAll("optgroup");
    expect(optgroups).toHaveLength(2);
    expect(optgroups[0].getAttribute("label")).toBe(
      "resumeBuilder.design.freeFonts",
    );
    expect(optgroups[1].getAttribute("label")).toBe(
      "resumeBuilder.design.premiumFonts",
    );
  });

  it("renders free fonts as options", () => {
    render(<TypographyPopover {...defaultProps} />);
    expect(screen.getByText("Georgia")).toBeInTheDocument();
    expect(screen.getByText("Arial")).toBeInTheDocument();
    expect(screen.getByText("Times New Roman")).toBeInTheDocument();
  });

  it("renders premium fonts as options", () => {
    render(<TypographyPopover {...defaultProps} />);
    expect(screen.getByText("Roboto")).toBeInTheDocument();
    expect(screen.getByText("Inter")).toBeInTheDocument();
  });

  it("calls updateDesign when font is changed", async () => {
    const user = userEvent.setup();
    render(<TypographyPopover {...defaultProps} />);

    const select = screen.getByDisplayValue("Georgia");
    await user.selectOptions(select, "Arial");

    expect(mockState.updateDesign).toHaveBeenCalledWith({
      font_family: "Arial",
    });
  });

  it("renders spacing slider with current value", () => {
    render(<TypographyPopover {...defaultProps} />);
    const sliders = screen.getAllByRole("slider");
    const spacingSlider = sliders.find(
      (s) => s.getAttribute("value") === "100",
    );
    expect(spacingSlider).toBeTruthy();
  });

  it("displays spacing percentage in label", () => {
    const { container } = render(<TypographyPopover {...defaultProps} />);
    // The label contains "resumeBuilder.design.spacing" + ": " + "100" + "%"
    // as separate text nodes, so use textContent check
    const labels = container.querySelectorAll("label");
    const spacingLabel = Array.from(labels).find((l) =>
      l.textContent?.includes("100%"),
    );
    expect(spacingLabel).toBeTruthy();
  });

  it("calls updateDesign when spacing changes", () => {
    render(<TypographyPopover {...defaultProps} />);
    const sliders = screen.getAllByRole("slider");
    const spacingSlider = sliders.find(
      (s) => s.getAttribute("value") === "100",
    )!;

    fireEvent.change(spacingSlider, { target: { value: "120" } });

    expect(mockState.updateDesign).toHaveBeenCalledWith({ spacing: 120 });
  });
});
