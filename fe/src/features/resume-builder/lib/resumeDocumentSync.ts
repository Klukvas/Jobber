import { resumeBuilderService } from "@/services/resumeBuilderService";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import {
  createDocumentFetchTracker,
  createDocumentSyncQueue,
  drainWrites,
  DocumentSwitchedError,
  hasDocumentMutationFailures,
  retryDocumentMutations,
  trackPersistedTitles,
  waitForDocumentMutations,
  type DocumentSyncQueue,
  type DocumentSyncSnapshot,
} from "@/shared/lib/documentSync";
import { getQueryClient } from "@/shared/lib/queryClient";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import type { FullResumeDTO } from "@/shared/types/resume-builder";
import {
  initServerIds,
  captureResumeSectionIds,
  clearResumeHistoryIntent,
  getServerIds,
  reconcilePendingSectionMutations,
  reconcileResumeSectionStructure,
  resolveResumeSectionIds,
} from "../hooks/useSectionPersistence";

/** Array section keys that are saved item by item. */
type ArraySectionKey =
  | "experiences"
  | "educations"
  | "skills"
  | "languages"
  | "certifications"
  | "projects"
  | "volunteering"
  | "custom_sections";

/** Maps section key → service update method name. */
const SECTION_UPDATE_MAP: Record<
  ArraySectionKey,
  keyof typeof resumeBuilderService
> = {
  experiences: "updateExperience",
  educations: "updateEducation",
  skills: "updateSkill",
  languages: "updateLanguage",
  certifications: "updateCertification",
  projects: "updateProject",
  volunteering: "updateVolunteering",
  custom_sections: "updateCustomSection",
} as const;

const SECTION_KEYS = Object.keys(SECTION_UPDATE_MAP) as ArraySectionKey[];

type SectionItem = Record<string, unknown> & { id: string };
/** Serialized section arrays, used to send only the items that changed. */
type SectionSnapshot = Record<ArraySectionKey, string>;

type ResumeSnapshot = DocumentSyncSnapshot<FullResumeDTO>;

// The scope belongs to this enqueue, rather than to a mutable server registry.
const sectionIdsAtSnapshot = new WeakMap<ResumeSnapshot, ReadonlySet<string>>();

/**
 * Section state as the server last accepted it, per resume.
 *
 * It is built from the document that was actually sent — never from the live
 * store — so an edit made while a save is in flight still counts as changed on
 * the next flush instead of being diffed away.
 */
const sentSections = new Map<string, SectionSnapshot>();
const queues = new Map<string, DocumentSyncQueue<FullResumeDTO>>();

/**
 * Marks the builder list stale once a save stored a new title.
 *
 * The resumes page and the job pickers read that list, and renaming inside the
 * editor goes through auto-save, so both kept showing the old name. The list
 * key is deliberately not a prefix of the editor's own document key, so this
 * cannot refetch the resume out from under the editor.
 */
const persistedTitles = trackPersistedTitles(() => {
  void getQueryClient().invalidateQueries({
    queryKey: resumeQueryKeys.builderList(),
  });
});

/**
 * Marks the resume's own cache entry stale, without refetching it.
 *
 * The editor opens whatever that entry holds, and a save never touched it: for
 * as long as the payload counted as fresh, leaving the editor and coming back
 * re-opened the resume as it had been *before* the save — and the next edit
 * saved that revision back over the server. The thumbnails on the resumes page
 * read the same entry and were a save behind for the same reason.
 *
 * `refetchType: "none"` is deliberate. Refetching here would replace the
 * document under the user's cursor on every debounced save, clearing the undo
 * history with it; marking it stale instead means the next time the resume is
 * opened it is fetched again.
 */
function markStoredResumeStale(resumeId: string): void {
  void getQueryClient().invalidateQueries({
    queryKey: resumeQueryKeys.builderDetail(resumeId),
    refetchType: "none",
  });
}

const resumeFetches = createDocumentFetchTracker();

