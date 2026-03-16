import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

// Mock i18n
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

// Mock store
const mockSetSectionOrder = vi.fn();
let mockSectionOrder: Array<{
  section_key: string;
  is_visible: boolean;
  sort_order: number;
  column: string;
}> = [];

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      resume: { section_order: mockSectionOrder },
      setSectionOrder: mockSetSectionOrder,
    }),
}));

// Mock section labels
vi.mock("../../constants/sectionLabels", () => ({
  SECTION_LABEL_KEYS: {
    skills: "resumeBuilder.sections.skills",
    languages: "resumeBuilder.sections.languages",
  },
}));

import { SectionDivider } from "./SectionDivider";

beforeEach(() => {
  vi.clearAllMocks();
  mockSectionOrder = [];
});

describe("SectionDivider", () => {
  // ---------------------------------------------------------------------------
  // Null returns
  // ---------------------------------------------------------------------------

  it("returns null when not editable", () => {
    mockSectionOrder = [
      {
        section_key: "skills",
        is_visible: false,
        sort_order: 0,
        column: "main",
      },
    ];
    const { container } = render(
      <SectionDivider insertAtOrder={1} editable={false} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("returns null when all sections are visible", () => {
    mockSectionOrder = [
      {
        section_key: "experience",
        is_visible: true,
        sort_order: 0,
        column: "main",
      },
      {
        section_key: "education",
        is_visible: true,
        sort_order: 1,
        column: "main",
      },
    ];
    const { container } = render(
      <SectionDivider insertAtOrder={1} editable={true} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("returns null when section_order is empty", () => {
    mockSectionOrder = [];
    const { container } = render(
      <SectionDivider insertAtOrder={0} editable={true} />,
    );
    expect(container.innerHTML).toBe("");
  });

  // ---------------------------------------------------------------------------
  // Rendering when hidden sections exist
  // ---------------------------------------------------------------------------

  it("renders add button when hidden sections exist", () => {
    mockSectionOrder = [
      {
        section_key: "experience",
        is_visible: true,
        sort_order: 0,
        column: "main",
      },
      {
        section_key: "skills",
        is_visible: false,
        sort_order: 1,
        column: "main",
      },
    ];
    render(<SectionDivider insertAtOrder={1} editable={true} />);
    expect(
      screen.getByRole("button", { name: "resumeBuilder.layout.addSection" }),
    ).toBeInTheDocument();
  });

  it("shows dropdown with hidden sections when button clicked", () => {
    mockSectionOrder = [
      {
        section_key: "experience",
        is_visible: true,
        sort_order: 0,
        column: "main",
      },
      {
        section_key: "skills",
        is_visible: false,
        sort_order: 1,
        column: "main",
      },
      {
        section_key: "languages",
        is_visible: false,
        sort_order: 2,
        column: "main",
      },
    ];
    render(<SectionDivider insertAtOrder={1} editable={true} />);

    fireEvent.click(
      screen.getByRole("button", { name: "resumeBuilder.layout.addSection" }),
    );

    // Should show hidden section buttons
    expect(
      screen.getByText("resumeBuilder.sections.skills"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.sections.languages"),
    ).toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // addSectionAtPosition logic
  // ---------------------------------------------------------------------------

  it("calls setSectionOrder with correct values when adding a section", () => {
    mockSectionOrder = [
      {
        section_key: "experience",
        is_visible: true,
        sort_order: 0,
        column: "main",
      },
      {
        section_key: "education",
        is_visible: true,
        sort_order: 1,
        column: "main",
      },
      {
        section_key: "skills",
        is_visible: false,
        sort_order: 2,
        column: "main",
      },
    ];
    render(<SectionDivider insertAtOrder={1} editable={true} />);

    // Open dropdown
    fireEvent.click(
      screen.getByRole("button", { name: "resumeBuilder.layout.addSection" }),
    );
    // Click skills
    fireEvent.click(screen.getByText("resumeBuilder.sections.skills"));

    expect(mockSetSectionOrder).toHaveBeenCalledTimes(1);
    const result = mockSetSectionOrder.mock.calls[0][0];

    // Skills should be visible now
    const skillsEntry = result.find(
      (e: { section_key: string }) => e.section_key === "skills",
    );
    expect(skillsEntry.is_visible).toBe(true);

    // Sort orders should be normalized (contiguous 0, 1, 2)
    const sortOrders = result.map((e: { sort_order: number }) => e.sort_order);
    expect(sortOrders).toEqual([0, 1, 2]);
  });

  it("shifts existing sections in the same column", () => {
    mockSectionOrder = [
      {
        section_key: "experience",
        is_visible: true,
        sort_order: 0,
        column: "main",
      },
      {
        section_key: "education",
        is_visible: true,
        sort_order: 1,
        column: "main",
      },
      {
        section_key: "skills",
        is_visible: false,
        sort_order: 5,
        column: "main",
      },
    ];
    render(<SectionDivider insertAtOrder={1} editable={true} />);

    fireEvent.click(
      screen.getByRole("button", { name: "resumeBuilder.layout.addSection" }),
    );
    fireEvent.click(screen.getByText("resumeBuilder.sections.skills"));

    const result = mockSetSectionOrder.mock.calls[0][0];

    // Education (was sort_order=1, same column, >= insertAtOrder=1) should have been shifted
    const education = result.find(
      (e: { section_key: string }) => e.section_key === "education",
    );
    // After normalization, education should be after skills (which was inserted at 1)
    const skills = result.find(
      (e: { section_key: string }) => e.section_key === "skills",
    );
    expect(skills.sort_order).toBeLessThan(education.sort_order);
  });

  it("applies correct button color", () => {
    mockSectionOrder = [
      {
        section_key: "skills",
        is_visible: false,
        sort_order: 0,
        column: "main",
      },
    ];
    render(
      <SectionDivider insertAtOrder={0} editable={true} color="#3b82f6" />,
    );
    const button = screen.getByRole("button", {
      name: "resumeBuilder.layout.addSection",
    });
    expect(button.style.backgroundColor).toBe("rgb(59, 130, 246)");
    expect(button.style.borderColor).toBe("rgb(59, 130, 246)");
  });
});
