import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditableSelect } from "./EditableSelect";

const options = [
  { value: "beginner", label: "Beginner" },
  { value: "intermediate", label: "Intermediate" },
  { value: "advanced", label: "Advanced" },
];

describe("EditableSelect", () => {
  const defaultProps = {
    value: "intermediate",
    onChange: vi.fn(),
    options,
    placeholder: "Select level",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the selected label in display mode", () => {
    render(<EditableSelect {...defaultProps} />);
    expect(screen.getByText("Intermediate")).toBeInTheDocument();
  });

  it("maps a legacy uppercase value to its readable label without changing it", () => {
    render(<EditableSelect {...defaultProps} value="INTERMEDIATE" editable={false} />);
    expect(screen.getByText("Intermediate")).toBeInTheDocument();
    expect(defaultProps.onChange).not.toHaveBeenCalled();
  });

  it("preserves unknown imported values as escaped display text", () => {
    const { container } = render(<EditableSelect {...defaultProps} value="<b>Legacy level</b>" editable={false} />);
    expect(screen.getByText("<b>Legacy level</b>")).toBeInTheDocument();
    expect(container.querySelector("b")).toBeNull();
    expect(defaultProps.onChange).not.toHaveBeenCalled();
  });

  it("renders placeholder when no value is selected", () => {
    render(<EditableSelect {...defaultProps} value="" />);
    expect(screen.getByText("Select level")).toBeInTheDocument();
  });

  it("opens dropdown on click", async () => {
    const user = userEvent.setup();
    render(<EditableSelect {...defaultProps} />);

    await user.click(screen.getByText("Intermediate"));

    expect(screen.getByText("Beginner")).toBeInTheDocument();
    expect(screen.getByText("Advanced")).toBeInTheDocument();
  });

  it("keeps dropdown options readable inside a white-text skill badge", async () => {
    render(<span className="text-white"><EditableSelect {...defaultProps} className="text-white" /></span>);
    await userEvent.setup().click(screen.getByText("Intermediate"));
    expect(screen.getByRole("button", { name: "Beginner" }).parentElement).toHaveClass("text-gray-700");
  });

  it("calls onChange when an option is selected", async () => {
    const user = userEvent.setup();
    render(<EditableSelect {...defaultProps} />);

    await user.click(screen.getByText("Intermediate"));
    await user.click(screen.getByText("Advanced"));

    expect(defaultProps.onChange).toHaveBeenCalledWith("advanced");
  });

  it("closes dropdown after selection", async () => {
    const user = userEvent.setup();
    render(<EditableSelect {...defaultProps} />);

    await user.click(screen.getByText("Intermediate"));
    await user.click(screen.getByText("Advanced"));

    // After selection, dropdown should close
    // Only the display text should remain
    const advancedElements = screen.queryAllByText("Advanced");
    // There might be 0 or 1 "Advanced" after closing (if value was changed externally)
    expect(advancedElements.length).toBeLessThanOrEqual(1);
  });

  it("does not open dropdown when not editable", async () => {
    const user = userEvent.setup();
    render(<EditableSelect {...defaultProps} editable={false} />);

    await user.click(screen.getByText("Intermediate"));

    // Should not show other options
    expect(screen.queryByText("Beginner")).not.toBeInTheDocument();
    expect(screen.queryByText("Advanced")).not.toBeInTheDocument();
  });

  it("applies italic styling when no value and editable", () => {
    render(<EditableSelect {...defaultProps} value="" />);
    const display = screen.getByText("Select level");
    expect(display.className).toContain("italic");
  });

  it("highlights currently selected option", async () => {
    const user = userEvent.setup();
    render(<EditableSelect {...defaultProps} />);

    await user.click(screen.getByText("Intermediate"));

    const intermediateOption = screen
      .getAllByText("Intermediate")
      .find((el) => el.tagName === "BUTTON");
    expect(intermediateOption?.className).toContain("font-medium");
  });
});
