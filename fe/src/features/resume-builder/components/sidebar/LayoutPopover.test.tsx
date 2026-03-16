import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LayoutPopover } from "./LayoutPopover";
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

vi.mock("../SectionOrderPanel", () => ({
  SectionOrderPanel: () => (
    <div data-testid="section-order-panel">Section Order</div>
  ),
}));

vi.mock("./LayoutConfigurator", () => ({
  LayoutConfigurator: () => (
    <div data-testid="layout-configurator">Layout Configurator</div>
  ),
}));

describe("LayoutPopover", () => {
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
      <LayoutPopover isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the popover when open", () => {
    render(<LayoutPopover {...defaultProps} />);
    expect(screen.getByTestId("sidebar-popover")).toBeInTheDocument();
  });

  it("renders resume title input with current value", () => {
    render(<LayoutPopover {...defaultProps} />);
    const titleInput = screen.getByDisplayValue("My Resume");
    expect(titleInput).toBeInTheDocument();
  });

  it("calls updateDesign when title changes", async () => {
    const user = userEvent.setup();
    render(<LayoutPopover {...defaultProps} />);

    const titleInput = screen.getByDisplayValue("My Resume");
    await user.clear(titleInput);
    await user.type(titleInput, "New Title");

    expect(mockState.updateDesign).toHaveBeenCalled();
  });

  it("renders margin inputs", () => {
    render(<LayoutPopover {...defaultProps} />);
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

  it("renders margin inputs with correct values", () => {
    render(<LayoutPopover {...defaultProps} />);
    const numberInputs = screen.getAllByRole("spinbutton");
    // Should have 4 margin inputs
    expect(numberInputs).toHaveLength(4);
    numberInputs.forEach((input) => {
      expect(input).toHaveValue(40);
    });
  });

  it("calls updateDesign when margin changes", () => {
    render(<LayoutPopover {...defaultProps} />);
    const numberInputs = screen.getAllByRole("spinbutton");

    fireEvent.change(numberInputs[0], { target: { value: "50" } });

    expect(mockState.updateDesign).toHaveBeenCalledWith({ margin_top: 50 });
  });

  it("renders the LayoutConfigurator component", () => {
    render(<LayoutPopover {...defaultProps} />);
    expect(screen.getByTestId("layout-configurator")).toBeInTheDocument();
  });

  it("renders the SectionOrderPanel component", () => {
    render(<LayoutPopover {...defaultProps} />);
    expect(screen.getByTestId("section-order-panel")).toBeInTheDocument();
  });
});
