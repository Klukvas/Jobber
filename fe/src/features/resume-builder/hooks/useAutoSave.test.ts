import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useAutoSave } from "./useAutoSave";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import type { FullResumeDTO } from "@/shared/types/resume-builder";
import { markServerIds } from "./useSectionPersistence";

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    update: vi.fn().mockResolvedValue({}),
    upsertContact: vi.fn().mockResolvedValue({}),
    upsertSummary: vi.fn().mockResolvedValue({}),
    updateSectionOrder: vi.fn().mockResolvedValue([]),
    updateExperience: vi.fn().mockResolvedValue({}),
    updateEducation: vi.fn().mockResolvedValue({}),
    updateSkill: vi.fn().mockResolvedValue({}),
    updateLanguage: vi.fn().mockResolvedValue({}),
    updateCertification: vi.fn().mockResolvedValue({}),
    updateProject: vi.fn().mockResolvedValue({}),
    updateVolunteering: vi.fn().mockResolvedValue({}),
    updateCustomSection: vi.fn().mockResolvedValue({}),
  },
}));

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
    created_at: "2024-01-01",
    updated_at: "2024-01-01",
    contact: {
      full_name: "John",
      email: "j@j.com",
      phone: "",
      location: "",
      website: "",
      linkedin: "",
      github: "",
    },
    summary: { content: "A summary" },
    experiences: [],
    educations: [],
    skills: [],
    languages: [],
    certifications: [],
    projects: [],
    volunteering: [],
    custom_sections: [],
    section_order: [
      {
        section_key: "experience",
        sort_order: 0,
        is_visible: true,
        column: "main",
      },
    ],
    ...overrides,
  };
}

describe("useAutoSave", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useResumeBuilderStore.setState({
      resume: null,
      isDirty: false,
      saveStatus: "idle",
    });
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("does not trigger save when resume is null", () => {
    useResumeBuilderStore.setState({ resume: null, isDirty: true });
    renderHook(() => useAutoSave());

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(resumeBuilderService.update).not.toHaveBeenCalled();
  });

  it("does not trigger save when not dirty", () => {
    useResumeBuilderStore.setState({ resume: makeResume(), isDirty: false });
    renderHook(() => useAutoSave());

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(resumeBuilderService.update).not.toHaveBeenCalled();
  });

  it("triggers save after debounce when dirty", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      // Let the async save resolve
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.update).toHaveBeenCalledWith(
      "resume-1",
      expect.objectContaining({
        title: "My Resume",
        template_id: "modern",
      }),
    );
  });

  it("saves contact when present", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.upsertContact).toHaveBeenCalledWith(
      "resume-1",
      resume.contact,
    );
  });

  it("saves summary when present", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.upsertSummary).toHaveBeenCalledWith(
      "resume-1",
      { content: "A summary" },
    );
  });

  it("saves section order when non-empty", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.updateSectionOrder).toHaveBeenCalledWith(
      "resume-1",
      { sections: resume.section_order },
    );
  });

  it("sets saveStatus to saving then saved on success", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    const statuses: string[] = [];
    const unsub = useResumeBuilderStore.subscribe((state) => {
      statuses.push(state.saveStatus);
    });

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    unsub();
    expect(statuses).toContain("saving");
    expect(statuses).toContain("saved");
  });

  it("sets saveStatus to error on failure", async () => {
    vi.mocked(resumeBuilderService.update).mockRejectedValueOnce(
      new Error("fail"),
    );
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");
  });

  it("debounces multiple rapid changes", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    // First change
    act(() => {
      vi.advanceTimersByTime(500);
    });

    // Second change triggers re-render with new resume
    act(() => {
      useResumeBuilderStore.setState({
        resume: makeResume({ title: "Updated" }),
        isDirty: true,
      });
    });

    // Not enough time yet
    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(resumeBuilderService.update).not.toHaveBeenCalled();

    // Now wait for full debounce after last change
    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.update).toHaveBeenCalledTimes(1);
  });

  it("marks store as clean after successful save", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
  });

  it("saves server-tracked experience items on change", async () => {
    const exp = {
      id: "exp-server-1",
      company: "Acme",
      position: "Dev",
      location: "",
      start_date: "",
      end_date: "",
      is_current: false,
      description: "",
      sort_order: 0,
    };
    // Start with no experiences so the snapshot is initialized as empty
    const resumeNoExp = makeResume({ experiences: [] });
    markServerIds("resume-1", ["exp-server-1"]);

    useResumeBuilderStore.setState({ resume: resumeNoExp, isDirty: false });
    renderHook(() => useAutoSave());

    // Now add the experience and mark as dirty so the snapshot differs
    act(() => {
      useResumeBuilderStore.setState({
        resume: makeResume({ experiences: [exp] }),
        isDirty: true,
      });
    });

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.updateExperience).toHaveBeenCalledWith(
      "resume-1",
      "exp-server-1",
      exp,
    );
  });

  it("exposes save function that can be called manually", async () => {
    const resume = makeResume();
    useResumeBuilderStore.setState({ resume, isDirty: false });
    const { result } = renderHook(() => useAutoSave());

    await act(async () => {
      await result.current.save();
    });

    expect(resumeBuilderService.update).toHaveBeenCalled();
  });

  it("does not save contact when null", async () => {
    const resume = makeResume({ contact: null });
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.upsertContact).not.toHaveBeenCalled();
  });

  it("does not save summary when null", async () => {
    const resume = makeResume({ summary: null });
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.upsertSummary).not.toHaveBeenCalled();
  });

  it("does not save section order when empty", async () => {
    const resume = makeResume({ section_order: [] });
    useResumeBuilderStore.setState({ resume, isDirty: true });
    renderHook(() => useAutoSave());

    await act(async () => {
      vi.advanceTimersByTime(1500);
      await vi.runAllTimersAsync();
    });

    expect(resumeBuilderService.updateSectionOrder).not.toHaveBeenCalled();
  });
});
