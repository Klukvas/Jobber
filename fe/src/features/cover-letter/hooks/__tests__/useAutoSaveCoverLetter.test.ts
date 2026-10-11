import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

const { mockUpdate, mockInvalidateQueries, mockStore } = vi.hoisted(() => {
  const mockUpdate = vi.fn();
  const mockInvalidateQueries = vi.fn();

  const mockStore = {
    coverLetter: null as CoverLetterDTO | null,
    isDirty: false,
    setSaveStatus: vi.fn(),
    markClean: vi.fn(),
  };

  return { mockUpdate, mockInvalidateQueries, mockStore };
});

vi.mock("@/services/coverLetterService", () => ({
  coverLetterService: { update: mockUpdate },
}));

// The sync module reaches the app's shared cache to refresh the letters list
// after a rename, so the stand-in has to offer a client to construct as well.
vi.mock("@tanstack/react-query", () => ({
  QueryClient: class {
    invalidateQueries = mockInvalidateQueries;
  },
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}));

vi.mock("@/stores/coverLetterStore", () => ({
  useCoverLetterStore: Object.assign(
    (selector: (s: typeof mockStore) => unknown) => selector(mockStore),
    { getState: () => mockStore },
  ),
}));

import { useAutoSaveCoverLetter } from "../useAutoSaveCoverLetter";

function createMockCoverLetter(
  overrides?: Partial<CoverLetterDTO>,
): CoverLetterDTO {
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

describe("useAutoSaveCoverLetter", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    mockStore.coverLetter = null;
    mockStore.isDirty = false;
    mockStore.setSaveStatus = vi.fn();
    mockStore.markClean = vi.fn();
    mockUpdate.mockResolvedValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not save when isDirty is false", () => {
    mockStore.coverLetter = createMockCoverLetter();
    mockStore.isDirty = false;

    renderHook(() => useAutoSaveCoverLetter());

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(mockUpdate).not.toHaveBeenCalled();
  });

  it("does not save when coverLetter is null", () => {
    mockStore.coverLetter = null;
    mockStore.isDirty = true;

    renderHook(() => useAutoSaveCoverLetter());

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(mockUpdate).not.toHaveBeenCalled();
  });

  it("triggers debounced save after 1500ms when dirty", async () => {
    const letter = createMockCoverLetter();
    mockStore.coverLetter = letter;
    mockStore.isDirty = true;

    renderHook(() => useAutoSaveCoverLetter());

    // Should not save before 1500ms
    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(mockUpdate).not.toHaveBeenCalled();

    // Should save after 1500ms
    await act(async () => {
      vi.advanceTimersByTime(500);
    });

    expect(mockUpdate).toHaveBeenCalled();
  });

  it("does not save when JSON is unchanged (dedup)", async () => {
    const letter = createMockCoverLetter();
    mockStore.coverLetter = letter;
    mockStore.isDirty = true;

    const { rerender } = renderHook(() => useAutoSaveCoverLetter());

    // First render sets prevCoverLetterRef, timer fires
    await act(async () => {
      vi.advanceTimersByTime(1500);
    });

    vi.clearAllMocks();

    // Re-render with same coverLetter object (isDirty still true)
    // The JSON is the same so it should be deduped
    rerender();

    act(() => {
      vi.advanceTimersByTime(1500);
    });

    expect(mockUpdate).not.toHaveBeenCalled();
  });

  it("calls coverLetterService.update with correct fields", async () => {
    const letter = createMockCoverLetter({
      id: "cl-42",
      title: "Test Title",
      template: "modern",
      recipient_name: "John",
      recipient_title: "CTO",
      company_name: "TestCo",
      company_address: "456 Oak Ave",
      greeting: "Hi John,",
      paragraphs: ["Para 1", "Para 2"],
      closing: "Regards,",
      font_family: "Arial",
      font_size: 14,
      primary_color: "#ff0000",
      job_id: "job-7",
    });
    mockStore.coverLetter = letter;
    mockStore.isDirty = true;

    renderHook(() => useAutoSaveCoverLetter());

    await act(async () => {
      vi.advanceTimersByTime(1500);
    });

    expect(mockUpdate).toHaveBeenCalledWith("cl-42", {
      title: "Test Title",
      template: "modern",
      recipient_name: "John",
      recipient_title: "CTO",
      company_name: "TestCo",
      company_address: "456 Oak Ave",
      greeting: "Hi John,",
      paragraphs: ["Para 1", "Para 2"],
      closing: "Regards,",
      font_family: "Arial",
      font_size: 14,
      primary_color: "#ff0000",
      job_id: "job-7",
    });
  });

  it("marks clean and sets 'saved' status on success", async () => {
    mockStore.coverLetter = createMockCoverLetter();
    mockStore.isDirty = true;
    mockUpdate.mockResolvedValueOnce({});

    renderHook(() => useAutoSaveCoverLetter());

    await act(async () => {
      vi.advanceTimersByTime(1500);
    });

    expect(mockStore.setSaveStatus).toHaveBeenCalledWith("saving");
    expect(mockStore.markClean).toHaveBeenCalled();
    expect(mockStore.setSaveStatus).toHaveBeenCalledWith("saved");
  });

  it("sets 'error' status on failure", async () => {
    mockStore.coverLetter = createMockCoverLetter();
    mockStore.isDirty = true;
    mockUpdate.mockRejectedValueOnce(new Error("Save failed"));

    renderHook(() => useAutoSaveCoverLetter());

    await act(async () => {
      vi.advanceTimersByTime(1500);
    });

    expect(mockStore.setSaveStatus).toHaveBeenCalledWith("saving");
    expect(mockStore.setSaveStatus).toHaveBeenCalledWith("error");
    expect(mockStore.markClean).not.toHaveBeenCalled();
  });

  it("stores the pending revision when the editor asks for a flush", async () => {
    mockStore.coverLetter = createMockCoverLetter({ title: "Flush me" });
    mockStore.isDirty = true;

    const { result } = renderHook(() => useAutoSaveCoverLetter());

    await act(async () => {
      await result.current.flush();
    });

    expect(mockUpdate).toHaveBeenCalledWith(
      "cl-1",
      expect.objectContaining({ title: "Flush me" }),
    );
  });

  it("flushes pending save on unmount", async () => {
    mockStore.coverLetter = createMockCoverLetter();
    mockStore.isDirty = true;

    const { unmount } = renderHook(() => useAutoSaveCoverLetter());

    // Unmount before debounce fires -- should flush
    await act(async () => {
      unmount();
    });

    // The unmount cleanup checks isDirty via getState() and calls save()
    expect(mockUpdate).toHaveBeenCalled();
  });
});

