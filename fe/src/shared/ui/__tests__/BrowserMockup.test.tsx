import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { BrowserMockup } from "../BrowserMockup";

describe("BrowserMockup", () => {
  const defaultProps = {
    url: "https://example.com",
    src: "/screenshot.png",
    alt: "Screenshot",
  };

  it("renders image with alt text", () => {
    render(<BrowserMockup {...defaultProps} />);
    const img = screen.getByAltText("Screenshot");
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute("src", "/screenshot.png");
  });

  it("renders URL in address bar", () => {
    render(<BrowserMockup {...defaultProps} />);
    expect(screen.getByText("https://example.com")).toBeInTheDocument();
  });

  it("renders light variant by default", () => {
    const { container } = render(<BrowserMockup {...defaultProps} />);
    // Light variant has bg-muted/60 in the toolbar
    expect(container.querySelector(".bg-muted\\/60")).toBeTruthy();
  });

  it("renders dark variant when dark=true", () => {
    const { container } = render(<BrowserMockup {...defaultProps} dark />);
    // Dark variant has bg-slate-800/80 in the toolbar
    expect(container.querySelector(".bg-slate-800\\/80")).toBeTruthy();
  });
});
