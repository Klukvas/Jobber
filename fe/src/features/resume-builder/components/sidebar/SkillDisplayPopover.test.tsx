import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { SkillDisplayPopover } from "./SkillDisplayPopover";
import {
  createMockStoreState,
} from "../__tests__/testHelpers";

const mockState = createMockStoreState();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
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
    fullscreen?: boolean;
  }) =>
    isOpen ? (
      <div data-testid="sidebar-popover">
        <h3>{title}</h3>
        {children}
      </div>
    ) : null,
}));

describe("SkillDisplayPopover", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <SkillDisplayPopover isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the popover with title when open", () => {
    render(<SkillDisplayPopover isOpen={true} onClose={vi.fn()} />);
    expect(
      screen.getByText("resumeBuilder.skillDisplay.title"),
    ).toBeInTheDocument();
  });

  it("renders all 13 skill display option buttons", () => {
    render(<SkillDisplayPopover isOpen={true} onClose={vi.fn()} />);
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(13);
  });

  it("renders template default option", () => {
    render(<SkillDisplayPopover isOpen={true} onClose={vi.fn()} />);
    expect(
      screen.getByText("resumeBuilder.skillDisplay.templateDefault"),
    ).toBeInTheDocument();
  });

  it("renders pill option", () => {
    render(<SkillDisplayPopover isOpen={true} onClose={vi.fn()} />);
    expect(
      screen.getByText("resumeBuilder.skillDisplay.pill"),
    ).toBeInTheDocument();
  });

  it("renders bar option", () => {
    render(<SkillDisplayPopover isOpen={true} onClose={vi.fn()} />);
    expect(
      screen.getByText("resumeBuilder.skillDisplay.bar"),
    ).toBeInTheDocument();
  });
});
