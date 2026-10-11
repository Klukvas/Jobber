import { beforeEach, describe, expect, it, vi } from "vitest";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { DocumentSwitchedError } from "@/shared/lib/documentSync";
import { getQueryClient } from "@/shared/lib/queryClient";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";
import {
  enqueueCoverLetterSave,
  enqueueOutgoingCoverLetterSave,
  flushCoverLetterDocument,
  forgetCoverLetterBaseline,
  markCoverLetterLoaded,
  reconcileLoadedCoverLetter,
  resetCoverLetterDocumentSync,
} from "./coverLetterDocumentSync";
import { coverLetterQueryKeys } from "./coverLetterQueryKeys";

/** The key the letters list is cached under, as `CoverLetters.tsx` spells it. */
const COVER_LETTER_LIST_KEY = ["cover-letters"];

const { mockUpdate } = vi.hoisted(() => ({ mockUpdate: vi.fn() }));

vi.mock("@/services/coverLetterService", () => ({
  coverLetterService: { update: mockUpdate },
}));

function makeLetter(overrides: Partial<CoverLetterDTO> = {}): CoverLetterDTO {
  return {
    id: "cl-1",
    resume_builder_id: null,
    job_id: null,
    title: "My Cover Letter",
    template: "professional",
    recipient_name: "Jane Smith",
    recipient_title: "Hiring Manager",
    company_name: "Acme Corp",
    company_address: "123 Main St",
    greeting: "Dear Jane Smith,",
    paragraphs: ["First paragraph"],
    closing: "Sincerely,",
    font_family: "Georgia",
    font_size: 12,
    primary_color: "#2563eb",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>((resolvePromise) => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
}

/** Lets every pending promise reaction run before asserting. */
function settleMicrotasks(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

function loadLetter(letter: CoverLetterDTO) {
  useCoverLetterStore.setState({
    coverLetter: letter,
    isDirty: false,
    saveStatus: "idle",
  });
  markCoverLetterLoaded(letter);
}

beforeEach(() => {
  vi.clearAllMocks();
  mockUpdate.mockResolvedValue({});
  resetCoverLetterDocumentSync();
  useCoverLetterStore.setState({
    coverLetter: null,
    isDirty: false,
    saveStatus: "idle",
  });
});

// Resolving a flush for a letter the editor has left reported success, and the
// export then rendered a letter whose own save had never been awaited.
describe("cover letter document sync — flush after a letter switch", () => {
  it("refuses to flush a letter the editor has left", async () => {
    loadLetter(makeLetter());
    useCoverLetterStore.getState().updateField("title", "Edited");
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });

    await expect(flushCoverLetterDocument("cl-1")).rejects.toBeInstanceOf(
      DocumentSwitchedError,
    );
    expect(mockUpdate).not.toHaveBeenCalled();
  });

  it("still flushes the letter the editor shows", async () => {
    loadLetter(makeLetter());
    useCoverLetterStore.getState().updateField("title", "Flush me");

    await flushCoverLetterDocument("cl-1");

    expect(mockUpdate).toHaveBeenCalledWith(
      "cl-1",
      expect.objectContaining({ title: "Flush me" }),
    );
  });
});

// The fresh payload used to replace unsaved text without a word — on a refetch
// and on coming back from another letter.
describe("cover letter document sync — unsaved drafts", () => {
  it("hands a failed edit back when the editor returns to the letter", async () => {
    mockUpdate.mockRejectedValue(new Error("offline"));
    const stored = makeLetter({ title: "Stored title" });
    loadLetter(stored);
    useCoverLetterStore
      .getState()
      .updateField("title", "Typed but never stored");

    await expect(enqueueCoverLetterSave()).rejects.toThrow("offline");

    // SPA navigation to another letter, then back to this one.
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });
    const loaded = reconcileLoadedCoverLetter(stored);

    expect(loaded.coverLetter.title).toBe("Typed but never stored");
    expect(loaded.unsaved).toBe("failed");
  });

  it("stores the recovered draft when the save is retried", async () => {
    mockUpdate.mockRejectedValueOnce(new Error("offline"));
    const stored = makeLetter({ title: "Stored title" });
    loadLetter(stored);
    useCoverLetterStore.getState().updateField("title", "Recovered");
    await expect(enqueueCoverLetterSave()).rejects.toThrow("offline");
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });

    const loaded = reconcileLoadedCoverLetter(stored);
    expect(loaded.coverLetter.title).toBe("Recovered");

    // The editor reopens the letter the way the page does: dirty, with nothing
    // assumed about what the server already holds.
    forgetCoverLetterBaseline("cl-1");
    useCoverLetterStore.setState({
      coverLetter: loaded.coverLetter,
      isDirty: true,
    });
    await enqueueCoverLetterSave();

    const sentTitles = mockUpdate.mock.calls.map(
      (call) => (call[1] as { title: string }).title,
    );
    expect(sentTitles).toEqual(["Recovered", "Recovered"]);
    expect(reconcileLoadedCoverLetter(stored).unsaved).toBe("none");
  });

  it("prefers edits typed during the fetch over the payload", () => {
    loadLetter(makeLetter({ title: "Stored title" }));
    useCoverLetterStore.getState().updateField("title", "Still typing");

    const loaded = reconcileLoadedCoverLetter(
      makeLetter({ title: "Stored title" }),
    );

    expect(loaded.coverLetter.title).toBe("Still typing");
    expect(loaded.unsaved).toBe("pending");
  });

  it("takes the payload for a letter with nothing unsaved", () => {
    loadLetter(makeLetter({ title: "Stored title" }));

    const loaded = reconcileLoadedCoverLetter(
      makeLetter({ title: "Changed elsewhere" }),
    );

    expect(loaded.coverLetter.title).toBe("Changed elsewhere");
    expect(loaded.unsaved).toBe("none");
  });
});

