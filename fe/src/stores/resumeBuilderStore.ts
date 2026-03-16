import { create } from "zustand";
import { temporal } from "zundo";
import type {
  FullResumeDTO,
  SectionKey,
  ContactDTO,
  SummaryDTO,
  ExperienceDTO,
  EducationDTO,
  SkillDTO,
  LanguageDTO,
  CertificationDTO,
  ProjectDTO,
  VolunteeringDTO,
  CustomSectionDTO,
  SectionOrderDTO,
} from "@/shared/types/resume-builder";

type SaveStatus = "idle" | "saving" | "saved" | "error";

/** Keys of FullResumeDTO that hold arrays with { id: string } items. */
type ArraySectionKey =
  | "experiences"
  | "educations"
  | "skills"
  | "languages"
  | "certifications"
  | "projects"
  | "volunteering"
  | "custom_sections";

interface ResumeBuilderState {
  resume: FullResumeDTO | null;
  activeSection: SectionKey;
  saveStatus: SaveStatus;
  isDirty: boolean;

  // Actions
  setResume: (resume: FullResumeDTO) => void;
  setActiveSection: (section: SectionKey) => void;
  setSaveStatus: (status: SaveStatus) => void;
  markDirty: () => void;
  markClean: () => void;

  // Immutable section updates
  updateContact: (contact: ContactDTO) => void;
  updateSummary: (summary: SummaryDTO) => void;
  updateDesign: (
    updates: Partial<
      Pick<
        FullResumeDTO,
        | "template_id"
        | "font_family"
        | "primary_color"
        | "text_color"
        | "spacing"
        | "margin_top"
        | "margin_bottom"
        | "margin_left"
        | "margin_right"
        | "title"
        | "layout_mode"
        | "sidebar_width"
        | "font_size"
        | "skill_display"
      >
    >,
  ) => void;

  // Array section updates (immutable)
  setExperiences: (experiences: ExperienceDTO[]) => void;
  setEducations: (educations: EducationDTO[]) => void;
  setSkills: (skills: SkillDTO[]) => void;
  setLanguages: (languages: LanguageDTO[]) => void;
  setCertifications: (certifications: CertificationDTO[]) => void;
  setProjects: (projects: ProjectDTO[]) => void;
  setVolunteering: (volunteering: VolunteeringDTO[]) => void;
  setCustomSections: (customSections: CustomSectionDTO[]) => void;
  setSectionOrder: (sectionOrder: SectionOrderDTO[]) => void;

  // Entry-level updates
  addExperience: (exp: ExperienceDTO) => void;
  updateExperience: (id: string, updates: Partial<ExperienceDTO>) => void;
  removeExperience: (id: string) => void;

  addEducation: (edu: EducationDTO) => void;
  updateEducation: (id: string, updates: Partial<EducationDTO>) => void;
  removeEducation: (id: string) => void;

  addSkill: (skill: SkillDTO) => void;
  updateSkill: (id: string, updates: Partial<SkillDTO>) => void;
  removeSkill: (id: string) => void;

  addLanguage: (lang: LanguageDTO) => void;
  updateLanguage: (id: string, updates: Partial<LanguageDTO>) => void;
  removeLanguage: (id: string) => void;

  addCertification: (cert: CertificationDTO) => void;
  updateCertification: (id: string, updates: Partial<CertificationDTO>) => void;
  removeCertification: (id: string) => void;

  addProject: (proj: ProjectDTO) => void;
  updateProject: (id: string, updates: Partial<ProjectDTO>) => void;
  removeProject: (id: string) => void;

  addVolunteering: (vol: VolunteeringDTO) => void;
  updateVolunteering: (id: string, updates: Partial<VolunteeringDTO>) => void;
  removeVolunteering: (id: string) => void;

  addCustomSection: (cs: CustomSectionDTO) => void;
  updateCustomSection: (id: string, updates: Partial<CustomSectionDTO>) => void;
  removeCustomSection: (id: string) => void;
}

// ---------------------------------------------------------------------------
// Generic helpers — single implementation for all array sections
// ---------------------------------------------------------------------------

type SetFn = (
  updater:
    | Partial<ResumeBuilderState>
    | ((state: ResumeBuilderState) => Partial<ResumeBuilderState>),
) => void;

function addEntry<T>(set: SetFn, key: ArraySectionKey, item: T) {
  set((state) => {
    if (!state.resume) return state;
    return {
      resume: {
        ...state.resume,
        [key]: [...(state.resume[key] as unknown as T[]), item],
      },
      isDirty: true,
    };
  });
}

function updateEntry<T extends { id: string }>(
  set: SetFn,
  key: ArraySectionKey,
  id: string,
  updates: Partial<T>,
) {
  set((state) => {
    if (!state.resume) return state;
    return {
      resume: {
        ...state.resume,
        [key]: (state.resume[key] as unknown as T[]).map((item) =>
          item.id === id ? { ...item, ...updates } : item,
        ),
      },
      isDirty: true,
    };
  });
}