/**
 * Reads a resume for the editor and the thumbnails, dating the request.
 *
 * Both read the same cache entry, so both have to go through here: the payload
 * the editor opens can only be told apart from a write this client made while
 * it was on its way if the request that asked for it is dated. See
 * `createDocumentFetchTracker`.
 */
export function fetchResumeDocument(resumeId: string): Promise<FullResumeDTO> {
  return resumeFetches.track(resumeId, () =>
    resumeBuilderService.getById(resumeId),
  );
}

/**
 * When the resume the editor is opening was asked for, if that is known.
 *
 * The arrival time the cache records is the fallback: it dates a payload this
 * client never fetched itself, and it is all there is for one put into the
 * cache by hand.
 */
function payloadRequestedAt(resumeId: string): number | undefined {
  const arrivedAt = getQueryClient().getQueryState(
    resumeQueryKeys.builderDetail(resumeId),
  )?.dataUpdatedAt;
  return resumeFetches.requestStartOf(resumeId, arrivedAt) ?? arrivedAt;
}

function buildSectionSnapshot(resume: FullResumeDTO): SectionSnapshot {
  const snapshot = {} as SectionSnapshot;
  for (const key of SECTION_KEYS) {
    snapshot[key] = JSON.stringify(resume[key]);
  }
  return snapshot;
}

/** The document the editor currently holds, or null when none is loaded. */
function resumeSnapshot(): ResumeSnapshot | null {
  const resume = useResumeBuilderStore.getState().resume;
  if (!resume) return null;
  const snapshot: ResumeSnapshot = {
    documentId: resume.id,
    revision: JSON.stringify(resume),
    document: resume,
  };
  sectionIdsAtSnapshot.set(snapshot, captureResumeSectionIds(resume.id));
  return snapshot;
}

function persistTopLevel(resume: FullResumeDTO): Promise<unknown>[] {
  const writes: Promise<unknown>[] = [
    resumeBuilderService.update(resume.id, {
      title: resume.title,
      template_id: resume.template_id,
      font_family: resume.font_family,
      primary_color: resume.primary_color,
      text_color: resume.text_color,
      spacing: resume.spacing,
      margin_top: resume.margin_top,
      margin_bottom: resume.margin_bottom,
      margin_left: resume.margin_left,
      margin_right: resume.margin_right,
      layout_mode: resume.layout_mode,
      sidebar_width: resume.sidebar_width,
      font_size: resume.font_size,
      skill_display: resume.skill_display,
    }),
  ];

  if (resume.contact) {
    writes.push(resumeBuilderService.upsertContact(resume.id, resume.contact));
  }

  if (resume.summary) {
    writes.push(
      resumeBuilderService.upsertSummary(resume.id, {
        content: resume.summary.content,
      }),
    );
  }

  if (resume.section_order.length > 0) {
    writes.push(
      resumeBuilderService.updateSectionOrder(resume.id, {
        sections: resume.section_order,
      }),
    );
  }

  return writes;
}

/**
 * Sends the section items that differ from the last accepted state.
 *
 * An item that lost its last piece of content is still sent: clearing every
 * field is an edit like any other, and skipping it left the old text on the
 * server.
 */
async function persistSectionItems(resume: FullResumeDTO): Promise<void> {
  const baseline = sentSections.get(resume.id) ?? null;
  const current = buildSectionSnapshot(resume);

  for (const key of SECTION_KEYS) {
    if (baseline && current[key] === baseline[key]) continue;

    const updateFn = resumeBuilderService[SECTION_UPDATE_MAP[key]] as (
      id: string,
      entryId: string,
      data: unknown,
    ) => Promise<unknown>;

    const sentItems: SectionItem[] = baseline ? JSON.parse(baseline[key]) : [];
    const sentById = new Map(
      sentItems.map((item) => [item.id, JSON.stringify(item)]),
    );

    const writes = (resume[key] as unknown as SectionItem[])
      .filter(
        (item) =>
          getServerIds(resume.id).has(item.id) &&
          JSON.stringify(item) !== sentById.get(item.id),
      )
      .map((item) =>
        updateFn.call(resumeBuilderService, resume.id, item.id, item),
      );

    if (writes.length > 0) await drainWrites(writes);
  }
}

