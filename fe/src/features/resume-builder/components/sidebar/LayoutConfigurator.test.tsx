import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { LayoutConfigurator } from "./LayoutConfigurator";
import { createMockStoreState } from "../__tests__/testHelpers";

const mockState = createMockStoreState({
  section_order: [
    { section_key: "contact", sort_order: 0, is_visible: true, column: "main" },
    {
      section_key: "experience",
      sort_order: 1,
      is_visible: true,
      column: "main",
    },
    { section_key: "skills", sort_order: 2, is_visible: true, column: "main" },
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
  LAYOUT_PRESETS: {
    single: {
      layout_mode: "single",
      sidebar_width: 35,
      assignments: {},
    },
    "double-left": {
      layout_mode: "double-left",
      sidebar_width: 35,
      assignments: { skills: "sidebar", contact: "sidebar" },
    },
    "double-right": {
      layout_mode: "double-right",
      sidebar_width: 35,
      assignments: { skills: "sidebar" },
    },
  },
}));

vi.mock("./LayoutPresetThumbnail", () => ({
  LayoutPresetThumbnail: ({ mode }: { mode: string }) => (
    <div data-testid={`thumbnail-${mode}`}>Thumbnail</div>
  ),
}));

vi.mock("../../constants/sectionLabels", () => ({
  SECTION_LABEL_KEYS: {
    contact: "resumeBuilder.sections.contact",
    experience: "resumeBuilder.sections.experience",
    skills: "resumeBuilder.sections.skills",
  },
}));

describe("LayoutConfigurator", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState({
      section_order: [
        {
          section_key: "contact",
          sort_order: 0,
          is_visible: true,
          column: "main",
        },
        {
          section_key: "experience",
          sort_order: 1,
          is_visible: true,
          column: "main",
        },
        {
          section_key: "skills",
          sort_order: 2,
          is_visible: true,
          column: "main",
        },
      ],
    });
    Object.assign(mockState, freshState);
  });

  it("renders nothing when resume is null", () => {
    Object.assign(mockState, { resume: null });
    const { container } = render(<LayoutConfigurator />);
    expect(container.innerHTML).toBe("");
  });

  it("renders layout presets label", () => {
    render(<LayoutConfigurator />);
    expect(
      screen.getByText("resumeBuilder.layout.layoutPresets"),
    ).toBeInTheDocument();
  });

  it("renders layout preset thumbnails", () => {
    render(<LayoutConfigurator />);
    expect(screen.getByTestId("thumbnail-single")).toBeInTheDocument();
    expect(screen.getByTestId("thumbnail-double-left")).toBeInTheDocument();
    expect(screen.getByTestId("thumbnail-double-right")).toBeInTheDocument();
    expect(screen.getByTestId("thumbnail-custom")).toBeInTheDocument();
  });
});
