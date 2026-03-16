import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { DesignSidebar } from "./DesignSidebar";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("./TemplatePickerPopover", () => ({
  TemplatePickerPopover: ({
    isOpen,
  }: {
    isOpen: boolean;
    onClose: () => void;
  }) => (isOpen ? <div data-testid="template-popover">Template</div> : null),
}));

vi.mock("./ColorPickerPopover", () => ({
  ColorPickerPopover: ({ isOpen }: { isOpen: boolean; onClose: () => void }) =>
    isOpen ? <div data-testid="color-popover">Color</div> : null,
}));

vi.mock("./TypographyPopover", () => ({
  TypographyPopover: ({ isOpen }: { isOpen: boolean; onClose: () => void }) =>
    isOpen ? <div data-testid="typography-popover">Typography</div> : null,
}));

vi.mock("./LayoutPopover", () => ({
  LayoutPopover: ({ isOpen }: { isOpen: boolean; onClose: () => void }) =>
    isOpen ? <div data-testid="layout-popover">Layout</div> : null,
}));

vi.mock("./SectionsPopover", () => ({
  SectionsPopover: ({ isOpen }: { isOpen: boolean; onClose: () => void }) =>
    isOpen ? <div data-testid="sections-popover">Sections</div> : null,
}));

vi.mock("./SkillDisplayPopover", () => ({
  SkillDisplayPopover: ({
    isOpen,
  }: {
    isOpen: boolean;
    onClose: () => void;
  }) =>
    isOpen ? (
      <div data-testid="skill-display-popover">Skill Display</div>
    ) : null,
}));

describe("DesignSidebar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders six sidebar icon buttons", () => {
    render(<DesignSidebar />);
    const buttons = screen.getAllByRole("button");
    expect(buttons).toHaveLength(6);
  });

  it("opens template popover when template button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const templateBtn = screen.getByLabelText(
      "resumeBuilder.sidebar.templates",
    );
    await user.click(templateBtn);

    expect(screen.getByTestId("template-popover")).toBeInTheDocument();
  });

  it("opens color popover when color button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const colorBtn = screen.getByLabelText("resumeBuilder.sidebar.colors");
    await user.click(colorBtn);

    expect(screen.getByTestId("color-popover")).toBeInTheDocument();
  });

  it("opens typography popover when typography button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const typoBtn = screen.getByLabelText("resumeBuilder.sidebar.typography");
    await user.click(typoBtn);

    expect(screen.getByTestId("typography-popover")).toBeInTheDocument();
  });

  it("opens layout popover when layout button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const layoutBtn = screen.getByLabelText("resumeBuilder.sidebar.layout");
    await user.click(layoutBtn);

    expect(screen.getByTestId("layout-popover")).toBeInTheDocument();
  });

  it("opens sections popover when sections button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const sectionsBtn = screen.getByLabelText(
      "resumeBuilder.sidebar.sections",
    );
    await user.click(sectionsBtn);

    expect(screen.getByTestId("sections-popover")).toBeInTheDocument();
  });

  it("opens skill display popover when skill display button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const btn = screen.getByLabelText("resumeBuilder.sidebar.skillDisplay");
    await user.click(btn);

    expect(screen.getByTestId("skill-display-popover")).toBeInTheDocument();
  });

  it("closes popover when the same button is clicked again (toggle)", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const templateBtn = screen.getByLabelText(
      "resumeBuilder.sidebar.templates",
    );
    await user.click(templateBtn);
    expect(screen.getByTestId("template-popover")).toBeInTheDocument();

    await user.click(templateBtn);
    expect(screen.queryByTestId("template-popover")).not.toBeInTheDocument();
  });

  it("switches popover when a different button is clicked", async () => {
    const user = userEvent.setup();
    render(<DesignSidebar />);

    const templateBtn = screen.getByLabelText(
      "resumeBuilder.sidebar.templates",
    );
    const colorBtn = screen.getByLabelText("resumeBuilder.sidebar.colors");

    await user.click(templateBtn);
    expect(screen.getByTestId("template-popover")).toBeInTheDocument();

    await user.click(colorBtn);
    expect(screen.queryByTestId("template-popover")).not.toBeInTheDocument();
    expect(screen.getByTestId("color-popover")).toBeInTheDocument();
  });

  it("no popover is open by default", () => {
    render(<DesignSidebar />);
    expect(screen.queryByTestId("template-popover")).not.toBeInTheDocument();
    expect(screen.queryByTestId("color-popover")).not.toBeInTheDocument();
    expect(screen.queryByTestId("typography-popover")).not.toBeInTheDocument();
    expect(screen.queryByTestId("layout-popover")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sections-popover")).not.toBeInTheDocument();
  });
});
