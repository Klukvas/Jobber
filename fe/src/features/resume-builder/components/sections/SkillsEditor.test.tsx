import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SkillsEditor } from "./SkillsEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { SkillDTO } from "@/shared/types/resume-builder";

const mockState = createMockStoreState();
const mockPersistAdd = vi.fn();
const mockPersistRemove = vi.fn();

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

vi.mock("../../hooks/useSectionPersistence", () => ({
  useSectionPersistence: () => ({
    add: mockPersistAdd,
    remove: mockPersistRemove,
  }),
}));

const sampleSkill: SkillDTO = {
  id: "skill-1",
  name: "TypeScript",
  level: "advanced",
  sort_order: 0,
};

describe("SkillsEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<SkillsEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.skills"),
    ).toBeInTheDocument();
  });

  it("shows Add button", () => {
    render(<SkillsEditor />);
    expect(screen.getByText("resumeBuilder.actions.add")).toBeInTheDocument();
  });

  it("shows empty state when no skills", () => {
    render(<SkillsEditor />);
    expect(screen.getByText("resumeBuilder.skills.empty")).toBeInTheDocument();
  });

  it("adds a skill when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<SkillsEditor />);

    await user.click(screen.getByText("resumeBuilder.actions.add"));
    expect(mockState.addSkill).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders skill entry with name and level", () => {
    const stateWithSkills = createMockStoreState({
      skills: [sampleSkill],
    });
    Object.assign(mockState, stateWithSkills);

    render(<SkillsEditor />);
    expect(screen.getByDisplayValue("TypeScript")).toBeInTheDocument();
    // The select has value="advanced" but the displayed text is the i18n key
    const levelSelect = screen.getByDisplayValue(
      "resumeBuilder.skills.levels.advanced",
    );
    expect(levelSelect).toBeInTheDocument();
  });

  it("renders level select with all options", () => {
    const stateWithSkills = createMockStoreState({
      skills: [sampleSkill],
    });
    Object.assign(mockState, stateWithSkills);

    render(<SkillsEditor />);
    const select = screen.getByDisplayValue(
      "resumeBuilder.skills.levels.advanced",
    );
    expect(select).toBeInTheDocument();
    expect(select.tagName).toBe("SELECT");
  });

  it("calls updateSkill when name changes", async () => {
    const user = userEvent.setup();
    const stateWithSkills = createMockStoreState({
      skills: [sampleSkill],
    });
    Object.assign(mockState, stateWithSkills);

    render(<SkillsEditor />);
    const nameInput = screen.getByDisplayValue("TypeScript");
    await user.clear(nameInput);
    await user.type(nameInput, "React");

    expect(mockState.updateSkill).toHaveBeenCalled();
  });

  it("calls removeSkill when remove button is clicked", async () => {
    const user = userEvent.setup();
    const stateWithSkills = createMockStoreState({
      skills: [sampleSkill],
    });
    Object.assign(mockState, stateWithSkills);

    render(<SkillsEditor />);
    const removeBtn = screen.getByLabelText("resumeBuilder.actions.remove");
    await user.click(removeBtn);

    expect(mockState.removeSkill).toHaveBeenCalledWith("skill-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("skill-1");
  });

  it("does not show empty state when skills exist", () => {
    const stateWithSkills = createMockStoreState({
      skills: [sampleSkill],
    });
    Object.assign(mockState, stateWithSkills);

    render(<SkillsEditor />);
    expect(
      screen.queryByText("resumeBuilder.skills.empty"),
    ).not.toBeInTheDocument();
  });
});