describe("useAutoSaveCoverLetter — saves that overlap", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    mockStore.coverLetter = null;
    mockStore.isDirty = false;
    mockStore.setSaveStatus = vi.fn();
    mockStore.markClean = vi.fn();
    mockUpdate.mockResolvedValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function deferred() {
    let resolve!: () => void;
    const promise = new Promise<void>((resolvePromise) => {
      resolve = resolvePromise;
    });
    return { promise, resolve };
  }

  // Editing during the request used to be lost: the save came back, marked the
  // letter clean, and the newer paragraph was never sent.
  it("sends an edit made while the save is in flight", async () => {
    const inFlight = deferred();
    const sentTitles: string[] = [];
    mockUpdate.mockImplementation(
      async (_id: string, data: { title: string }) => {
        sentTitles.push(data.title);
        if (sentTitles.length === 1) await inFlight.promise;
        return {};
      },
    );
    mockStore.coverLetter = createMockCoverLetter({ title: "First" });
    mockStore.isDirty = true;

    renderHook(() => useAutoSaveCoverLetter());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await Promise.resolve();
    });

    mockStore.coverLetter = createMockCoverLetter({ title: "Second" });

    await act(async () => {
      inFlight.resolve();
      await vi.runAllTimersAsync();
    });

    expect(sentTitles).toEqual(["First", "Second"]);
    expect(mockStore.markClean).toHaveBeenCalledTimes(1);
  });

  it("does not mark a letter clean when the editor moved to another one", async () => {
    const inFlight = deferred();
    mockUpdate.mockImplementation(async () => {
      await inFlight.promise;
      return {};
    });
    mockStore.coverLetter = createMockCoverLetter({ title: "Letter one" });
    mockStore.isDirty = true;

    renderHook(() => useAutoSaveCoverLetter());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await Promise.resolve();
    });

    mockStore.coverLetter = createMockCoverLetter({
      id: "cl-2",
      title: "Letter two",
    });

    await act(async () => {
      inFlight.resolve();
      await vi.runAllTimersAsync();
    });

    expect(mockStore.markClean).not.toHaveBeenCalled();
    expect(mockStore.setSaveStatus).not.toHaveBeenCalledWith("saved");
  });
});
