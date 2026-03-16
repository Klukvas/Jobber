import { useCallback } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import type {
  ExperienceDTO,
  EducationDTO,
  SkillDTO,
  LanguageDTO,
  CertificationDTO,
  ProjectDTO,
  VolunteeringDTO,
  CustomSectionDTO,
} from "@/shared/types/resume-builder";

/** Stable reference to avoid infinite re-renders when store.resume is null. */
const EMPTY: readonly never[] = [];

// Factory functions for empty DTOs
export function createEmptyExperience(sortOrder: number): ExperienceDTO {
  return {
    id: crypto.randomUUID(),
    company: "",
    position: "",
    location: "",
    start_date: "",
    end_date: "",
    is_current: false,
    description: "",
    sort_order: sortOrder,
  };
}

export function createEmptyEducation(sortOrder: number): EducationDTO {
  return {
    id: crypto.randomUUID(),
    institution: "",
    degree: "",
    field_of_study: "",
    start_date: "",
    end_date: "",
    is_current: false,
    gpa: "",
    description: "",
    sort_order: sortOrder,
  };
}

export function createEmptySkill(sortOrder: number): SkillDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    level: "",
    sort_order: sortOrder,
  };
}

export function createEmptyLanguage(sortOrder: number): LanguageDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    proficiency: "",
    sort_order: sortOrder,
  };
}

export function createEmptyCertification(sortOrder: number): CertificationDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    issuer: "",
    issue_date: "",
    expiry_date: "",
    url: "",
    sort_order: sortOrder,
  };
}

export function createEmptyProject(sortOrder: number): ProjectDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    url: "",
    start_date: "",
    end_date: "",
    description: "",
    sort_order: sortOrder,
  };
}

export function createEmptyVolunteering(sortOrder: number): VolunteeringDTO {
  return {
    id: crypto.randomUUID(),
    organization: "",
    role: "",
    start_date: "",
    end_date: "",
    description: "",
    sort_order: sortOrder,
  };
}

export function createEmptyCustomSection(sortOrder: number): CustomSectionDTO {
  return {
    id: crypto.randomUUID(),
    title: "",
    content: "",
    sort_order: sortOrder,
  };
}

function getMaxSortOrder(items: readonly { sort_order: number }[]): number {
  if (items.length === 0) return -1;
  return Math.max(...items.map((i) => i.sort_order));
}

export function useExperienceInline() {
  const experiences = useResumeBuilderStore(
    (s) => s.resume?.experiences ?? EMPTY,
  );
  const addExperience = useResumeBuilderStore((s) => s.addExperience);
  const removeExperience = useResumeBuilderStore((s) => s.removeExperience);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<ExperienceDTO>("experiences");

  const handleAdd = useCallback(() => {
    const item = createEmptyExperience(getMaxSortOrder(experiences) + 1);
    addExperience(item);
    persistAdd(item);
  }, [experiences, addExperience, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeExperience(id);
      persistRemove(id);
    },
    [removeExperience, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useEducationInline() {
  const educations = useResumeBuilderStore(
    (s) => s.resume?.educations ?? EMPTY,
  );
  const addEducation = useResumeBuilderStore((s) => s.addEducation);
  const removeEducation = useResumeBuilderStore((s) => s.removeEducation);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<EducationDTO>("educations");

  const handleAdd = useCallback(() => {
    const item = createEmptyEducation(getMaxSortOrder(educations) + 1);
    addEducation(item);
    persistAdd(item);
  }, [educations, addEducation, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeEducation(id);
      persistRemove(id);
    },
    [removeEducation, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useSkillsInline() {
  const skills = useResumeBuilderStore((s) => s.resume?.skills ?? EMPTY);
  const addSkill = useResumeBuilderStore((s) => s.addSkill);
  const removeSkill = useResumeBuilderStore((s) => s.removeSkill);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<SkillDTO>("skills");

  const handleAdd = useCallback(() => {
    const item = createEmptySkill(getMaxSortOrder(skills) + 1);
    addSkill(item);
    persistAdd(item);
  }, [skills, addSkill, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeSkill(id);
      persistRemove(id);
    },
    [removeSkill, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useLanguagesInline() {
  const languages = useResumeBuilderStore((s) => s.resume?.languages ?? EMPTY);
  const addLanguage = useResumeBuilderStore((s) => s.addLanguage);
  const removeLanguage = useResumeBuilderStore((s) => s.removeLanguage);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<LanguageDTO>("languages");

  const handleAdd = useCallback(() => {
    const item = createEmptyLanguage(getMaxSortOrder(languages) + 1);
    addLanguage(item);
    persistAdd(item);
  }, [languages, addLanguage, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeLanguage(id);
      persistRemove(id);
    },
    [removeLanguage, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useCertificationsInline() {
  const certifications = useResumeBuilderStore(
    (s) => s.resume?.certifications ?? EMPTY,
  );
  const addCertification = useResumeBuilderStore((s) => s.addCertification);
  const removeCertification = useResumeBuilderStore(
    (s) => s.removeCertification,
  );
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<CertificationDTO>("certifications");

  const handleAdd = useCallback(() => {
    const item = createEmptyCertification(getMaxSortOrder(certifications) + 1);
    addCertification(item);
    persistAdd(item);
  }, [certifications, addCertification, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeCertification(id);
      persistRemove(id);
    },
    [removeCertification, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useProjectsInline() {
  const projects = useResumeBuilderStore((s) => s.resume?.projects ?? EMPTY);
  const addProject = useResumeBuilderStore((s) => s.addProject);
  const removeProject = useResumeBuilderStore((s) => s.removeProject);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<ProjectDTO>("projects");

  const handleAdd = useCallback(() => {
    const item = createEmptyProject(getMaxSortOrder(projects) + 1);
    addProject(item);
    persistAdd(item);
  }, [projects, addProject, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeProject(id);
      persistRemove(id);
    },
    [removeProject, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useVolunteeringInline() {
  const volunteering = useResumeBuilderStore(
    (s) => s.resume?.volunteering ?? EMPTY,
  );
  const addVolunteering = useResumeBuilderStore((s) => s.addVolunteering);
  const removeVolunteering = useResumeBuilderStore((s) => s.removeVolunteering);
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<VolunteeringDTO>("volunteering");

  const handleAdd = useCallback(() => {
    const item = createEmptyVolunteering(getMaxSortOrder(volunteering) + 1);
    addVolunteering(item);
    persistAdd(item);
  }, [volunteering, addVolunteering, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeVolunteering(id);
      persistRemove(id);
    },
    [removeVolunteering, persistRemove],
  );

  return { handleAdd, handleRemove };
}

export function useCustomSectionsInline() {
  const customSections = useResumeBuilderStore(
    (s) => s.resume?.custom_sections ?? EMPTY,
  );
  const addCustomSection = useResumeBuilderStore((s) => s.addCustomSection);
  const removeCustomSection = useResumeBuilderStore(
    (s) => s.removeCustomSection,
  );
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<CustomSectionDTO>("custom-sections");

  const handleAdd = useCallback(() => {
    const item = createEmptyCustomSection(getMaxSortOrder(customSections) + 1);
    addCustomSection(item);
    persistAdd(item);
  }, [customSections, addCustomSection, persistAdd]);

  const handleRemove = useCallback(
    (id: string) => {
      removeCustomSection(id);
      persistRemove(id);
    },
    [removeCustomSection, persistRemove],
  );

  return { handleAdd, handleRemove };
}
