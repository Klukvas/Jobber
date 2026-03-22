import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { TwoColumnLayout } from "./TwoColumnLayout";

describe("TwoColumnLayout", () => {
  it("renders sidebar on the left for double-left mode", () => {
    render(
      <TwoColumnLayout
        sidebarWidth={35}
        layoutMode="double-left"
        mainContent={<div data-testid="main">Main</div>}
        sidebarContent={<div data-testid="sidebar">Sidebar</div>}
      />,
    );
    expect(screen.getByTestId("main")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
  });

  it("renders sidebar on the right for double-right mode", () => {
    render(
      <TwoColumnLayout
        sidebarWidth={35}
        layoutMode="double-right"
        mainContent={<div data-testid="main">Main</div>}
        sidebarContent={<div data-testid="sidebar">Sidebar</div>}
      />,
    );
    expect(screen.getByTestId("main")).toBeInTheDocument();
    expect(screen.getByTestId("sidebar")).toBeInTheDocument();
  });

  it("applies correct width percentages", () => {
    const { container } = render(
      <TwoColumnLayout
        sidebarWidth={30}
        layoutMode="double-left"
        mainContent={<span>Main</span>}
        sidebarContent={<span>Sidebar</span>}
      />,
    );
    const flexContainer = container.firstChild as HTMLElement;
    const children = flexContainer.children;
    expect((children[0] as HTMLElement).style.width).toBe("30%");
    expect((children[1] as HTMLElement).style.width).toBe("70%");
  });
});
