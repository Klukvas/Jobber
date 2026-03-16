import { useEffect, useCallback } from "react";
import { useStore } from "zustand";
import { useCoverLetterStore } from "@/stores/coverLetterStore";

export function useUndoRedoCoverLetter() {
  const { undo, redo, pastStates, futureStates } = useStore(
    useCoverLetterStore.temporal,
  );
  const markDirty = useCoverLetterStore((s) => s.markDirty);

  const canUndo = pastStates.length > 0;
  const canRedo = futureStates.length > 0;

  const handleUndo = useCallback(() => {
    undo();
    markDirty();
  }, [undo, markDirty]);

  const handleRedo = useCallback(() => {
    redo();
    markDirty();
  }, [redo, markDirty]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (!mod) return;

      const target = e.target as HTMLElement | null;
      const tag = target?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || target?.isContentEditable) {
        return;
      }

      if (e.key === "z" && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
      } else if (e.key === "y" || (e.key === "z" && e.shiftKey)) {
        e.preventDefault();
        handleRedo();
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [handleUndo, handleRedo]);

  return { undo: handleUndo, redo: handleRedo, canUndo, canRedo };
}
