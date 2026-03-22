import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { SectionOrderPanel } from "./SectionOrderPanel";
import { createMockStoreState } from "./__tests__/testHelpers";

const mockState = createMockStoreState({
  section_order: [
    { section_key: "contact", sort_order: 0, is_visible: true },
    { section_key: "summary", sort_order: 1, is_visible: true },
    { section_key: "experience", sort_order: 2, is_visible: false },
  ],
});

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

vi.mock("../constants/sectionLabels", () => ({
  SECTION_LABEL_KEYS: {
    contact: "resumeBuilder.sections.contact",
    summary: "resumeBuilder.sections.summary",
    experience: "resumeBuilder.sections.experience",
  },
}));

describe("SectionOrderPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState({
      section_order: [
        { section_key: "contact", sort_order: 0, is_visible: true },
        { section_key: "summary", sort_order: 1, is_visible: true },
        { section_key: "experience", sort_order: 2, is_visible: false },
      ],
    });
    Object.assign(mockState, freshState);
  });

  it("renders the section order heading", () => {
    render(<SectionOrderPanel />);
    expect(
      screen.getByText("resumeBuilder.design.sectionOrder"),
    ).toBeInTheDocument();
  });

  it("renders section items", () => {
    render(<SectionOrderPanel />);
    expect(
      screen.getByText("resumeBuilder.sections.contact"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.sections.summary"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.sections.experience"),
    ).toBeInTheDocument();
  });

  it("renders move up and move down buttons for each section", () => {
    render(<SectionOrderPanel />);
    const upButtons = screen.getAllByLabelText("Move up");
    const downButtons = screen.getAllByLabelText("Move down");
    expect(upButtons).toHaveLength(3);
    expect(downButtons).toHaveLength(3);
  });

  it("disables move up button for the first section", () => {
    render(<SectionOrderPanel />);
    const upButtons = screen.getAllByLabelText("Move up");
    expect(upButtons[0]).toBeDisabled();
  });

  it("disables move down button for the last section", () => {
    render(<SectionOrderPanel />);
    const downButtons = screen.getAllByLabelText("Move down");
    expect(downButtons[downButtons.length - 1]).toBeDisabled();
  });

  it("renders visibility toggle buttons for each section", () => {
    render(<SectionOrderPanel />);
    // Two visible sections get "Hide section" label, one hidden gets "Show section"
    const hideButtons = screen.getAllByLabelText("Hide section");
    const showButtons = screen.getAllByLabelText("Show section");
    expect(hideButtons).toHaveLength(2);
    expect(showButtons).toHaveLength(1);
  });
});
