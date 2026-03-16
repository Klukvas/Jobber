import { describe, it, expect, beforeEach } from "vitest";
import { useResumeBuilderStore } from "../resumeBuilderStore";
import type {
  FullResumeDTO,
  ContactDTO,
  SummaryDTO,
  ExperienceDTO,
  SkillDTO,
  SectionOrderDTO,
} from "@/shared/types/resume-builder";

// ---------------------------------------------------------------------------
// Mock factory
// ---------------------------------------------------------------------------

function createMockResume(
  overrides?: Partial<FullResumeDTO>,
): FullResumeDTO {
  return {
    id: "resume-1",
    title: "My Resume",
    template_id: "t1",
    font_family: "Georgia",
    primary_color: "#2563eb",
    text_color: "#111827",
    spacing: 100,
    margin_top: 40,
    margin_bottom: 40,
    margin_left: 40,
    margin_right: 40,
    layout_mode: "single",
    sidebar_width: 35,
    font_size: 12,
    skill_display: "",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    contact: null,
    summary: null,
    experiences: [],
    educations: [],
    skills: [],
    languages: [],
    certifications: [],
    projects: [],
    volunteering: [],
    custom_sections: [],
    section_order: [],
    ...overrides,
  };
}

function createMockExperience(
  overrides?: Partial<ExperienceDTO>,
): ExperienceDTO {
  return {
    id: "exp-1",
    company: "Acme Corp",
    position: "Engineer",
    location: "Remote",
    start_date: "2023-01-01",
    end_date: "2024-01-01",
    is_current: false,
    description: "Built things",
    sort_order: 0,
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("resumeBuilderStore", () => {
  beforeEach(() => {
    useResumeBuilderStore.setState({
      resume: null,
      activeSection: "contact",
      saveStatus: "idle",
      isDirty: false,
    });
  });

  // 1. Initial state
  it("has correct initial state", () => {
    const state = useResumeBuilderStore.getState();

    expect(state.resume).toBeNull();
    expect(state.activeSection).toBe("contact");
    expect(state.saveStatus).toBe("idle");
    expect(state.isDirty).toBe(false);
  });

  // 2. setResume sets resume and resets dirty/status
  it("setResume sets resume and resets isDirty and saveStatus", () => {
    // Pre-condition: mark store as dirty with non-idle status
    useResumeBuilderStore.setState({ isDirty: true, saveStatus: "error" });

    const mockResume = createMockResume();
    useResumeBuilderStore.getState().setResume(mockResume);

    const state = useResumeBuilderStore.getState();
    expect(state.resume).toEqual(mockResume);
    expect(state.isDirty).toBe(false);
    expect(state.saveStatus).toBe("idle");
  });

  // 3. markDirty sets isDirty=true and saveStatus="idle"
  it("markDirty sets isDirty to true and saveStatus to idle", () => {
    useResumeBuilderStore.setState({ saveStatus: "saved" });

    useResumeBuilderStore.getState().markDirty();

    const state = useResumeBuilderStore.getState();
    expect(state.isDirty).toBe(true);
    expect(state.saveStatus).toBe("idle");
  });

  // 4. markClean sets isDirty=false
  it("markClean sets isDirty to false", () => {
    useResumeBuilderStore.setState({ isDirty: true });

    useResumeBuilderStore.getState().markClean();

    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
  });

  // 5. setSaveStatus changes status
  it("setSaveStatus updates the saveStatus field", () => {
    useResumeBuilderStore.getState().setSaveStatus("saving");
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saving");

    useResumeBuilderStore.getState().setSaveStatus("saved");
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");

    useResumeBuilderStore.getState().setSaveStatus("error");
    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");
  });

  // 6. updateContact - immutably replaces contact, marks dirty
  it("updateContact immutably replaces contact and marks dirty", () => {
    const mockResume = createMockResume();
    useResumeBuilderStore.getState().setResume(mockResume);

    const newContact: ContactDTO = {
      full_name: "Jane Doe",
      email: "jane@example.com",
      phone: "+1234567890",
      location: "NYC",
      website: "https://jane.dev",
      linkedin: "linkedin.com/in/jane",
      github: "github.com/jane",
    };

    useResumeBuilderStore.getState().updateContact(newContact);

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.contact).toEqual(newContact);
    expect(state.isDirty).toBe(true);
    // Original mock resume should not be mutated
    expect(mockResume.contact).toBeNull();
  });

  // 7. updateSummary - immutably replaces summary, marks dirty
  it("updateSummary immutably replaces summary and marks dirty", () => {
    const mockResume = createMockResume();
    useResumeBuilderStore.getState().setResume(mockResume);

    const newSummary: SummaryDTO = { content: "Experienced engineer." };
    useResumeBuilderStore.getState().updateSummary(newSummary);

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.summary).toEqual(newSummary);
    expect(state.isDirty).toBe(true);
    expect(mockResume.summary).toBeNull();
  });

  // 8. updateDesign - spreads updates onto resume, marks dirty
  it("updateDesign spreads design updates onto resume and marks dirty", () => {
    const mockResume = createMockResume();
    useResumeBuilderStore.getState().setResume(mockResume);

    useResumeBuilderStore.getState().updateDesign({
      font_family: "Inter",
      primary_color: "#ff0000",
      spacing: 120,
    });

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.font_family).toBe("Inter");
    expect(state.resume?.primary_color).toBe("#ff0000");
    expect(state.resume?.spacing).toBe(120);
    // Other fields remain unchanged
    expect(state.resume?.template_id).toBe("t1");
    expect(state.isDirty).toBe(true);
  });

  // 9. addExperience - appends to experiences array, marks dirty
  it("addExperience appends to experiences array and marks dirty", () => {
    const mockResume = createMockResume();
    useResumeBuilderStore.getState().setResume(mockResume);

    const exp = createMockExperience();
    useResumeBuilderStore.getState().addExperience(exp);

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.experiences).toHaveLength(1);
    expect(state.resume?.experiences[0]).toEqual(exp);
    expect(state.isDirty).toBe(true);
    // Original array not mutated
    expect(mockResume.experiences).toHaveLength(0);
  });

  // 10. updateExperience - updates matching experience by id, marks dirty
  it("updateExperience updates matching experience by id and marks dirty", () => {
    const exp1 = createMockExperience({ id: "exp-1" });
    const exp2 = createMockExperience({ id: "exp-2", company: "Other Corp" });
    const mockResume = createMockResume({ experiences: [exp1, exp2] });
    useResumeBuilderStore.getState().setResume(mockResume);

    useResumeBuilderStore.getState().updateExperience("exp-1", {
      company: "Updated Corp",
      position: "Senior Engineer",
    });

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.experiences).toHaveLength(2);
    expect(state.resume?.experiences[0].company).toBe("Updated Corp");
    expect(state.resume?.experiences[0].position).toBe("Senior Engineer");
    // Other experience untouched
    expect(state.resume?.experiences[1].company).toBe("Other Corp");
    expect(state.isDirty).toBe(true);
    // Original not mutated
    expect(exp1.company).toBe("Acme Corp");
  });

  // 11. removeExperience - filters out by id, marks dirty
  it("removeExperience filters out experience by id and marks dirty", () => {
    const exp1 = createMockExperience({ id: "exp-1" });
    const exp2 = createMockExperience({ id: "exp-2", company: "Other Corp" });
    const mockResume = createMockResume({ experiences: [exp1, exp2] });
    useResumeBuilderStore.getState().setResume(mockResume);

    useResumeBuilderStore.getState().removeExperience("exp-1");

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.experiences).toHaveLength(1);
    expect(state.resume?.experiences[0].id).toBe("exp-2");
    expect(state.isDirty).toBe(true);
  });

  // 12. setSectionOrder - replaces section_order array, marks dirty
  it("setSectionOrder replaces section_order array and marks dirty", () => {
    const mockResume = createMockResume();
    useResumeBuilderStore.getState().setResume(mockResume);

    const newOrder: SectionOrderDTO[] = [
      { section_key: "experience", sort_order: 0, is_visible: true, column: "main" },
      { section_key: "education", sort_order: 1, is_visible: true, column: "main" },
      { section_key: "skills", sort_order: 2, is_visible: false, column: "sidebar" },
    ];

    useResumeBuilderStore.getState().setSectionOrder(newOrder);

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.section_order).toEqual(newOrder);
    expect(state.resume?.section_order).toHaveLength(3);
    expect(state.isDirty).toBe(true);
  });

  // 13. Edge: operations on null resume are no-ops
  it("operations on null resume are no-ops and do not throw", () => {
    // resume is null by default after beforeEach
    const stateBefore = useResumeBuilderStore.getState();

    useResumeBuilderStore.getState().updateContact({
      full_name: "Test",
      email: "",
      phone: "",
      location: "",
      website: "",
      linkedin: "",
      github: "",
    });
    useResumeBuilderStore.getState().updateSummary({ content: "Test" });
    useResumeBuilderStore.getState().updateDesign({ font_family: "Arial" });
    useResumeBuilderStore.getState().addExperience(createMockExperience());
    useResumeBuilderStore.getState().updateExperience("exp-1", { company: "X" });
    useResumeBuilderStore.getState().removeExperience("exp-1");
    useResumeBuilderStore.getState().setSectionOrder([]);

    const stateAfter = useResumeBuilderStore.getState();
    expect(stateAfter.resume).toBeNull();
    expect(stateAfter.isDirty).toBe(stateBefore.isDirty);
  });

  // 14. Edge: updateExperience with non-existent id leaves array unchanged but marks dirty
  it("updateExperience with non-existent id leaves array unchanged but marks dirty", () => {
    const exp = createMockExperience({ id: "exp-1" });
    const mockResume = createMockResume({ experiences: [exp] });
    useResumeBuilderStore.getState().setResume(mockResume);

    useResumeBuilderStore.getState().updateExperience("non-existent", {
      company: "Ghost Corp",
    });

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.experiences).toHaveLength(1);
    expect(state.resume?.experiences[0]).toEqual(exp);
    expect(state.isDirty).toBe(true);
  });

  // 15. Edge: removeExperience with non-existent id leaves array unchanged
  it("removeExperience with non-existent id leaves array unchanged", () => {
    const exp = createMockExperience({ id: "exp-1" });
    const mockResume = createMockResume({ experiences: [exp] });
    useResumeBuilderStore.getState().setResume(mockResume);

    useResumeBuilderStore.getState().removeExperience("non-existent");

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.experiences).toHaveLength(1);
    expect(state.resume?.experiences[0].id).toBe("exp-1");
    // isDirty is true because removeEntry always marks dirty when resume exists
    expect(state.isDirty).toBe(true);
  });
});
