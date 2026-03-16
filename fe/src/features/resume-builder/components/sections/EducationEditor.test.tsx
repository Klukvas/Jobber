import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EducationEditor } from "./EducationEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { EducationDTO } from "@/shared/types/resume-builder";

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

const sampleEducation: EducationDTO = {
  id: "edu-1",
  institution: "MIT",
  degree: "Bachelor of Science",
  field_of_study: "Computer Science",
  start_date: "2016-09-01",
  end_date: "2020-06-15",
  is_current: false,
  gpa: "3.8",
  description: "Studied algorithms and systems",
  sort_order: 0,
};

describe("EducationEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<EducationEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.education"),
    ).toBeInTheDocument();
  });

  it("shows Add button", () => {
    render(<EducationEditor />);
    expect(
      screen.getByText("resumeBuilder.actions.add"),
    ).toBeInTheDocument();
  });

  it("shows empty state when no educations", () => {
    render(<EducationEditor />);
    expect(
      screen.getByText("resumeBuilder.education.empty"),
    ).toBeInTheDocument();
  });

  it("adds education when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<EducationEditor />);

    await user.click(screen.getByText("resumeBuilder.actions.add"));
    expect(mockState.addEducation).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders education card with degree and institution", () => {
    const stateWithEdu = createMockStoreState({
      educations: [sampleEducation],
    });
    Object.assign(mockState, stateWithEdu);

    render(<EducationEditor />);
    expect(
      screen.getByText("Bachelor of Science - MIT"),
    ).toBeInTheDocument();
  });

  it("displays input fields for expanded education card", () => {
    const stateWithEdu = createMockStoreState({
      educations: [sampleEducation],
    });
    Object.assign(mockState, stateWithEdu);

    render(<EducationEditor />);
    expect(screen.getByDisplayValue("MIT")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("Bachelor of Science"),
    ).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("Computer Science"),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue("3.8")).toBeInTheDocument();
  });

  it("calls removeEducation when remove is clicked", async () => {
    const user = userEvent.setup();
    const stateWithEdu = createMockStoreState({
      educations: [sampleEducation],
    });
    Object.assign(mockState, stateWithEdu);

    render(<EducationEditor />);
    await user.click(screen.getByText("resumeBuilder.actions.remove"));

    expect(mockState.removeEducation).toHaveBeenCalledWith("edu-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("edu-1");
  });

  it("hides end date when is_current is true", () => {
    const currentEdu: EducationDTO = {
      ...sampleEducation,
      is_current: true,
      end_date: "",
    };
    const stateWithCurrent = createMockStoreState({
      educations: [currentEdu],
    });
    Object.assign(mockState, stateWithCurrent);

    render(<EducationEditor />);
    expect(
      screen.queryByLabelText("resumeBuilder.education.endDate"),
    ).not.toBeInTheDocument();
  });

  it("shows fallback title for empty education", () => {
    const emptyEdu: EducationDTO = {
      ...sampleEducation,
      degree: "",
      institution: "",
    };
    const stateWithEmpty = createMockStoreState({
      educations: [emptyEdu],
    });
    Object.assign(mockState, stateWithEmpty);

    render(<EducationEditor />);
    expect(
      screen.getByText("resumeBuilder.education.newEntry"),
    ).toBeInTheDocument();
  });

  it("toggles card open/closed", async () => {
    const user = userEvent.setup();
    const stateWithEdu = createMockStoreState({
      educations: [sampleEducation],
    });
    Object.assign(mockState, stateWithEdu);

    render(<EducationEditor />);
    const toggleBtn = screen.getByText("Bachelor of Science - MIT");
    await user.click(toggleBtn);

    expect(screen.queryByDisplayValue("MIT")).not.toBeInTheDocument();
  });
});
