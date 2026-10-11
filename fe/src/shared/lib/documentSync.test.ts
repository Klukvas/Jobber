import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearDocumentMutationState,
  createDocumentFetchTracker,
  createDocumentSyncQueue,
  hasDocumentMutationFailures,
  retryDocumentMutations,
  trackDocumentMutation,
  trackPersistedTitles,
  type DocumentSyncSnapshot,
  waitForDocumentMutations,
} from "./documentSync";

interface TestDocument {
  readonly value: string;
}

function snapshot(value: string): DocumentSyncSnapshot<TestDocument> {
  return {
    documentId: "document-1",
    revision: value,
    document: { value },
  };
}

function deferred() {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

const otherDocument: DocumentSyncSnapshot<TestDocument> = {
  documentId: "document-2",
  revision: "B",
  document: { value: "B" },
};

/** Lets every pending promise reaction run before asserting. */
function settleMicrotasks(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

describe("createDocumentSyncQueue", () => {
  it("persists an edit made while an older revision is in flight", async () => {
    const firstSave = deferred();
    const persisted: string[] = [];
    let latest = snapshot("A");
    const persist = vi.fn(
      async (current: DocumentSyncSnapshot<TestDocument>) => {
        if (current.revision === "A") await firstSave.promise;
        persisted.push(current.document.value);
      },
    );
    const onSynced = vi.fn();
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
      onSynced,
    });

    const completion = queue.enqueue(latest);
    latest = snapshot("B");
    firstSave.resolve();
    await completion;

    expect(persisted).toEqual(["A", "B"]);
    expect(onSynced).toHaveBeenCalledTimes(1);
    expect(onSynced).toHaveBeenCalledWith(latest);
  });

  it("retries the latest snapshot after a failed save", async () => {
    const latest = snapshot("latest");
    const persisted: string[] = [];
    const persist = vi
      .fn<(current: DocumentSyncSnapshot<TestDocument>) => Promise<void>>()
      .mockRejectedValueOnce(new Error("offline"))
      .mockImplementationOnce(async (current) => {
        persisted.push(current.document.value);
      });
    const onError = vi.fn();
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
      onError,
    });

    await expect(queue.enqueue(latest)).rejects.toThrow("offline");
    await queue.enqueue(latest);

    expect(persist).toHaveBeenCalledTimes(2);
    expect(persisted).toEqual(["latest"]);
    expect(onError).toHaveBeenCalledTimes(1);
  });

  it("does not treat a different document as the next revision", async () => {
    const save = deferred();
    let latest = snapshot("A");
    const persist = vi.fn(async () => save.promise);
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    const completion = queue.enqueue(latest);
    latest = {
      documentId: "document-2",
      revision: "B",
      document: { value: "B" },
    };
    save.resolve();
    await completion;

    expect(persist).toHaveBeenCalledTimes(1);
  });

  it("waits for an in-flight revision before persisting a revert", async () => {
    const middleSave = deferred();
    const persisted: string[] = [];
    let latest = snapshot("A");
    const persist = vi.fn(
      async (current: DocumentSyncSnapshot<TestDocument>) => {
        if (current.revision === "B") await middleSave.promise;
        persisted.push(current.document.value);
      },
    );
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    await queue.enqueue(latest);
    latest = snapshot("B");
    const middleCompletion = queue.enqueue(latest);
    latest = snapshot("A");
    let revertCompleted = false;
    const revertCompletion = queue.enqueue(latest).then(() => {
      revertCompleted = true;
    });

    await Promise.resolve();
    expect(revertCompleted).toBe(false);
    middleSave.resolve();
    await Promise.all([middleCompletion, revertCompletion]);

    expect(persisted).toEqual(["A", "B", "A"]);
  });

  // Reporting the save as synchronized would mark the *new* document clean,
  // hiding unsaved work the user can still see on screen.
  it("does not report a save as synced once another document is open", async () => {
    const save = deferred();
    let latest: DocumentSyncSnapshot<TestDocument> = snapshot("A");
    const onSynced = vi.fn();
    const queue = createDocumentSyncQueue({
      persist: () => save.promise,
      getLatestSnapshot: () => latest,
      onSynced,
    });

    const completion = queue.enqueue(latest);
    latest = {
      documentId: "document-2",
      revision: "B",
      document: { value: "B" },
    };
    save.resolve();
    await completion;

    expect(onSynced).not.toHaveBeenCalled();
  });

  it("skips a revision the server already has, unless forced", async () => {
    const latest = snapshot("A");
    const persist = vi.fn(async () => {});
    const onSynced = vi.fn();
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
      onSynced,
    });

    await queue.enqueue(latest);
    await queue.enqueue(latest);
    expect(persist).toHaveBeenCalledTimes(1);
    expect(onSynced).toHaveBeenCalledTimes(2);

    await queue.enqueue(latest, { force: true });
    expect(persist).toHaveBeenCalledTimes(2);
  });

  // A failed save used to leave nothing behind, so the editor had nothing to
  // offer back once the page reloaded the stored document over it.
  it("keeps the newest unsaved snapshot after a failed save", async () => {
    let latest = snapshot("A");
    const persist = vi
      .fn<(current: DocumentSyncSnapshot<TestDocument>) => Promise<void>>()
      .mockRejectedValue(new Error("offline"));
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    const saving = queue.enqueue(latest);
    // The user keeps typing while the save is failing.
    latest = snapshot("B");
    await expect(saving).rejects.toThrow("offline");

    expect(queue.unsavedSnapshot()?.document).toEqual({ value: "B" });
  });

  it("forgets the unsaved snapshot once the document is stored", async () => {
    const latest = snapshot("A");
    const persist = vi
      .fn<(current: DocumentSyncSnapshot<TestDocument>) => Promise<void>>()
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(undefined);
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    await expect(queue.enqueue(latest)).rejects.toThrow("offline");
    expect(queue.unsavedSnapshot()).not.toBeNull();

    await queue.enqueue(latest);

    expect(queue.unsavedSnapshot()).toBeNull();
  });

  // Undoing back to the stored revision leaves nothing unsaved; offering the
  // old draft back later would resurrect text the user removed on purpose.
  it("drops the draft when the editor matches what the server holds", async () => {
    let latest = snapshot("A");
    const persist = vi
      .fn<(current: DocumentSyncSnapshot<TestDocument>) => Promise<void>>()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error("offline"));
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    await queue.enqueue(latest);
    latest = snapshot("B");
    await expect(queue.enqueue(latest)).rejects.toThrow("offline");
    expect(queue.unsavedSnapshot()?.document).toEqual({ value: "B" });

    latest = snapshot("A");
    await queue.enqueue(latest);

    expect(queue.unsavedSnapshot()).toBeNull();
  });

  // An unmount or a route change hands the newest revision over explicitly.
  // Reporting the in-flight failure used to end the run with that revision
  // still queued, so it was never sent — and the next enqueue overwrote it.
  it("stores a revision queued before the editor moved to another document", async () => {
    const firstSave = deferred();
    const persisted: string[] = [];
    let latest = snapshot("A1");
    const persist = vi.fn(
      async (current: DocumentSyncSnapshot<TestDocument>) => {
        if (current.revision === "A1") {
          await firstSave.promise;
          throw new Error("offline");
        }
        persisted.push(current.revision);
      },
    );
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    const saving = queue.enqueue(latest);
    latest = snapshot("A2");
    const queued = queue.enqueue(latest);
    latest = otherDocument;
    firstSave.resolve();
    await Promise.all([saving, queued]);

    expect(persisted).toEqual(["A2"]);
    expect(queue.unsavedSnapshot()).toBeNull();
  });

  // Keeping the payload that happened to fail handed the editor back a draft
  // the user had already typed past.
  it("remembers the queued revision rather than the one that failed", async () => {
    const firstSave = deferred();
    const secondSave = deferred();
    let latest = snapshot("A1");
    const persist = vi.fn(
      async (current: DocumentSyncSnapshot<TestDocument>) => {
        if (current.revision === "A1") {
          await firstSave.promise;
          throw new Error("offline");
        }
        await secondSave.promise;
        throw new Error("still offline");
      },
    );
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    const saving = queue.enqueue(latest);
    latest = snapshot("A2");
    const queued = queue.enqueue(latest);
    latest = otherDocument;
    firstSave.resolve();
    await settleMicrotasks();

    expect(queue.unsavedSnapshot()?.revision).toBe("A2");

    secondSave.resolve();
    await expect(Promise.all([saving, queued])).rejects.toThrow(
      "still offline",
    );
    expect(queue.unsavedSnapshot()?.revision).toBe("A2");
  });

  // Coming back to a document whose save is still running needs the revision
  // that is on its way: it has not failed, so there is no unsaved snapshot.
  it("exposes the revision whose save has not landed yet", async () => {
    const save = deferred();
    const latest = snapshot("A");
    const queue = createDocumentSyncQueue({
      persist: () => save.promise,
      getLatestSnapshot: () => latest,
    });

    const saving = queue.enqueue(latest);

    expect(queue.queuedSnapshot()?.revision).toBe("A");

    save.resolve();
    await saving;

    expect(queue.queuedSnapshot()).toBeNull();
  });

  it("treats a document loaded from the server as already stored", async () => {
    const latest = snapshot("A");
    const persist = vi.fn(async () => {});
    const queue = createDocumentSyncQueue({
      persist,
      getLatestSnapshot: () => latest,
    });

    queue.markPersisted(latest);
    await queue.enqueue(latest);
    expect(persist).not.toHaveBeenCalled();

    queue.markPersisted(null);
    await queue.enqueue(latest);
    expect(persist).toHaveBeenCalledTimes(1);
  });

  // A cached payload kept being served for minutes after a save, so reopening
  // the document handed the editor the revision from before it.
  describe("dating a cached payload against the last stored write", () => {
    const storedQueue = () => {
      const latest = snapshot("A");
      const queue = createDocumentSyncQueue({
        persist: async () => {},
        getLatestSnapshot: () => latest,
      });
      return { queue, latest };
    };

    it("knows nothing before a write of its own has landed", () => {
      const { queue } = storedQueue();

      expect(queue.storedRevisionNewerThan(Date.now() - 10_000)).toBeNull();
    });

    it("offers its stored revision for a payload fetched before the write", async () => {
      const { queue, latest } = storedQueue();
      const beforeSave = Date.now() - 1;

      await queue.enqueue(latest);

      expect(queue.storedRevisionNewerThan(beforeSave)).toEqual(latest);
    });

    // The queue keeps the revision it actually sent, so coming back to a
    // document the editor has since left still opens that one.
    it("keeps offering the newest revision it stored", async () => {
      const first = snapshot("A");
      const second = snapshot("B");
      let current = first;
      const queue = createDocumentSyncQueue({
        persist: async () => {},
        getLatestSnapshot: () => current,
      });
      const beforeSaves = Date.now() - 1;

      await queue.enqueue(first);
      current = second;
      await queue.enqueue(second);

      expect(queue.storedRevisionNewerThan(beforeSaves)).toEqual(second);
    });

    it("lets a payload fetched after the write stand", async () => {
      const { queue, latest } = storedQueue();

      await queue.enqueue(latest);

      expect(queue.storedRevisionNewerThan(Date.now() + 1_000)).toBeNull();
    });

    it("assumes nothing when the payload has no fetch time", async () => {
      const { queue, latest } = storedQueue();

      await queue.enqueue(latest);

      expect(queue.storedRevisionNewerThan(undefined)).toBeNull();
    });

    // Only a write of this client's own dates a payload. A document handed
    // over as "already stored" says what the server holds, not when this
    // client put it there.
    it("does not count a loaded document as a write", () => {
      const { queue, latest } = storedQueue();

      queue.markPersisted(latest);

      expect(queue.storedRevisionNewerThan(Date.now() - 10_000)).toBeNull();
    });

    it("stays quiet while the only save has failed", async () => {
      const latest = snapshot("A");
      const queue = createDocumentSyncQueue({
        persist: async () => {
          throw new Error("offline");
        },
        getLatestSnapshot: () => latest,
      });
      const beforeSave = Date.now() - 1;

      await expect(queue.enqueue(latest)).rejects.toThrow("offline");

      expect(queue.storedRevisionNewerThan(beforeSave)).toBeNull();
    });
  });
});

