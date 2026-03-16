import { useCallback, useEffect, useRef } from "react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { isServerItem } from "./useSectionPersistence";

const DEBOUNCE_MS = 1500;

/** Snapshot of array section items for dirty-checking. */
type SectionSnapshot = Record<ArraySectionKey, string>;

/** Array section keys that support auto-save updates. */
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

function buildSnapshot(
  resume: NonNullable<
    ReturnType<typeof useResumeBuilderStore.getState>["resume"]
  >,
): SectionSnapshot {
  const snapshot = {} as SectionSnapshot;
  for (const key of SECTION_KEYS) {
    snapshot[key] = JSON.stringify(resume[key]);
  }
  return snapshot;
}

export function useAutoSave() {
  const resume = useResumeBuilderStore((s) => s.resume);
  const isDirty = useResumeBuilderStore((s) => s.isDirty);
  const setSaveStatus = useResumeBuilderStore((s) => s.setSaveStatus);
  const markClean = useResumeBuilderStore((s) => s.markClean);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const savingRef = useRef(false);
  const prevResumeRef = useRef<string>("");
  const lastSnapshotRef = useRef<SectionSnapshot | null>(null);
  const cancelledRef = useRef(false);

  const save = useCallback(async () => {
    const currentResume = useResumeBuilderStore.getState().resume;
    if (!currentResume || savingRef.current) return;

    savingRef.current = true;
    setSaveStatus("saving");

    try {
      // Save independent top-level fields in parallel
      const topLevelPromises: Promise<unknown>[] = [
        resumeBuilderService.update(currentResume.id, {
          title: currentResume.title,
          template_id: currentResume.template_id,
          font_family: currentResume.font_family,
          primary_color: currentResume.primary_color,
          text_color: currentResume.text_color,
          spacing: currentResume.spacing,
          margin_top: currentResume.margin_top,
          margin_bottom: currentResume.margin_bottom,
          margin_left: currentResume.margin_left,
          margin_right: currentResume.margin_right,
          layout_mode: currentResume.layout_mode,
          sidebar_width: currentResume.sidebar_width,
          font_size: currentResume.font_size,
          skill_display: currentResume.skill_display,
        }),
      ];

      if (currentResume.contact) {
        topLevelPromises.push(
          resumeBuilderService.upsertContact(
            currentResume.id,
            currentResume.contact,
          ),
        );
      }

      if (currentResume.summary) {
        topLevelPromises.push(
          resumeBuilderService.upsertSummary(currentResume.id, {
            content: currentResume.summary.content,
          }),
        );
      }

      if (currentResume.section_order.length > 0) {
        topLevelPromises.push(
          resumeBuilderService.updateSectionOrder(currentResume.id, {
            sections: currentResume.section_order,
          }),
        );
      }

      await Promise.all(topLevelPromises);

      // Save array section updates (only items that actually changed)
      const currentSnapshot = buildSnapshot(currentResume);
      const lastSnapshot = lastSnapshotRef.current;

      await saveArraySections(currentResume, currentSnapshot, lastSnapshot);

      lastSnapshotRef.current = buildSnapshot(
        useResumeBuilderStore.getState().resume ?? currentResume,
      );

      // markClean updates Zustand store which is safe after unmount.
      // Only guard React state updates (setSaveStatus) with cancelledRef.
      markClean();
      if (!cancelledRef.current) {
        setSaveStatus("saved");
      }
    } catch {
      if (!cancelledRef.current) {
        setSaveStatus("error");
      }
    } finally {
      savingRef.current = false;
    }
  }, [setSaveStatus, markClean]);

  useEffect(() => {
    if (!resume || !isDirty) return;

    const serialized = JSON.stringify(resume);
    if (serialized === prevResumeRef.current) return;
    prevResumeRef.current = serialized;

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    timerRef.current = setTimeout(save, DEBOUNCE_MS);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [resume, isDirty, save]);

  // Initialize snapshot when resume loads
  useEffect(() => {
    if (resume && !lastSnapshotRef.current) {
      lastSnapshotRef.current = buildSnapshot(resume);
    }
  }, [resume]);

  // Save pending changes on SPA navigation (component unmount)
  useEffect(() => {
    cancelledRef.current = false;
    return () => {
      cancelledRef.current = true;
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      // Fire-and-forget save of any pending dirty state
      const { isDirty: dirty, resume: r } = useResumeBuilderStore.getState();
      if (dirty && r && !savingRef.current) {
        save().catch(() => {
          // Error state already surfaced via setSaveStatus("error")
        });
      }
    };
  }, [save]);

  return { save };
}

/** Check if an entry has any non-empty string content (ignoring id, sort_order, booleans). */
function hasContent(entry: Record<string, unknown>): boolean {
  for (const [key, value] of Object.entries(entry)) {
    if (key === "id" || key === "sort_order") continue;
    if (typeof value === "string" && value.trim() !== "") return true;
  }
  return false;
}

async function saveArraySections(
  currentResume: NonNullable<
    ReturnType<typeof useResumeBuilderStore.getState>["resume"]
  >,
  currentSnapshot: SectionSnapshot,
  lastSnapshot: SectionSnapshot | null,
) {
  const resumeId = currentResume.id;

  for (const key of SECTION_KEYS) {
    if (lastSnapshot && currentSnapshot[key] === lastSnapshot[key]) continue;

    const methodName = SECTION_UPDATE_MAP[key];
    const updateFn = resumeBuilderService[methodName] as (
      id: string,
      entryId: string,
      data: unknown,
    ) => Promise<unknown>;

    const items = currentResume[key] as unknown as Array<
      Record<string, unknown> & { id: string }
    >;

    // Build a map of previous item JSON for item-level diffing
    const prevItems: Array<Record<string, unknown> & { id: string }> =
      lastSnapshot ? JSON.parse(lastSnapshot[key]) : [];
    const prevMap = new Map(
      prevItems.map((item) => [item.id, JSON.stringify(item)]),
    );

    // Only update items that actually changed (item-level diff)
    const itemPromises: Promise<unknown>[] = [];
    for (const item of items) {
      if (!isServerItem(item.id) || !hasContent(item)) continue;
      if (JSON.stringify(item) === prevMap.get(item.id)) continue;
      itemPromises.push(
        updateFn.call(resumeBuilderService, resumeId, item.id, item),
      );
    }

    if (itemPromises.length > 0) {
      await Promise.all(itemPromises);
    }
  }
}
