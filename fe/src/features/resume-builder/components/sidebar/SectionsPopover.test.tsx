import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SectionsPopover } from "./SectionsPopover";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("./SidebarPopover", () => ({
  SidebarPopover: ({
    isOpen,
    children,
    title,
  }: {
    isOpen: boolean;
    children: React.ReactNode;
    title: string;
    onClose: () => void;
  }) =>
    isOpen ? (
      <div data-testid="sidebar-popover">
        <h3>{title}</h3>
        {children}
      </div>
    ) : null,
}));

vi.mock("./SectionsConfigurator", () => ({
  SectionsConfigurator: () => (
    <div data-testid="sections-configurator">SectionsConfigurator</div>
  ),
}));

describe("SectionsPopover", () => {
  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <SectionsPopover isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the popover with title and configurator when open", () => {
    render(<SectionsPopover isOpen={true} onClose={vi.fn()} />);
    expect(
      screen.getByText("resumeBuilder.sections.title"),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId("sections-configurator"),
    ).toBeInTheDocument();
  });
});
