import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SidebarPopover } from "./SidebarPopover";

describe("SidebarPopover", () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    title: "Test Panel",
    children: <div>Panel Content</div>,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <SidebarPopover {...defaultProps} isOpen={false} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the title when open", () => {
    render(<SidebarPopover {...defaultProps} />);
    expect(screen.getByText("Test Panel")).toBeInTheDocument();
  });

  it("renders children when open", () => {
    render(<SidebarPopover {...defaultProps} />);
    expect(screen.getByText("Panel Content")).toBeInTheDocument();
  });

  it("renders close button", () => {
    render(<SidebarPopover {...defaultProps} />);
    expect(screen.getByLabelText("Close")).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", async () => {
    const user = userEvent.setup();
    render(<SidebarPopover {...defaultProps} />);

    await user.click(screen.getByLabelText("Close"));
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it("renders with proper heading structure", () => {
    render(<SidebarPopover {...defaultProps} />);
    const heading = screen.getByText("Test Panel");
    expect(heading.tagName).toBe("H3");
  });
});
