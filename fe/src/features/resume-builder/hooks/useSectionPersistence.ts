import { useCallback, useRef } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
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

type SectionType =
  | "experiences"
  | "educations"
  | "skills"
  | "languages"
  | "certifications"
  | "projects"
  | "volunteering"
  | "custom-sections";

type DTOMap = {
  experiences: ExperienceDTO;
  educations: EducationDTO;
  skills: SkillDTO;
  languages: LanguageDTO;
  certifications: CertificationDTO;
  projects: ProjectDTO;
  volunteering: VolunteeringDTO;
  "custom-sections": CustomSectionDTO;
};

/**
 * Tracks which item IDs exist on the server (were fetched or successfully created).
 * Items with IDs NOT in this set are considered "local-only" and need CREATE.
 * Items with IDs IN this set need UPDATE.
 *
 * Keyed by resumeId to prevent cross-resume contamination when navigating
 * between different resumes in the same SPA session.
 */
const serverIdsByResume = new Map<string, Set<string>>();

function getOrCreateSet(resumeId: string): Set<string> {
  let set = serverIdsByResume.get(resumeId);
  if (!set) {
    set = new Set<string>();
    serverIdsByResume.set(resumeId, set);
  }
  return set;
}

/** Initialize known server IDs from a loaded resume. */
export function initServerIds(resume: {
  id: string;
  experiences: { id: string }[];
  educations: { id: string }[];
  skills: { id: string }[];
  languages: { id: string }[];
  certifications: { id: string }[];
  projects: { id: string }[];
  volunteering: { id: string }[];
  custom_sections: { id: string }[];
}) {
  const set = new Set<string>();
  for (const arr of [
    resume.experiences,
    resume.educations,
    resume.skills,
    resume.languages,
    resume.certifications,
    resume.projects,
    resume.volunteering,
    resume.custom_sections,
  ]) {
    for (const item of arr) {
      set.add(item.id);
    }
  }
  serverIdsByResume.set(resume.id, set);
}

export function markServerIds(resumeId: string, ids: string[]) {
  const set = getOrCreateSet(resumeId);
  for (const id of ids) {
    set.add(id);
  }
}

export function removeServerId(resumeId: string, id: string) {
  serverIdsByResume.get(resumeId)?.delete(id);
}

export function isServerItem(id: string): boolean {
  for (const set of serverIdsByResume.values()) {
    if (set.has(id)) return true;
  }
  return false;
}

/** Returns all server IDs for the given resume. */
export function getServerIds(resumeId: string): ReadonlySet<string> {
  return serverIdsByResume.get(resumeId) ?? new Set<string>();
}

/** Clean up stored IDs for a resume that is no longer active. */
export function clearServerIds(resumeId: string) {
  serverIdsByResume.delete(resumeId);
}

type CreateFn<T> = (resumeId: string, data: Omit<T, "id">) => Promise<T>;
type DeleteFn = (resumeId: string, entryId: string) => Promise<void>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const createFns: Record<SectionType, CreateFn<any>> = {
  experiences: (id, data) => resumeBuilderService.createExperience(id, data),
  educations: (id, data) => resumeBuilderService.createEducation(id, data),
  skills: (id, data) => resumeBuilderService.createSkill(id, data),
  languages: (id, data) => resumeBuilderService.createLanguage(id, data),
  certifications: (id, data) =>
    resumeBuilderService.createCertification(id, data),
  projects: (id, data) => resumeBuilderService.createProject(id, data),
  volunteering: (id, data) => resumeBuilderService.createVolunteering(id, data),
  "custom-sections": (id, data) =>
    resumeBuilderService.createCustomSection(id, data),
};

const deleteFns: Record<SectionType, DeleteFn> = {
  experiences: (id, eid) => resumeBuilderService.deleteExperience(id, eid),
  educations: (id, eid) => resumeBuilderService.deleteEducation(id, eid),
  skills: (id, eid) => resumeBuilderService.deleteSkill(id, eid),
  languages: (id, eid) => resumeBuilderService.deleteLanguage(id, eid),
  certifications: (id, eid) =>
    resumeBuilderService.deleteCertification(id, eid),
  projects: (id, eid) => resumeBuilderService.deleteProject(id, eid),
  volunteering: (id, eid) => resumeBuilderService.deleteVolunteering(id, eid),
  "custom-sections": (id, eid) =>
    resumeBuilderService.deleteCustomSection(id, eid),
};

type StoreKey =
  | "experiences"
  | "educations"
  | "skills"
  | "languages"
  | "certifications"
  | "projects"
  | "volunteering"
  | "custom_sections";

const sectionToStoreKey: Record<SectionType, StoreKey> = {
  experiences: "experiences",
  educations: "educations",
  skills: "skills",
  languages: "languages",
  certifications: "certifications",
  projects: "projects",
  volunteering: "volunteering",
  "custom-sections": "custom_sections",
};

/** Maps store key to the batch setter method on the store. */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const storeSetters: Record<StoreKey, (items: any[]) => void> = {
  experiences: (v) => useResumeBuilderStore.getState().setExperiences(v),
  educations: (v) => useResumeBuilderStore.getState().setEducations(v),
  skills: (v) => useResumeBuilderStore.getState().setSkills(v),
  languages: (v) => useResumeBuilderStore.getState().setLanguages(v),
  certifications: (v) => useResumeBuilderStore.getState().setCertifications(v),
  projects: (v) => useResumeBuilderStore.getState().setProjects(v),
  volunteering: (v) => useResumeBuilderStore.getState().setVolunteering(v),
  custom_sections: (v) => useResumeBuilderStore.getState().setCustomSections(v),
};

/**
 * Hook that provides add/remove functions that immediately sync with the API.
 * Returns a function pair { add, remove } for the given section type.
 * On add: calls API create, replaces local item with server item (with server ID).
 * On remove: calls API delete if item exists on server.
 */
export function useSectionPersistence<T extends DTOMap[SectionType]>(
  sectionType: SectionType,
) {
  const pendingRef = useRef(false);

  const add = useCallback(
    async (localItem: T) => {
      const resume = useResumeBuilderStore.getState().resume;
      if (!resume || pendingRef.current) return;

      pendingRef.current = true;
      try {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unused-vars
        const { id: _localId, ...data } = localItem as any;
        const serverItem = await (createFns[sectionType] as CreateFn<T>)(
          resume.id,
          data,
        );

        // Replace local item with server item in store.
        // Use the store's batch setter so zundo temporal middleware tracks the change.
        const storeKey = sectionToStoreKey[sectionType];
        const currentResume = useResumeBuilderStore.getState().resume;
        if (!currentResume) return;

        const items = [...(currentResume[storeKey] as T[])];
        const idx = items.findIndex(
          (i) => (i as { id: string }).id === (localItem as { id: string }).id,
        );
        if (idx !== -1) {
          items[idx] = serverItem;
        }

        storeSetters[storeKey](items);

        getOrCreateSet(resume.id).add((serverItem as { id: string }).id);
      } catch {
        useResumeBuilderStore.getState().setSaveStatus("error");
      } finally {
        pendingRef.current = false;
      }
    },
    [sectionType],
  );

  const remove = useCallback(
    async (itemId: string) => {
      const resume = useResumeBuilderStore.getState().resume;
      if (!resume) return;

      if (isServerItem(itemId)) {
        try {
          await deleteFns[sectionType](resume.id, itemId);
          serverIdsByResume.get(resume.id)?.delete(itemId);
        } catch {
          useResumeBuilderStore.getState().setSaveStatus("error");
        }
      }
    },
    [sectionType],
  );

  return { add, remove };
}
