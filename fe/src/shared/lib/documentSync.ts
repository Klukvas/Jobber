export interface DocumentSyncSnapshot<TDocument> {
  readonly documentId: string;
  readonly revision: string;
  readonly document: TDocument;
}

interface DocumentSyncOptions<TDocument> {
  readonly persist: (
    snapshot: DocumentSyncSnapshot<TDocument>,
  ) => Promise<void>;
  readonly getLatestSnapshot: () => DocumentSyncSnapshot<TDocument> | null;
  readonly onSaving?: (snapshot: DocumentSyncSnapshot<TDocument>) => void;
  readonly onSynced?: (snapshot: DocumentSyncSnapshot<TDocument>) => void;
  readonly onError?: (
    error: unknown,
    snapshot: DocumentSyncSnapshot<TDocument>,
  ) => void;
}

interface EnqueueOptions {
  /** Persists even when this exact revision was already stored. */
  readonly force?: boolean;
}

export interface DocumentSyncQueue<TDocument> {
  enqueue: (
    snapshot: DocumentSyncSnapshot<TDocument>,
    options?: EnqueueOptions,
  ) => Promise<void>;
  /** Records what the server holds; `null` when that is no longer known. */
  markPersisted: (snapshot: DocumentSyncSnapshot<TDocument> | null) => void;
  isSaving: () => boolean;
  /**
   * The newest editor state this queue failed to store, or `null` once the
   * server is known to hold what the editor showed.
   *
   * It is the in-memory draft the editor falls back to: a failed save used to
   * leave nothing behind, so a document switch plus a fresh query silently
   * replaced the customer's unsaved work with the stored version.
   */
  unsavedSnapshot: () => DocumentSyncSnapshot<TDocument> | null;
  /**
   * The newest revision this queue took on but has not stored yet — the one in
   * flight, or the one waiting behind it.
   *
   * `unsavedSnapshot` only knows about saves that already failed. Coming back
   * to a document whose save is still running needs this one too, otherwise the
   * fresh payload replaces work that was merely still on its way.
   */
  queuedSnapshot: () => DocumentSyncSnapshot<TDocument> | null;
  /**
   * The revision this queue stored, when the request behind the payload a
   * query handed over left the client before that write — so the server
   * cannot have put the write in it.
   *
   * A cached document keeps being served while it counts as fresh, and the
   * editor's own save does not refresh it. Reopening a document inside that
   * window handed the editor a pre-save revision, which the next edit then
   * saved back over the server. `null` means the payload may be opened:
   * either it was asked for after the write, or nothing is known about when
   * it was asked for.
   */
  storedRevisionNewerThan: (
    requestedAt: number | undefined,
  ) => DocumentSyncSnapshot<TDocument> | null;
}

/** When the request behind a delivered payload left the client. */
interface DocumentFetchDelivery {
  readonly startedAt: number;
  readonly settledAt: number;
}

export interface DocumentFetchTracker {
  /** Runs a document fetch, recording when its request left the client. */
  track: <TPayload>(
    documentId: string,
    fetchDocument: () => Promise<TPayload>,
  ) => Promise<TPayload>;
  /**
   * When the request behind the cached payload left, if that can be told.
   *
   * `dataUpdatedAt` is the cache entry the answer has to describe; `undefined`
   * comes back when no tracked delivery matches it.
   */
  requestStartOf: (
    documentId: string,
    dataUpdatedAt: number | undefined,
  ) => number | undefined;
  /** Test seam: forgets every recorded delivery. */
  clear: () => void;
}

/**
 * Dates the payload a query serves by when its request left the client.
 *
 * `dataUpdatedAt` is the moment a response *arrived*, which proves nothing
 * about what the server put in it. A GET issued before a save and delivered
 * after it looks newer than that save, so the editor adopted a revision the
 * server had already replaced — and the next edit saved that revision back,
 * undoing the save. The moment the request left is the honest stamp: the
 * server cannot have answered it with a write that had not happened yet.
 *
 * Only a delivery updates the record, and the cache holds whatever was
 * delivered last, so the record describes the payload the cache is serving.
 *
 * The request is dated rather than cancelled on purpose. Cancelling the
 * in-flight GET would also work, but React Query reverts a cancelled fetch,
 * and a cancelled *first* load has no data to revert to — that leaves the
 * editor on a document it never received.
 */
