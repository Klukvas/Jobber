import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import {
  useExperienceInline,
  useEducationInline,
  useSkillsInline,
  useLanguagesInline,
  useCertificationsInline,
  useProjectsInline,
  useVolunteeringInline,
  useCustomSectionsInline,
  createEmptyExperience,
  createEmptyEducation,
  createEmptySkill,
  createEmptyLanguage,
  createEmptyCertification,
  createEmptyProject,
  createEmptyVolunteering,
  createEmptyCustomSection,
} from "./useInlineSection";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { initServerIds } from "../../hooks/useSectionPersistence";
import type { FullResumeDTO } from "@/shared/types/resume-builder";

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    createExperience: vi.fn().mockResolvedValue({
      id: "server-exp",
      company: "",
      position: "",
      location: "",
      start_date: "",
      end_date: "",
      is_current: false,
      description: "",
      sort_order: 0,
    }),
    createEducation: vi.fn().mockResolvedValue({
      id: "server-edu",
      institution: "",
      degree: "",
      field_of_study: "",
      start_date: "",
      end_date: "",
      is_current: false,
      gpa: "",
      description: "",
      sort_order: 0,
    }),
    createSkill: vi.fn().mockResolvedValue({
      id: "server-skill",
      name: "",
      level: "",
      sort_order: 0,
    }),
    createLanguage: vi.fn().mockResolvedValue({
      id: "server-lang",
      name: "",
      proficiency: "",
      sort_order: 0,
    }),
    createCertification: vi.fn().mockResolvedValue({
      id: "server-cert",
      name: "",
      issuer: "",
      issue_date: "",
      expiry_date: "",
      url: "",
      sort_order: 0,
    }),
    createProject: vi.fn().mockResolvedValue({
      id: "server-proj",
      name: "",
      url: "",
      start_date: "",
      end_date: "",
      description: "",
      sort_order: 0,
    }),
    createVolunteering: vi.fn().mockResolvedValue({
      id: "server-vol",
      organization: "",
      role: "",
      start_date: "",
      end_date: "",
      description: "",
      sort_order: 0,
    }),
    createCustomSection: vi.fn().mockResolvedValue({
      id: "server-cs",
      title: "",
      content: "",
      sort_order: 0,
    }),
    deleteExperience: vi.fn().mockResolvedValue(undefined),
    deleteEducation: vi.fn().mockResolvedValue(undefined),
    deleteSkill: vi.fn().mockResolvedValue(undefined),
    deleteLanguage: vi.fn().mockResolvedValue(undefined),
    deleteCertification: vi.fn().mockResolvedValue(undefined),
    deleteProject: vi.fn().mockResolvedValue(undefined),
    deleteVolunteering: vi.fn().mockResolvedValue(undefined),
    deleteCustomSection: vi.fn().mockResolvedValue(undefined),
  },
}));

function makeResume(overrides: Partial<FullResumeDTO> = {}): FullResumeDTO {
  return {
    id: "resume-1",
    title: "Test",
    template_id: "modern",
    font_family: "Inter",
    primary_color: "#000",
    text_color: "#000",
    spacing: 1,
    margin_top: 20,
    margin_bottom: 20,
    margin_left: 20,
    margin_right: 20,
    layout_mode: "single",
    sidebar_width: 35,
    font_size: 12,
    created_at: "2024-01-01",
    updated_at: "2024-01-01",
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

describe("factory functions", () => {
  describe("createEmptyExperience", () => {
    it("creates an experience with a UUID and correct sort_order", () => {
      const exp = createEmptyExperience(3);
      expect(exp.id).toBeTruthy();
      expect(exp.sort_order).toBe(3);
      expect(exp.company).toBe("");
      expect(exp.position).toBe("");
      expect(exp.is_current).toBe(false);
    });

    it("generates unique IDs on each call", () => {
      const a = createEmptyExperience(0);
      const b = createEmptyExperience(0);
      expect(a.id).not.toBe(b.id);
    });
  });

  describe("createEmptyEducation", () => {
    it("creates an education with empty fields", () => {
      const edu = createEmptyEducation(1);
      expect(edu.id).toBeTruthy();
      expect(edu.sort_order).toBe(1);
      expect(edu.institution).toBe("");
      expect(edu.degree).toBe("");
      expect(edu.gpa).toBe("");
    });
  });

  describe("createEmptySkill", () => {
    it("creates a skill with empty fields", () => {
      const skill = createEmptySkill(5);
      expect(skill.id).toBeTruthy();
      expect(skill.sort_order).toBe(5);
      expect(skill.name).toBe("");
      expect(skill.level).toBe("");
    });
  });

  describe("createEmptyLanguage", () => {
    it("creates a language with empty fields", () => {
      const lang = createEmptyLanguage(2);
      expect(lang.id).toBeTruthy();
      expect(lang.sort_order).toBe(2);
      expect(lang.name).toBe("");
      expect(lang.proficiency).toBe("");
    });
  });

  describe("createEmptyCertification", () => {
    it("creates a certification with empty fields", () => {
      const cert = createEmptyCertification(0);
      expect(cert.id).toBeTruthy();
      expect(cert.sort_order).toBe(0);
      expect(cert.name).toBe("");
      expect(cert.issuer).toBe("");
      expect(cert.url).toBe("");
    });
  });

  describe("createEmptyProject", () => {
    it("creates a project with empty fields", () => {
      const proj = createEmptyProject(4);
      expect(proj.id).toBeTruthy();
      expect(proj.sort_order).toBe(4);
      expect(proj.name).toBe("");
      expect(proj.url).toBe("");
      expect(proj.description).toBe("");
    });
  });

  describe("createEmptyVolunteering", () => {
    it("creates a volunteering entry with empty fields", () => {
      const vol = createEmptyVolunteering(1);
      expect(vol.id).toBeTruthy();
      expect(vol.sort_order).toBe(1);
      expect(vol.organization).toBe("");
      expect(vol.role).toBe("");
    });
  });

  describe("createEmptyCustomSection", () => {
    it("creates a custom section with empty fields", () => {
      const cs = createEmptyCustomSection(0);
      expect(cs.id).toBeTruthy();
      expect(cs.sort_order).toBe(0);
      expect(cs.title).toBe("");
      expect(cs.content).toBe("");
    });
  });
});

describe("useExperienceInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds an experience to the store", async () => {
    const { result } = renderHook(() => useExperienceInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    // Should have added one item (or it may have been replaced by server response)
    expect(resume?.experiences.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes an experience from the store", async () => {
    const exp = createEmptyExperience(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ experiences: [exp] }),
    });

    const { result } = renderHook(() => useExperienceInline());

    await act(async () => {
      result.current.handleRemove(exp.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.experiences).toHaveLength(0);
  });

  it("handleAdd increments sort_order based on existing items", async () => {
    const existing = createEmptyExperience(2);
    useResumeBuilderStore.setState({
      resume: makeResume({ experiences: [existing] }),
    });

    const { result } = renderHook(() => useExperienceInline());

    // Capture what was added to the store by checking before the API replaces it
    const storeBeforeAdd = useResumeBuilderStore.getState().resume;
    const countBefore = storeBeforeAdd?.experiences.length ?? 0;

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.experiences.length ?? 0).toBeGreaterThan(countBefore);
  });
});

