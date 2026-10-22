import { beforeEach, describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider, useQuery } from "@tanstack/react-query";
import { useUndoRedo } from "../hooks/useUndoRedo";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { getQueryClient } from "@/shared/lib/queryClient";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import {
  DocumentSwitchedError,
  trackDocumentMutation,
  waitForDocumentMutations,
} from "@/shared/lib/documentSync";
import type {
  ContactDTO,
  FullResumeDTO,
  SkillDTO,
} from "@/shared/types/resume-builder";
import {
  clearServerIds,
  initServerIds,
  isServerItem,
  useSectionPersistence,
} from "../hooks/useSectionPersistence";
import {
  enqueueOutgoingResumeSave,
  isResumeSaving,
  enqueueResumeSave,
  flushResumeDocument,
  forgetResumeBaseline,
  markResumeLoaded,
  reconcileLoadedResume,
  resetResumeDocumentSync,
} from "./resumeDocumentSync";

const {
  mockUpdate,
  mockUpsertContact,
  mockUpsertSummary,
  mockUpdateSectionOrder,
  mockUpdateSkill,
  mockCreateSkill,
  mockDeleteSkill,
} = vi.hoisted(() => ({
  mockUpdate: vi.fn(),
  mockUpsertContact: vi.fn(),
  mockUpsertSummary: vi.fn(),
  mockUpdateSectionOrder: vi.fn(),
  mockUpdateSkill: vi.fn(),
  mockCreateSkill: vi.fn(),
  mockDeleteSkill: vi.fn(),
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    update: mockUpdate,
    upsertContact: mockUpsertContact,
    upsertSummary: mockUpsertSummary,
    updateSectionOrder: mockUpdateSectionOrder,
    updateSkill: mockUpdateSkill,
    createSkill: mockCreateSkill,
    deleteSkill: mockDeleteSkill,
    updateExperience: vi.fn(),
    updateEducation: vi.fn(),
    updateLanguage: vi.fn(),
    updateCertification: vi.fn(),
    updateProject: vi.fn(),
    updateVolunteering: vi.fn(),
    updateCustomSection: vi.fn(),
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

function emptyServerIds(resumeId: string) {
  initServerIds({
    id: resumeId,
    experiences: [],
    educations: [],
    skills: [],
    languages: [],
    certifications: [],
    projects: [],
    volunteering: [],
    custom_sections: [],
  });
}

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>((resolvePromise) => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
}

function loadResume(resume: FullResumeDTO) {
  useResumeBuilderStore.setState({
    resume,
    isDirty: false,
    saveStatus: "idle",
  });
  markResumeLoaded(resume);
}

const skill = (overrides: Partial<SkillDTO> = {}): SkillDTO => ({
  id: "skill-1",
  name: "React",
  level: "Expert",
  sort_order: 0,
  ...overrides,
});

const contact = (): ContactDTO => ({
  full_name: "John",
  email: "john@example.com",
  phone: "",
  location: "",
  website: "",
  linkedin: "",
  github: "",
});

/** Lets every pending promise reaction run before asserting. */
function settleMicrotasks(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

beforeEach(() => {
  vi.clearAllMocks();
  mockUpdate.mockResolvedValue({});
  mockUpsertContact.mockResolvedValue({});
  mockUpsertSummary.mockResolvedValue({});
  mockUpdateSectionOrder.mockResolvedValue([]);
  mockUpdateSkill.mockResolvedValue({});
  mockDeleteSkill.mockResolvedValue(undefined);
  resetResumeDocumentSync();
  clearServerIds("resume-1");
  clearServerIds("resume-2");
  emptyServerIds("resume-1");
  useResumeBuilderStore.setState({
    resume: null,
    isDirty: false,
    saveStatus: "idle",
  });
});

describe("resume document sync", () => {
  // The old save read the store again after the request came back, so a
  // keystroke made while it was in flight was marked clean and never sent.
  it("sends an edit made while an older revision was in flight", async () => {
    const inFlight = deferred();
    const sentTitles: string[] = [];
    mockUpdate.mockImplementation(
      async (_id: string, data: { title: string }) => {
        sentTitles.push(data.title);
        if (sentTitles.length === 1) await inFlight.promise;
        return {};
      },
    );
    loadResume(makeResume({ title: "First" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Second" });

    const saving = enqueueResumeSave();
    useResumeBuilderStore.getState().updateDesign({ title: "Third" });
    inFlight.resolve();
    await saving;

    expect(sentTitles).toEqual(["Second", "Third"]);
    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");
  });

  it("keeps the document dirty while a newer revision is still unsent", async () => {
    const inFlight = deferred();
    let calls = 0;
    mockUpdate.mockImplementation(async () => {
      calls += 1;
      if (calls === 1) await inFlight.promise;
      return {};
    });
    loadResume(makeResume());
    useResumeBuilderStore.getState().updateDesign({ title: "Typing" });

    const saving = enqueueResumeSave();
    useResumeBuilderStore.getState().updateDesign({ title: "Typing more" });

    expect(useResumeBuilderStore.getState().isDirty).toBe(true);

    inFlight.resolve();
    await saving;

    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
  });

  it("coalesces overlapping flushes into one trailing save", async () => {
    const inFlight = deferred();
    let calls = 0;
    mockUpdate.mockImplementation(async () => {
      calls += 1;
      if (calls === 1) await inFlight.promise;
      return {};
    });
    loadResume(makeResume());
    useResumeBuilderStore.getState().updateDesign({ title: "Edited" });

    const first = flushResumeDocument("resume-1");
    const second = flushResumeDocument("resume-1");
    inFlight.resolve();
    await Promise.all([first, second]);

    expect(calls).toBe(1);
  });

  it("keeps the failure visible and retries it on the next flush", async () => {
    mockUpdate.mockRejectedValueOnce(new Error("offline"));
    loadResume(makeResume());
    useResumeBuilderStore.getState().updateDesign({ title: "Edited" });

    await expect(enqueueResumeSave()).rejects.toThrow("offline");
    expect(useResumeBuilderStore.getState().isDirty).toBe(true);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");

    await flushResumeDocument("resume-1");

    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");
  });

  it("does not mark a newly opened resume clean when the previous save lands", async () => {
    const inFlight = deferred();
    mockUpdate.mockImplementation(async () => {
      await inFlight.promise;
      return {};
    });
    loadResume(makeResume({ id: "resume-1", title: "First resume" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Edited" });
    const saving = enqueueResumeSave();

    // The editor navigates to another resume while the request is in flight.
    useResumeBuilderStore
      .getState()
      .setResume(makeResume({ id: "resume-2", title: "Second resume" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Other edit" });

    inFlight.resolve();
    await saving;

    expect(useResumeBuilderStore.getState().resume?.id).toBe("resume-2");
    expect(useResumeBuilderStore.getState().isDirty).toBe(true);
    expect(useResumeBuilderStore.getState().saveStatus).not.toBe("saved");
  });

  // Emptying the last field used to look like "no content worth saving", so
  // the cleared text stayed on the server.
  it("sends a section item whose last field was cleared", async () => {
    const resume = makeResume({ skills: [skill({ level: "" })] });
    initServerIds({ ...resume, skills: [{ id: "skill-1" }] });
    loadResume(resume);

    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "" });
    await enqueueResumeSave();

    expect(mockUpdateSkill).toHaveBeenCalledWith("resume-1", "skill-1", {
      id: "skill-1",
      name: "",
      level: "",
      sort_order: 0,
    });
  });

  it("diffs section items against what was sent, not against the live store", async () => {
    const inFlight = deferred();
    let calls = 0;
    mockUpdate.mockImplementation(async () => {
      calls += 1;
      if (calls === 1) await inFlight.promise;
      return {};
    });
    const resume = makeResume({ skills: [skill()] });
    initServerIds({ ...resume, skills: [{ id: "skill-1" }] });
    loadResume(resume);

    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "Vue" });
    const saving = enqueueResumeSave();
    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "Svelte" });
    inFlight.resolve();
    await saving;

    const names = mockUpdateSkill.mock.calls.map(
      (call) => (call[2] as SkillDTO).name,
    );
    expect(names).toEqual(["Vue", "Svelte"]);
  });
});

// A rejected `Promise.all` hands the failure back while its siblings are still
// running. The next attempt then raced those leftovers over the same fields,
// and the older payload could land last.
describe("resume document sync — overlapping writes", () => {
  it("drains the writes of a failed attempt before starting the next", async () => {
    const contactWrite = deferred();
    let activeContactWrites = 0;
    let overlappingContactWrites = 0;
    mockUpsertContact.mockImplementation(async () => {
      activeContactWrites += 1;
      if (activeContactWrites > 1) overlappingContactWrites += 1;
      try {
        if (mockUpsertContact.mock.calls.length === 1) {
          await contactWrite.promise;
        }
        return {};
      } finally {
        activeContactWrites -= 1;
      }
    });
    const sentTitles: string[] = [];
    mockUpdate.mockImplementation(
      async (_id: string, data: { title: string }) => {
        sentTitles.push(data.title);
        if (sentTitles.length === 1) throw new Error("offline");
        return {};
      },
    );
    loadResume(makeResume({ contact: contact(), title: "First" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Second" });

    const saving = enqueueResumeSave();
    // The user keeps typing while the first attempt is already failing.
    useResumeBuilderStore.getState().updateDesign({ title: "Third" });
    await settleMicrotasks();

    expect(mockUpsertContact).toHaveBeenCalledTimes(1);

    contactWrite.resolve();
    await saving;

    expect(overlappingContactWrites).toBe(0);
    expect(sentTitles).toEqual(["Second", "Third"]);
    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
  });

  it("drains section item writes before retrying a failed one", async () => {
    const secondItemWrite = deferred();
    const writes: { id: string; name: string }[] = [];
    mockUpdateSkill.mockImplementation(
      async (_resumeId: string, skillId: string, data: SkillDTO) => {
        writes.push({ id: skillId, name: data.name });
        const attempt = writes.filter((write) => write.id === skillId).length;
        if (skillId === "skill-1" && attempt === 1) {
          throw new Error("offline");
        }
        if (skillId === "skill-2" && attempt === 1) {
          await secondItemWrite.promise;
        }
        return {};
      },
    );
    const resume = makeResume({
      skills: [skill(), skill({ id: "skill-2", name: "Vue", sort_order: 1 })],
    });
    initServerIds({
      ...resume,
      skills: [{ id: "skill-1" }, { id: "skill-2" }],
    });
    loadResume(resume);

    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "Svelte" });
    useResumeBuilderStore
      .getState()
      .updateSkill("skill-2", { name: "Angular" });
    const saving = enqueueResumeSave();
    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "Solid" });
    await settleMicrotasks();

    expect(writes.map((write) => write.id)).toEqual(["skill-1", "skill-2"]);

    secondItemWrite.resolve();
    await saving;

    // The second attempt only ran once the first had drained, so the newest
    // text is what the server was left with.
    const skillOneWrites = writes
      .filter((write) => write.id === "skill-1")
      .map((write) => write.name);
    expect(skillOneWrites).toEqual(["Svelte", "Solid"]);
    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
  });
});

// Resolving a flush for a document the editor has left reported success, and
// the toolbar it came from then exported a resume nobody had saved.
describe("resume document sync — flush after a document switch", () => {
  it("refuses to flush a document the editor has left", async () => {
    const pendingWrite = deferred();
    void trackDocumentMutation(
      "resume-1",
      pendingWrite.promise,
      "create:skills:local-1",
    );
    loadResume(makeResume());

    const flushing = flushResumeDocument("resume-1");
    // The user navigates to another resume while the section write settles.
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    pendingWrite.resolve();

    await expect(flushing).rejects.toBeInstanceOf(DocumentSwitchedError);
    expect(mockUpdate).not.toHaveBeenCalled();
  });
});

// A failed save used to leave nothing behind: the queue dropped the snapshot,
// and the page then replaced the editor with the stored payload.
describe("resume document sync — unsaved drafts", () => {
  it("hands a failed edit back when the editor returns to the document", async () => {
    mockUpdate.mockRejectedValue(new Error("offline"));
    const stored = makeResume({ title: "Stored title" });
    loadResume(stored);
    useResumeBuilderStore
      .getState()
      .updateDesign({ title: "Typed but never stored" });

    await expect(enqueueResumeSave()).rejects.toThrow("offline");

    // SPA navigation to another resume, then back to this one.
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    const loaded = reconcileLoadedResume(stored);

    expect(loaded.resume.title).toBe("Typed but never stored");
    expect(loaded.unsaved).toBe("failed");
  });

  it("stores the recovered draft when the save is retried", async () => {
    mockUpdate.mockRejectedValueOnce(new Error("offline"));
    const stored = makeResume({ title: "Stored title" });
    loadResume(stored);
    useResumeBuilderStore.getState().updateDesign({ title: "Recovered" });
    await expect(enqueueResumeSave()).rejects.toThrow("offline");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));

    const loaded = reconcileLoadedResume(stored);
    expect(loaded.resume.title).toBe("Recovered");

    // The editor reopens the document the way the page does: dirty, with
    // nothing assumed about what the server already holds.
    forgetResumeBaseline("resume-1");
    useResumeBuilderStore.setState({ resume: loaded.resume, isDirty: true });
    await enqueueResumeSave();

    const sentTitles = mockUpdate.mock.calls.map(
      (call) => (call[1] as { title: string }).title,
    );
    expect(sentTitles).toEqual(["Recovered", "Recovered"]);
    expect(reconcileLoadedResume(stored).unsaved).toBe("none");
  });

  it("prefers edits typed during the fetch over the payload", () => {
    loadResume(makeResume({ title: "Stored title" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Still typing" });

    const loaded = reconcileLoadedResume(makeResume({ title: "Stored title" }));

    expect(loaded.resume.title).toBe("Still typing");
    expect(loaded.unsaved).toBe("pending");
  });

  it("takes the payload for a document with nothing unsaved", () => {
    loadResume(makeResume({ title: "Stored title" }));

    const loaded = reconcileLoadedResume(
      makeResume({ title: "Changed elsewhere" }),
    );

    expect(loaded.resume.title).toBe("Changed elsewhere");
    expect(loaded.unsaved).toBe("none");
  });

  it("keeps an unsent section edit when the document is reloaded", async () => {
    mockUpdateSkill.mockRejectedValue(new Error("offline"));
    const stored = makeResume({ skills: [skill({ name: "React" })] });
    initServerIds({ ...stored, skills: [{ id: "skill-1" }] });
    loadResume(stored);
    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "Vue" });

    await expect(enqueueResumeSave()).rejects.toThrow("offline");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    const loaded = reconcileLoadedResume(stored);

    expect(loaded.resume.skills.map((item) => item.name)).toEqual(["Vue"]);
    expect(loaded.unsaved).toBe("failed");
  });

  it("brings a row whose create failed back to the editor", async () => {
    mockCreateSkill.mockRejectedValue(new Error("create failed"));
    const local = skill({ id: "local-1", name: "Rust" });
    loadResume(makeResume());

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      useResumeBuilderStore.getState().addSkill(local);
      await result.current.add(local);
    });
    await expect(enqueueResumeSave()).rejects.toThrow("create failed");

    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    // The payload predates the create, so the row exists only in the draft.
    const loaded = reconcileLoadedResume(makeResume({ skills: [] }));

    expect(loaded.resume.skills.map((item) => item.id)).toEqual(["local-1"]);
    expect(loaded.unsaved).toBe("failed");
  });
});

// Renaming happens inside the editor and goes through auto-save, so the list
// the resumes page and the job pickers read kept showing the old name.
describe("resume document sync — the list behind the editor", () => {
  const useBuilderList = (titleOnServer: () => string) =>
    useQuery({
      queryKey: resumeQueryKeys.builderList(),
      queryFn: () => [{ id: "resume-1", title: titleOnServer() }],
    });

  const withQueryClient = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: getQueryClient() }, children);

  const listState = () =>
    getQueryClient().getQueryState(resumeQueryKeys.builderList());

  beforeEach(() => {
    getQueryClient().clear();
  });

  it("shows the new title on the list once the rename is stored", async () => {
    let titleOnServer = "Stored title";
    mockUpdate.mockImplementation(
      async (_id: string, data: { title: string }) => {
        titleOnServer = data.title;
        return {};
      },
    );
    const { result } = renderHook(() => useBuilderList(() => titleOnServer), {
      wrapper: withQueryClient,
    });
    await waitFor(() =>
      expect(result.current.data?.[0].title).toBe("Stored title"),
    );

    loadResume(makeResume({ title: "Stored title" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Renamed here" });
    await act(async () => {
      await enqueueResumeSave();
    });

    await waitFor(() =>
      expect(result.current.data?.[0].title).toBe("Renamed here"),
    );
  });

  // Every save stores the title along with the rest of the document, so
  // refreshing the list for each one would be traffic for nothing.
  it("leaves the list alone when only the design changed", async () => {
    getQueryClient().setQueryData(resumeQueryKeys.builderList(), []);
    loadResume(makeResume({ title: "Stored title" }));
    useResumeBuilderStore.getState().updateDesign({ primary_color: "#123456" });

    await enqueueResumeSave();

    expect(listState()?.isInvalidated).toBe(false);
  });

  // Refetching a list that still holds the old name as though it were the kept
  // one is worse than leaving it: the server never took the new title.
  it("leaves the list alone when the rename could not be stored", async () => {
    mockUpdate.mockRejectedValue(new Error("offline"));
    getQueryClient().setQueryData(resumeQueryKeys.builderList(), []);
    loadResume(makeResume({ title: "Stored title" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Renamed here" });

    await expect(enqueueResumeSave()).rejects.toThrow("offline");

    expect(listState()?.isInvalidated).toBe(false);
  });

  it("marks the list stale for a resume the editor has already left", async () => {
    getQueryClient().setQueryData(resumeQueryKeys.builderList(), []);
    loadResume(makeResume({ title: "Stored title" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Renamed on exit" });

    enqueueOutgoingResumeSave("resume-2");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    await settleMicrotasks();

    expect(listState()?.isInvalidated).toBe(true);
  });
});

// The editor's id changes without the page unmounting — React Router keeps the
// same element for `/resume-builder/:id` — so the unmount flush never runs and
// the debounce timer dies with the store update.
describe("resume document sync — the document on its way out", () => {
  it("stores what was typed before the route change replaces it", async () => {
    loadResume(makeResume({ title: "Stored title" }));
    useResumeBuilderStore.getState().updateDesign({ title: "Typed on exit" });

    enqueueOutgoingResumeSave("resume-2");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    await settleMicrotasks();

    expect(mockUpdate).toHaveBeenCalledWith(
      "resume-1",
      expect.objectContaining({ title: "Typed on exit" }),
    );
  });

  it("leaves a document with nothing unsaved alone", async () => {
    loadResume(makeResume({ title: "Stored title" }));

    enqueueOutgoingResumeSave("resume-2");
    await settleMicrotasks();

    expect(mockUpdate).not.toHaveBeenCalled();
  });

  // Returning before the request comes back used to show the stored payload:
  // the revision was on its way, so nothing had failed yet and the editor had
  // no draft to fall back on.
  it("offers back a revision whose save is still on its way", async () => {
    const inFlight = deferred();
    mockUpdate.mockImplementation(async () => {
      await inFlight.promise;
      return {};
    });
    const stored = makeResume({ title: "Stored title" });
    loadResume(stored);
    useResumeBuilderStore.getState().updateDesign({ title: "Typed on exit" });

    enqueueOutgoingResumeSave("resume-2");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));

    const loaded = reconcileLoadedResume(stored);
    expect(loaded.resume.title).toBe("Typed on exit");
    expect(loaded.unsaved).toBe("pending");

    inFlight.resolve();
    await settleMicrotasks();

    expect(reconcileLoadedResume(stored).unsaved).toBe("none");
  });

  // The queue used to keep the payload that failed and leave the newer one it
  // had already been handed sitting in the queue unsent.
  it("keeps the newest revision when the save that overlapped it fails", async () => {
    const inFlight = deferred();
    mockUpdate.mockImplementation(async () => {
      await inFlight.promise;
      throw new Error("offline");
    });
    const stored = makeResume({ title: "Stored title" });
    loadResume(stored);
    useResumeBuilderStore.getState().updateDesign({ title: "Edited once" });
    const saving = enqueueResumeSave();

    // The user types on, then opens another resume from a job screen.
    useResumeBuilderStore.getState().updateDesign({ title: "Edited twice" });
    enqueueOutgoingResumeSave("resume-2");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));

    inFlight.resolve();
    await expect(saving).rejects.toThrow("offline");

    const loaded = reconcileLoadedResume(stored);
    expect(loaded.resume.title).toBe("Edited twice");
    expect(loaded.unsaved).toBe("failed");
  });

  // Reconciling the older failed revision and saving it would have overwritten
  // the newer one the queue was still holding.
  it("sends the newest revision when the recovered editor saves again", async () => {
    mockUpdate.mockRejectedValue(new Error("offline"));
    const stored = makeResume({ title: "Stored title" });
    loadResume(stored);
    useResumeBuilderStore.getState().updateDesign({ title: "Edited once" });
    const saving = enqueueResumeSave();
    useResumeBuilderStore.getState().updateDesign({ title: "Edited twice" });
    enqueueOutgoingResumeSave("resume-2");
    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    await expect(saving).rejects.toThrow("offline");

    // The page reopens the resume the way it does after a failed save.
    const loaded = reconcileLoadedResume(stored);
    forgetResumeBaseline("resume-1");
    useResumeBuilderStore.setState({ resume: loaded.resume, isDirty: true });
    mockUpdate.mockResolvedValue({});
    await enqueueResumeSave();

    const sentTitles = mockUpdate.mock.calls.map(
      (call) => (call[1] as { title: string }).title,
    );
    expect(sentTitles.at(-1)).toBe("Edited twice");
    expect(reconcileLoadedResume(stored).unsaved).toBe("none");
  });
});

describe("resume document sync — section writes", () => {
  it("waits for a pending create and keeps the typed content", async () => {
    const inFlight = deferred();
    mockCreateSkill.mockImplementation(async () => {
      await inFlight.promise;
      return skill({ id: "skill-server", name: "", level: "" });
    });
    const local = skill({ id: "local-1", name: "", level: "" });
    loadResume(makeResume({ skills: [local] }));

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    const adding = result.current.add(local);

    // The user types while the create is still on its way.
    await act(async () => {
      useResumeBuilderStore
        .getState()
        .updateSkill("local-1", { name: "Typed while creating" });
      inFlight.resolve();
      await adding;
    });

    const stored = useResumeBuilderStore.getState().resume?.skills ?? [];
    expect(stored).toEqual([
      {
        id: "skill-server",
        name: "Typed while creating",
        level: "",
        sort_order: 0,
      },
    ]);
  });

  it("refuses an export flush while a section create keeps failing", async () => {
    mockCreateSkill.mockRejectedValue(new Error("create failed"));
    const local = skill({ id: "local-1" });
    loadResume(makeResume({ skills: [local] }));

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      await result.current.add(local);
    });

    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");
    await expect(flushResumeDocument("resume-1")).rejects.toThrow(
      "create failed",
    );
    expect(mockUpdate).not.toHaveBeenCalled();
  });

  it("retries a failed create on the next flush", async () => {
    mockCreateSkill
      .mockRejectedValueOnce(new Error("create failed"))
      .mockResolvedValueOnce(skill({ id: "skill-server" }));
    const local = skill({ id: "local-1" });
    loadResume(makeResume({ skills: [local] }));

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      await result.current.add(local);
    });

    await act(async () => {
      await flushResumeDocument("resume-1");
    });

    expect(mockCreateSkill).toHaveBeenCalledTimes(2);
    expect(useResumeBuilderStore.getState().resume?.skills[0].id).toBe(
      "skill-server",
    );
  });

  it("takes a row back off the server when it is removed mid-create", async () => {
    const inFlight = deferred();
    mockCreateSkill.mockImplementation(async () => {
      await inFlight.promise;
      return skill({ id: "skill-server" });
    });
    const local = skill({ id: "local-1" });
    loadResume(makeResume({ skills: [local] }));

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    const adding = result.current.add(local);

    await act(async () => {
      useResumeBuilderStore.getState().removeSkill("local-1");
      await result.current.remove("local-1");
      inFlight.resolve();
      await adding;
      await waitForDocumentMutations("resume-1");
    });

    expect(mockDeleteSkill).toHaveBeenCalledWith("resume-1", "skill-server");
    expect(useResumeBuilderStore.getState().resume?.skills).toEqual([]);
  });

  // A payload the editor had cached before a delete still carries the removed
  // row. Applying it put the row back on screen and marked it as a server row
  // again, so the next save patched an entry the server no longer had — the
  // 404 left the editor stuck on "save failed".
  it("never patches a row the server has already dropped", async () => {
    mockUpdateSkill.mockImplementation(async (_id: string, skillId: string) => {
      if (skillId === "skill-2") throw new Error("not found");
      return {};
    });
    const deleting = deferred();
    mockDeleteSkill.mockImplementation(async () => {
      await deleting.promise;
    });
    const stored = makeResume({
      skills: [skill(), skill({ id: "skill-2", name: "Vue", sort_order: 1 })],
    });
    initServerIds(stored);
    loadResume(stored);

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      useResumeBuilderStore.getState().removeSkill("skill-2");
      const removing = result.current.remove("skill-2");
      deleting.resolve();
      await removing;
    });

    expect(mockDeleteSkill).toHaveBeenCalledWith("resume-1", "skill-2");

    // Reopening the document hands the page the payload the query had cached
    // before the delete went out.
    initServerIds(stored);
    const reopened = reconcileLoadedResume(stored);
    expect(reopened.resume.skills.map((item) => item.id)).toEqual(["skill-1"]);

    forgetResumeBaseline("resume-1");
    useResumeBuilderStore.setState({ resume: reopened.resume, isDirty: true });
    await enqueueResumeSave();

    expect(mockUpdateSkill.mock.calls.map((call) => call[1])).toEqual([
      "skill-1",
    ]);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");
  });

  // A flush retries the delete itself, so the bookkeeping that marks the row
  // gone has to travel with the request rather than with the first attempt.
  it("counts a delete retried by a flush as gone for good", async () => {
    mockDeleteSkill
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(undefined);
    const stored = makeResume({ skills: [skill()] });
    initServerIds(stored);
    loadResume(stored);

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      useResumeBuilderStore.getState().removeSkill("skill-1");
      await result.current.remove("skill-1");
    });

    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");

    await act(async () => {
      await flushResumeDocument("resume-1");
    });

    expect(mockDeleteSkill).toHaveBeenCalledTimes(2);
    expect(isServerItem("skill-1")).toBe(false);

    // The payload the query still holds predates the delete.
    initServerIds(stored);
    expect(reconcileLoadedResume(stored).resume.skills).toEqual([]);
  });

  it("creates every row added in quick succession", async () => {
    mockCreateSkill.mockImplementation(async (_id: string, data: SkillDTO) =>
      skill({ id: `server-${data.name}`, name: data.name }),
    );
    const first = skill({ id: "local-1", name: "React" });
    const second = skill({ id: "local-2", name: "Vue", sort_order: 1 });
    loadResume(makeResume({ skills: [first, second] }));

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    await act(async () => {
      await Promise.all([
        result.current.add(first),
        result.current.add(second),
      ]);
    });

    expect(mockCreateSkill).toHaveBeenCalledTimes(2);
    const ids = (useResumeBuilderStore.getState().resume?.skills ?? []).map(
      (item) => item.id,
    );
    expect(ids).toEqual(["server-React", "server-Vue"]);
  });
});

