import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import {
  useSectionPersistence,
  initServerIds,
  markServerIds,
  removeServerId,
  isServerItem,
  getServerIds,
  clearServerIds,
  reconcilePendingSectionMutations,
} from "./useSectionPersistence";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import type {
  FullResumeDTO,
  ExperienceDTO,
  SkillDTO,
} from "@/shared/types/resume-builder";

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    createExperience: vi.fn(),
    createEducation: vi.fn(),
    createSkill: vi.fn(),
    createLanguage: vi.fn(),
    createCertification: vi.fn(),
    createProject: vi.fn(),
    createVolunteering: vi.fn(),
    createCustomSection: vi.fn(),
    deleteExperience: vi.fn(),
    deleteEducation: vi.fn(),
    deleteSkill: vi.fn(),
    deleteLanguage: vi.fn(),
    deleteCertification: vi.fn(),
    deleteProject: vi.fn(),
    deleteVolunteering: vi.fn(),
    deleteCustomSection: vi.fn(),
  },
}));

const RESUME_ID = "resume-1";

function makeResume(overrides: Partial<FullResumeDTO> = {}): FullResumeDTO {
  return {
    id: RESUME_ID,
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

describe("useSectionPersistence", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearServerIds(RESUME_ID);
    initServerIds({
      id: RESUME_ID,
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
    useResumeBuilderStore.setState({
      resume: makeResume(),
      isDirty: false,
      saveStatus: "idle",
    });
  });

  describe("add", () => {
    it("calls createExperience API and replaces local item with server item", async () => {
      const localItem: ExperienceDTO = {
        id: "local-1",
        company: "Acme",
        position: "Dev",
        location: "",
        start_date: "",
        end_date: "",
        is_current: false,
        description: "",
        sort_order: 0,
      };
      const serverItem: ExperienceDTO = { ...localItem, id: "server-1" };
      vi.mocked(resumeBuilderService.createExperience).mockResolvedValueOnce(
        serverItem,
      );

      // Add local item to store
      useResumeBuilderStore.setState({
        resume: makeResume({ experiences: [localItem] }),
      });

      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      await act(async () => {
        await result.current.add(localItem);
      });

      expect(resumeBuilderService.createExperience).toHaveBeenCalledWith(
        RESUME_ID,
        expect.objectContaining({ company: "Acme", position: "Dev" }),
      );

      // Server item should replace local in store
      const storeResume = useResumeBuilderStore.getState().resume;
      expect(storeResume?.experiences[0].id).toBe("server-1");
      expect(isServerItem("server-1")).toBe(true);
    });

    it("does not call API when resume is null", async () => {
      useResumeBuilderStore.setState({ resume: null });

      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      const localItem: ExperienceDTO = {
        id: "local-2",
        company: "",
        position: "",
        location: "",
        start_date: "",
        end_date: "",
        is_current: false,
        description: "",
        sort_order: 0,
      };

      await act(async () => {
        await result.current.add(localItem);
      });

      expect(resumeBuilderService.createExperience).not.toHaveBeenCalled();
    });

    it("handles API error gracefully (item stays local)", async () => {
      vi.mocked(resumeBuilderService.createExperience).mockRejectedValueOnce(
        new Error("API error"),
      );

      const localItem: ExperienceDTO = {
        id: "local-3",
        company: "Test",
        position: "",
        location: "",
        start_date: "",
        end_date: "",
        is_current: false,
        description: "",
        sort_order: 0,
      };

      useResumeBuilderStore.setState({
        resume: makeResume({ experiences: [localItem] }),
      });

      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      // Should not throw
      await act(async () => {
        await result.current.add(localItem);
      });

      // Item remains local
      expect(isServerItem("local-3")).toBe(false);
    });

    it("works with skills section type", async () => {
      const localSkill: SkillDTO = {
        id: "local-skill-1",
        name: "React",
        level: "Expert",
        sort_order: 0,
      };
      const serverSkill: SkillDTO = { ...localSkill, id: "server-skill-1" };
      vi.mocked(resumeBuilderService.createSkill).mockResolvedValueOnce(
        serverSkill,
      );

      useResumeBuilderStore.setState({
        resume: makeResume({ skills: [localSkill] }),
      });

      const { result } = renderHook(() =>
        useSectionPersistence<SkillDTO>("skills"),
      );

      await act(async () => {
        await result.current.add(localSkill);
      });

      expect(resumeBuilderService.createSkill).toHaveBeenCalledWith(
        RESUME_ID,
        expect.objectContaining({ name: "React", level: "Expert" }),
      );
    });
  });

  describe("remove", () => {
    it("calls deleteExperience for server-tracked items", async () => {
      markServerIds(RESUME_ID, ["exp-server-1"]);
      vi.mocked(resumeBuilderService.deleteExperience).mockResolvedValueOnce(
        undefined,
      );

      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      await act(async () => {
        await result.current.remove("exp-server-1");
      });

      expect(resumeBuilderService.deleteExperience).toHaveBeenCalledWith(
        RESUME_ID,
        "exp-server-1",
      );
      expect(isServerItem("exp-server-1")).toBe(false);
    });

    it("does not call API for local-only items", async () => {
      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      await act(async () => {
        await result.current.remove("local-only-id");
      });

      expect(resumeBuilderService.deleteExperience).not.toHaveBeenCalled();
    });

    it("does not call API when resume is null", async () => {
      markServerIds(RESUME_ID, ["some-id"]);
      useResumeBuilderStore.setState({ resume: null });

      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      await act(async () => {
        await result.current.remove("some-id");
      });

      expect(resumeBuilderService.deleteExperience).not.toHaveBeenCalled();
    });

    it("handles delete API error gracefully", async () => {
      markServerIds(RESUME_ID, ["exp-fail"]);
      vi.mocked(resumeBuilderService.deleteExperience).mockRejectedValueOnce(
        new Error("Delete failed"),
      );

      const { result } = renderHook(() =>
        useSectionPersistence<ExperienceDTO>("experiences"),
      );

      // Should not throw
      await act(async () => {
        await result.current.remove("exp-fail");
      });

      // ID remains in server set since delete failed
      expect(isServerItem("exp-fail")).toBe(true);
    });
  });
});

// A payload fetched around a create or a delete must not undo it: the new row
// would vanish and the deleted one would come back the moment it is applied.
describe("reconcilePendingSectionMutations", () => {
  const serverSkill: SkillDTO = {
    id: "skill-server",
    name: "Go",
    level: "",
    sort_order: 0,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    clearServerIds(RESUME_ID);
    useResumeBuilderStore.setState({
      resume: makeResume(),
      isDirty: false,
      saveStatus: "idle",
    });
  });

  it("keeps a row whose create has not landed yet", async () => {
    vi.mocked(resumeBuilderService.createSkill).mockRejectedValueOnce(
      new Error("offline"),
    );
    const local: SkillDTO = {
      id: "local-skill",
      name: "React",
      level: "",
      sort_order: 1,
    };
    useResumeBuilderStore.setState({ resume: makeResume({ skills: [local] }) });

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      await result.current.add(local);
    });

    const fresh = makeResume({ skills: [serverSkill] });
    initServerIds(fresh);
    const merged = reconcilePendingSectionMutations(fresh);

    expect(merged.skills).toEqual([serverSkill, local]);
    expect(isServerItem("local-skill")).toBe(false);
  });

  it("hides a row whose delete has not landed yet", async () => {
    markServerIds(RESUME_ID, [serverSkill.id]);
    vi.mocked(resumeBuilderService.deleteSkill).mockRejectedValueOnce(
      new Error("offline"),
    );

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      await result.current.remove(serverSkill.id);
    });

    const fresh = makeResume({ skills: [serverSkill] });
    initServerIds(fresh);
    const merged = reconcilePendingSectionMutations(fresh);

    expect(merged.skills).toEqual([]);
  });

  // The payload simply predates the create that already succeeded.
  it("keeps a row created after the payload was fetched and remembers it", () => {
    const created: SkillDTO = {
      id: "skill-new",
      name: "Vue",
      level: "",
      sort_order: 1,
    };
    useResumeBuilderStore.setState({
      resume: makeResume({ skills: [created] }),
    });

    const fresh = makeResume({ skills: [serverSkill] });
    initServerIds(fresh);
    const merged = reconcilePendingSectionMutations(fresh);

    expect(merged.skills).toEqual([serverSkill, created]);
    expect(isServerItem("skill-new")).toBe(true);
  });

  it("leaves a payload for another resume untouched", () => {
    const other = makeResume({
      id: "resume-2",
      skills: [serverSkill],
    });

    expect(reconcilePendingSectionMutations(other)).toEqual(other);
  });

  // The payload predates the keystrokes in the draft, so taking its copy of a
  // row would hand the customer back the text they had already replaced.
  it("prefers the draft's copy of a row the payload also has", () => {
    const edited: SkillDTO = { ...serverSkill, name: "Rust" };
    const fresh = makeResume({ skills: [serverSkill] });
    initServerIds(fresh);

    const merged = reconcilePendingSectionMutations(
      fresh,
      makeResume({ skills: [edited] }),
    );

    expect(merged.skills).toEqual([edited]);
  });

  /** Removes a row the server holds and lets the delete land. */
  async function removeServerSkill(id: string) {
    markServerIds(RESUME_ID, [id]);
    vi.mocked(resumeBuilderService.deleteSkill).mockResolvedValueOnce(
      undefined,
    );
    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      await result.current.remove(id);
    });
  }

  // The delete landed, so nothing is pending any more — but a payload fetched
  // before it still lists the row, and taking it back put the row the customer
  // deleted back on screen.
  it("drops a deleted row from a payload fetched before the delete", async () => {
    await removeServerSkill(serverSkill.id);

    const stale = makeResume({ skills: [serverSkill] });
    initServerIds(stale);
    const merged = reconcilePendingSectionMutations(stale);

    expect(merged.skills).toEqual([]);
    expect(isServerItem(serverSkill.id)).toBe(false);
  });

  // Same row, the other way round: the payload no longer has it and a draft
  // typed before the delete still does. Rescuing it marked a row the server
  // had dropped as one it holds, and the next save patched it.
  it("does not rescue a deleted row from a draft that predates the delete", async () => {
    await removeServerSkill(serverSkill.id);

    const fresh = makeResume({ skills: [] });
    initServerIds(fresh);
    const merged = reconcilePendingSectionMutations(
      fresh,
      makeResume({ skills: [serverSkill] }),
    );

    expect(merged.skills).toEqual([]);
    expect(isServerItem(serverSkill.id)).toBe(false);
  });

  it("still takes the payload's rows when there is no draft", () => {
    useResumeBuilderStore.setState({
      resume: makeResume({ skills: [{ ...serverSkill, name: "Stale" }] }),
    });
    const fresh = makeResume({ skills: [serverSkill] });
    initServerIds(fresh);

    expect(reconcilePendingSectionMutations(fresh).skills).toEqual([
      serverSkill,
    ]);
  });
});

