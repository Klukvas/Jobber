import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EntryWrapper } from "./EntryWrapper";

describe("EntryWrapper", () => {
  const defaultProps = {
    entryId: "entry-1",
    onRemove: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders children", () => {
    render(
      <EntryWrapper {...defaultProps}>
        <div>Child Content</div>
      </EntryWrapper>,
    );
    expect(screen.getByText("Child Content")).toBeInTheDocument();
  });

  it("renders remove button when editable", () => {
    render(
      <EntryWrapper {...defaultProps}>
        <div>Content</div>
      </EntryWrapper>,
    );
    expect(screen.getByLabelText("Remove entry")).toBeInTheDocument();
  });

  it("calls onRemove with entry ID when remove button is clicked", async () => {
    const user = userEvent.setup();
    render(
      <EntryWrapper {...defaultProps}>
        <div>Content</div>
      </EntryWrapper>,
    );

    await user.click(screen.getByLabelText("Remove entry"));
    expect(defaultProps.onRemove).toHaveBeenCalledWith("entry-1");
  });

  it("does not render remove button when not editable", () => {
    render(
      <EntryWrapper {...defaultProps} editable={false}>
        <div>Content</div>
      </EntryWrapper>,
    );
    expect(screen.queryByLabelText("Remove entry")).not.toBeInTheDocument();
  });

  it("renders as plain div when not editable", () => {
    const { container } = render(
      <EntryWrapper {...defaultProps} editable={false}>
        <div>Content</div>
      </EntryWrapper>,
    );
    // When not editable, no group hover class
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).not.toContain("group");
  });

  it("applies custom className", () => {
    const { container } = render(
      <EntryWrapper {...defaultProps} className="custom-class">
        <div>Content</div>
      </EntryWrapper>,
    );
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).toContain("custom-class");
  });

  it("has hover group class when editable", () => {
    const { container } = render(
      <EntryWrapper {...defaultProps}>
        <div>Content</div>
      </EntryWrapper>,
    );
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).toContain("group");
  });
});