/**
 * The editor opens whatever the resume's cache entry holds, and `staleTime`
 * kept a payload fresh for five minutes while a save never touched it. Leaving
 * the editor and coming back inside that window re-opened the resume as it had
 * been *before* the save — and the next edit saved that revision back over the
 * server, undoing work the server had already accepted.
 */
describe("resume document sync — a cached resume older than the save", () => {
  const detailKey = resumeQueryKeys.builderDetail("resume-1");
  const detailState = () => getQueryClient().getQueryState(detailKey);

  beforeEach(() => {
    getQueryClient().clear();
  });

  /** The resume as the editor fetched it, before anything was typed. */
  function openFromCache(resume: FullResumeDTO) {
    getQueryClient().setQueryData(detailKey, resume, { updatedAt: 1_000 });
    loadResume(resume);
  }

  it("keeps a saved section edit when the cache serves the pre-save payload", async () => {
    const fetched = makeResume({ skills: [skill({ name: "React" })] });
    initServerIds({ ...fetched, skills: [{ id: "skill-1" }] });
    openFromCache(fetched);
    useResumeBuilderStore.getState().updateSkill("skill-1", { name: "Vue" });

    await enqueueResumeSave();

    // Back to the resumes page and in again: same page, same cache entry.
    const loaded = reconcileLoadedResume(fetched);

    expect(loaded.resume.skills.map((item) => item.name)).toEqual(["Vue"]);
    expect(loaded.unsaved).toBe("none");
  });

  it("keeps a saved rename when the cache serves the pre-save payload", async () => {
    const fetched = makeResume({ title: "Stored title" });
    openFromCache(fetched);
    useResumeBuilderStore.getState().updateDesign({ title: "Saved rename" });

    await enqueueResumeSave();

    expect(reconcileLoadedResume(fetched).resume.title).toBe("Saved rename");
  });

  // Opening another resume and coming back changes the route parameter without
  // unmounting the page, so the store holds the other resume when the cached
  // payload arrives — the queue's own record of what it stored is all there is.
  it("opens the revision it stored when another resume is on screen", async () => {
    const fetched = makeResume({ title: "Stored title" });
    openFromCache(fetched);
    useResumeBuilderStore.getState().updateDesign({ title: "Saved rename" });
    await enqueueResumeSave();

    useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
    const loaded = reconcileLoadedResume(fetched);

    expect(loaded.resume.title).toBe("Saved rename");
    expect(loaded.unsaved).toBe("none");
  });

  it("marks the resume stale after a save without refetching it", async () => {
    const fetched = makeResume({ title: "Stored title" });
    openFromCache(fetched);
    useResumeBuilderStore.getState().updateDesign({ title: "Saved rename" });

    await enqueueResumeSave();

    // Stale, so reopening the resume fetches it again. That a mounted editor
    // is *not* refetched over is proved where there is one to observe it:
    // `pages/__tests__/EditorReopenFreshness.test.tsx`.
    expect(detailState()?.isInvalidated).toBe(true);
    expect(detailState()?.fetchStatus).toBe("idle");
  });

  it("opens a payload the server sent after the save", async () => {
    const fetched = makeResume({ title: "Stored title" });
    openFromCache(fetched);
    useResumeBuilderStore.getState().updateDesign({ title: "Saved rename" });
    await enqueueResumeSave();

    // The refetch that reopening triggers, carrying a rename from elsewhere.
    const refetched = makeResume({ title: "Renamed on another device" });
    getQueryClient().setQueryData(detailKey, refetched, {
      updatedAt: Date.now() + 1_000,
    });

    expect(reconcileLoadedResume(refetched).resume.title).toBe(
      "Renamed on another device",
    );
  });

  it("still prefers unsaved text over its own stored revision", async () => {
    const fetched = makeResume({ title: "Stored title" });
    openFromCache(fetched);
    useResumeBuilderStore.getState().updateDesign({ title: "Saved rename" });
    await enqueueResumeSave();
    useResumeBuilderStore
      .getState()
      .updateDesign({ title: "Typed after the save" });

    const loaded = reconcileLoadedResume(fetched);

    expect(loaded.resume.title).toBe("Typed after the save");
    expect(loaded.unsaved).toBe("pending");
  });

  // The known server ids are seeded from the payload, so ignoring an older one
  // has to leave them alone: a row created since would otherwise look local
  // again, and its edits would never be sent.
  it("keeps a row created after the payload writable", async () => {
    mockCreateSkill.mockResolvedValue(skill({ id: "server-1", name: "Rust" }));
    const fetched = makeResume();
    openFromCache(fetched);

    const { result } = renderHook(() =>
      useSectionPersistence<SkillDTO>("skills"),
    );
    const local = skill({ id: "local-1", name: "Rust" });
    await act(async () => {
      useResumeBuilderStore.getState().addSkill(local);
      await result.current.add(local);
    });
    await enqueueResumeSave();

    const loaded = reconcileLoadedResume(fetched);

    expect(loaded.resume.skills.map((item) => item.id)).toEqual(["server-1"]);
    expect(isServerItem("server-1")).toBe(true);
  });
});