describe("server ID utility functions", () => {
  beforeEach(() => {
    clearServerIds(RESUME_ID);
    initServerIds({
      id: RESUME_ID,
      experiences: [],
      educations: [],
      skills: [],
      languages: [],
      certifications: [],
      projects: [],
      volunteering: [],
      custom_sections: [],
    });
  });

  describe("initServerIds", () => {
    it("populates server IDs from a loaded resume", () => {
      initServerIds({
        id: RESUME_ID,
        experiences: [{ id: "exp-1" }, { id: "exp-2" }],
        educations: [{ id: "edu-1" }],
        skills: [],
        languages: [],
        certifications: [],
        projects: [],
        volunteering: [],
        custom_sections: [],
      });

      expect(isServerItem("exp-1")).toBe(true);
      expect(isServerItem("exp-2")).toBe(true);
      expect(isServerItem("edu-1")).toBe(true);
      expect(isServerItem("unknown")).toBe(false);
    });

    it("clears previous server IDs on re-init", () => {
      markServerIds(RESUME_ID, ["old-id"]);
      expect(isServerItem("old-id")).toBe(true);

      initServerIds({
        id: RESUME_ID,
        experiences: [],
        educations: [],
        skills: [],
        languages: [],
        certifications: [],
        projects: [],
        volunteering: [],
        custom_sections: [],
      });

      expect(isServerItem("old-id")).toBe(false);
    });
  });

  describe("markServerIds", () => {
    it("adds multiple IDs to the server set", () => {
      markServerIds(RESUME_ID, ["a", "b", "c"]);

      expect(isServerItem("a")).toBe(true);
      expect(isServerItem("b")).toBe(true);
      expect(isServerItem("c")).toBe(true);
    });
  });

  describe("removeServerId", () => {
    it("removes a single ID from the server set", () => {
      markServerIds(RESUME_ID, ["x"]);
      expect(isServerItem("x")).toBe(true);

      removeServerId(RESUME_ID, "x");
      expect(isServerItem("x")).toBe(false);
    });
  });

  describe("getServerIds", () => {
    it("returns a readonly set of all server IDs for a resume", () => {
      markServerIds(RESUME_ID, ["id-1", "id-2"]);

      const ids = getServerIds(RESUME_ID);
      expect(ids.has("id-1")).toBe(true);
      expect(ids.has("id-2")).toBe(true);
      expect(ids.size).toBe(2);
    });
  });

  describe("isServerItem", () => {
    it("returns false for unknown IDs", () => {
      expect(isServerItem("nonexistent")).toBe(false);
    });

    it("returns true for known IDs", () => {
      markServerIds(RESUME_ID, ["known"]);
      expect(isServerItem("known")).toBe(true);
    });
  });
});
