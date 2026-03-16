import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ColorPickerPopover } from "./ColorPickerPopover";
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

vi.mock("./colorPalettes", () => ({
  COLOR_PALETTES: [
    {
      nameKey: "resumeBuilder.colors.corporate",
      colors: ["#1e3a5f", "#1d4ed8"],
    },
    {
      nameKey: "resumeBuilder.colors.creative",
      colors: ["#e11d48", "#9333ea"],
    },
  ],
  TEMPLATE_RECOMMENDED_PALETTE: {
    professional: "resumeBuilder.colors.corporate",
  },
}));

describe("ColorPickerPopover", () => {
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
      <ColorPickerPopover isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the color picker when open", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    expect(screen.getByTestId("sidebar-popover")).toBeInTheDocument();
  });

  it("renders both panel and text color sections", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    expect(
      screen.getByText("resumeBuilder.design.panelColor"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.design.textColor"),
    ).toBeInTheDocument();
  });

  it("renders palette names in both sections", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    // Each palette appears twice (once per section)
    expect(screen.getAllByText("resumeBuilder.colors.corporate")).toHaveLength(
      2,
    );
    expect(screen.getAllByText("resumeBuilder.colors.creative")).toHaveLength(
      2,
    );
  });

  it("renders color swatch buttons in both sections", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    // Each color appears twice (once in panel section, once in text section)
    expect(screen.getAllByLabelText("#1e3a5f")).toHaveLength(2);
    expect(screen.getAllByLabelText("#e11d48")).toHaveLength(2);
  });

  it("calls updateDesign with primary_color when panel swatch is clicked", async () => {
    const user = userEvent.setup();
    render(<ColorPickerPopover {...defaultProps} />);

    // First occurrence is in the panel color section
    const swatches = screen.getAllByLabelText("#e11d48");
    await user.click(swatches[0]);

    expect(mockState.updateDesign).toHaveBeenCalledWith({
      primary_color: "#e11d48",
    });
  });

  it("calls updateDesign with text_color when text swatch is clicked", async () => {
    const user = userEvent.setup();
    render(<ColorPickerPopover {...defaultProps} />);

    // Second occurrence is in the text color section
    const swatches = screen.getAllByLabelText("#e11d48");
    await user.click(swatches[1]);

    expect(mockState.updateDesign).toHaveBeenCalledWith({
      text_color: "#e11d48",
    });
  });

  it("shows recommended badge for recommended palette", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    // Badge appears in both sections
    expect(
      screen.getAllByText("resumeBuilder.colors.recommended"),
    ).toHaveLength(2);
  });

  it("renders custom color section for both panel and text", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    expect(screen.getAllByText("resumeBuilder.colors.custom")).toHaveLength(2);
  });

  it("renders color inputs with current colors", () => {
    render(<ColorPickerPopover {...defaultProps} />);
    // Panel color section: 1 color input + 1 text input = 2 inputs with #2563eb
    // Text color section uses text_color (#111827), not primary_color
    const inputs = screen.getAllByDisplayValue("#2563eb");
    expect(inputs.length).toBe(2);
  });
});
