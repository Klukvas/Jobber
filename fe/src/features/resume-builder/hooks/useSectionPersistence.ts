import { useCallback } from "react";
import { adoptResumeSectionId, useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import {
  clearDocumentMutationState,
  trackDocumentMutation,
  waitForDocumentMutations,
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
const serverSectionsByResume = new Map<string, Map<string, SectionType>>();
const adoptedIdsByResume = new Map<string, Map<string, string>>();
const historyOwnsStructure = new Set<string>();
const historyRestoredIds = new Map<string, Set<string>>();

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
  const sections = new Map<string, SectionType>();
  for (const [sectionType, storeKey] of Object.entries(sectionToStoreKey) as [SectionType, StoreKey][]) {
    const arr = resume[storeKey];
    for (const item of arr) {
      // A payload older than the delete still lists the row; adopting its id
      // would make the editor write to a row the server no longer has.
      if (!removed.has(item.id)) {
        set.add(item.id);
        sections.set(item.id, sectionType);
      }
    }
  }
  serverIdsByResume.set(resume.id, set);
  serverSectionsByResume.set(resume.id, sections);
}

export function markServerIds(resumeId: string, ids: string[], sectionType?: SectionType) {
  const set = getOrCreateSet(resumeId);
  const removed = removedFor(resumeId);
  for (const id of ids) {
    if (!removed.has(id)) {
      set.add(id);
      if (sectionType) {
        const sections = serverSectionsByResume.get(resumeId) ?? new Map<string, SectionType>();
        sections.set(id, sectionType);
        serverSectionsByResume.set(resumeId, sections);
      }
      const live = useResumeBuilderStore.getState().resume;
      if (live?.id === resumeId) {
        for (const [section, key] of Object.entries(sectionToStoreKey) as [SectionType, StoreKey][]) {
          if (!live[key].some(item => item.id === id)) continue;
          const sections = serverSectionsByResume.get(resumeId) ?? new Map<string, SectionType>();
          sections.set(id, section);
          serverSectionsByResume.set(resumeId, sections);
        }
      }
    }
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
  serverSectionsByResume.delete(resumeId);
  adoptedIdsByResume.delete(resumeId);
  clearResumeHistoryIntent(resumeId);
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
  fallback?: SectionItem,
): Promise<void> {
  const storeKey = sectionToStoreKey[sectionType];
  const liveItems = liveSectionItems(resumeId, storeKey);
  const localItem = liveItems
    ? liveItems.find(item => item.id === localId)
    : fallback;

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
  const sections = serverSectionsByResume.get(resumeId) ?? new Map<string, SectionType>();
  sections.set(serverItem.id, sectionType);
  serverSectionsByResume.set(resumeId, sections);
  const aliases = adoptedIdsByResume.get(resumeId) ?? new Map<string, string>();
  aliases.set(localId, serverItem.id);
  adoptedIdsByResume.set(resumeId, aliases);
  untrackPending(pendingCreateIds, resumeId, localId);

  const items = liveSectionItems(resumeId, storeKey);
  // Even an undone CREATE gave this row a server identity. Redo must restore
  // that identity from history after the compensating DELETE has settled.
  adoptResumeSectionId(resumeId, storeKey, localId, serverItem.id);
  // A different active document does not mean the original row was removed.
  const explicitlyRemoved = removedFor(resumeId).has(localId) && !historyRestoredIds.get(resumeId)?.has(localId);
  if (explicitlyRemoved || (items && !items.some(item => item.id === localId))) {
    await deleteRemotely(sectionType, resumeId, serverItem.id);
    return;
  }

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
    serverSectionsByResume.get(resumeId)?.delete(entryId);
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
  const rescuedIds: {id: string; section: SectionType}[] = [];

  /** Removed for good, or on its way out — either way the editor is done with it. */
  const isGone = (id: string): boolean =>
    pendingDeletes.has(id) || removed.has(id);

  function merge<T extends { id: string; sort_order: number }>(
    freshItems: readonly T[],
    localItems: readonly T[] | undefined,
    section: SectionType,
  ): T[] {
    const localById = new Map(
      (localItems ?? []).map((item) => [item.id, item]),
    );
    const restored = (id: string) => historyRestoredIds.get(fresh.id)?.has(id) && localById.has(id);
    const kept = freshItems
      .filter(item => (!isGone(item.id) || restored(item.id)) &&
        (!historyOwnsStructure.has(fresh.id) || !localItems || localById.has(item.id)))
      .map(item => (draftWins || restored(item.id) ? (localById.get(item.id) ?? item) : item));
    const freshIds = new Set(freshItems.map((item) => item.id));
    const localOnly = (localItems ?? []).filter(
      (item) => !freshIds.has(item.id) && (!isGone(item.id) || restored(item.id)),
    );
    for (const item of localOnly) {
      // Absent from the payload and not waiting on a create means the payload
      // simply predates a successful one — the row does exist on the server.
      if (!pendingCreates.has(item.id) && !isGone(item.id)) rescuedIds.push({id:item.id,section});
    }

    if (localOnly.length === 0) return kept;
    return [...kept, ...localOnly].sort((a, b) => a.sort_order - b.sort_order);
  }

  const merged: FullResumeDTO = {
    ...fresh,
    experiences: merge(fresh.experiences, local?.experiences, "experiences"),
    educations: merge(fresh.educations, local?.educations, "educations"),
    skills: merge(fresh.skills, local?.skills, "skills"),
    languages: merge(fresh.languages, local?.languages, "languages"),
    certifications: merge(fresh.certifications, local?.certifications, "certifications"),
    projects: merge(fresh.projects, local?.projects, "projects"),
    volunteering: merge(fresh.volunteering, local?.volunteering, "volunteering"),
    custom_sections: merge(fresh.custom_sections, local?.custom_sections, "custom-sections"),
  };

  for (const {id, section} of rescuedIds) markServerIds(fresh.id, [id], section);

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
      const run = () => createRemotely(sectionType, resumeId, localId, localItem as unknown as SectionItem);

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


/** IDs whose presence this snapshot can legitimately reconcile, including pending creates. */
export function captureResumeSectionIds(resumeId: string): ReadonlySet<string> {
  return new Set([
    ...getServerIds(resumeId),
    ...(pendingCreateIds.get(resumeId) ?? NO_IDS),
  ]);
}

function resolveSectionId(resumeId: string, initialId: string): string {
  const aliases = adoptedIdsByResume.get(resumeId);
  const visited = new Set<string>();
  let id = initialId;
  while (aliases?.has(id) && !visited.has(id)) {
    visited.add(id);
    id = aliases.get(id) ?? id;
  }
  return id;
}

/** Resolves IDs in outgoing snapshots retained while another document is open. */
export function resolveResumeSectionIds(resume: FullResumeDTO): FullResumeDTO {
  if (!adoptedIdsByResume.has(resume.id)) return resume;
  let resolved = resume;
  for (const key of Object.values(sectionToStoreKey)) {
    if (!resolved[key].some(item => resolveSectionId(resume.id, item.id) !== item.id)) continue;
    resolved = {
      ...resolved,
      [key]: resolved[key].map(item => ({
        ...item,
        id: resolveSectionId(resume.id, item.id),
      })),
    };
  }
  return resolved;
}

/** History changes arrays without add/remove hooks; flush reconciles their structure. */
export async function reconcileResumeSectionStructure(
  resume: FullResumeDTO,
  knownAtSnapshot: ReadonlySet<string>,
  getLatestDesired: () => FullResumeDTO,
): Promise<FullResumeDTO> {
  const latestDesired = () => resolveResumeSectionIds(getLatestDesired());
  for (const [sectionType, storeKey] of Object.entries(sectionToStoreKey) as [SectionType, StoreKey][]) {
    for (const candidate of latestDesired()[storeKey]) {
      if (getServerIds(resume.id).has(candidate.id)) continue;
      // An immediate CREATE may have started after this save's initial wait.
      // Let it settle before deciding this row still needs a structural CREATE.
      await waitForDocumentMutations(resume.id);
      const currentId = resolveSectionId(resume.id, candidate.id);
      const item = latestDesired()[storeKey].find(current => current.id === currentId);
      // A newer Undo or outgoing revision may have removed it during the wait.
      if (!item || getServerIds(resume.id).has(item.id)) continue;
      if (removedFor(resume.id).has(item.id)) trackPending(historyRestoredIds, resume.id, item.id);
      await createRemotely(sectionType, resume.id, item.id, item as unknown as SectionItem);
    }
    const eligible = new Set(
      [...knownAtSnapshot].map(id => resolveSectionId(resume.id, id)),
    );
    for (const [id, section] of serverSectionsByResume.get(resume.id) ?? []) {
      if (section !== sectionType || !eligible.has(id)) continue;
      // A save only owns rows it knew about when it was queued. Re-read the
      // current intent after each await; newer rows and Redo must survive.
      if (latestDesired()[storeKey].some(item => item.id === id)) continue;
      await deleteRemotely(sectionType, resume.id, id);
    }
  }
  return resolveResumeSectionIds(resume);
}


/** Keeps history's explicit row presence authoritative while a stale fetch lands. */
export function noteResumeHistoryChange(resume: FullResumeDTO): void {
  historyOwnsStructure.add(resume.id);
  const restored = new Set<string>();
  for (const key of Object.values(sectionToStoreKey)) {
    for (const item of resume[key]) {
      if (removedFor(resume.id).has(item.id) || pendingDeleteIds.get(resume.id)?.has(item.id)) restored.add(item.id);
    }
  }
  historyRestoredIds.set(resume.id, restored);
}

export function clearResumeHistoryIntent(resumeId: string): void {
  historyOwnsStructure.delete(resumeId);
  historyRestoredIds.delete(resumeId);
}
