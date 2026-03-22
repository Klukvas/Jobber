import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { SectionsConfigurator } from "./SectionsConfigurator";
import { createMockStoreState } from "../__tests__/testHelpers";

const mockState = createMockStoreState({
  section_order: [
    { section_key: "contact", sort_order: 0, is_visible: true },
    { section_key: "summary", sort_order: 1, is_visible: true },
    { section_key: "experience", sort_order: 2, is_visible: true },
    { section_key: "education", sort_order: 3, is_visible: true },
    { section_key: "skills", sort_order: 4, is_visible: true },
    { section_key: "languages", sort_order: 5, is_visible: false },
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

vi.mock("./layoutPresets", () => ({
  LAYOUT_PRESETS: {},
}));

vi.mock("../../constants/sectionLabels", () => ({
  SECTION_LABEL_KEYS: {
    contact: "resumeBuilder.sections.contact",
    summary: "resumeBuilder.sections.summary",
    experience: "resumeBuilder.sections.experience",
    education: "resumeBuilder.sections.education",
    skills: "resumeBuilder.sections.skills",
    languages: "resumeBuilder.sections.languages",
  },
}));

describe("SectionsConfigurator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState({
      section_order: [
        { section_key: "contact", sort_order: 0, is_visible: true },
        { section_key: "summary", sort_order: 1, is_visible: true },
        { section_key: "experience", sort_order: 2, is_visible: true },
        { section_key: "education", sort_order: 3, is_visible: true },
        { section_key: "skills", sort_order: 4, is_visible: true },
        { section_key: "languages", sort_order: 5, is_visible: false },
      ],
    });
    Object.assign(mockState, freshState);
  });

  it("renders without crash", () => {
    const { container } = render(<SectionsConfigurator />);
    expect(container.innerHTML).not.toBe("");
  });

  it("renders section toggles for summary and contact", () => {
    render(<SectionsConfigurator />);
    // Summary and contact appear in multiple places (toggles + preset cards)
    const summaryElements = screen.getAllByText(
      "resumeBuilder.sections.summary",
    );
    expect(summaryElements.length).toBeGreaterThanOrEqual(1);
    const contactElements = screen.getAllByText(
      "resumeBuilder.sections.contact",
    );
    expect(contactElements.length).toBeGreaterThanOrEqual(1);
  });
});