export function createDocumentFetchTracker(): DocumentFetchTracker {
  const deliveries = new Map<string, DocumentFetchDelivery>();

  return {
    track: async (documentId, fetchDocument) => {
      const startedAt = Date.now();
      const payload = await fetchDocument();
      deliveries.set(documentId, { startedAt, settledAt: Date.now() });
      return payload;
    },
    requestStartOf: (documentId, dataUpdatedAt) => {
      const delivery = deliveries.get(documentId);
      if (!delivery || dataUpdatedAt === undefined) return undefined;

      // A delivery that settled after the cache entry was written is not the
      // payload the cache holds — a fetch React Query discarded, whose request
      // resolved late, or data put there by hand. Dating the cached payload by
      // it would be a guess, so the caller falls back to what the cache knows.
      return delivery.settledAt <= dataUpdatedAt
        ? delivery.startedAt
        : undefined;
    },
    clear: () => deliveries.clear(),
  };
}

/** Keeps track of the document titles the server is known to hold. */
export interface PersistedTitles {
  /** The title a freshly loaded document arrived with. */
  setKnown: (documentId: string, title: string) => void;
  forget: (documentId: string) => void;
  /** Records a title a save just stored, reporting it when it changed. */
  recordPersisted: (documentId: string, title: string) => void;
  /** Test seam: drops every known title. */
  clear: () => void;
}

/**
 * Reports a stored title change, so the lists that show it can be refreshed.
 *
 * The screens outside an editor read a list that carries the title. Renaming
 * happens inside the editor and goes through auto-save, so nothing invalidated
 * those lists and they kept showing the old name. `onTitleChanged` fires only
 * once a write came back and only when the title really moved: a refetch per
 * keystroke would be pointless traffic, and one for a save that failed would
 * present the old name as though it had been kept.
 */
export function trackPersistedTitles(
  onTitleChanged: (documentId: string) => void,
): PersistedTitles {
  const titles = new Map<string, string>();

  return {
    setKnown: (documentId, title) => {
      titles.set(documentId, title);
    },
    forget: (documentId) => {
      titles.delete(documentId);
    },
    recordPersisted: (documentId, title) => {
      const previousTitle = titles.get(documentId);
      titles.set(documentId, title);
      if (previousTitle !== title) onTitleChanged(documentId);
    },
    clear: () => titles.clear(),
  };
}

/** Thrown when a flush is asked for a document the editor no longer shows. */
export class DocumentSwitchedError extends Error {
  readonly documentId: string;

  constructor(documentId: string) {
    super(`document ${documentId} is no longer open in the editor`);
    this.name = "DocumentSwitchedError";
    this.documentId = documentId;
  }
}

/**
 * Awaits every write, then reports the first failure.
 *
 * `Promise.all` hands back the first rejection while its siblings are still on
 * their way. A retry or a newer revision starting in that window raced those
 * leftovers, and whichever landed last won — sometimes the older payload.
 */
export async function drainWrites(
  writes: readonly Promise<unknown>[],
): Promise<void> {
  const results = await Promise.allSettled(writes);
  for (const result of results) {
    if (result.status === "rejected") throw result.reason;
  }
}

const pendingDocumentMutations = new Map<string, Set<Promise<unknown>>>();
interface DocumentMutationFailure {
  readonly error: unknown;
  readonly retry?: () => Promise<unknown>;
}

const documentMutationErrors = new Map<
  string,
  Map<string, DocumentMutationFailure>
>();

/** Registers immediate section creates/deletes that a full-document flush must await. */
export function trackDocumentMutation<TResult>(
  documentId: string,
  operation: Promise<TResult>,
  mutationKey = "document",
  retry?: () => Promise<TResult>,
): Promise<TResult> {
  const mutations = pendingDocumentMutations.get(documentId) ?? new Set();
  pendingDocumentMutations.set(documentId, mutations);
  mutations.add(operation);
  documentMutationErrors.get(documentId)?.delete(mutationKey);

  const remove = () => {
    mutations.delete(operation);
    if (mutations.size === 0) pendingDocumentMutations.delete(documentId);
  };
  void operation.then(
    () => {
      remove();
      const errors = documentMutationErrors.get(documentId);
      errors?.delete(mutationKey);
      if (errors?.size === 0) documentMutationErrors.delete(documentId);
    },
    (error: unknown) => {
      remove();
      const errors = documentMutationErrors.get(documentId) ?? new Map();
      errors.set(mutationKey, { error, retry });
      documentMutationErrors.set(documentId, errors);
    },
  );
  return operation;
}

