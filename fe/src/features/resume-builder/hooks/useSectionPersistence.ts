import { useCallback } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import {
  clearDocumentMutationState,
  trackDocumentMutation,
} from "@/shared/lib/documentSync";
import type {
  FullResumeDTO,
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

type SectionItem = { id: string; sort_order: number } & Record<string, unknown>;

/**
 * Tracks which item IDs exist on the server (were fetched or successfully created).
 * Items with IDs NOT in this set are considered "local-only" and need CREATE.
 * Items with IDs IN this set need UPDATE.
 *
 * Keyed by resumeId to prevent cross-resume contamination when navigating
 * between different resumes in the same SPA session.
 */
const serverIdsByResume = new Map<string, Set<string>>();

/** Local ids whose create has not landed yet — still in flight, or failed. */
const pendingCreateIds = new Map<string, Set<string>>();

/** Server ids whose delete has not landed yet — still in flight, or failed. */
const pendingDeleteIds = new Map<string, Set<string>>();

/**
 * Ids the user removed for good — a delete the server confirmed, or a row that
 * never reached it.
 *
 * A payload fetched before the removal still carries the row, and nothing
 * pending says so any more. Taking it back put the deleted row on screen again
 * and marked it as one the server holds, so the next save patched an entry that
 * was gone — the 404 left the editor stuck on "save failed".
 */
const removedIds = new Map<string, Set<string>>();

const NO_IDS: ReadonlySet<string> = new Set<string>();

function getOrCreateSet(resumeId: string): Set<string> {
  let set = serverIdsByResume.get(resumeId);
  if (!set) {
    set = new Set<string>();
    serverIdsByResume.set(resumeId, set);
  }
  return set;
}

function trackPending(
  registry: Map<string, Set<string>>,
  resumeId: string,
  id: string,
) {
  const ids = registry.get(resumeId) ?? new Set<string>();
  ids.add(id);
  registry.set(resumeId, ids);
}

function untrackPending(
  registry: Map<string, Set<string>>,
  resumeId: string,
  id: string,
) {
  const ids = registry.get(resumeId);
  if (!ids) return;
  ids.delete(id);
  if (ids.size === 0) registry.delete(resumeId);
}

/** Rows the editor must never show or write again for this resume. */
function removedFor(resumeId: string): ReadonlySet<string> {
  return removedIds.get(resumeId) ?? NO_IDS;
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
  const removed = removedFor(resume.id);
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
      // A payload older than the delete still lists the row; adopting its id
      // would make the editor write to a row the server no longer has.
      if (!removed.has(item.id)) set.add(item.id);
    }
  }
  serverIdsByResume.set(resume.id, set);
}

export function markServerIds(resumeId: string, ids: string[]) {
  const set = getOrCreateSet(resumeId);
  const removed = removedFor(resumeId);
  for (const id of ids) {
    if (!removed.has(id)) set.add(id);
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
  pendingCreateIds.delete(resumeId);
  pendingDeleteIds.delete(resumeId);
  removedIds.delete(resumeId);
  clearDocumentMutationState(resumeId);
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

function createMutationKey(sectionType: SectionType, localId: string): string {
  return `create:${sectionType}:${localId}`;
}

function deleteMutationKey(sectionType: SectionType, entryId: string): string {
  return `delete:${sectionType}:${entryId}`;
}

function liveSectionItems(
  resumeId: string,
  storeKey: StoreKey,
): SectionItem[] | null {
  const resume = useResumeBuilderStore.getState().resume;
  if (!resume || resume.id !== resumeId) return null;
  return resume[storeKey] as unknown as SectionItem[];
}

function withoutId(item: SectionItem): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(item).filter(([key]) => key !== "id"),
  );
}

/**
 * Creates the row the user just added, carrying its newest local content.
 *
 * The payload is read from the store rather than from the value the caller
 * held, so a retry sends what the user has typed since the first attempt.
 */
async function createRemotely(
  sectionType: SectionType,
  resumeId: string,
  localId: string,
): Promise<void> {
  const storeKey = sectionToStoreKey[sectionType];
  const localItem = liveSectionItems(resumeId, storeKey)?.find(
    (item) => item.id === localId,
  );
  if (!localItem) {
    // The row is gone — removed, or the editor moved to another document.
    // Creating it now would resurrect content the user dropped.
    untrackPending(pendingCreateIds, resumeId, localId);
    return;
  }

  const serverItem = (await createFns[sectionType](
    resumeId,
    withoutId(localItem),
  )) as SectionItem;
  getOrCreateSet(resumeId).add(serverItem.id);
  untrackPending(pendingCreateIds, resumeId, localId);

  const items = liveSectionItems(resumeId, storeKey);
  const index = items?.findIndex((item) => item.id === localId) ?? -1;
  if (!items || index === -1) {
    // Removed while the create was in flight — take the new row back off the
    // server instead of leaving an orphan behind.
    void deleteRemotely(sectionType, resumeId, serverItem.id);
    return;
  }

  // Adopt the server id but keep everything the user typed in the meantime.
  // Overwriting the row with the server copy threw those keystrokes away.
  const next = [...items];
  next[index] = { ...items[index], id: serverItem.id };
  storeSetters[storeKey](next);
}

