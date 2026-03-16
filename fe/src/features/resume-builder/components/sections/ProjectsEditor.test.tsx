import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ProjectsEditor } from "./ProjectsEditor";
import { createMockStoreState } from "../__tests__/testHelpers";
import type { ProjectDTO } from "@/shared/types/resume-builder";

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

const sampleProject: ProjectDTO = {
  id: "proj-1",
  name: "Open Source CLI",
  url: "https://github.com/cli",
  start_date: "2022-01-01",
  end_date: "2023-12-31",
  description: "Built a command-line interface tool",
  sort_order: 0,
};

describe("ProjectsEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders the section heading", () => {
    render(<ProjectsEditor />);
    expect(
      screen.getByText("resumeBuilder.sections.projects"),
    ).toBeInTheDocument();
  });

  it("shows empty state when no projects", () => {
    render(<ProjectsEditor />);
    expect(
      screen.getByText("resumeBuilder.projects.empty"),
    ).toBeInTheDocument();
  });

  it("adds a project when Add is clicked", async () => {
    const user = userEvent.setup();
    render(<ProjectsEditor />);

    await user.click(screen.getByText("resumeBuilder.projects.add"));
    expect(mockState.addProject).toHaveBeenCalledTimes(1);
    expect(mockPersistAdd).toHaveBeenCalledTimes(1);
  });

  it("renders project card with name", () => {
    const stateWithProjects = createMockStoreState({
      projects: [sampleProject],
    });
    Object.assign(mockState, stateWithProjects);

    render(<ProjectsEditor />);
    expect(screen.getByText("Open Source CLI")).toBeInTheDocument();
  });

  it("expands project card on click and shows fields", async () => {
    const user = userEvent.setup();
    const stateWithProjects = createMockStoreState({
      projects: [sampleProject],
    });
    Object.assign(mockState, stateWithProjects);

    render(<ProjectsEditor />);
    // Projects use expandedId pattern - initially collapsed (null)
    await user.click(screen.getByText("Open Source CLI"));

    expect(screen.getByDisplayValue("Open Source CLI")).toBeInTheDocument();
    expect(
      screen.getByDisplayValue("https://github.com/cli"),
    ).toBeInTheDocument();
  });

  it("calls removeProject when remove is clicked", async () => {
    const user = userEvent.setup();
    const stateWithProjects = createMockStoreState({
      projects: [sampleProject],
    });
    Object.assign(mockState, stateWithProjects);

    render(<ProjectsEditor />);
    // Expand first
    await user.click(screen.getByText("Open Source CLI"));
    await user.click(screen.getByText("resumeBuilder.projects.remove"));

    expect(mockState.removeProject).toHaveBeenCalledWith("proj-1");
    expect(mockPersistRemove).toHaveBeenCalledWith("proj-1");
  });

  it("shows untitled for project without a name", () => {
    const unnamedProject: ProjectDTO = {
      ...sampleProject,
      name: "",
    };
    const stateWithUntitled = createMockStoreState({
      projects: [unnamedProject],
    });
    Object.assign(mockState, stateWithUntitled);

    render(<ProjectsEditor />);
    expect(
      screen.getByText("resumeBuilder.projects.untitled"),
    ).toBeInTheDocument();
  });

  it("collapses expanded card when clicked again", async () => {
    const user = userEvent.setup();
    const stateWithProjects = createMockStoreState({
      projects: [sampleProject],
    });
    Object.assign(mockState, stateWithProjects);

    render(<ProjectsEditor />);
    const header = screen.getByText("Open Source CLI");

    // Expand
    await user.click(header);
    expect(screen.getByDisplayValue("Open Source CLI")).toBeInTheDocument();

    // Collapse
    await user.click(header);
    expect(
      screen.queryByDisplayValue("Open Source CLI"),
    ).not.toBeInTheDocument();
  });
});
