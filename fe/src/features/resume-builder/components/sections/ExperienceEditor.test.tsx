import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ExperienceEditor } from "./ExperienceEditor";
import {
  createMockStoreState,
} from "../__tests__/testHelpers";
import type { ExperienceDTO } from "@/shared/types/resume-builder";

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

const sampleExperience: ExperienceDTO = {
  id: "exp-1",
  company: "Acme Corp",
  position: "Software Engineer",
  location: "San Francisco",
  start_date: "2020-01-15",
  end_date: "2023-06-30",
  is_current: false,
  description: "Built web applications",
  sort_order: 0,
};

describe("ExperienceEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
    mockPersistAdd.mockReset();
    mockPersistRemove.mockReset();
  });

  it("renders the section heading", () => {
    render(<ExperienceEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.experience"),
    ).toBeInTheDocument();
  });

  it("shows the Add button", () => {
    render(<ExperienceEditor />);
    expect(
      screen.getByText("resumeBuilder.actions.add"),
    ).toBeInTheDocument();
  });

  it("shows empty state message when no experiences", () => {
    render(<ExperienceEditor />);
    expect(
      screen.getByText("resumeBuilder.experience.empty"),
    ).toBeInTheDocument();
  });

  it("adds an experience entry when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<ExperienceEditor />);

    await user.click(screen.getByText("resumeBuilder.actions.add"));

    expect(mockState.addExperience).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders experience cards when data exists", () => {
    const stateWithExps = createMockStoreState({
      experiences: [sampleExperience],
    });
    Object.assign(mockState, stateWithExps);

    render(<ExperienceEditor />);
    expect(
      screen.getByText("Software Engineer - Acme Corp"),
    ).toBeInTheDocument();
  });

  it("shows input fields when an experience card is expanded", () => {
    const stateWithExps = createMockStoreState({
      experiences: [sampleExperience],
    });
    Object.assign(mockState, stateWithExps);

    render(<ExperienceEditor />);
    // Cards start open (isOpen defaults to true)
    expect(
      screen.getByDisplayValue("Acme Corp"),
    ).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("Software Engineer"),
    ).toBeInTheDocument();
  });

  it("toggles experience card open/closed on click", async () => {
    const user = userEvent.setup();
    const stateWithExps = createMockStoreState({
      experiences: [sampleExperience],
    });
    Object.assign(mockState, stateWithExps);

    render(<ExperienceEditor />);
    // Initially open, click to collapse
    const toggleButton = screen.getByText("Software Engineer - Acme Corp");
    await user.click(toggleButton);

    // After collapsing, the company input should not be visible
    expect(screen.queryByDisplayValue("Acme Corp")).not.toBeInTheDocument();
  });

  it("calls removeExperience and persistRemove when remove is clicked", async () => {
    const user = userEvent.setup();
    const stateWithExps = createMockStoreState({
      experiences: [sampleExperience],
    });
    Object.assign(mockState, stateWithExps);

    render(<ExperienceEditor />);
    await user.click(screen.getByText("resumeBuilder.actions.remove"));

    expect(mockState.removeExperience).toHaveBeenCalledWith("exp-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("exp-1");
  });

  it("hides end date field when is_current is true", () => {
    const currentExperience: ExperienceDTO = {
      ...sampleExperience,
      is_current: true,
      end_date: "",
    };
    const stateWithCurrent = createMockStoreState({
      experiences: [currentExperience],
    });
    Object.assign(mockState, stateWithCurrent);

    render(<ExperienceEditor />);
    expect(
      screen.queryByLabelText("resumeBuilder.experience.endDate"),
    ).not.toBeInTheDocument();
  });

  it("displays fallback title for empty experience", () => {
    const emptyExperience: ExperienceDTO = {
      ...sampleExperience,
      company: "",
      position: "",
    };
    const stateWithEmpty = createMockStoreState({
      experiences: [emptyExperience],
    });
    Object.assign(mockState, stateWithEmpty);

    render(<ExperienceEditor />);
    expect(
      screen.getByText("resumeBuilder.experience.newEntry"),
    ).toBeInTheDocument();
  });
});
