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
      saveStatus: "idle",
      isDirty: false,

      setCoverLetter: (coverLetter) =>
        set({ coverLetter, isDirty: false, saveStatus: "idle" }),

      setSaveStatus: (saveStatus) => set({ saveStatus }),

      markDirty: () => set({ isDirty: true, saveStatus: "idle" }),

      markClean: () => set({ isDirty: false }),

      updateField: (key, value) =>
        set((state) => {
          if (!state.coverLetter) return state;
          return {
            coverLetter: { ...state.coverLetter, [key]: value },
            isDirty: true,
          };
        }),

      updateFields: (updates) =>
        set((state) => {
          if (!state.coverLetter) return state;
          return {
            coverLetter: { ...state.coverLetter, ...updates },
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
            isDirty: true,
          };
        }),
    }),
    {
      partialize: (state) => ({ coverLetter: state.coverLetter }),
      limit: 50,
      equality: (pastState, currentState) =>
        pastState.coverLetter === currentState.coverLetter,
    },
  ),
);
