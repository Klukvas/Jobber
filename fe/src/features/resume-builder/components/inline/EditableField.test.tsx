import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditableField } from "./EditableField";

describe("EditableField", () => {
  const defaultProps = {
    value: "John Doe",
    onChange: vi.fn(),
    placeholder: "Enter name",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the value in display mode", () => {
    render(<EditableField {...defaultProps} />);
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("renders placeholder when value is empty", () => {
    render(<EditableField {...defaultProps} value="" />);
    expect(screen.getByText("Enter name")).toBeInTheDocument();
  });

  it("has textbox role when editable", () => {
    render(<EditableField {...defaultProps} />);
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("does not have textbox role when not editable", () => {
    render(<EditableField {...defaultProps} editable={false} />);
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it("enters edit mode on click", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} />);

    await user.click(screen.getByText("John Doe"));

    const input = screen.getByRole("textbox");
    expect(input.tagName).toBe("INPUT");
  });

  it("commits value on blur", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} />);

    await user.click(screen.getByText("John Doe"));
    const input = screen.getByDisplayValue("John Doe");
    await user.clear(input);
    await user.type(input, "Jane Smith");
    await user.tab(); // blur

    expect(defaultProps.onChange).toHaveBeenCalledWith("Jane Smith");
  });

  it("commits value on Enter key", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} />);

    await user.click(screen.getByText("John Doe"));
    const input = screen.getByDisplayValue("John Doe");
    await user.clear(input);
    await user.type(input, "New Name{Enter}");

    expect(defaultProps.onChange).toHaveBeenCalledWith("New Name");
  });

  it("cancels edit on Escape key", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} />);

    await user.click(screen.getByText("John Doe"));
    const input = screen.getByDisplayValue("John Doe");
    await user.clear(input);
    await user.type(input, "Changed");
    await user.keyboard("{Escape}");

    expect(defaultProps.onChange).not.toHaveBeenCalled();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("does not call onChange when value has not changed", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} />);

    await user.click(screen.getByText("John Doe"));
    const input = screen.getByDisplayValue("John Doe");
    await user.tab(); // blur without changing

    expect(defaultProps.onChange).not.toHaveBeenCalled();
  });

  it("does not enter edit mode when editable is false", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} editable={false} />);

    await user.click(screen.getByText("John Doe"));
    expect(screen.queryByDisplayValue("John Doe")).not.toBeInTheDocument();
  });

  it("renders with custom tag", () => {
    render(<EditableField {...defaultProps} as="h1" />);
    const heading = screen.getByText("John Doe");
    expect(heading.tagName).toBe("H1");
  });

  it("applies italic styling when empty and editable", () => {
    render(<EditableField {...defaultProps} value="" />);
    const element = screen.getByText("Enter name");
    expect(element.className).toContain("italic");
  });

  it("enters edit mode on Enter key in display mode", async () => {
    const user = userEvent.setup();
    render(<EditableField {...defaultProps} />);

    const display = screen.getByRole("textbox");
    display.focus();
    await user.keyboard("{Enter}");

    const input = screen.getByDisplayValue("John Doe");
    expect(input.tagName).toBe("INPUT");
  });
});