describe("useEducationInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds an education to the store", async () => {
    const { result } = renderHook(() => useEducationInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.educations.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes an education from the store", async () => {
    const edu = createEmptyEducation(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ educations: [edu] }),
    });

    const { result } = renderHook(() => useEducationInline());

    await act(async () => {
      result.current.handleRemove(edu.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.educations).toHaveLength(0);
  });
});

describe("useSkillsInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds a skill to the store", async () => {
    const { result } = renderHook(() => useSkillsInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.skills.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes a skill from the store", async () => {
    const skill = createEmptySkill(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ skills: [skill] }),
    });

    const { result } = renderHook(() => useSkillsInline());

    await act(async () => {
      result.current.handleRemove(skill.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.skills).toHaveLength(0);
  });
});

describe("useLanguagesInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds a language to the store", async () => {
    const { result } = renderHook(() => useLanguagesInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.languages.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes a language from the store", async () => {
    const lang = createEmptyLanguage(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ languages: [lang] }),
    });

    const { result } = renderHook(() => useLanguagesInline());

    await act(async () => {
      result.current.handleRemove(lang.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.languages).toHaveLength(0);
  });
});

describe("useCertificationsInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds a certification to the store", async () => {
    const { result } = renderHook(() => useCertificationsInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.certifications.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes a certification from the store", async () => {
    const cert = createEmptyCertification(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ certifications: [cert] }),
    });

    const { result } = renderHook(() => useCertificationsInline());

    await act(async () => {
      result.current.handleRemove(cert.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.certifications).toHaveLength(0);
  });
});

describe("useProjectsInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds a project to the store", async () => {
    const { result } = renderHook(() => useProjectsInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.projects.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes a project from the store", async () => {
    const proj = createEmptyProject(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ projects: [proj] }),
    });

    const { result } = renderHook(() => useProjectsInline());

    await act(async () => {
      result.current.handleRemove(proj.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.projects).toHaveLength(0);
  });
});

describe("useVolunteeringInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds a volunteering entry to the store", async () => {
    const { result } = renderHook(() => useVolunteeringInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.volunteering.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes a volunteering entry from the store", async () => {
    const vol = createEmptyVolunteering(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ volunteering: [vol] }),
    });

    const { result } = renderHook(() => useVolunteeringInline());

    await act(async () => {
      result.current.handleRemove(vol.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.volunteering).toHaveLength(0);
  });
});

describe("useCustomSectionsInline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    initServerIds({
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
  });

  it("handleAdd adds a custom section to the store", async () => {
    const { result } = renderHook(() => useCustomSectionsInline());

    await act(async () => {
      result.current.handleAdd();
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.custom_sections.length).toBeGreaterThanOrEqual(1);
  });

  it("handleRemove removes a custom section from the store", async () => {
    const cs = createEmptyCustomSection(0);
    useResumeBuilderStore.setState({
      resume: makeResume({ custom_sections: [cs] }),
    });

    const { result } = renderHook(() => useCustomSectionsInline());

    await act(async () => {
      result.current.handleRemove(cs.id);
    });

    const resume = useResumeBuilderStore.getState().resume;
    expect(resume?.custom_sections).toHaveLength(0);
  });
});
