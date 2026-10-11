import { useCallback, useEffect, useRef } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import {
  enqueueResumeSave,
  flushResumeDocument,
  forgetResumeBaseline,
  isResumeSaving,
  markResumeLoaded,
  saveResumeNow,
} from "../lib/resumeDocumentSync";

const DEBOUNCE_MS = 1500;

/**
 * Debounced auto-save for the resume editor.
 *
 * All the ordering lives in the document sync queue — this hook only decides
 * *when* to hand the live document over, so a save in flight, an unmount and a
 * document switch cannot drop an edit between them.
 */
export function useAutoSave() {
  const resume = useResumeBuilderStore((s) => s.resume);
  const isDirty = useResumeBuilderStore((s) => s.isDirty);
  const resumeId = resume?.id ?? null;
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const loadedIdRef = useRef<string | null>(null);

  const save = useCallback(() => saveResumeNow(), []);

  const flush = useCallback(
    () => (resumeId ? flushResumeDocument(resumeId) : Promise.resolve()),
    [resumeId],
  );

  // A newly opened document defines what the server holds — unless the editor
  // opened on top of unsaved work, in which case nothing may be assumed.
  useEffect(() => {
    if (!resume || loadedIdRef.current === resume.id) return;
    loadedIdRef.current = resume.id;

    if (useResumeBuilderStore.getState().isDirty) {
      forgetResumeBaseline(resume.id);
    } else {
      markResumeLoaded(resume);
    }
  }, [resume]);

  useEffect(() => {
    if (!resume || !isDirty) return;

    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      void enqueueResumeSave().catch(() => {
        // Surfaced through saveStatus; the queue keeps the document dirty.
      });
    }, DEBOUNCE_MS);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [resume, isDirty]);

  // A reload inside the debounce window used to drop whatever had been typed
  // silently. There is no way to finish an async save during unload, so the
  // browser is asked to confirm instead — and only while there is something to
  // lose, so the prompt never appears for a saved document.
  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      const { isDirty: dirty, resume: current } =
        useResumeBuilderStore.getState();
      if (!dirty && !(current && isResumeSaving(current.id))) return;
      event.preventDefault();
      // Legacy browsers need a returnValue to show the prompt; the string is
      // never displayed.
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, []);

  // Save pending changes on SPA navigation (component unmount). The queue is
  // not tied to this component, so the request outlives it.
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      const { isDirty: dirty, resume: current } =
        useResumeBuilderStore.getState();
      if (dirty && current) {
        void enqueueResumeSave().catch(() => {
          // Error state already surfaced via setSaveStatus("error").
        });
      }
    };
  }, []);

  return { save, flush };
}