describe("trackPersistedTitles", () => {
  it("reports a stored title the server did not have before", () => {
    const onTitleChanged = vi.fn();
    const titles = trackPersistedTitles(onTitleChanged);

    titles.setKnown("document-1", "Loaded title");
    titles.recordPersisted("document-1", "Renamed");

    expect(onTitleChanged).toHaveBeenCalledWith("document-1");
  });

  // Every auto-save stores the title along with the rest of the document, so
  // reporting each one would refresh the lists for nothing.
  it("stays quiet when the stored title is the one it already had", () => {
    const onTitleChanged = vi.fn();
    const titles = trackPersistedTitles(onTitleChanged);

    titles.setKnown("document-1", "Loaded title");
    titles.recordPersisted("document-1", "Loaded title");

    expect(onTitleChanged).not.toHaveBeenCalled();
  });

  // A document opened on top of unsaved work has no known stored title, so the
  // first save that lands has to be treated as a change.
  it("reports the first stored title of a document it knows nothing about", () => {
    const onTitleChanged = vi.fn();
    const titles = trackPersistedTitles(onTitleChanged);

    titles.setKnown("document-1", "Loaded title");
    titles.forget("document-1");
    titles.recordPersisted("document-1", "Loaded title");

    expect(onTitleChanged).toHaveBeenCalledTimes(1);
  });
});

