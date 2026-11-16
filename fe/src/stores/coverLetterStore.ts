import { create } from "zustand";
import { temporal } from "zundo";
import type {
  CoverLetterDTO,
  UpdateCoverLetterRequest,
} from "@/shared/types/cover-letter";

/** Keys that can be individually updated via updateField. */
type EditableCoverLetterKey = keyof UpdateCoverLetterRequest &
  keyof CoverLetterDTO;

type SaveStatus = "idle" | "saving" | "saved" | "error";

interface CoverLetterState {
  coverLetter: CoverLetterDTO | null;
  paragraphIds: string[];
  saveStatus: SaveStatus;
  isDirty: boolean;

  // Actions
  setCoverLetter: (coverLetter: CoverLetterDTO) => void;
  setSaveStatus: (status: SaveStatus) => void;
  markDirty: () => void;
  markClean: () => void;

  // Immutable field updates
  updateField: <K extends EditableCoverLetterKey>(
    key: K,
    value: CoverLetterDTO[K],
  ) => void;
  updateFields: (
    updates: Partial<
      Pick<
        CoverLetterDTO,
        | "title"
        | "template"
        | "recipient_name"
        | "recipient_title"
        | "company_name"
        | "company_address"
        | "greeting"
        | "paragraphs"
        | "closing"
        | "font_family"
        | "font_size"
        | "primary_color"
        | "job_id"
      >
    >,
  ) => void;

  // Paragraph management (immutable)
  addParagraph: () => void;
  updateParagraph: (index: number, value: string) => void;
  removeParagraph: (index: number) => void;
}

export const useCoverLetterStore = create<CoverLetterState>()(
  temporal(
    (set) => ({
      coverLetter: null,
      paragraphIds: [],
      saveStatus: "idle",
      isDirty: false,

      setCoverLetter: (coverLetter) =>
        set((state) => ({
          coverLetter,
          paragraphIds: state.coverLetter?.id === coverLetter.id &&
            state.coverLetter.paragraphs.length === coverLetter.paragraphs.length &&
            state.coverLetter.paragraphs.every((text, index) => text === coverLetter.paragraphs[index])
              ? state.paragraphIds
              : coverLetter.paragraphs.map(() => crypto.randomUUID()),
          isDirty: false,
          saveStatus: "idle",
        })),

      setSaveStatus: (saveStatus) => set({ saveStatus }),

      markDirty: () => set({ isDirty: true, saveStatus: "idle" }),

      markClean: () => set({ isDirty: false }),

      updateField: (key, value) =>
        set((state) => {
          if (!state.coverLetter) return state;
          return {
            coverLetter: { ...state.coverLetter, [key]: value },
            ...(key === "paragraphs" ? {
              paragraphIds: (value as string[]).map(() => crypto.randomUUID()),
            } : {}),
            isDirty: true,
          };
        }),

      updateFields: (updates) =>
        set((state) => {
          if (!state.coverLetter) return state;
          return {
            coverLetter: { ...state.coverLetter, ...updates },
            ...(updates.paragraphs ? {
              paragraphIds: updates.paragraphs.map(() => crypto.randomUUID()),
            } : {}),
            isDirty: true,
          };
        }),

      addParagraph: () =>
        set((state) => {
          if (!state.coverLetter) return state;
          return {
            coverLetter: {
              ...state.coverLetter,
              paragraphs: [...state.coverLetter.paragraphs, ""],
            },
            paragraphIds: [...state.paragraphIds, crypto.randomUUID()],
            isDirty: true,
          };
        }),

      updateParagraph: (index, value) =>
        set((state) => {
          if (!state.coverLetter) return state;
          const paragraphs = state.coverLetter.paragraphs.map((p, i) =>
            i === index ? value : p,
          );
          return {
            coverLetter: { ...state.coverLetter, paragraphs },
            isDirty: true,
          };
        }),

      removeParagraph: (index) =>
        set((state) => {
          if (!state.coverLetter) return state;
          const paragraphs = state.coverLetter.paragraphs.filter(
            (_, i) => i !== index,
          );
          return {
            coverLetter: { ...state.coverLetter, paragraphs },
            paragraphIds: state.paragraphIds.filter((_, i) => i !== index),
            isDirty: true,
          };
        }),
    }),
    {
      partialize: (state) => ({ coverLetter: state.coverLetter, paragraphIds: state.paragraphIds }),
      limit: 50,
      equality: (pastState, currentState) =>
        pastState.coverLetter === currentState.coverLetter,
    },
  ),
);
