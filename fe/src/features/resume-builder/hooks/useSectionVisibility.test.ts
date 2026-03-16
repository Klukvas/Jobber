import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSectionVisibility } from "./useSectionVisibility";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type {
  FullResumeDTO,
  SectionOrderDTO,
} from "@/shared/types/resume-builder";

function makeResume(sectionOrder: SectionOrderDTO[]): FullResumeDTO {
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
    section_order: sectionOrder,
  };
}

function makeSection(
  key: string,
  order: number,
  visible = true,
): SectionOrderDTO {
  return {
    section_key: key,
    sort_order: order,
    is_visible: visible,
    column: "main",
  };
}

describe("useSectionVisibility", () => {
  beforeEach(() => {
    useResumeBuilderStore.setState({
      resume: null,
      isDirty: false,
      saveStatus: "idle",
    });
  });

  describe("hideSection", () => {
    it("sets is_visible to false for the given section", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
        makeSection("skills", 2),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      act(() => {
        result.current.hideSection("education");
      });

      const updated =
        useResumeBuilderStore.getState().resume?.section_order ?? [];
      const edu = updated.find((s) => s.section_key === "education");
      expect(edu?.is_visible).toBe(false);
    });

    it("does not affect other sections", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      act(() => {
        result.current.hideSection("education");
      });

      const updated =
        useResumeBuilderStore.getState().resume?.section_order ?? [];
      const exp = updated.find((s) => s.section_key === "experience");
      expect(exp?.is_visible).toBe(true);
    });

    it("normalizes sort order after hiding", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
        makeSection("skills", 2),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      act(() => {
        result.current.hideSection("education");
      });

      const updated =
        useResumeBuilderStore.getState().resume?.section_order ?? [];
      const orders = updated.map((s) => s.sort_order).sort((a, b) => a - b);
      expect(orders).toEqual([0, 1, 2]);
    });
  });

  describe("moveSection", () => {
    it("moves a section up by swapping sort_order with the previous visible section", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
        makeSection("skills", 2),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      act(() => {
        result.current.moveSection("education", "up");
      });

      const updated =
        useResumeBuilderStore.getState().resume?.section_order ?? [];
      const edu = updated.find((s) => s.section_key === "education");
      const exp = updated.find((s) => s.section_key === "experience");
      // After swap and normalization, education should come before experience
      expect(edu!.sort_order).toBeLessThan(exp!.sort_order);
    });

    it("moves a section down by swapping sort_order with the next visible section", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
        makeSection("skills", 2),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      act(() => {
        result.current.moveSection("education", "down");
      });

      const updated =
        useResumeBuilderStore.getState().resume?.section_order ?? [];
      const edu = updated.find((s) => s.section_key === "education");
      const skills = updated.find((s) => s.section_key === "skills");
      expect(edu!.sort_order).toBeGreaterThan(skills!.sort_order);
    });

    it("does nothing when moving the first visible section up", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());
      const beforeOrder =
        useResumeBuilderStore.getState().resume?.section_order;

      act(() => {
        result.current.moveSection("experience", "up");
      });

      const afterOrder = useResumeBuilderStore.getState().resume?.section_order;
      // The section_order reference stays the same since no swap was made
      expect(afterOrder).toBe(beforeOrder);
    });

    it("does nothing when moving the last visible section down", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());
      const beforeOrder =
        useResumeBuilderStore.getState().resume?.section_order;

      act(() => {
        result.current.moveSection("education", "down");
      });

      const afterOrder = useResumeBuilderStore.getState().resume?.section_order;
      expect(afterOrder).toBe(beforeOrder);
    });

    it("skips hidden sections when determining swap target", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1, false), // hidden
        makeSection("skills", 2),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      act(() => {
        result.current.moveSection("skills", "up");
      });

      const updated =
        useResumeBuilderStore.getState().resume?.section_order ?? [];
      const skills = updated.find((s) => s.section_key === "skills");
      const exp = updated.find((s) => s.section_key === "experience");
      // Skills should now be before experience (education was hidden, so it swaps with experience)
      expect(skills!.sort_order).toBeLessThan(exp!.sort_order);
    });

    it("does nothing for a non-existent section key", () => {
      const sections = [makeSection("experience", 0)];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());
      const beforeOrder =
        useResumeBuilderStore.getState().resume?.section_order;

      act(() => {
        result.current.moveSection("nonexistent", "up");
      });

      const afterOrder = useResumeBuilderStore.getState().resume?.section_order;
      expect(afterOrder).toBe(beforeOrder);
    });
  });

  describe("canMoveUp", () => {
    it("returns false for the first visible section", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveUp("experience")).toBe(false);
    });

    it("returns true for non-first visible sections", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveUp("education")).toBe(true);
    });

    it("returns false for unknown section key", () => {
      const sections = [makeSection("experience", 0)];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveUp("nonexistent")).toBe(false);
    });
  });

  describe("canMoveDown", () => {
    it("returns false for the last visible section", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveDown("education")).toBe(false);
    });

    it("returns true for non-last visible sections", () => {
      const sections = [
        makeSection("experience", 0),
        makeSection("education", 1),
      ];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveDown("experience")).toBe(true);
    });

    it("returns false for unknown section key", () => {
      const sections = [makeSection("experience", 0)];
      useResumeBuilderStore.setState({ resume: makeResume(sections) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveDown("nonexistent")).toBe(false);
    });
  });

  describe("with empty section order", () => {
    it("returns canMoveUp false for any section", () => {
      useResumeBuilderStore.setState({ resume: makeResume([]) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveUp("any")).toBe(false);
    });

    it("returns canMoveDown false for any section", () => {
      useResumeBuilderStore.setState({ resume: makeResume([]) });

      const { result } = renderHook(() => useSectionVisibility());

      expect(result.current.canMoveDown("any")).toBe(false);
    });
  });
});