describe("document mutation barrier", () => {
  it("keeps a settled failure until the same mutation succeeds", async () => {
    const failedMutation = Promise.reject(new Error("create failed"));
    await expect(
      trackDocumentMutation("document-failure", failedMutation, "section:item"),
    ).rejects.toThrow("create failed");

    await expect(waitForDocumentMutations("document-failure")).rejects.toThrow(
      "create failed",
    );

    await trackDocumentMutation(
      "document-failure",
      Promise.resolve(),
      "section:item",
    );
    await expect(
      waitForDocumentMutations("document-failure"),
    ).resolves.toBeUndefined();
  });

  it("retries a failed write with the registered retry", async () => {
    const retry = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error("still offline"))
      .mockResolvedValueOnce(undefined);
    await expect(
      trackDocumentMutation(
        "document-retry",
        Promise.reject(new Error("offline")),
        "section:item",
        retry,
      ),
    ).rejects.toThrow("offline");

    await expect(retryDocumentMutations("document-retry")).rejects.toThrow(
      "still offline",
    );
    expect(hasDocumentMutationFailures("document-retry")).toBe(true);

    await retryDocumentMutations("document-retry");

    expect(retry).toHaveBeenCalledTimes(2);
    expect(hasDocumentMutationFailures("document-retry")).toBe(false);
  });

  // Leaving on the first rejection let a document save start while another
  // section write was still running, so the two overlapped on the same rows.
  it("waits for every tracked write before reporting a failure", async () => {
    const slowWrite = deferred();
    const failingWrite = deferred();
    void trackDocumentMutation(
      "document-fence",
      slowWrite.promise,
      "section:slow",
    );
    void trackDocumentMutation(
      "document-fence",
      failingWrite.promise,
      "section:failing",
    ).catch(() => {});

    const waiting = waitForDocumentMutations("document-fence");
    let barrierSettled = false;
    void waiting.then(
      () => (barrierSettled = true),
      () => (barrierSettled = true),
    );

    failingWrite.reject(new Error("boom"));
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(barrierSettled).toBe(false);

    slowWrite.resolve();
    await expect(waiting).rejects.toThrow("boom");
    expect(barrierSettled).toBe(true);
  });

  // The row the failed create belonged to is gone, so retrying it would bring
  // deleted content back.
  it("forgets a single failure whose item no longer exists", async () => {
    await expect(
      trackDocumentMutation(
        "document-dropped",
        Promise.reject(new Error("create failed")),
        "section:dropped",
      ),
    ).rejects.toThrow("create failed");

    clearDocumentMutationState("document-dropped", "section:dropped");

    expect(hasDocumentMutationFailures("document-dropped")).toBe(false);
    await expect(
      waitForDocumentMutations("document-dropped"),
    ).resolves.toBeUndefined();
  });
});