// The editor's id changes without the page unmounting — React Router keeps the
// same element for `/cover-letters/:id` — so the unmount flush never runs and
// the debounce timer dies with the store update.
describe("cover letter document sync — the letter on its way out", () => {
  it("stores what was typed before the route change replaces it", async () => {
    loadLetter(makeLetter({ title: "Stored title" }));
    useCoverLetterStore.getState().updateField("title", "Typed on exit");

    enqueueOutgoingCoverLetterSave("cl-2");
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });
    await settleMicrotasks();

    expect(mockUpdate).toHaveBeenCalledWith(
      "cl-1",
      expect.objectContaining({ title: "Typed on exit" }),
    );
  });

  it("leaves a letter with nothing unsaved alone", async () => {
    loadLetter(makeLetter({ title: "Stored title" }));

    enqueueOutgoingCoverLetterSave("cl-2");
    await settleMicrotasks();

    expect(mockUpdate).not.toHaveBeenCalled();
  });

  // Returning before the request came back used to show the stored payload:
  // nothing had failed yet, so the editor had no draft to fall back on.
  it("offers back a revision whose save is still on its way", async () => {
    const inFlight = deferred();
    mockUpdate.mockImplementation(async () => {
      await inFlight.promise;
      return {};
    });
    const stored = makeLetter({ title: "Stored title" });
    loadLetter(stored);
    useCoverLetterStore.getState().updateField("title", "Typed on exit");

    enqueueOutgoingCoverLetterSave("cl-2");
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });

    const loaded = reconcileLoadedCoverLetter(stored);
    expect(loaded.coverLetter.title).toBe("Typed on exit");
    expect(loaded.unsaved).toBe("pending");

    inFlight.resolve();
    await settleMicrotasks();

    expect(reconcileLoadedCoverLetter(stored).unsaved).toBe("none");
  });

  it("keeps the newest revision when the save that overlapped it fails", async () => {
    const inFlight = deferred();
    mockUpdate.mockImplementation(async () => {
      await inFlight.promise;
      throw new Error("offline");
    });
    const stored = makeLetter({ title: "Stored title" });
    loadLetter(stored);
    useCoverLetterStore.getState().updateField("title", "Edited once");
    const saving = enqueueCoverLetterSave();

    useCoverLetterStore.getState().updateField("title", "Edited twice");
    enqueueOutgoingCoverLetterSave("cl-2");
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });

    inFlight.resolve();
    await expect(saving).rejects.toThrow("offline");

    const loaded = reconcileLoadedCoverLetter(stored);
    expect(loaded.coverLetter.title).toBe("Edited twice");
    expect(loaded.unsaved).toBe("failed");
  });
});

// Renaming happens inside the editor, so the letters list kept showing the old
// name for every save the editor did not live to see through.
describe("cover letter document sync — the list behind the editor", () => {
  const listState = () => getQueryClient().getQueryState(COVER_LETTER_LIST_KEY);

  beforeEach(() => {
    getQueryClient().clear();
  });

  it("marks the list stale for a letter the editor has already left", async () => {
    getQueryClient().setQueryData(COVER_LETTER_LIST_KEY, []);
    loadLetter(makeLetter({ title: "Stored title" }));
    useCoverLetterStore.getState().updateField("title", "Renamed on exit");

    enqueueOutgoingCoverLetterSave("cl-2");
    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });
    await settleMicrotasks();

    expect(listState()?.isInvalidated).toBe(true);
  });

  it("leaves the list alone when the rename could not be stored", async () => {
    mockUpdate.mockRejectedValue(new Error("offline"));
    getQueryClient().setQueryData(COVER_LETTER_LIST_KEY, []);
    loadLetter(makeLetter({ title: "Stored title" }));
    useCoverLetterStore.getState().updateField("title", "Renamed here");

    await expect(enqueueCoverLetterSave()).rejects.toThrow("offline");

    expect(listState()?.isInvalidated).toBe(false);
  });
});

