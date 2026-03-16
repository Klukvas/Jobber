import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { VolunteeringEditor } from "./VolunteeringEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { VolunteeringDTO } from "@/shared/types/resume-builder";

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

const sampleVolunteering: VolunteeringDTO = {
  id: "vol-1",
  organization: "Red Cross",
  role: "Volunteer Coordinator",
  start_date: "2021-03-01",
  end_date: "2022-06-30",
  description: "Coordinated disaster relief efforts",
  sort_order: 0,
};

describe("VolunteeringEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<VolunteeringEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.volunteering"),
    ).toBeInTheDocument();
  });

  it("shows empty state when no volunteering entries", () => {
    render(<VolunteeringEditor />);
    expect(
      screen.getByText("resumeBuilder.volunteering.empty"),
    ).toBeInTheDocument();
  });

  it("adds volunteering when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<VolunteeringEditor />);

    await user.click(screen.getByText("resumeBuilder.volunteering.add"));
    expect(mockState.addVolunteering).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders volunteering card with organization name", () => {
    const stateWithVol = createMockStoreState({
      volunteering: [sampleVolunteering],
    });
    Object.assign(mockState, stateWithVol);

    render(<VolunteeringEditor />);
    expect(screen.getByText("Red Cross")).toBeInTheDocument();
  });

  it("expands card on click and shows fields", async () => {
    const user = userEvent.setup();
    const stateWithVol = createMockStoreState({
      volunteering: [sampleVolunteering],
    });
    Object.assign(mockState, stateWithVol);

    render(<VolunteeringEditor />);
    await user.click(screen.getByText("Red Cross"));

    expect(screen.getByDisplayValue("Red Cross")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("Volunteer Coordinator"),
    ).toBeInTheDocument();
  });

  it("calls removeVolunteering when remove is clicked", async () => {
    const user = userEvent.setup();
    const stateWithVol = createMockStoreState({
      volunteering: [sampleVolunteering],
    });
    Object.assign(mockState, stateWithVol);

    render(<VolunteeringEditor />);
    await user.click(screen.getByText("Red Cross"));
    await user.click(screen.getByText("resumeBuilder.volunteering.remove"));

    expect(mockState.removeVolunteering).toHaveBeenCalledWith("vol-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("vol-1");
  });

  it("shows untitled for volunteering without organization", () => {
    const unnamed: VolunteeringDTO = {
      ...sampleVolunteering,
      organization: "",
    };
    const stateWithUntitled = createMockStoreState({
      volunteering: [unnamed],
    });
    Object.assign(mockState, stateWithUntitled);

    render(<VolunteeringEditor />);
    expect(
      screen.getByText("resumeBuilder.volunteering.untitled"),
    ).toBeInTheDocument();
  });
});
