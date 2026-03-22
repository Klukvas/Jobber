import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LayoutPresetThumbnail } from "./LayoutPresetThumbnail";

describe("LayoutPresetThumbnail", () => {
  it("renders a button with the label", () => {
    render(
      <LayoutPresetThumbnail
        mode="single"
        isActive={false}
        onClick={vi.fn()}
        label="Single"
      />,
    );
    expect(screen.getByText("Single")).toBeInTheDocument();
  });

  it("renders SVG with single column layout", () => {
    const { container } = render(
      <LayoutPresetThumbnail
        mode="single"
        isActive={false}
        onClick={vi.fn()}
        label="Single"
      />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders SVG with double-left layout", () => {
    const { container } = render(
      <LayoutPresetThumbnail
        mode="double-left"
        isActive={false}
        onClick={vi.fn()}
        label="Two Column Left"
      />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders SVG with double-right layout", () => {
    const { container } = render(
      <LayoutPresetThumbnail
        mode="double-right"
        isActive={false}
        onClick={vi.fn()}
        label="Two Column Right"
      />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders SVG with custom layout", () => {
    const { container } = render(
      <LayoutPresetThumbnail
        mode="custom"
        isActive={false}
        onClick={vi.fn()}
        label="Custom"
      />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("applies active border styling when isActive is true", () => {
    const { container } = render(
      <LayoutPresetThumbnail
        mode="single"
        isActive={true}
        onClick={vi.fn()}
        label="Single"
      />,
    );
    const button = container.querySelector("button");
    expect(button?.className).toContain("border-primary");
  });

  it("calls onClick when button is clicked", async () => {
    const onClick = vi.fn();
    const user = userEvent.setup();
    render(
      <LayoutPresetThumbnail
        mode="single"
        isActive={false}
        onClick={onClick}
        label="Single"
      />,
    );
    await user.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });
});
