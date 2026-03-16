import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AddSectionButton } from "./AddSectionButton";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { SectionOrderDTO } from "@/shared/types/resume-builder";

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

vi.mock("../../constants/sectionLabels", () => ({
  SECTION_LABEL_KEYS: {
    experience: "resumeBuilder.sections.experience",
    skills: "resumeBuilder.sections.skills",
  },
}));

const hiddenSection: SectionOrderDTO = {
  section_key: "skills",
  sort_order: 1,
  is_visible: false,
  column: "main",
};

const visibleSection: SectionOrderDTO = {
  section_key: "experience",
  sort_order: 0,
  is_visible: true,
  column: "main",
};

describe("AddSectionButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders nothing when not editable", () => {
    const { container } = render(<AddSectionButton />);
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when all sections are visible", () => {
    const stateWithAll = createMockStoreState({
      section_order: [visibleSection],
    });
    Object.assign(mockState, stateWithAll);

    const { container } = render(<AddSectionButton editable />);
    expect(container.innerHTML).toBe("");
  });

  it("renders button when there are hidden sections", () => {
    const stateWithHidden = createMockStoreState({
      section_order: [visibleSection, hiddenSection],
    });
    Object.assign(mockState, stateWithHidden);

    render(<AddSectionButton editable />);
    expect(
      screen.getByText("resumeBuilder.layout.addSection"),
    ).toBeInTheDocument();
  });

  it("shows hidden section options when button is clicked", async () => {
    const user = userEvent.setup();
    const stateWithHidden = createMockStoreState({
      section_order: [visibleSection, hiddenSection],
    });
    Object.assign(mockState, stateWithHidden);

    render(<AddSectionButton editable />);
    await user.click(screen.getByText("resumeBuilder.layout.addSection"));

    expect(
      screen.getByText("resumeBuilder.sections.skills"),
    ).toBeInTheDocument();
  });

  it("calls setSectionOrder when a hidden section is clicked", async () => {
    const user = userEvent.setup();
    const stateWithHidden = createMockStoreState({
      section_order: [visibleSection, hiddenSection],
    });
    Object.assign(mockState, stateWithHidden);

    render(<AddSectionButton editable />);
    await user.click(screen.getByText("resumeBuilder.layout.addSection"));
    await user.click(screen.getByText("resumeBuilder.sections.skills"));

    expect(mockState.setSectionOrder).toHaveBeenCalledTimes(1);
    const updatedOrder = mockState.setSectionOrder.mock.calls[0][0];
    const skillsEntry = updatedOrder.find(
      (s: SectionOrderDTO) => s.section_key === "skills",
    );
    expect(skillsEntry?.is_visible).toBe(true);
  });

  it("toggles menu open and closed", async () => {
    const user = userEvent.setup();
    const stateWithHidden = createMockStoreState({
      section_order: [visibleSection, hiddenSection],
    });
    Object.assign(mockState, stateWithHidden);

    render(<AddSectionButton editable />);
    const btn = screen.getByText("resumeBuilder.layout.addSection");

    // Open
    await user.click(btn);
    expect(
      screen.getByText("resumeBuilder.sections.skills"),
    ).toBeInTheDocument();

    // Close
    await user.click(btn);
    expect(
      screen.queryByText("resumeBuilder.sections.skills"),
    ).not.toBeInTheDocument();
  });
});
