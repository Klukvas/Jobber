import { coverLetterService } from "@/services/coverLetterService";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import {
  createDocumentFetchTracker,
  createDocumentSyncQueue,
  DocumentSwitchedError,
  trackPersistedTitles,
  type DocumentSyncQueue,
  type DocumentSyncSnapshot,
} from "@/shared/lib/documentSync";
import { getQueryClient } from "@/shared/lib/queryClient";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";
import { coverLetterQueryKeys } from "./coverLetterQueryKeys";

type CoverLetterSnapshot = DocumentSyncSnapshot<CoverLetterDTO>;

const queues = new Map<string, DocumentSyncQueue<CoverLetterDTO>>();

/**
 * Marks the letters list stale once a save stored a new title.
 *
 * The editor's own auto-save already refreshes that list while the page is
 * mounted. This covers the saves that outlive it — the revision a route change
 * hands over on the way out — which would otherwise leave the list showing the
 * name the letter was renamed away from.
 */
const persistedTitles = trackPersistedTitles(() => {
  void getQueryClient().invalidateQueries({
    queryKey: coverLetterQueryKeys.list(),
  });
});

/**
 * Marks the letter's own cache entry stale, without refetching it.
 *
 * The editor opens whatever that entry holds, and a save never touched it: for
 * as long as the payload counted as fresh, leaving the editor and coming back
 * re-opened the letter as it had been *before* the save — and the next edit
 * saved that revision back over the server.
 *
 * `refetchType: "none"` is deliberate. Refetching here would replace the
 * document under the user's cursor on every debounced save, clearing the undo
 * history with it; marking it stale instead means the next time the letter is
 * opened it is fetched again.
 */
function markStoredLetterStale(documentId: string): void {
  void getQueryClient().invalidateQueries({
    queryKey: coverLetterQueryKeys.detail(documentId),
    refetchType: "none",
  });
}

const letterFetches = createDocumentFetchTracker();

/**
 * Reads a letter for the editor, dating the request that asks for it.
 *
 * Every read of this cache entry goes through here, so the payload the editor
 * opens can be told apart from a write this client made while it was on its
 * way; see `createDocumentFetchTracker`.
 */
export function fetchCoverLetterDocument(
  documentId: string,
): Promise<CoverLetterDTO> {
  return letterFetches.track(documentId, () =>
    coverLetterService.getById(documentId),
  );
}

/**
 * When the letter the editor is opening was asked for, if that is known.
 *
 * The arrival time the cache records is the fallback: it dates a payload this
 * client never fetched itself, and it is all there is for one put into the
 * cache by hand.
 */
function payloadRequestedAt(documentId: string): number | undefined {
  const arrivedAt = getQueryClient().getQueryState(
    coverLetterQueryKeys.detail(documentId),
  )?.dataUpdatedAt;
  return letterFetches.requestStartOf(documentId, arrivedAt) ?? arrivedAt;
}

/** The document the editor currently holds, or null when none is loaded. */
function coverLetterSnapshot(): CoverLetterSnapshot | null {
  const coverLetter = useCoverLetterStore.getState().coverLetter;
  if (!coverLetter) return null;
  return {
    documentId: coverLetter.id,
    revision: JSON.stringify(coverLetter),
    document: coverLetter,
  };
}

async function persistCoverLetter(snapshot: CoverLetterSnapshot) {
  const letter = snapshot.document;
  await coverLetterService.update(letter.id, {
    title: letter.title,
    template: letter.template,
    recipient_name: letter.recipient_name,
    recipient_title: letter.recipient_title,
    company_name: letter.company_name,
    company_address: letter.company_address,
    greeting: letter.greeting,
    paragraphs: letter.paragraphs,
    closing: letter.closing,
    font_family: letter.font_family,
    font_size: letter.font_size,
    primary_color: letter.primary_color,
    job_id: letter.job_id,
  });
  persistedTitles.recordPersisted(letter.id, letter.title);
  markStoredLetterStale(letter.id);
}

function isLiveDocument(documentId: string): boolean {
  return useCoverLetterStore.getState().coverLetter?.id === documentId;
}

function queueFor(documentId: string): DocumentSyncQueue<CoverLetterDTO> {
  const existing = queues.get(documentId);
  if (existing) return existing;

  const queue = createDocumentSyncQueue<CoverLetterDTO>({
    persist: persistCoverLetter,
    getLatestSnapshot: coverLetterSnapshot,
    onSaving: (snapshot) => {
      if (isLiveDocument(snapshot.documentId)) {
        useCoverLetterStore.getState().setSaveStatus("saving");
      }
    },
    onSynced: (snapshot) => {
      // Only the exact revision that was sent may be called clean. Anything
      // newer — or another letter — still has unsaved work in it.
      const live = coverLetterSnapshot();
      if (!live || live.documentId !== snapshot.documentId) return;
      if (live.revision !== snapshot.revision) return;

      const { markClean, setSaveStatus } = useCoverLetterStore.getState();
      markClean();
      setSaveStatus("saved");
    },
    onError: (_error, snapshot) => {
      if (isLiveDocument(snapshot.documentId)) {
        useCoverLetterStore.getState().setSaveStatus("error");
      }
    },
  });
  queues.set(documentId, queue);
  return queue;
}