/** Waits until all section-level writes registered for this document settle. */
export async function waitForDocumentMutations(
  documentId: string,
): Promise<void> {
  while (true) {
    const mutations = pendingDocumentMutations.get(documentId);
    if (!mutations?.size) break;
    // Every write has to settle, failures included. Leaving early on the first
    // rejection let a document save start while a section write was still
    // running, so the two overlapped on the same rows.
    await Promise.allSettled([...mutations]);
  }

  const errors = documentMutationErrors.get(documentId);
  if (errors?.size) {
    const firstError = errors.values().next();
    if (!firstError.done) throw firstError.value.error;
  }
}

/** Retries section writes that failed before a user-requested flush/export. */
export async function retryDocumentMutations(
  documentId: string,
): Promise<void> {
  const failures = documentMutationErrors.get(documentId);
  if (!failures?.size) return;

  await drainWrites(
    [...failures.entries()].map(([mutationKey, failure]) => {
      if (!failure.retry) return Promise.reject(failure.error);
      return trackDocumentMutation(
        documentId,
        failure.retry(),
        mutationKey,
        failure.retry,
      );
    }),
  );
}

/**
 * Forgets section-write failures that can no longer be retried.
 *
 * With a `mutationKey` it drops that single failure — the item it belonged to
 * is gone, so retrying it would resurrect deleted content. Without one it
 * clears the whole document, but only while nothing is still in flight.
 */
export function clearDocumentMutationState(
  documentId: string,
  mutationKey?: string,
): void {
  const errors = documentMutationErrors.get(documentId);
  if (!errors) return;

  if (mutationKey !== undefined) {
    errors.delete(mutationKey);
    if (errors.size === 0) documentMutationErrors.delete(documentId);
    return;
  }

  if (pendingDocumentMutations.get(documentId)?.size) return;
  documentMutationErrors.delete(documentId);
}

export function hasDocumentMutationFailures(documentId: string): boolean {
  return Boolean(documentMutationErrors.get(documentId)?.size);
}

function snapshotKey<TDocument>(
  snapshot: DocumentSyncSnapshot<TDocument>,
): string {
  return `${snapshot.documentId}:${snapshot.revision}`;
}

/**
 * Serializes saves for one editor and coalesces intermediate revisions.
 *
 * A successful request is only considered synchronized when the live document
 * still matches the request snapshot. If the user edited while it was in
 * flight, the queue immediately persists the newest snapshot before resolving.
 * When the editor moved on to a different document the result is neither
 * reported as synchronized nor as failed — the save belonged to a document
 * nobody is looking at any more.
 */