/**
 * A response the server composed before a save, delivered after it.
 *
 * Dating a payload by its arrival said that response was the newer revision,
 * so the editor adopted it and the next edit sent it back — the save was
 * undone. What the request left with is the only honest date for it.
 */
describe("createDocumentFetchTracker", () => {
  let now = 1_000;

  beforeEach(() => {
    now = 1_000;
    vi.spyOn(Date, "now").mockImplementation(() => now);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  /** A request that takes `duration` to come back with `payload`. */
  const answersIn =
    <TPayload>(payload: TPayload, duration: number) =>
    async () => {
      now += duration;
      return payload;
    };

  it("hands the payload straight back", async () => {
    const tracker = createDocumentFetchTracker();

    await expect(
      tracker.track("document-1", answersIn({ value: "A" }, 500)),
    ).resolves.toEqual({ value: "A" });
  });

  it("dates a delivered payload by the moment its request left", async () => {
    const tracker = createDocumentFetchTracker();

    await tracker.track("document-1", answersIn({ value: "A" }, 500));

    // 1_500 is when the cache took the response in — half a second after the
    // server was asked, and after any write made in between.
    expect(tracker.requestStartOf("document-1", 1_500)).toBe(1_000);
  });

  it("knows nothing about a document it never fetched", () => {
    const tracker = createDocumentFetchTracker();

    expect(tracker.requestStartOf("document-1", 1_500)).toBeUndefined();
  });

  it("assumes nothing when the cache entry has no date", async () => {
    const tracker = createDocumentFetchTracker();

    await tracker.track("document-1", answersIn({ value: "A" }, 500));

    expect(tracker.requestStartOf("document-1", undefined)).toBeUndefined();
  });

  it("keeps no date for a request that failed", async () => {
    const tracker = createDocumentFetchTracker();

    await expect(
      tracker.track("document-1", async () => {
        now += 500;
        throw new Error("offline");
      }),
    ).rejects.toThrow("offline");

    expect(tracker.requestStartOf("document-1", 1_500)).toBeUndefined();
  });

  // The cache entry is older than the delivery on record, so that delivery is
  // not what the cache is holding — a response React Query discarded, or data
  // put there by hand. Dating the payload by it would be a guess.
  it("stays quiet when its delivery is newer than the cache entry", async () => {
    const tracker = createDocumentFetchTracker();

    await tracker.track("document-1", answersIn({ value: "A" }, 500));

    expect(tracker.requestStartOf("document-1", 1_200)).toBeUndefined();
  });

  // Two requests in flight: the cache keeps whichever answers last, so that is
  // the one the date has to describe — even though it left first.
  it("follows the last delivery, not the latest request", async () => {
    const tracker = createDocumentFetchTracker();
    let releaseFirst!: (payload: { value: string }) => void;
    const first = tracker.track("document-1", () => {
      return new Promise<{ value: string }>((resolve) => {
        releaseFirst = resolve;
      });
    });

    now = 1_100;
    await tracker.track("document-1", answersIn({ value: "B" }, 100));
    expect(tracker.requestStartOf("document-1", 1_200)).toBe(1_100);

    now = 1_300;
    releaseFirst({ value: "A" });
    await first;

    expect(tracker.requestStartOf("document-1", 1_300)).toBe(1_000);
  });

  it("forgets every delivery when cleared", async () => {
    const tracker = createDocumentFetchTracker();

    await tracker.track("document-1", answersIn({ value: "A" }, 500));
    tracker.clear();

    expect(tracker.requestStartOf("document-1", 1_500)).toBeUndefined();
  });
});
