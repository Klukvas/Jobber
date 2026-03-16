import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useUndoRedo } from "./useUndoRedo";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type { FullResumeDTO } from "@/shared/types/resume-builder";

function makeResume(overrides: Partial<FullResumeDTO> = {}): FullResumeDTO {
  return {
    id: "resume-1",
    title: "My Resume",
    template_id: "modern",
    font_family: "Inter",
    primary_color: "#000",
    text_color: "#000",
    spacing: 1,
    margin_top: 20,
    margin_bottom: 20,
    margin_left: 20,
    margin_right: 20,
    layout_mode: "single",
    sidebar_width: 35,
    font_size: 12,
    skill_display: "",
    created_at: "2024-01-01",
    updated_at: "2024-01-01",
    contact: null,
    summary: null,
    experiences: [],
    educations: [],
    skills: [],
    languages: [],
    certifications: [],
    projects: [],
    volunteering: [],
    custom_sections: [],
    section_order: [],
    ...overrides,
  };
}

describe("useUndoRedo", () => {
  beforeEach(() => {
    // Reset the store and clear temporal history
    useResumeBuilderStore.setState({
      resume: null,
      isDirty: false,
      saveStatus: "idle",
    });
    useResumeBuilderStore.temporal.getState().clear();
  });

  afterEach(() => {
    // Clean up keydown listeners
    vi.restoreAllMocks();
  });

  it("reports canUndo=false and canRedo=false when no history", () => {
    const { result } = renderHook(() => useUndoRedo());

    expect(result.current.canUndo).toBe(false);
    expect(result.current.canRedo).toBe(false);
  });

  it("reports canUndo=true after a state change", () => {
    useResumeBuilderStore.setState({ resume: makeResume() });

    // Make a change so temporal stores a past state
    useResumeBuilderStore.setState({
      resume: makeResume({ title: "Changed" }),
    });

    const { result } = renderHook(() => useUndoRedo());

    expect(result.current.canUndo).toBe(true);
  });

  it("undo reverts to previous state and marks dirty", () => {
    useResumeBuilderStore.setState({
      resume: makeResume({ title: "Original" }),
    });
    useResumeBuilderStore.setState({
      resume: makeResume({ title: "Modified" }),
    });

    const { result } = renderHook(() => useUndoRedo());

    act(() => {
      result.current.undo();
    });

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.title).toBe("Original");
    expect(state.isDirty).toBe(true);
  });

  it("redo restores a previously undone state", () => {
    useResumeBuilderStore.setState({
      resume: makeResume({ title: "Original" }),
    });
    useResumeBuilderStore.setState({
      resume: makeResume({ title: "Modified" }),
    });

    const { result } = renderHook(() => useUndoRedo());

    act(() => {
      result.current.undo();
    });

    expect(result.current.canRedo).toBe(true);

    act(() => {
      result.current.redo();
    });

    const state = useResumeBuilderStore.getState();
    expect(state.resume?.title).toBe("Modified");
    expect(state.isDirty).toBe(true);
  });

  it("canRedo becomes false after redo with no more future states", () => {
    useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
    useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

    const { result } = renderHook(() => useUndoRedo());

    act(() => {
      result.current.undo();
    });

    act(() => {
      result.current.redo();
    });

    expect(result.current.canRedo).toBe(false);
  });

  describe("keyboard shortcuts", () => {
    it("handles Ctrl+Z for undo", () => {
      useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
      useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

      renderHook(() => useUndoRedo());

      act(() => {
        window.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "z",
            ctrlKey: true,
            bubbles: true,
          }),
        );
      });

      expect(useResumeBuilderStore.getState().resume?.title).toBe("A");
    });

    it("handles Ctrl+Y for redo", () => {
      useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
      useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

      renderHook(() => useUndoRedo());

      // First undo
      act(() => {
        window.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "z",
            ctrlKey: true,
            bubbles: true,
          }),
        );
      });

      // Then redo
      act(() => {
        window.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "y",
            ctrlKey: true,
            bubbles: true,
          }),
        );
      });

      expect(useResumeBuilderStore.getState().resume?.title).toBe("B");
    });

    it("handles Ctrl+Shift+Z for redo", () => {
      useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
      useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

      renderHook(() => useUndoRedo());

      // First undo
      act(() => {
        window.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "z",
            ctrlKey: true,
            bubbles: true,
          }),
        );
      });

      // Then redo with Shift+Z
      act(() => {
        window.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "z",
            ctrlKey: true,
            shiftKey: true,
            bubbles: true,
          }),
        );
      });

      expect(useResumeBuilderStore.getState().resume?.title).toBe("B");
    });

    it("does not intercept keyboard shortcut when target is INPUT", () => {
      useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
      useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

      renderHook(() => useUndoRedo());

      const input = document.createElement("input");
      document.body.appendChild(input);

      act(() => {
        const event = new KeyboardEvent("keydown", {
          key: "z",
          ctrlKey: true,
          bubbles: true,
        });
        Object.defineProperty(event, "target", { value: input });
        window.dispatchEvent(event);
      });

      // State should NOT have changed since input was focused
      expect(useResumeBuilderStore.getState().resume?.title).toBe("B");

      document.body.removeChild(input);
    });

    it("does not intercept keyboard shortcut when target is TEXTAREA", () => {
      useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
      useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

      renderHook(() => useUndoRedo());

      const textarea = document.createElement("textarea");
      document.body.appendChild(textarea);

      act(() => {
        const event = new KeyboardEvent("keydown", {
          key: "z",
          ctrlKey: true,
          bubbles: true,
        });
        Object.defineProperty(event, "target", { value: textarea });
        window.dispatchEvent(event);
      });

      expect(useResumeBuilderStore.getState().resume?.title).toBe("B");

      document.body.removeChild(textarea);
    });

    it("does not trigger undo without modifier key", () => {
      useResumeBuilderStore.setState({ resume: makeResume({ title: "A" }) });
      useResumeBuilderStore.setState({ resume: makeResume({ title: "B" }) });

      renderHook(() => useUndoRedo());

      act(() => {
        window.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "z",
            bubbles: true,
          }),
        );
      });

      expect(useResumeBuilderStore.getState().resume?.title).toBe("B");
    });
  });

  it("exposes undo and redo as callable functions", () => {
    const { result } = renderHook(() => useUndoRedo());

    expect(typeof result.current.undo).toBe("function");
    expect(typeof result.current.redo).toBe("function");
  });
});