export function createDocumentSyncQueue<TDocument>(
  options: DocumentSyncOptions<TDocument>,
): DocumentSyncQueue<TDocument> {
  let pendingSnapshot: DocumentSyncSnapshot<TDocument> | null = null;
  let activeSnapshot: DocumentSyncSnapshot<TDocument> | null = null;
  let activeRun: Promise<void> | null = null;
  let running = false;
  let persistedKey: string | null = null;
  let unsavedSnapshot: DocumentSyncSnapshot<TDocument> | null = null;
  // The revision this queue last had accepted, and when. Only a write of its
  // own counts: a document handed over by `markPersisted` says what the server
  // holds, not that this client put it there, so it cannot date a payload.
  let storedSnapshot: DocumentSyncSnapshot<TDocument> | null = null;
  let storedAt: number | null = null;

  /**
   * Keeps the work the server does not have, so the editor can offer it back.
   *
   * The newest revision of that document wins. The live document is it while
   * the editor still shows it; once the editor has moved on, a revision that
   * was handed over explicitly — by an unmount or a route change — is still
   * newer than the payload that happened to fail, and keeping the failed one
   * instead offered the customer back a draft they had already typed past.
   */
  const rememberUnsaved = (
    failedSnapshot: DocumentSyncSnapshot<TDocument>,
  ): void => {
    const latestSnapshot = options.getLatestSnapshot();
    if (latestSnapshot?.documentId === failedSnapshot.documentId) {
      unsavedSnapshot = latestSnapshot;
      return;
    }

    unsavedSnapshot =
      pendingSnapshot?.documentId === failedSnapshot.documentId
        ? pendingSnapshot
        : failedSnapshot;
  };

  const queueLatestRevision = (
    savedSnapshot: DocumentSyncSnapshot<TDocument>,
  ): boolean => {
    const latestSnapshot = options.getLatestSnapshot();
    if (
      !latestSnapshot ||
      latestSnapshot.documentId !== savedSnapshot.documentId ||
      latestSnapshot.revision === savedSnapshot.revision
    ) {
      return false;
    }

    pendingSnapshot = latestSnapshot;
    return true;
  };

  /** True while the editor still shows the document this snapshot came from. */
  const stillCurrent = (snapshot: DocumentSyncSnapshot<TDocument>): boolean => {
    const latestSnapshot = options.getLatestSnapshot();
    return !latestSnapshot || latestSnapshot.documentId === snapshot.documentId;
  };

  const run = async (): Promise<void> => {
    running = true;
    // Releasing the run and reading `pendingSnapshot` has to happen without an
    // await in between, otherwise a save enqueued in that gap would be left
    // behind by a run that is already finishing.
    try {
      while (pendingSnapshot) {
        const snapshot = pendingSnapshot;
        pendingSnapshot = null;
        activeSnapshot = snapshot;
        options.onSaving?.(snapshot);

        try {
          await options.persist(snapshot);
        } catch (error) {
          activeSnapshot = null;
          rememberUnsaved(snapshot);
          if (queueLatestRevision(snapshot)) {
            continue;
          }
          if (stillCurrent(snapshot)) options.onError?.(error, snapshot);
          // A revision handed over explicitly — an unmount, a route change —
          // still has to be stored, even though the editor shows another
          // document by now. Throwing left it sitting in the queue, where the
          // next enqueue overwrote it with an older revision of the same
          // document.
          if (pendingSnapshot) continue;
          throw error;
        }

        activeSnapshot = null;
        persistedKey = snapshotKey(snapshot);
        storedSnapshot = snapshot;
        storedAt = Date.now();
        unsavedSnapshot = null;
        if (queueLatestRevision(snapshot)) {
          continue;
        }

        if (stillCurrent(snapshot)) options.onSynced?.(snapshot);
      }
    } finally {
      running = false;
      activeRun = null;
    }
  };

  const enqueue = (
    snapshot: DocumentSyncSnapshot<TDocument>,
    { force = false }: EnqueueOptions = {},
  ): Promise<void> => {
    const key = snapshotKey(snapshot);
    const isIdle = !running && !pendingSnapshot;

    if (persistedKey === key && isIdle && !force) {
      // What the editor shows is what the server holds — an older draft kept
      // from a failed save is text the user has since undone.
      unsavedSnapshot = null;
      options.onSynced?.(snapshot);
      return Promise.resolve();
    }

    const alreadyInFlight =
      activeSnapshot !== null && snapshotKey(activeSnapshot) === key;
    if (!alreadyInFlight) {
      pendingSnapshot = snapshot;
    }

    if (!running) {
      activeRun = run();
    }

    return activeRun ?? Promise.resolve();
  };

  return {
    enqueue,
    markPersisted: (snapshot) => {
      persistedKey = snapshot ? snapshotKey(snapshot) : null;
      // A revision the server is known to hold is not unsaved work any more.
      // `markPersisted(null)` says the opposite — keep the draft then.
      if (
        snapshot &&
        unsavedSnapshot &&
        snapshotKey(unsavedSnapshot) === persistedKey
      ) {
        unsavedSnapshot = null;
      }
    },
    isSaving: () => running,
    unsavedSnapshot: () => unsavedSnapshot,
    queuedSnapshot: () => pendingSnapshot ?? activeSnapshot,
    // A payload asked for in the same millisecond as the write landed is
    // treated as the older one: the stored revision is then either the same
    // document or newer, so preferring it never loses anything.
    storedRevisionNewerThan: (requestedAt) =>
      requestedAt !== undefined && storedAt !== null && requestedAt <= storedAt
        ? storedSnapshot
        : null,
  };
}