async function deleteRemotely(
  sectionType: SectionType,
  resumeId: string,
  entryId: string,
): Promise<void> {
  trackPending(pendingDeleteIds, resumeId, entryId);
  // The bookkeeping travels with the request: a flush retries `run` on its
  // own, and leaving it out there kept the row listed as one the server holds.
  const run = async () => {
    await deleteFns[sectionType](resumeId, entryId);
    serverIdsByResume.get(resumeId)?.delete(entryId);
    untrackPending(pendingDeleteIds, resumeId, entryId);
    trackPending(removedIds, resumeId, entryId);
  };

  try {
    await trackDocumentMutation(
      resumeId,
      run(),
      deleteMutationKey(sectionType, entryId),
      run,
    );
  } catch {
    // The id stays known and pending so a flush retries the delete.
    useResumeBuilderStore.getState().setSaveStatus("error");
  }
}

/**
 * Rebuilds a freshly fetched document around section writes still in flight.
 *
 * A payload fetched before (or while) a row was created or deleted would
 * otherwise undo the user's work on `setResume`: the new row disappears and
 * the deleted one comes back. Call it after `initServerIds`, which resets the
 * known server ids from the same payload.
 *
 * `draft` is editor state the server never accepted — a failed save, or edits
 * made while the payload was on its way. Its copy of a row wins over the
 * payload's, because the payload predates those keystrokes. Without a draft
 * the live document only decides which rows exist, never their content.
 */
export function reconcilePendingSectionMutations(
  fresh: FullResumeDTO,
  draft?: FullResumeDTO | null,
): FullResumeDTO {
  const live = useResumeBuilderStore.getState().resume;
  const local = draft ?? (live?.id === fresh.id ? live : null);
  const draftWins = draft != null;
  const pendingDeletes = pendingDeleteIds.get(fresh.id) ?? NO_IDS;
  const pendingCreates = pendingCreateIds.get(fresh.id) ?? NO_IDS;
  const removed = removedFor(fresh.id);
  const rescuedIds: string[] = [];

  /** Removed for good, or on its way out — either way the editor is done with it. */
  const isGone = (id: string): boolean =>
    pendingDeletes.has(id) || removed.has(id);

  function merge<T extends { id: string; sort_order: number }>(
    freshItems: readonly T[],
    localItems: readonly T[] | undefined,
  ): T[] {
    const localById = new Map(
      (localItems ?? []).map((item) => [item.id, item]),
    );
    const kept = freshItems
      .filter((item) => !isGone(item.id))
      .map((item) => (draftWins ? (localById.get(item.id) ?? item) : item));
    const freshIds = new Set(freshItems.map((item) => item.id));
    const localOnly = (localItems ?? []).filter(
      (item) => !freshIds.has(item.id) && !isGone(item.id),
    );
    for (const item of localOnly) {
      // Absent from the payload and not waiting on a create means the payload
      // simply predates a successful one — the row does exist on the server.
      if (!pendingCreates.has(item.id)) rescuedIds.push(item.id);
    }

    if (localOnly.length === 0) return kept;
    return [...kept, ...localOnly].sort((a, b) => a.sort_order - b.sort_order);
  }

  const merged: FullResumeDTO = {
    ...fresh,
    experiences: merge(fresh.experiences, local?.experiences),
    educations: merge(fresh.educations, local?.educations),
    skills: merge(fresh.skills, local?.skills),
    languages: merge(fresh.languages, local?.languages),
    certifications: merge(fresh.certifications, local?.certifications),
    projects: merge(fresh.projects, local?.projects),
    volunteering: merge(fresh.volunteering, local?.volunteering),
    custom_sections: merge(fresh.custom_sections, local?.custom_sections),
  };

  if (rescuedIds.length > 0) markServerIds(fresh.id, rescuedIds);

  return merged;
}

/**
 * Hook that provides add/remove functions that immediately sync with the API.
 *
 * `add` expects the item to already be in the store — it tracks it by id and
 * sends the store copy, so content typed before the request goes out is kept.
 * Both writes are registered as document mutations, which a flush or an export
 * waits for (and retries) before it treats the document as saved.
 */
export function useSectionPersistence<T extends DTOMap[SectionType]>(
  sectionType: SectionType,
) {
  const add = useCallback(
    async (localItem: T) => {
      const resume = useResumeBuilderStore.getState().resume;
      if (!resume) return;

      const resumeId = resume.id;
      const localId = (localItem as unknown as SectionItem).id;
      trackPending(pendingCreateIds, resumeId, localId);
      const run = () => createRemotely(sectionType, resumeId, localId);

      try {
        await trackDocumentMutation(
          resumeId,
          run(),
          createMutationKey(sectionType, localId),
          run,
        );
      } catch {
        useResumeBuilderStore.getState().setSaveStatus("error");
      }
    },
    [sectionType],
  );

  const remove = useCallback(
    async (itemId: string) => {
      const resume = useResumeBuilderStore.getState().resume;
      if (!resume) return;

      if (isServerItem(itemId)) {
        await deleteRemotely(sectionType, resume.id, itemId);
        return;
      }

      // Nothing on the server yet. A create that failed earlier has nothing
      // left to retry now that the user removed the row, and a draft still
      // holding it must not put it back.
      untrackPending(pendingCreateIds, resume.id, itemId);
      trackPending(removedIds, resume.id, itemId);
      clearDocumentMutationState(
        resume.id,
        createMutationKey(sectionType, itemId),
      );
    },
    [sectionType],
  );

  return { add, remove };
}