async function persistResume(snapshot: ResumeSnapshot): Promise<void> {
  const { documentId, document } = snapshot;

  // Section creates/deletes are fired immediately, outside this queue. The
  // document flush has to let them settle first: until a create resolves the
  // item has no server id, and a failed one means the document on the server
  // is incomplete — which is a failed save, not a successful one.
  await retryDocumentMutations(documentId);
  await waitForDocumentMutations(documentId);

  // Every write of this revision settles before the queue moves on, failures
  // included. Returning on the first rejection left siblings in flight, and
  // the next attempt then raced them with a different revision of the same
  // fields — the loser of that race decided what the server kept.
  await drainWrites(persistTopLevel(document));
  const latestDesired = () => {
    const live = useResumeBuilderStore.getState().resume;
    if (live?.id === documentId) return live;
    // A route change hands its newest outgoing revision to this same queue.
    // The active store alone cannot describe document A once B is on screen.
    return queues.get(documentId)?.queuedSnapshot()?.document ?? document;
  };
  const structured = await reconcileResumeSectionStructure(
    document,
    sectionIdsAtSnapshot.get(snapshot) ?? new Set<string>(),
    latestDesired,
  );
  await waitForDocumentMutations(documentId);
  await persistSectionItems(structured);

  sentSections.set(documentId, buildSectionSnapshot(structured));
  persistedTitles.recordPersisted(documentId, document.title);
  markStoredResumeStale(documentId);
}

function isLiveDocument(documentId: string): boolean {
  return useResumeBuilderStore.getState().resume?.id === documentId;
}

function queueFor(resumeId: string): DocumentSyncQueue<FullResumeDTO> {
  const existing = queues.get(resumeId);
  if (existing) return existing;

  const queue = createDocumentSyncQueue<FullResumeDTO>({
    persist: persistResume,
    getLatestSnapshot: resumeSnapshot,
    onSaving: (snapshot) => {
      if (isLiveDocument(snapshot.documentId)) {
        useResumeBuilderStore.getState().setSaveStatus("saving");
      }
    },
    onSynced: (snapshot) => {
      // Marking clean is only honest while the live document is still the one
      // that was sent, down to the revision. Anything else means the user has
      // newer work — or another document open — that this save did not cover.
      const live = resumeSnapshot();
      if (!live || live.documentId !== snapshot.documentId) return;
      if (live.revision !== snapshot.revision) return;

      const { markClean, setSaveStatus } = useResumeBuilderStore.getState();
      clearResumeHistoryIntent(snapshot.documentId);
      markClean();
      setSaveStatus("saved");
    },
    onError: (_error, snapshot) => {
      if (isLiveDocument(snapshot.documentId)) {
        useResumeBuilderStore.getState().setSaveStatus("error");
      }
    },
  });
  queues.set(resumeId, queue);
  return queue;
}

/**
 * Records a freshly loaded document as the state the server holds.
 *
 * Without it the first flush of an untouched resume would re-send everything.
 */
export function markResumeLoaded(resume: FullResumeDTO): void {
  sentSections.set(resume.id, buildSectionSnapshot(resume));
  persistedTitles.setKnown(resume.id, resume.title);
  queueFor(resume.id).markPersisted({
    documentId: resume.id,
    revision: JSON.stringify(resume),
    document: resume,
  });
}

/**
 * Forgets what the server is known to hold for this document.
 *
 * Used when the editor opens on top of unsaved work: nothing may be diffed
 * away on the assumption that the server already has it.
 */
export function forgetResumeBaseline(resumeId: string): void {
  sentSections.delete(resumeId);
  persistedTitles.forget(resumeId);
  queues.get(resumeId)?.markPersisted(null);
}

/** How much of what the editor holds the server has not accepted. */
export type ResumeUnsavedState = "none" | "pending" | "failed";

export interface LoadedResume {
  readonly resume: FullResumeDTO;
  readonly unsaved: ResumeUnsavedState;
}