/**
 * The editor opens whatever the letter's cache entry holds, and `staleTime`
 * kept a payload fresh for five minutes while a save never touched it. Leaving
 * the editor and coming back inside that window re-opened the letter as it had
 * been *before* the save — and the next edit saved that revision back over the
 * server, undoing work the server had already accepted.
 */
describe("cover letter document sync — a cached letter older than the save", () => {
  const detailKey = coverLetterQueryKeys.detail("cl-1");
  const detailState = () => getQueryClient().getQueryState(detailKey);

  beforeEach(() => {
    getQueryClient().clear();
  });

  /** The letter as the editor fetched it, before anything was typed. */
  function openFromCache(letter: CoverLetterDTO) {
    getQueryClient().setQueryData(detailKey, letter, { updatedAt: 1_000 });
    loadLetter(letter);
  }

  it("keeps the saved revision when the cache serves the pre-save payload", async () => {
    const fetched = makeLetter({ paragraphs: ["First", "Original second"] });
    openFromCache(fetched);
    useCoverLetterStore.getState().updateParagraph(1, "Saved second");

    await enqueueCoverLetterSave();

    // Back to the list and in again: same page, same cache entry, no refetch.
    const loaded = reconcileLoadedCoverLetter(fetched);

    expect(loaded.coverLetter.paragraphs).toEqual(["First", "Saved second"]);
    expect(loaded.unsaved).toBe("none");
  });

  // Opening another letter and coming back changes the route parameter without
  // unmounting the page, so the store holds the other letter when the cached
  // payload arrives — the queue's own record of what it stored is all there is.
  it("opens the revision it stored when another letter is on screen", async () => {
    const fetched = makeLetter({ paragraphs: ["First", "Original second"] });
    openFromCache(fetched);
    useCoverLetterStore.getState().updateParagraph(1, "Saved second");
    await enqueueCoverLetterSave();

    useCoverLetterStore.setState({ coverLetter: makeLetter({ id: "cl-2" }) });
    const loaded = reconcileLoadedCoverLetter(fetched);

    expect(loaded.coverLetter.paragraphs).toEqual(["First", "Saved second"]);
    expect(loaded.unsaved).toBe("none");
  });

  it("marks the letter stale after a save without refetching it", async () => {
    const fetched = makeLetter({ title: "Stored title" });
    openFromCache(fetched);
    useCoverLetterStore.getState().updateField("title", "Renamed here");

    await enqueueCoverLetterSave();

    // Stale, so reopening the letter fetches it again. That a mounted editor
    // is *not* refetched over is proved where there is one to observe it:
    // `pages/__tests__/EditorReopenFreshness.test.tsx`.
    expect(detailState()?.isInvalidated).toBe(true);
    expect(detailState()?.fetchStatus).toBe("idle");
  });

  it("opens a payload the server sent after the save", async () => {
    const fetched = makeLetter({ title: "Stored title" });
    openFromCache(fetched);
    useCoverLetterStore.getState().updateField("title", "Renamed here");
    await enqueueCoverLetterSave();

    // The refetch that reopening triggers, carrying a rename from elsewhere.
    const refetched = makeLetter({ title: "Renamed on another device" });
    getQueryClient().setQueryData(detailKey, refetched, {
      updatedAt: Date.now() + 1_000,
    });

    expect(reconcileLoadedCoverLetter(refetched).coverLetter.title).toBe(
      "Renamed on another device",
    );
  });

  it("still prefers unsaved text over its own stored revision", async () => {
    const fetched = makeLetter({ title: "Stored title" });
    openFromCache(fetched);
    useCoverLetterStore.getState().updateField("title", "Saved rename");
    await enqueueCoverLetterSave();
    useCoverLetterStore.getState().updateField("title", "Typed after the save");

    const loaded = reconcileLoadedCoverLetter(fetched);

    expect(loaded.coverLetter.title).toBe("Typed after the save");
    expect(loaded.unsaved).toBe("pending");
  });

  it("opens the payload for a letter this client never wrote to", () => {
    const fetched = makeLetter({ title: "Stored title" });
    openFromCache(fetched);

    const loaded = reconcileLoadedCoverLetter(
      makeLetter({ title: "Changed elsewhere" }),
    );

    expect(loaded.coverLetter.title).toBe("Changed elsewhere");
  });
});
