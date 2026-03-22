import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { SectionIcon } from "./SectionIcon";

function MockIcon({ className }: { className?: string }) {
  return <svg data-testid="mock-icon" className={className} />;
}

describe("SectionIcon", () => {
  it("renders the icon inside a colored circle", () => {
    const { container } = render(
      <SectionIcon icon={MockIcon} color="#2563eb" />,
    );
    expect(screen.getByTestId("mock-icon")).toBeInTheDocument();
    const span = container.querySelector("span");
    expect(span?.style.backgroundColor).toBe("rgb(37, 99, 235)");
  });

  it("applies the correct background color", () => {
    const { container } = render(
      <SectionIcon icon={MockIcon} color="#ff0000" />,
    );
    const span = container.querySelector("span");
    expect(span?.style.backgroundColor).toBe("rgb(255, 0, 0)");
  });
});
