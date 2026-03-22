import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditorPanel } from "./EditorPanel";
import { createMockStoreState } from "./__tests__/testHelpers";

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

// Mock all section editors to avoid complex dependencies
vi.mock("./sections/ContactEditor", () => ({
  ContactEditor: () => <div data-testid="contact-editor">ContactEditor</div>,
}));
vi.mock("./sections/SummaryEditor", () => ({
  SummaryEditor: () => <div data-testid="summary-editor">SummaryEditor</div>,
}));
vi.mock("./sections/ExperienceEditor", () => ({
  ExperienceEditor: () => (
    <div data-testid="experience-editor">ExperienceEditor</div>
  ),
}));
vi.mock("./sections/EducationEditor", () => ({
  EducationEditor: () => (
    <div data-testid="education-editor">EducationEditor</div>
  ),
}));
vi.mock("./sections/SkillsEditor", () => ({
  SkillsEditor: () => <div data-testid="skills-editor">SkillsEditor</div>,
}));
vi.mock("./sections/LanguagesEditor", () => ({
  LanguagesEditor: () => (
    <div data-testid="languages-editor">LanguagesEditor</div>
  ),
}));
vi.mock("./sections/CertificationsEditor", () => ({
  CertificationsEditor: () => (
    <div data-testid="certifications-editor">CertificationsEditor</div>
  ),
}));
vi.mock("./sections/ProjectsEditor", () => ({
  ProjectsEditor: () => (
    <div data-testid="projects-editor">ProjectsEditor</div>
  ),
}));
vi.mock("./sections/VolunteeringEditor", () => ({
  VolunteeringEditor: () => (
    <div data-testid="volunteering-editor">VolunteeringEditor</div>
  ),
}));
vi.mock("./sections/CustomSectionsEditor", () => ({
  CustomSectionsEditor: () => (
    <div data-testid="custom-sections-editor">CustomSectionsEditor</div>
  ),
}));
vi.mock("./DesignPanel", () => ({
  DesignPanel: () => <div data-testid="design-panel">DesignPanel</div>,
}));

describe("EditorPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders section navigation buttons", () => {
    render(<EditorPanel />);
    const buttons = screen.getAllByRole("button");
    // 10 section nav items + 1 design button = 11
    expect(buttons).toHaveLength(11);
  });

  it("renders the active section editor (contact by default)", () => {
    render(<EditorPanel />);
    expect(screen.getByTestId("contact-editor")).toBeInTheDocument();
  });

  it("calls setActiveSection when a navigation button is clicked", async () => {
    const user = userEvent.setup();
    render(<EditorPanel />);

    // Click the design button (last button)
    const buttons = screen.getAllByRole("button");
    await user.click(buttons[buttons.length - 1]);

    expect(mockState.setActiveSection).toHaveBeenCalled();
  });

  it("shows design panel when activeSection is design", () => {
    Object.assign(mockState, { activeSection: "design" });
    render(<EditorPanel />);
    expect(screen.getByTestId("design-panel")).toBeInTheDocument();
  });

  it("shows experience editor when activeSection is experience", () => {
    Object.assign(mockState, { activeSection: "experience" });
    render(<EditorPanel />);
    expect(screen.getByTestId("experience-editor")).toBeInTheDocument();
  });

  it("shows education editor when activeSection is education", () => {
    Object.assign(mockState, { activeSection: "education" });
    render(<EditorPanel />);
    expect(screen.getByTestId("education-editor")).toBeInTheDocument();
  });
});
