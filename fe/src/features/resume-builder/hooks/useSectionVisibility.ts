import { useCallback } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type { SectionOrderDTO } from "@/shared/types/resume-builder";

/** Stable reference to avoid infinite re-renders when store.resume is null. */
const EMPTY: readonly never[] = [];

function normalizeSortOrder(entries: SectionOrderDTO[]): SectionOrderDTO[] {
  return [...entries]
    .sort((a, b) => a.sort_order - b.sort_order)
    .map((entry, idx) => ({ ...entry, sort_order: idx }));
}

export function useSectionVisibility() {
  const sectionOrder = useResumeBuilderStore(
    (s) => s.resume?.section_order ?? EMPTY,
  );
  const setSectionOrder = useResumeBuilderStore((s) => s.setSectionOrder);

  const sorted = [...sectionOrder].sort((a, b) => a.sort_order - b.sort_order);

  const hideSection = useCallback(
    (sectionKey: string) => {
      const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
        if (entry.section_key !== sectionKey) return entry;
        return { ...entry, is_visible: false };
      });
      setSectionOrder(normalizeSortOrder(updated));
    },
    [sectionOrder, setSectionOrder],
  );

  const moveSection = useCallback(
    (sectionKey: string, direction: "up" | "down") => {
      const visibleSorted = sorted.filter((s) => s.is_visible);
      const idx = visibleSorted.findIndex((s) => s.section_key === sectionKey);
      if (idx < 0) return;

      const swapIdx = direction === "up" ? idx - 1 : idx + 1;
      if (swapIdx < 0 || swapIdx >= visibleSorted.length) return;

      const currentOrder = visibleSorted[idx].sort_order;
      const swapOrder = visibleSorted[swapIdx].sort_order;

      const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
        if (entry.section_key === sectionKey) {
          return { ...entry, sort_order: swapOrder };
        }
        if (entry.section_key === visibleSorted[swapIdx].section_key) {
          return { ...entry, sort_order: currentOrder };
        }
        return entry;
      });
      setSectionOrder(normalizeSortOrder(updated));
    },
    [sectionOrder, sorted, setSectionOrder],
  );

  const canMoveUp = useCallback(
    (sectionKey: string) => {
      const visibleSorted = sorted.filter((s) => s.is_visible);
      const idx = visibleSorted.findIndex((s) => s.section_key === sectionKey);
      return idx > 0;
    },
    [sorted],
  );

  const canMoveDown = useCallback(
    (sectionKey: string) => {
      const visibleSorted = sorted.filter((s) => s.is_visible);
      const idx = visibleSorted.findIndex((s) => s.section_key === sectionKey);
      return idx >= 0 && idx < visibleSorted.length - 1;
    },
    [sorted],
  );

  return { hideSection, moveSection, canMoveUp, canMoveDown };
}
