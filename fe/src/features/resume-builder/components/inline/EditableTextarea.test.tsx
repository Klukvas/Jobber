import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditableTextarea } from "./EditableTextarea";

describe("EditableTextarea", () => {
  const defaultProps = {
    value: "A summary of experience",
    onChange: vi.fn(),
    placeholder: "Enter summary",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the value in display mode", () => {
    render(<EditableTextarea {...defaultProps} />);
    expect(
      screen.getByText("A summary of experience"),
    ).toBeInTheDocument();
  });

  it("renders placeholder when value is empty", () => {
    render(<EditableTextarea {...defaultProps} value="" />);
    expect(screen.getByText("Enter summary")).toBeInTheDocument();
  });

  it("has textbox role when editable", () => {
    render(<EditableTextarea {...defaultProps} />);
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("does not have textbox role when not editable", () => {
    render(<EditableTextarea {...defaultProps} editable={false} />);
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it("enters edit mode on click", async () => {
    const user = userEvent.setup();
    render(<EditableTextarea {...defaultProps} />);

    await user.click(screen.getByText("A summary of experience"));

    const textarea = screen.getByRole("textbox");
    expect(textarea.tagName).toBe("TEXTAREA");
  });

  it("commits value on blur", async () => {
    const user = userEvent.setup();
    render(<EditableTextarea {...defaultProps} />);

    await user.click(screen.getByText("A summary of experience"));
    const textarea = screen.getByDisplayValue("A summary of experience");
    await user.clear(textarea);
    await user.type(textarea, "Updated summary");
    await user.tab();

    expect(defaultProps.onChange).toHaveBeenCalledWith("Updated summary");
  });

  it("cancels edit on Escape", async () => {
    const user = userEvent.setup();
    render(<EditableTextarea {...defaultProps} />);

    await user.click(screen.getByText("A summary of experience"));
    const textarea = screen.getByDisplayValue("A summary of experience");
    await user.clear(textarea);
    await user.type(textarea, "Changed");
    await user.keyboard("{Escape}");

    expect(defaultProps.onChange).not.toHaveBeenCalled();
    expect(
      screen.getByText("A summary of experience"),
    ).toBeInTheDocument();
  });

  it("does not call onChange when value has not changed", async () => {
    const user = userEvent.setup();
    render(<EditableTextarea {...defaultProps} />);

    await user.click(screen.getByText("A summary of experience"));
    await user.tab();

    expect(defaultProps.onChange).not.toHaveBeenCalled();
  });

  it("does not enter edit mode when not editable", async () => {
    const user = userEvent.setup();
    render(<EditableTextarea {...defaultProps} editable={false} />);

    await user.click(screen.getByText("A summary of experience"));
    expect(
      screen.queryByDisplayValue("A summary of experience"),
    ).not.toBeInTheDocument();
  });

  it("applies italic and gray styling when empty and editable", () => {
    render(<EditableTextarea {...defaultProps} value="" />);
    const element = screen.getByText("Enter summary");
    expect(element.className).toContain("italic");
    expect(element.className).toContain("text-gray-400");
  });

  it("renders as a p element in display mode", () => {
    render(<EditableTextarea {...defaultProps} />);
    const display = screen.getByText("A summary of experience");
    expect(display.tagName).toBe("P");
  });
});