/** The newest editor state for this resume that a save could not store. */
function failedResumeDraft(resumeId: string): FullResumeDTO | null {
  return queues.get(resumeId)?.unsavedSnapshot()?.document ?? null;
}

/** The newest editor state handed to the queue whose save has not landed yet. */
function queuedResumeDraft(resumeId: string): FullResumeDTO | null {
  return queues.get(resumeId)?.queuedSnapshot()?.document ?? null;
}

/**
 * The draft's own edits on top of a freshly fetched payload.
 *
 * Everything the editor owns comes from the draft. The per-item section arrays
 * come from `base`, which already reconciled the payload with the section
 * writes that are still on their way, and the server owns the timestamps.
 */
function withDraftEdits(
  base: FullResumeDTO,
  draft: FullResumeDTO,
): FullResumeDTO {
  return {
    ...draft,
    id: base.id,
    created_at: base.created_at,
    updated_at: base.updated_at,
    experiences: base.experiences,
    educations: base.educations,
    skills: base.skills,
    languages: base.languages,
    certifications: base.certifications,
    projects: base.projects,
    volunteering: base.volunteering,
    custom_sections: base.custom_sections,
  };
}

/**
 * What to open instead of a payload older than this client's own last save.
 *
 * `fresh` is what the editor's own query handed over, and the request that
 * asked for it is dated. A payload asked for before the last stored write is a
 * revision behind — the server could not have answered with that write — and
 * opening it would undo a save the server has already accepted, which the next
 * edit would then make permanent.
 *
 * The copy the editor holds comes first: it is that revision plus anything
 * typed since, including the server ids its section creates swapped in. With
 * another resume on screen, the revision the queue recorded as stored is the
 * next best thing, and still newer than the payload.
 */
function revisionAheadOfPayload(fresh: FullResumeDTO): FullResumeDTO | null {
  const stored = queues
    .get(fresh.id)
    ?.storedRevisionNewerThan(payloadRequestedAt(fresh.id));
  if (!stored) return null;

  const live = useResumeBuilderStore.getState().resume;
  return live?.id === fresh.id ? live : stored.document;
}

/**
 * Merges a freshly fetched resume with editor state the server never accepted.
 *
 * Three kinds of unsaved work can exist when a payload arrives: edits sitting
 * in the store (a refetch landing mid-typing), the draft a failed save left
 * behind, and a revision whose save is still running — the one a route change
 * handed over on the way out. All three outlive the editor, and handing the
 * payload straight to the store dropped them without a word.
 *
 * A fourth case is the payload itself being behind; see
 * `revisionAheadOfPayload`. The known server ids are left alone then too —
 * re-seeding them from a payload older than this client's write would forget
 * every row created since it was fetched.
 */
export function reconcileLoadedResume(fresh: FullResumeDTO): LoadedResume {
  const { resume: live, isDirty } = useResumeBuilderStore.getState();
  const ahead = revisionAheadOfPayload(fresh);

  // Resets the known server ids from the payload, so it has to run before
  // `reconcilePendingSectionMutations` adds back the rows it rescues.
  if (!ahead) initServerIds(fresh);

  const liveDraft = isDirty && live?.id === fresh.id ? live : null;
  const failedDraft = failedResumeDraft(fresh.id);
  const draft = liveDraft ?? failedDraft ?? queuedResumeDraft(fresh.id);

  const base = reconcilePendingSectionMutations(resolveResumeSectionIds(ahead ?? fresh), draft ? resolveResumeSectionIds(draft) : draft);
  const failed = failedDraft !== null || hasDocumentMutationFailures(fresh.id);

  if (!draft) return { resume: base, unsaved: failed ? "failed" : "none" };
  return {
    resume: withDraftEdits(base, draft),
    unsaved: failed ? "failed" : "pending",
  };
}

/** True while a save for this document is in flight. */
export function isResumeSaving(resumeId: string): boolean {
  return queues.get(resumeId)?.isSaving() ?? false;
}

/** Queues the live document; resolves once it (or a newer edit) is stored. */
export function enqueueResumeSave(): Promise<void> {
  const live = resumeSnapshot();
  if (!live) return Promise.resolve();
  return queueFor(live.documentId).enqueue(live, { force: useResumeBuilderStore.getState().isDirty });
}