describe("structural history persistence", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetResumeDocumentSync();
    clearServerIds("resume-1");
    mockUpdate.mockResolvedValue({});
    mockUpdateSkill.mockResolvedValue({});
    mockDeleteSkill.mockResolvedValue(undefined);
  });

  it("restores a deleted row remotely on Undo and survives a reload", async () => {
    const original = skill();
    let remote: SkillDTO[] = [original];
    mockDeleteSkill.mockImplementation(async (_id, entryId) => { remote = remote.filter(row => row.id !== entryId); });
    mockCreateSkill.mockImplementation(async (_id, data) => {
      const created = { ...data, id: "restored" };
      remote = [...remote, created];
      return created;
    });
    const initial = makeResume({ skills: [original] });
    loadResume(initial);
    initServerIds(initial);
    useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const history = renderHook(() => useUndoRedo());
    act(() => useResumeBuilderStore.getState().removeSkill(original.id));
    await act(async () => { await persistence.result.current.remove(original.id); });
    act(() => history.result.current.undo());
    await act(async () => { await enqueueResumeSave(); });
    expect(remote.map(row => row.name)).toEqual([original.name]);
    act(() => useResumeBuilderStore.getState().setResume(makeResume({ skills: remote })));
    expect(useResumeBuilderStore.getState().resume?.skills).toEqual(remote);
    expect(useResumeBuilderStore.getState().saveStatus).not.toBe("error");
  });

  it("first Undo removes an added row after server ID adoption, Redo restores it", async () => {
    let remote: SkillDTO[] = [];
    mockCreateSkill.mockImplementation(async (_id, data) => {
      const created = { ...data, id: `created-${remote.length}` };
      remote = [...remote, created];
      return created;
    });
    mockDeleteSkill.mockImplementation(async (_id, entryId) => { remote = remote.filter(row => row.id !== entryId); });
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const history = renderHook(() => useUndoRedo());
    const added = skill({id:"local"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    await act(async () => { await persistence.result.current.add(added); });
    act(() => history.result.current.undo());
    expect(useResumeBuilderStore.getState().resume?.skills).toEqual([]);
    await act(async () => { await enqueueResumeSave(); });
    expect(remote).toEqual([]);
    act(() => history.result.current.redo());
    await act(async () => { await enqueueResumeSave(); });
    expect(remote.map(row => row.name)).toEqual([added.name]);
  });

  it.each(["delete", "add"] as const)("preserves Undo-%s structure when an older detail payload arrives before flush", async direction => {
    const original = skill();
    let remote: SkillDTO[] = direction === "delete" ? [original] : [];
    let nextId = 0;
    mockCreateSkill.mockImplementation(async (_id, data) => {
      const created = {...data, id:`remote-${++nextId}`}; remote = [...remote, created]; return created;
    });
    mockDeleteSkill.mockImplementation(async (_id, id) => {remote = remote.filter(row => row.id !== id);});
    const initial = makeResume({skills: remote}); loadResume(initial); initServerIds(initial);
    useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const history = renderHook(() => useUndoRedo());
    if (direction === "delete") {
      act(() => useResumeBuilderStore.getState().removeSkill(original.id));
      await act(async () => {await persistence.result.current.remove(original.id);});
    } else {
      act(() => useResumeBuilderStore.getState().addSkill(original));
      await act(async () => {await persistence.result.current.add(original);});
    }
    act(() => history.result.current.undo());
    const expectedNames = direction === "delete" ? [original.name] : [];
    const loaded = reconcileLoadedResume(makeResume({skills: remote}));
    expect(loaded.resume.skills.map(row => row.name)).toEqual(expectedNames);
    act(() => {useResumeBuilderStore.getState().setResume(loaded.resume); useResumeBuilderStore.getState().markDirty();});
    await act(async () => {await enqueueResumeSave();});
    expect(remote.map(row => row.name)).toEqual(expectedNames);
  });

  it("retries an Undo restoration failure without marking the document saved", async () => {
    const original = skill(); const initial = makeResume({skills:[original]});
    loadResume(initial); initServerIds(initial); useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills")); const history = renderHook(() => useUndoRedo());
    act(() => useResumeBuilderStore.getState().removeSkill(original.id));
    await act(async () => {await persistence.result.current.remove(original.id);});
    act(() => history.result.current.undo());
    mockCreateSkill.mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({...original, id:"restored-retry"});
    await act(async () => {await expect(enqueueResumeSave()).rejects.toThrow("offline");});
    expect(useResumeBuilderStore.getState().isDirty).toBe(true);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");
    await act(async () => {await enqueueResumeSave();});
    expect(useResumeBuilderStore.getState().resume?.skills[0].id).toBe("restored-retry");
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");
  });

  it("flushes a create with the latest edits into its original document after switching", async () => {
    const gate = deferred();
    mockCreateSkill.mockImplementation(async (_id, data) => {await gate.promise; return {...data,id:"outgoing-created"};});
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    const persistence = renderHook(() => useSectionPersistence("skills")); const added = skill({id:"outgoing-local"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    let creation:Promise<void> = Promise.resolve();
    act(() => {creation = persistence.result.current.add(added);});
    act(() => useResumeBuilderStore.getState().updateSkill(added.id,{name:"Latest typed"}));
    act(() => {enqueueOutgoingResumeSave("other");});
    act(() => useResumeBuilderStore.getState().setResume(makeResume({id:"other"})));
    await act(async () => {gate.resolve(); await creation;});
    await waitFor(() => expect(isResumeSaving(initial.id)).toBe(false));
    expect(mockDeleteSkill).not.toHaveBeenCalled();
    expect(mockUpdateSkill).toHaveBeenLastCalledWith(initial.id,"outgoing-created",expect.objectContaining({name:"Latest typed"}));
    const loaded = reconcileLoadedResume(makeResume());
    expect(loaded.resume.skills).toEqual([expect.objectContaining({id:"outgoing-created",name:"Latest typed"})]);
  });

  it("keeps a successful pending CREATE when another document becomes active", async () => {
    const gate = deferred();
    mockCreateSkill.mockImplementation(async (_id, data) => { await gate.promise; return {...data, id:"created-away"}; });
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const added = skill({id:"local-away"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    let pending: Promise<void> = Promise.resolve();
    act(() => { pending = persistence.result.current.add(added); });
    act(() => useResumeBuilderStore.getState().setResume(makeResume({id:"other"})));
    await act(async () => { gate.resolve(); await pending; });
    expect(mockDeleteSkill).not.toHaveBeenCalled();
    expect(useResumeBuilderStore.getState().resume?.id).toBe("other");
  });
});


describe("new section rows racing an older save", () => {
  it.each([false, true])("keeps a newer CREATE through old-save completion and refetch (outgoing=%s)", async outgoing => {
    const topWrite = deferred();
    const deleteWrite = deferred();
    let remote: SkillDTO[] = [];
    mockUpdate.mockImplementationOnce(async () => { await topWrite.promise; return {}; });
    mockDeleteSkill.mockImplementation(async (_resumeId, entryId) => {
      await deleteWrite.promise;
      remote = remote.filter(row => row.id !== entryId);
    });
    mockCreateSkill.mockImplementation(async (_resumeId, data) => {
      const created = { ...data, id: "server-new" };
      remote = [...remote, created];
      return created;
    });
    const initial = makeResume();
    loadResume(initial);
    initServerIds(initial);
    const persistence = renderHook(() => useSectionPersistence("skills"));
    act(() => useResumeBuilderStore.getState().updateDesign({ title: "New title" }));
    let saving: Promise<void> = Promise.resolve();
    act(() => { saving = enqueueResumeSave(); });
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1));
    const added = skill({ id: "new-local" });
    act(() => useResumeBuilderStore.getState().addSkill(added));
    await act(async () => { await persistence.result.current.add(added); });
    if (outgoing) {
      act(() => {
        enqueueOutgoingResumeSave("resume-2");
        useResumeBuilderStore.getState().setResume(makeResume({ id: "resume-2" }));
      });
    }
    // The old implementation starts DELETE(server-new) from this stale snapshot.
    await act(async () => { topWrite.resolve(); await settleMicrotasks(); });
    const loaded = reconcileLoadedResume(makeResume({ title: "New title", skills: remote }));
    act(() => {
      useResumeBuilderStore.getState().setResume(loaded.resume);
      useResumeBuilderStore.getState().markDirty();
    });
    await act(async () => { deleteWrite.resolve(); await saving; await enqueueResumeSave(); });
    await waitFor(() => expect(isResumeSaving(initial.id)).toBe(false));
    expect(mockDeleteSkill).not.toHaveBeenCalled();
    expect(remote.map(row => row.name)).toEqual([added.name]);
    expect(useResumeBuilderStore.getState().resume?.skills.map(row => row.name)).toEqual([added.name]);
    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");
  });
});


describe("structural reconciliation revision scope", () => {
  it("leaves a later successful CREATE intact when only the older save is queued after switching", async () => {
    const topWrite = deferred();
    mockUpdate.mockImplementationOnce(async () => { await topWrite.promise; return {}; });
    mockCreateSkill.mockImplementation(async (_id, data) => ({...data, id:"late-created"}));
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    const persistence = renderHook(() => useSectionPersistence("skills"));
    act(() => useResumeBuilderStore.getState().updateDesign({title:"Older save"}));
    let saving:Promise<void> = Promise.resolve();
    act(() => { saving = enqueueResumeSave(); });
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1));
    const added = skill({id:"local-late"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    await act(async () => { await persistence.result.current.add(added); });
    act(() => useResumeBuilderStore.getState().setResume(makeResume({id:"resume-2"})));
    await act(async () => { topWrite.resolve(); await saving; });
    expect(mockDeleteSkill).not.toHaveBeenCalled();
    expect(isServerItem("late-created")).toBe(true);
    expect(useResumeBuilderStore.getState().resume?.id).toBe("resume-2");
  });

  it("persists an explicit Undo of a pending CREATE after navigating away", async () => {
    const creation = deferred();
    let remote:SkillDTO[] = [];
    mockCreateSkill.mockImplementation(async (_id, data) => {
      await creation.promise;
      const created = {...data, id:"pending-undone"}; remote = [created]; return created;
    });
    mockDeleteSkill.mockImplementation(async (_id, entryId) => {remote = remote.filter(row => row.id !== entryId);});
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const history = renderHook(() => useUndoRedo());
    const added = skill({id:"pending-local"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    let adding:Promise<void> = Promise.resolve();
    act(() => { adding = persistence.result.current.add(added); });
    act(() => history.result.current.undo());
    let saving:Promise<void> = Promise.resolve();
    act(() => {
      saving = enqueueResumeSave();
      useResumeBuilderStore.getState().setResume(makeResume({id:"resume-2"}));
    });
    await act(async () => {creation.resolve(); await adding; await saving;});
    expect(mockDeleteSkill).toHaveBeenCalledExactlyOnceWith(initial.id,"pending-undone");
    expect(remote).toEqual([]);
    expect(useResumeBuilderStore.getState().resume?.id).toBe("resume-2");
  });

  it("uses a newer outgoing Redo instead of deleting from an older Undo snapshot", async () => {
    const topWrite = deferred();
    mockCreateSkill.mockImplementation(async (_id, data) => ({...data,id:"redo-present"}));
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const history = renderHook(() => useUndoRedo());
    const added = skill({id:"redo-local"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    await act(async () => {await persistence.result.current.add(added);});
    act(() => history.result.current.undo());
    mockUpdate.mockImplementationOnce(async () => {await topWrite.promise; return {};});
    let saving:Promise<void> = Promise.resolve();
    act(() => {saving = enqueueResumeSave();});
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1));
    act(() => {
      history.result.current.redo();
      enqueueOutgoingResumeSave("resume-2");
      useResumeBuilderStore.getState().setResume(makeResume({id:"resume-2"}));
    });
    await act(async () => {topWrite.resolve(); await saving;});
    expect(mockDeleteSkill).not.toHaveBeenCalled();
    expect(mockCreateSkill).toHaveBeenCalledTimes(1);
    expect(isServerItem("redo-present")).toBe(true);
    expect(useResumeBuilderStore.getState().resume?.id).toBe("resume-2");
  });
});


describe("pending CREATE racing older top-level writes", () => {
  it.each([false, true])("does not create a pending new row twice (outgoing=%s)", async outgoing => {
    const topWrite = deferred();
    const creation = deferred();
    let createNumber = 0;
    let remote:SkillDTO[] = [];
    mockUpdate.mockImplementationOnce(async () => {await topWrite.promise; return {};});
    mockCreateSkill.mockImplementation(async (_id, data) => {
      const id = `pending-server-${++createNumber}`;
      await creation.promise;
      const created = {...data,id}; remote = [...remote,created]; return created;
    });
    mockDeleteSkill.mockImplementation(async (_id, entryId) => {remote = remote.filter(row => row.id !== entryId);});
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    const persistence = renderHook(() => useSectionPersistence("skills"));
    act(() => useResumeBuilderStore.getState().updateDesign({title:"First snapshot"}));
    let saving:Promise<void> = Promise.resolve();
    act(() => {saving = enqueueResumeSave();});
    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1));
    const added = skill({id:"new-pending-local"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    let adding:Promise<void> = Promise.resolve();
    act(() => {adding = persistence.result.current.add(added);});
    expect(mockCreateSkill).toHaveBeenCalledTimes(1);
    if (outgoing) {
      act(() => {
        enqueueOutgoingResumeSave("resume-2");
        useResumeBuilderStore.getState().setResume(makeResume({id:"resume-2"}));
      });
    }
    await act(async () => {topWrite.resolve(); await settleMicrotasks();});
    const callsWhilePending = mockCreateSkill.mock.calls.length;
    await act(async () => {creation.resolve(); await adding; await saving;});
    expect(callsWhilePending).toBe(1);
    expect(mockCreateSkill).toHaveBeenCalledTimes(1);
    expect(mockDeleteSkill).not.toHaveBeenCalled();
    expect(remote).toEqual([expect.objectContaining({id:"pending-server-1",name:added.name})]);
    if (outgoing) {
      expect(useResumeBuilderStore.getState().resume?.id).toBe("resume-2");
    } else {
      expect(useResumeBuilderStore.getState().resume?.skills).toEqual(remote);
      expect(useResumeBuilderStore.getState().saveStatus).toBe("saved");
    }
  });
});


describe("Redo after compensating a pending CREATE", () => {
  it.each([false, true])("restores the row remotely after Undo compensated its CREATE (outgoing=%s)", async outgoing => {
    const creation = deferred();
    let createNumber = 0;
    let remote:SkillDTO[] = [];
    mockCreateSkill.mockImplementation(async (_id, data) => {
      const number = ++createNumber;
      if (number === 1) await creation.promise;
      const created = {...data,id:`compensated-server-${number}`};
      remote = [...remote,created];
      return created;
    });
    mockDeleteSkill.mockImplementation(async (_id, entryId) => {remote = remote.filter(row => row.id !== entryId);});
    const initial = makeResume(); loadResume(initial); initServerIds(initial);
    useResumeBuilderStore.temporal.getState().clear();
    const persistence = renderHook(() => useSectionPersistence("skills"));
    const history = renderHook(() => useUndoRedo());
    const added = skill({id:"compensated-local"});
    act(() => useResumeBuilderStore.getState().addSkill(added));
    let adding:Promise<void> = Promise.resolve();
    act(() => {adding = persistence.result.current.add(added);});
    act(() => history.result.current.undo());
    await act(async () => {creation.resolve(); await adding;});
    expect(remote).toEqual([]);
    expect(mockDeleteSkill).toHaveBeenCalledExactlyOnceWith(initial.id,"compensated-server-1");
    act(() => history.result.current.redo());
    await act(async () => {await enqueueResumeSave();});
    if (outgoing) {
      act(() => {
        enqueueOutgoingResumeSave("resume-2");
        useResumeBuilderStore.getState().setResume(makeResume({id:"resume-2"}));
      });
    }
    expect(mockCreateSkill).toHaveBeenCalledTimes(2);
    expect(remote).toEqual([expect.objectContaining({id:"compensated-server-2",name:added.name})]);
    const reloaded = reconcileLoadedResume(makeResume({skills:remote}));
    expect(reloaded.resume.skills.map(row => row.name)).toEqual([added.name]);
  });
});

describe("persisted sources for Job resume tailoring", () => {
  it("stores the selected active resume before analysis", async () => {
    const { ensureStoredResumeDocument } = await import("./resumeDocumentSync");
    const resume = makeResume();
    loadResume(resume);
    useResumeBuilderStore.setState({ resume: { ...resume, title: "Edited source" }, isDirty: true });
    await ensureStoredResumeDocument(resume.id);
    expect(mockUpdate).toHaveBeenCalledWith(resume.id, expect.objectContaining({ title: "Edited source" }));
    expect(useResumeBuilderStore.getState().isDirty).toBe(false);
  });
  it("blocks a failed inactive draft without replacing the current editor", async () => {
    const { ensureStoredResumeDocument } = await import("./resumeDocumentSync");
    const resume = makeResume();
    loadResume(resume);
    useResumeBuilderStore.setState({ resume: { ...resume, title: "Unsaved source" }, isDirty: true });
    mockUpdate.mockRejectedValueOnce(new Error("offline"));
    await expect(enqueueResumeSave()).rejects.toThrow("offline");
    const current = makeResume({ id: "resume-2", title: "Current editor" });
    loadResume(current);
    mockUpdate.mockClear();
    await expect(ensureStoredResumeDocument(resume.id)).rejects.toThrow("unsaved editor changes");
    expect(mockUpdate).not.toHaveBeenCalled();
    expect(useResumeBuilderStore.getState().resume).toEqual(current);
    expect(reconcileLoadedResume(resume).unsaved).toBe("failed");
  });
  it("blocks an inactive source while its outgoing save is still pending", async () => {
    const { ensureStoredResumeDocument } = await import("./resumeDocumentSync");
    const resume = makeResume();
    loadResume(resume);
    const pending = deferred();
    mockUpdate.mockReturnValueOnce(pending.promise);
    useResumeBuilderStore.setState({ resume: { ...resume, title: "Saving source" }, isDirty: true });
    const save = enqueueResumeSave();
    await waitFor(() => expect(mockUpdate).toHaveBeenCalled());
    loadResume(makeResume({ id: "resume-2" }));
    await expect(ensureStoredResumeDocument(resume.id)).rejects.toThrow("unsaved editor changes");
    pending.resolve();
    await save;
    await expect(ensureStoredResumeDocument(resume.id)).resolves.toBeUndefined();
  });
  it("waits for inactive section writes and preserves their failures", async () => {
    const { ensureStoredResumeDocument } = await import("./resumeDocumentSync");
    const pending = deferred();
    trackDocumentMutation("inactive-source", pending.promise);
    let settled = false;
    const ready = ensureStoredResumeDocument("inactive-source").then(() => { settled = true; });
    await Promise.resolve();
    expect(settled).toBe(false);
    pending.resolve();
    await ready;
    const failing = Promise.reject(new Error("section failed"));
    void failing.catch(() => {});
    trackDocumentMutation("inactive-source", failing);
    await expect(ensureStoredResumeDocument("inactive-source")).rejects.toThrow("section failed");
  });
});
