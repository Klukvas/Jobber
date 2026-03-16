import { useCallback, useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { coverLetterService } from "@/services/coverLetterService";

const DEBOUNCE_MS = 1500;

export function useAutoSaveCoverLetter() {
  const coverLetter = useCoverLetterStore((s) => s.coverLetter);
  const isDirty = useCoverLetterStore((s) => s.isDirty);
  const setSaveStatus = useCoverLetterStore((s) => s.setSaveStatus);
  const markClean = useCoverLetterStore((s) => s.markClean);
  const queryClient = useQueryClient();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const savingRef = useRef(false);
  const prevCoverLetterRef = useRef<string>("");

  const save = useCallback(async () => {
    const current = useCoverLetterStore.getState().coverLetter;
    if (!current || savingRef.current) return;

    savingRef.current = true;
    setSaveStatus("saving");

    try {
      await coverLetterService.update(current.id, {
        title: current.title,
        template: current.template,
        recipient_name: current.recipient_name,
        recipient_title: current.recipient_title,
        company_name: current.company_name,
        company_address: current.company_address,
        greeting: current.greeting,
        paragraphs: current.paragraphs,
        closing: current.closing,
        font_family: current.font_family,
        font_size: current.font_size,
        primary_color: current.primary_color,
        job_id: current.job_id,
      });

      markClean();
      setSaveStatus("saved");
      queryClient.invalidateQueries({ queryKey: ["cover-letters"] });
    } catch {
      setSaveStatus("error");
    } finally {
      savingRef.current = false;
    }
  }, [setSaveStatus, markClean, queryClient]);

  useEffect(() => {
    if (!coverLetter || !isDirty) return;

    const serialized = JSON.stringify(coverLetter);
    if (serialized === prevCoverLetterRef.current) return;
    prevCoverLetterRef.current = serialized;

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    timerRef.current = setTimeout(save, DEBOUNCE_MS);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [coverLetter, isDirty, save]);

  // Flush pending save on unmount (e.g. navigating back to list)
  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      const { isDirty: dirty } = useCoverLetterStore.getState();
      if (dirty) {
        save().catch(() => {
          // Error state already surfaced via setSaveStatus("error")
        });
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return { save };
}
