import { useCallback, useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import {
  enqueueCoverLetterSave,
  flushCoverLetterDocument,
  forgetCoverLetterBaseline,
  isCoverLetterSaving,
  markCoverLetterLoaded,
  saveCoverLetterNow,
} from "../lib/coverLetterDocumentSync";
import { coverLetterQueryKeys } from "../lib/coverLetterQueryKeys";

const DEBOUNCE_MS = 1500;

/**
 * Debounced auto-save for the cover letter editor.
 *
 * Ordering lives in the document sync queue: saves for one letter run one at a
 * time, an edit made while a request is in flight is sent straight after it,
 * and the letter is only marked clean when the stored revision is the one on
 * screen.
 */
export function useAutoSaveCoverLetter() {
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const isDirty = useCoverLetterStore((s) => s.isDirty);
  const documentId = coverLetter?.id ?? null;
  const queryClient = useQueryClient();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const loadedIdRef = useRef<string | null>(null);

  const persist = useCallback(async () => {
    try {
      await enqueueCoverLetterSave();
      queryClient.invalidateQueries({ queryKey: coverLetterQueryKeys.list() });
    } catch {
      // Surfaced through saveStatus; the letter stays dirty.
    }
  }, [queryClient]);

  const save = useCallback(async () => {
    await saveCoverLetterNow();
    queryClient.invalidateQueries({ queryKey: coverLetterQueryKeys.list() });
  }, [queryClient]);

  const flush = useCallback(
    () =>
      documentId ? flushCoverLetterDocument(documentId) : Promise.resolve(),
    [documentId],
  );

  // A newly opened letter defines what the server holds — unless the editor
  // opened on top of unsaved work, in which case nothing may be assumed.
  useEffect(() => {
    if (!coverLetter || loadedIdRef.current === coverLetter.id) return;
    loadedIdRef.current = coverLetter.id;

    if (useCoverLetterStore.getState().isDirty) {
      forgetCoverLetterBaseline(coverLetter.id);
    } else {
      markCoverLetterLoaded(coverLetter);
    }
  }, [coverLetter]);

  useEffect(() => {
    if (!coverLetter || !isDirty) return;

    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => void persist(), DEBOUNCE_MS);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [coverLetter, isDirty, persist]);

  // There is no way to finish an async save during unload, so the browser is
  // asked to confirm instead — and only while there is something to lose.
  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      const { isDirty: dirty, coverLetter: current } =
        useCoverLetterStore.getState();
      if (!dirty && !(current && isCoverLetterSaving(current.id))) return;
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, []);

  // Flush pending changes on SPA navigation (component unmount). The queue is
  // not tied to this component, so the request outlives it.
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      const { isDirty: dirty, coverLetter: current } =
        useCoverLetterStore.getState();
      if (dirty && current) void persist();
    };
  }, [persist]);

  return { save, flush };
}