/** Records a freshly loaded document as the state the server holds. */
export function markCoverLetterLoaded(coverLetter: CoverLetterDTO): void {
  persistedTitles.setKnown(coverLetter.id, coverLetter.title);
  queueFor(coverLetter.id).markPersisted({
    documentId: coverLetter.id,
    revision: JSON.stringify(coverLetter),
    document: coverLetter,
  });
}

/** Forgets what the server is known to hold for this document. */
export function forgetCoverLetterBaseline(documentId: string): void {
  persistedTitles.forget(documentId);
  queues.get(documentId)?.markPersisted(null);
}

/** How much of what the editor holds the server has not accepted. */
export type CoverLetterUnsavedState = "none" | "pending" | "failed";

export interface LoadedCoverLetter {
  readonly coverLetter: CoverLetterDTO;
  readonly unsaved: CoverLetterUnsavedState;
}

/**
 * What to open instead of a payload older than this client's own last save.
 *
 * `fresh` is what the editor's own query handed over, and the request that
 * asked for it is dated. A payload asked for before the last stored write is a
 * revision behind — the server could not have answered with that write —
 * and opening it would undo a save the server has already accepted, which the
 * next edit would then make permanent.
 *
 * The copy the editor holds comes first: it is that revision plus anything
 * typed since. With another letter on screen, the revision the queue recorded
 * as stored is the next best thing, and still newer than the payload.
 */
function revisionAheadOfPayload(fresh: CoverLetterDTO): CoverLetterDTO | null {
  const stored = queues
    .get(fresh.id)
    ?.storedRevisionNewerThan(payloadRequestedAt(fresh.id));
  if (!stored) return null;

  const live = useCoverLetterStore.getState().coverLetter;
  return live?.id === fresh.id ? live : stored.document;
}

/**
 * Merges a freshly fetched letter with editor state the server never accepted.
 *
 * Both the draft a failed save leaves behind and a revision whose save is still
 * running outlive the editor, so they are still there after a trip to another
 * letter — where the fresh payload used to replace the customer's unsaved text
 * without a word. A payload the cache kept from before the last save is
 * replaced as well; see `revisionAheadOfPayload`.
 */
export function reconcileLoadedCoverLetter(
  fresh: CoverLetterDTO,
): LoadedCoverLetter {
  const { coverLetter: live, isDirty } = useCoverLetterStore.getState();
  const queue = queues.get(fresh.id);
  const base = revisionAheadOfPayload(fresh) ?? fresh;

  const liveDraft = isDirty && live?.id === fresh.id ? live : null;
  const failedDraft = queue?.unsavedSnapshot()?.document ?? null;
  const draft =
    liveDraft ?? failedDraft ?? queue?.queuedSnapshot()?.document ?? null;

  if (!draft) return { coverLetter: base, unsaved: "none" };
  return {
    // The editor owns every field it can type in; the server owns the
    // identity and the timestamps.
    coverLetter: {
      ...draft,
      id: fresh.id,
      created_at: base.created_at,
      updated_at: base.updated_at,
    },
    unsaved: failedDraft ? "failed" : "pending",
  };
}

/** True while a save for this document is in flight. */
export function isCoverLetterSaving(documentId: string): boolean {
  return queues.get(documentId)?.isSaving() ?? false;
}

/** Queues the live document; resolves once it (or a newer edit) is stored. */
export function enqueueCoverLetterSave(): Promise<void> {
  const live = coverLetterSnapshot();
  if (!live) return Promise.resolve();
  return queueFor(live.documentId).enqueue(live);
}

/**
 * Hands the open letter to its queue before a different one replaces it.
 *
 * The editor's id changes without the page unmounting — React Router keeps the
 * same element for `/cover-letters/:id` — so the unmount flush never runs. The
 * debounce timer is cancelled as soon as the store holds the new letter, and
 * whatever had been typed went with it. Called *before* `setCoverLetter`, while
 * the outgoing revision is still the live one.
 */
export function enqueueOutgoingCoverLetterSave(nextDocumentId: string): void {
  const { coverLetter, isDirty } = useCoverLetterStore.getState();
  if (!coverLetter || !isDirty || coverLetter.id === nextDocumentId) return;

  void enqueueCoverLetterSave().catch(() => {
    // The queue keeps the revision as unsaved work, and the editor is offered
    // it back the next time it opens that letter.
  });
}

/** Saves the live document even when that exact revision was already stored. */
export async function saveCoverLetterNow(): Promise<void> {
  const live = coverLetterSnapshot();
  if (!live) return;
  try {
    await queueFor(live.documentId).enqueue(live, { force: true });
  } catch {
    // The failure is already reflected in saveStatus.
  }
}

/**
 * Brings the server up to date with the editor, errors included.
 *
 * Export renders the stored document, so a failure here has to surface rather
 * than produce a file from the previous version. A flush for a letter the
 * editor has left aborts for the same reason: reporting success would export a
 * letter this flush never saved.
 */
export async function flushCoverLetterDocument(
  documentId: string,
): Promise<void> {
  const live = coverLetterSnapshot();
  if (!live || live.documentId !== documentId) {
    throw new DocumentSwitchedError(documentId);
  }
  await queueFor(documentId).enqueue(live);
}

/** Test seam: drops the per-document queues, known titles and fetch dates. */
export function resetCoverLetterDocumentSync(): void {
  persistedTitles.clear();
  queues.clear();
  letterFetches.clear();
}
