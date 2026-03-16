import { vi } from "vitest";

export function createMockSetup(overrides?: Record<string, unknown>) {
  return {
    resume: {
      primary_color: "#e11d48",
      text_color: "#e11d48",
      section_order: [],
      layout_mode: "single",
      sidebar_width: 35,
    },
    color: "#e11d48",
    textColor: "#e11d48",
    contact: {
      full_name: "Jane Doe",
      email: "jane@example.com",
      phone: "+1234567890",
      location: "NYC",
      website: "jane.dev",
      linkedin: "linkedin.com/in/jane",
      github: "github.com/jane",
    },
    summary: { content: "Experienced dev" },
    layoutMode: "single",
    sidebarWidth: 35,
    visibleSections: [],
    mainSections: [],
    sidebarSections: [],
    isTwoColumn: false,
    updateContact: vi.fn(),
    updateSummary: vi.fn(),
    updateExperience: vi.fn(),
    updateEducation: vi.fn(),
    updateSkill: vi.fn(),
    updateLanguage: vi.fn(),
    updateCertification: vi.fn(),
    updateProject: vi.fn(),
    updateVolunteering: vi.fn(),
    updateCustomSection: vi.fn(),
    experienceSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    educationSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    skillsSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    languagesSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    certificationsSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    projectsSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    volunteeringSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    customSectionsSection: {
      handleAdd: vi.fn(),
      handleRemove: vi.fn(),
    },
    hideSection: vi.fn(),
    moveSection: vi.fn(),
    canMoveUp: vi.fn().mockReturnValue(false),
    canMoveDown: vi.fn().mockReturnValue(false),
    ...overrides,
  };
}
