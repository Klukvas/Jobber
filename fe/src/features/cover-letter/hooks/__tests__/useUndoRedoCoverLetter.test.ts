import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useUndoRedoCoverLetter } from "../useUndoRedoCoverLetter";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

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

describe("useUndoRedoCoverLetter", () => {
  beforeEach(() => {
    act(() => {
      useCoverLetterStore.setState({
        coverLetter: createMockCoverLetter(),
        isDirty: false,
        saveStatus: "idle",
      });
      useCoverLetterStore.temporal.getState().clear();
    });
  });

  it("canUndo is false initially", () => {
    const { result } = renderHook(() => useUndoRedoCoverLetter());

    expect(result.current.canUndo).toBe(false);
  });

  it("canRedo is false initially", () => {
    const { result } = renderHook(() => useUndoRedoCoverLetter());

    expect(result.current.canRedo).toBe(false);
  });

  it("after store change, canUndo becomes true", () => {
    const { result } = renderHook(() => useUndoRedoCoverLetter());

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Changed Title");
    });

    expect(result.current.canUndo).toBe(true);
  });

  it("undo reverts the change and calls markDirty", () => {
    const { result } = renderHook(() => useUndoRedoCoverLetter());

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Changed Title");
    });

    // markClean to reset isDirty, so we can verify undo sets it back
    act(() => {
      useCoverLetterStore.getState().markClean();
    });

    expect(useCoverLetterStore.getState().isDirty).toBe(false);

    act(() => {
      result.current.undo();
    });

    expect(useCoverLetterStore.getState().coverLetter?.title).toBe(
      "My Cover Letter",
    );
    expect(useCoverLetterStore.getState().isDirty).toBe(true);
  });

  it("after undo, canRedo becomes true", () => {
    const { result } = renderHook(() => useUndoRedoCoverLetter());

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Changed Title");
    });

    act(() => {
      result.current.undo();
    });

    expect(result.current.canRedo).toBe(true);
  });

  it("redo re-applies the change and calls markDirty", () => {
    const { result } = renderHook(() => useUndoRedoCoverLetter());

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Changed Title");
    });

    act(() => {
      result.current.undo();
    });

    // markClean to reset isDirty, so we can verify redo sets it back
    act(() => {
      useCoverLetterStore.getState().markClean();
    });

    expect(useCoverLetterStore.getState().isDirty).toBe(false);

    act(() => {
      result.current.redo();
    });

    expect(useCoverLetterStore.getState().coverLetter?.title).toBe(
      "Changed Title",
    );
    expect(useCoverLetterStore.getState().isDirty).toBe(true);
  });
});