/**
 * Hands the open document to its queue before a different one replaces it.
 *
 * The editor's id changes without the page unmounting — React Router keeps the
 * same element for `/resume-builder/:id` — so the unmount flush never runs. The
 * debounce timer is cancelled as soon as the store holds the new document, and
 * whatever had been typed went with it. Called *before* `setResume`, while the
 * outgoing revision is still the live one.
 */
export function enqueueOutgoingResumeSave(nextResumeId: string): void {
  const { resume, isDirty } = useResumeBuilderStore.getState();
  if (!resume || !isDirty || resume.id === nextResumeId) return;

  void enqueueResumeSave().catch(() => {
    // The queue keeps the revision as unsaved work, and the editor is offered
    // it back the next time it opens that resume.
  });
}

/** Saves the live document even when that exact revision was already stored. */
export async function saveResumeNow(): Promise<void> {
  const live = resumeSnapshot();
  if (!live) return;
  try {
    await queueFor(live.documentId).enqueue(live, { force: true });
  } catch {
    // The failure is already reflected in saveStatus; callers of an explicit
    // save have nothing extra to do with it.
  }
}

/**
 * Brings the server fully up to date with the editor, errors included.
 *
 * Export uses this: a PDF rendered from a half-saved document is wrong, so a
 * failure here has to surface instead of being swallowed. Pending and
 * previously failed section writes are retried first — awaiting only the
 * current document save would skip a create that never landed.
 *
 * Switching documents while those writes settle aborts the flush. Resolving
 * instead told the caller the document was stored, and the toolbar it came
 * from then exported a resume whose own save had never been awaited.
 */
export async function flushResumeDocument(resumeId: string): Promise<void> {
  await retryDocumentMutations(resumeId);
  await waitForDocumentMutations(resumeId);

  const live = resumeSnapshot();
  if (!live || live.documentId !== resumeId) {
    throw new DocumentSwitchedError(resumeId);
  }
  await queueFor(resumeId).enqueue(live, { force: useResumeBuilderStore.getState().isDirty });
}

/**
 * Prepares a persisted source for a workflow outside the editor. An inactive
 * draft must be reopened and saved explicitly; analysing the older stored
 * document would silently omit edits the user already made.
 */
export async function ensureStoredResumeDocument(resumeId: string): Promise<void> {
  if (isLiveDocument(resumeId)) {
    await flushResumeDocument(resumeId);
    return;
  }
  await waitForDocumentMutations(resumeId);
  const queue = queues.get(resumeId);
  if (queue?.isSaving() || queue?.queuedSnapshot() || queue?.unsavedSnapshot()) {
    throw new Error("Source resume has unsaved editor changes");
  }
}

/** Test seam: drops the per-document queues, baselines and fetch dates. */
export function resetResumeDocumentSync(): void {
  sentSections.clear();
  persistedTitles.clear();
  queues.clear();
  resumeFetches.clear();
}


/** Serializes a list rename after any outgoing editor save for this document. */
export async function renameResumeDocument(resumeId: string, title: string) {
  const queue = queues.get(resumeId);
  const queued = queue?.queuedSnapshot();
  if (queued && queue) await queue.enqueue(queued);
  const state = useResumeBuilderStore.getState();
  if (state.resume?.id === resumeId && state.isDirty) await enqueueResumeSave();
  const renamed = await resumeBuilderService.update(resumeId, { title });
  // The old stored revision must not take precedence over the renamed cache.
  queues.delete(resumeId);
  sentSections.delete(resumeId);
  persistedTitles.setKnown(resumeId, title);
  const live = useResumeBuilderStore.getState().resume;
  if (live?.id === resumeId) {
    const history = useResumeBuilderStore.temporal.getState();
    const wasTracking = history.isTracking;
    history.pause();
    useResumeBuilderStore.setState({ resume: { ...live, title } });
    if (wasTracking) history.resume();
  }
  return renamed;
}