function removeEntry<T extends { id: string }>(
  set: SetFn,
  key: ArraySectionKey,
  id: string,
) {
  set((state) => {
    if (!state.resume) return state;
    return {
      resume: {
        ...state.resume,
        [key]: (state.resume[key] as unknown as T[]).filter(
          (item) => item.id !== id,
        ),
      },
      isDirty: true,
    };
  });
}

function setSection<T>(set: SetFn, key: string, items: T) {
  set((state) => {
    if (!state.resume) return state;
    return { resume: { ...state.resume, [key]: items }, isDirty: true };
  });
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useResumeBuilderStore = create<ResumeBuilderState>()(
  temporal(
    (set) => ({
      resume: null,
      activeSection: "contact",
      saveStatus: "idle",
      isDirty: false,

      setResume: (resume) =>
        set({ resume, isDirty: false, saveStatus: "idle" }),
      setActiveSection: (activeSection) => set({ activeSection }),
      setSaveStatus: (saveStatus) => set({ saveStatus }),
      markDirty: () => set({ isDirty: true, saveStatus: "idle" }),
      markClean: () => set({ isDirty: false }),

      updateContact: (contact) =>
        set((state) => {
          if (!state.resume) return state;
          return { resume: { ...state.resume, contact }, isDirty: true };
        }),

      updateSummary: (summary) =>
        set((state) => {
          if (!state.resume) return state;
          return { resume: { ...state.resume, summary }, isDirty: true };
        }),

      updateDesign: (updates) =>
        set((state) => {
          if (!state.resume) return state;
          return { resume: { ...state.resume, ...updates }, isDirty: true };
        }),

      // Batch setters
      setExperiences: (v) => setSection(set, "experiences", v),
      setEducations: (v) => setSection(set, "educations", v),
      setSkills: (v) => setSection(set, "skills", v),
      setLanguages: (v) => setSection(set, "languages", v),
      setCertifications: (v) => setSection(set, "certifications", v),
      setProjects: (v) => setSection(set, "projects", v),
      setVolunteering: (v) => setSection(set, "volunteering", v),
      setCustomSections: (v) => setSection(set, "custom_sections", v),
      setSectionOrder: (v) => setSection(set, "section_order", v),

      // Experience
      addExperience: (item) => addEntry(set, "experiences", item),
      updateExperience: (id, updates) =>
        updateEntry<ExperienceDTO>(set, "experiences", id, updates),
      removeExperience: (id) =>
        removeEntry<ExperienceDTO>(set, "experiences", id),

      // Education
      addEducation: (item) => addEntry(set, "educations", item),
      updateEducation: (id, updates) =>
        updateEntry<EducationDTO>(set, "educations", id, updates),
      removeEducation: (id) => removeEntry<EducationDTO>(set, "educations", id),

      // Skill
      addSkill: (item) => addEntry(set, "skills", item),
      updateSkill: (id, updates) =>
        updateEntry<SkillDTO>(set, "skills", id, updates),
      removeSkill: (id) => removeEntry<SkillDTO>(set, "skills", id),

      // Language
      addLanguage: (item) => addEntry(set, "languages", item),
      updateLanguage: (id, updates) =>
        updateEntry<LanguageDTO>(set, "languages", id, updates),
      removeLanguage: (id) => removeEntry<LanguageDTO>(set, "languages", id),

      // Certification
      addCertification: (item) => addEntry(set, "certifications", item),
      updateCertification: (id, updates) =>
        updateEntry<CertificationDTO>(set, "certifications", id, updates),
      removeCertification: (id) =>
        removeEntry<CertificationDTO>(set, "certifications", id),

      // Project
      addProject: (item) => addEntry(set, "projects", item),
      updateProject: (id, updates) =>
        updateEntry<ProjectDTO>(set, "projects", id, updates),
      removeProject: (id) => removeEntry<ProjectDTO>(set, "projects", id),

      // Volunteering
      addVolunteering: (item) => addEntry(set, "volunteering", item),
      updateVolunteering: (id, updates) =>
        updateEntry<VolunteeringDTO>(set, "volunteering", id, updates),
      removeVolunteering: (id) =>
        removeEntry<VolunteeringDTO>(set, "volunteering", id),

      // Custom Section
      addCustomSection: (item) => addEntry(set, "custom_sections", item),
      updateCustomSection: (id, updates) =>
        updateEntry<CustomSectionDTO>(set, "custom_sections", id, updates),
      removeCustomSection: (id) =>
        removeEntry<CustomSectionDTO>(set, "custom_sections", id),
    }),
    {
      partialize: (state) => ({ resume: state.resume }),
      limit: 50,
      equality: (pastState, currentState) =>
        pastState.resume === currentState.resume,
    },
  ),
);
