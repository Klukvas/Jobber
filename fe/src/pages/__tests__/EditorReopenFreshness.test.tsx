import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import ResumeBuilderEditor from "../ResumeBuilderEditor";
import CoverLetterEditor from "../CoverLetterEditor";
import { getQueryClient } from "@/shared/lib/queryClient";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import { coverLetterQueryKeys } from "@/features/cover-letter/lib/coverLetterQueryKeys";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { resetResumeDocumentSync } from "@/features/resume-builder/lib/resumeDocumentSync";
import { clearServerIds } from "@/features/resume-builder/hooks/useSectionPersistence";
import { resetCoverLetterDocumentSync } from "@/features/cover-letter/lib/coverLetterDocumentSync";
import type { FullResumeDTO } from "@/shared/types/resume-builder";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

/**
 * Save, leave the editor, open the same document again.
 *
 * The editor opens whatever its own cache entry holds, and a save never
 * refreshed that entry — while `staleTime` kept it "fresh" for five minutes,
 * reopening a document showed the revision from *before* the save, and the next
 * edit saved that revision back over the server. The runtime repro behind these
 * tests is `runtime/scripts/ui-cl-stale-reopen.mjs`: a stored paragraph came
 * back as the pre-save text, and editing another paragraph reverted it.
 *
 * Everything here is real except the services and the heavy panels: the router
 * navigates, the stores hold the document, the app's own QueryClient caches it
 * and the document sync queues do the saving.
 */

const {
  mockGetResume,
  mockUpdateResume,
  mockUpsertSummary,
  mockGetLetter,
  mockUpdateLetter,
} = vi.hoisted(() => ({
  mockGetResume: vi.fn(),
  mockUpdateResume: vi.fn(),
  mockUpsertSummary: vi.fn(),
  mockGetLetter: vi.fn(),
  mockUpdateLetter: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));

vi.mock("@/shared/hooks/useMediaQuery", () => ({ useMediaQuery: () => true }));

vi.mock("@/shared/ui/Sheet", () => ({ Sheet: () => null }));

vi.mock("@/shared/ui/Tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    getById: mockGetResume,
    update: mockUpdateResume,
    upsertContact: vi.fn().mockResolvedValue({}),
    upsertSummary: mockUpsertSummary,
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

vi.mock("@/services/coverLetterService", () => ({
  coverLetterService: { getById: mockGetLetter, update: mockUpdateLetter },
}));

vi.mock("@/features/resume-builder/components/preview/PreviewPanel", () => ({
  PreviewPanel: () => <div data-testid="preview-panel" />,
}));

vi.mock("@/features/resume-builder/components/PreviewErrorBoundary", () => ({
  PreviewErrorBoundary: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

vi.mock("@/features/resume-builder/components/SaveIndicator", () => ({
  SaveIndicator: () => <div data-testid="save-indicator" />,
}));

vi.mock("@/features/resume-builder/components/EditorToolbar", () => ({
  EditorToolbar: () => <div data-testid="editor-toolbar" />,
}));

vi.mock("@/features/resume-builder/components/sidebar/DesignSidebar", () => ({
  DesignSidebar: () => <div data-testid="design-sidebar" />,
}));

vi.mock("@/features/resume-builder/components/AIAssistantPanel", () => ({
  AIAssistantPanel: () => null,
}));

vi.mock("@/features/resume-builder/components/ATSCheckerPanel", () => ({
  ATSCheckerPanel: () => null,
}));

vi.mock("@/features/resume-builder/components/ContentLibraryPanel", () => ({
  ContentLibraryPanel: () => null,
}));

vi.mock("@/features/cover-letter/components/CoverLetterPreview", () => ({
  CoverLetterPreview: () => <div data-testid="letter-preview" />,
  CoverLetterFullscreenPreview: () => null,
}));

vi.mock("@/features/cover-letter/components/CoverLetterSaveIndicator", () => ({
  CoverLetterSaveIndicator: () => <div data-testid="letter-save-indicator" />,
}));

vi.mock("@/features/cover-letter/components/CoverLetterAIPanel", () => ({
  CoverLetterAIPanel: () => null,
}));

vi.mock(
  "@/features/cover-letter/components/CoverLetterTemplateThumbnail",
  () => ({ CoverLetterTemplateThumbnail: () => null }),
);

vi.mock(
  "@/features/cover-letter/components/CoverLetterTemplatePickerModal",
  () => ({
    CoverLetterTemplatePickerModal: () => null,
    COVER_LETTER_TEMPLATES: [
      { id: "professional", labelKey: "coverLetter.templates.professional" },
    ],
  }),
);

vi.mock("@/features/cover-letter/hooks/useExportCoverLetterPDF", () => ({
  useExportCoverLetterPDF: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/features/cover-letter/hooks/useExportCoverLetterDOCX", () => ({
  useExportCoverLetterDOCX: () => ({ mutate: vi.fn(), isPending: false }),
}));

const LETTER_ID = "cl-a";
const RESUME_ID = "resume-a";

function makeResume(): FullResumeDTO {
  return {
    id: RESUME_ID,
    title: "Resume A",
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
    summary: { content: "Original summary" },
    experiences: [],
    educations: [],
    skills: [],
    languages: [],
    certifications: [],
    projects: [],
    volunteering: [],
    custom_sections: [],
    section_order: [],
  };
}

function makeLetter(): CoverLetterDTO {
  return {
    id: LETTER_ID,
    resume_builder_id: null,
    job_id: null,
    title: "Letter A",
    template: "professional",
    recipient_name: "Jane Smith",
    recipient_title: "Hiring Manager",
    company_name: "Acme Corp",
    company_address: "123 Main St",
    greeting: "Dear Jane Smith,",
    paragraphs: ["Original first", "Original second"],
    closing: "Sincerely,",
    font_family: "Georgia",
    font_size: 12,
    primary_color: "#2563eb",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  };
}

/** Stands in for the API's stored document: patched by writes, read by GETs. */
function server<T extends object>(initial: T) {
  let stored = initial;
  return {
    read: (): T => structuredClone(stored),
    patch: (changes: Partial<T>): T => {
      stored = { ...stored, ...changes };
      return structuredClone(stored);
    },
  };
}

/** Room for the editors' 1.5s auto-save debounce to fire and the write to land. */
const DEBOUNCED_SAVE_TIMEOUT = 4_000;

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
}

/**
 * A GET the server answers at once, held on its way back to the client.
 *
 * `payload()` is what the server composed when the request was made — the
 * revision the client would have got had nothing been saved in between — and
 * `release()` lets exactly that response through.
 */
function heldRead<T>(read: () => T) {
  const gate = deferred<void>();
  const answered: T[] = [];

  return {
    request: () => {
      const payload = read();
      answered.push(payload);
      return gate.promise.then(() => payload);
    },
    payload: () => answered[0],
    release: () => gate.resolve(undefined),
  };
}

/**
 * Puts a millisecond between a stored write and the next request.
 *
 * A request that leaves in the same millisecond as a write lands counts as the
 * older one — the editor cannot tell whether the server had the write yet — so
 * a test about a request that is genuinely later has to be genuinely later.
 */
const requestAfterTheWrite = () =>
  new Promise((resolve) => setTimeout(resolve, 2));

/**
 * Lets the render and the effect a cache write schedules run.
 *
 * React Query notifies its observers in a batch, so a payload that has reached
 * the cache has not reached the editor yet. Without this the assertions below
 * would pass on a screen the payload never had a chance to touch.
 */
const settleEffects = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 50));
  });

function renderRoute(path: string) {
  const router = createMemoryRouter(
    [
      { path: "/app/resume-builder", element: <div>resumes list</div> },
      { path: "/app/resume-builder/:id", element: <ResumeBuilderEditor /> },
      { path: "/app/cover-letters", element: <div>letters list</div> },
      { path: "/app/cover-letters/:id", element: <CoverLetterEditor /> },
    ],
    { initialEntries: [path] },
  );
  render(
    <QueryClientProvider client={getQueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
  return router;
}

async function navigateTo(
  router: ReturnType<typeof createMemoryRouter>,
  path: string,
) {
  await act(async () => {
    await router.navigate(path);
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  getQueryClient().clear();
  resetResumeDocumentSync();
  resetCoverLetterDocumentSync();
  clearServerIds(RESUME_ID);
  useResumeBuilderStore.setState({
    resume: null,
    isDirty: false,
    saveStatus: "idle",
  });
  useCoverLetterStore.setState({
    coverLetter: null,
    isDirty: false,
    saveStatus: "idle",
  });
});

describe("cover letter editor — reopening a letter that was saved", () => {
  const detailKey = coverLetterQueryKeys.detail(LETTER_ID);

  /**
   * Edits the letter, then leaves the editor. The unmount flush stores the
   * edit while the list is on screen — the editor is away by the time the
   * write lands, exactly as it is after a debounced save.
   */
  async function saveAndLeave(
    router: ReturnType<typeof createMemoryRouter>,
    edit: () => void,
  ) {
    const writesBefore = mockUpdateLetter.mock.calls.length;
    act(edit);
    await navigateTo(router, "/app/cover-letters");
    await waitFor(() =>
      expect(mockUpdateLetter.mock.calls.length).toBeGreaterThan(writesBefore),
    );
  }

  /** Edits the open letter and waits for auto-save to report it stored. */
  async function saveWhileOpen(edit: () => void) {
    const writesBefore = mockUpdateLetter.mock.calls.length;
    act(edit);
    await waitFor(
      () =>
        expect(mockUpdateLetter.mock.calls.length).toBeGreaterThan(
          writesBefore,
        ),
      { timeout: DEBOUNCED_SAVE_TIMEOUT },
    );
    await waitFor(() =>
      expect(useCoverLetterStore.getState().saveStatus).toBe("saved"),
    );
  }

  // The runtime repro, step for step: edit, wait for "Saved", back to the
  // list, open the same letter again.
  it("shows the stored revision, not the payload the cache still holds", async () => {
    const stored = server(makeLetter());
    mockGetLetter.mockImplementation(async () => stored.read());
    mockUpdateLetter.mockImplementation(
      async (_id: string, patch: Partial<CoverLetterDTO>) =>
        stored.patch(patch),
    );

    const router = renderRoute(`/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();

    await saveWhileOpen(() => {
      useCoverLetterStore.getState().updateField("title", "Letter A saved");
      useCoverLetterStore.getState().updateParagraph(1, "Saved second");
    });
    expect(stored.read().paragraphs[1]).toBe("Saved second");

    await navigateTo(router, "/app/cover-letters");
    expect(await screen.findByText("letters list")).toBeInTheDocument();

    // The refetch that reopening triggers is held back, so what the editor
    // puts on screen can only have come from the cache or from the editor's
    // own copy of the letter.
    const refetch = deferred<CoverLetterDTO>();
    mockGetLetter.mockImplementationOnce(() => refetch.promise);
    await navigateTo(router, `/app/cover-letters/${LETTER_ID}`);

    expect(
      await screen.findByDisplayValue("Letter A saved"),
    ).toBeInTheDocument();
    expect(
      getQueryClient().getQueryData<CoverLetterDTO>(detailKey)?.title,
    ).toBe("Letter A");
    expect(useCoverLetterStore.getState().coverLetter?.paragraphs[1]).toBe(
      "Saved second",
    );

    // And once the server answers, the same revision is still on screen.
    await act(async () => {
      refetch.resolve(stored.read());
    });
    expect(screen.getByDisplayValue("Letter A saved")).toBeInTheDocument();
  });

  // Reopening now refetches, so the refetch failing must not cost the
  // customer the letter they just saved.
  it("keeps the stored revision on screen when the refetch fails", async () => {
    const stored = server(makeLetter());
    mockGetLetter.mockImplementation(async () => stored.read());
    mockUpdateLetter.mockImplementation(
      async (_id: string, patch: Partial<CoverLetterDTO>) =>
        stored.patch(patch),
    );

    const router = renderRoute(`/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();
    await saveAndLeave(router, () =>
      useCoverLetterStore.getState().updateField("title", "Letter A saved"),
    );

    mockGetLetter.mockRejectedValue(new Error("network down"));
    await navigateTo(router, `/app/cover-letters/${LETTER_ID}`);

    await waitFor(
      () =>
        expect(getQueryClient().getQueryState(detailKey)?.status).toBe("error"),
      { timeout: DEBOUNCED_SAVE_TIMEOUT },
    );
    expect(screen.getByDisplayValue("Letter A saved")).toBeInTheDocument();
    expect(screen.queryByText("coverLetter.notFound")).not.toBeInTheDocument();
  });

  it("does not save the pre-save payload back over the server", async () => {
    const stored = server(makeLetter());
    mockGetLetter.mockImplementation(async () => stored.read());
    mockUpdateLetter.mockImplementation(
      async (_id: string, patch: Partial<CoverLetterDTO>) =>
        stored.patch(patch),
    );

    const router = renderRoute(`/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();
    await saveAndLeave(router, () =>
      useCoverLetterStore.getState().updateParagraph(1, "Saved second"),
    );

    await navigateTo(router, `/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();

    // The second edit touches another paragraph; the whole letter is sent.
    await saveAndLeave(router, () =>
      useCoverLetterStore.getState().updateParagraph(0, "Second edit first"),
    );

    expect(stored.read().paragraphs).toEqual([
      "Second edit first",
      "Saved second",
    ]);
  });

  /**
   * The response the server composed before a save, delivered after it.
   *
   * `runtime/scripts/ui-cl-reopen-delayed-get.mjs` (D2) holds a real GET
   * response until the next save has been stored, then lets it through. The
   * payload arrives last, so dating it by its arrival made it look like the
   * newer revision; the editor adopted it, and the next edit sent the
   * paragraph the save had just replaced back to the server.
   */
  it("keeps the stored edit when a pre-write payload arrives after it", async () => {
    const stored = server(makeLetter());
    mockGetLetter.mockImplementation(async () => stored.read());
    mockUpdateLetter.mockImplementation(
      async (_id: string, patch: Partial<CoverLetterDTO>) =>
        stored.patch(patch),
    );

    const router = renderRoute(`/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();
    await saveWhileOpen(() =>
      useCoverLetterStore.getState().updateParagraph(1, "Seeded second"),
    );

    await navigateTo(router, "/app/cover-letters");
    expect(await screen.findByText("letters list")).toBeInTheDocument();

    // The refetch reopening triggers reads the stored letter the moment the
    // request is made — as the server would — and is held on the way back.
    const held = heldRead(() => stored.read());
    mockGetLetter.mockImplementationOnce(held.request);
    await requestAfterTheWrite();
    await navigateTo(router, `/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();

    // Typed and stored while that response is still on its way, so the payload
    // being held cannot contain it.
    await saveWhileOpen(() => {
      useCoverLetterStore.getState().updateField("title", "Letter A held");
      useCoverLetterStore.getState().updateParagraph(1, "Held second");
    });
    expect(stored.read().paragraphs[1]).toBe("Held second");
    expect(held.payload()?.paragraphs[1]).toBe("Seeded second");

    await act(async () => {
      held.release();
    });
    // The pre-write payload really did arrive: the cache is holding it, and it
    // is the newest thing in there. Asserting before this point proves
    // nothing — the editor would not have seen it yet.
    await waitFor(() =>
      expect(
        getQueryClient().getQueryData<CoverLetterDTO>(detailKey)?.paragraphs[1],
      ).toBe("Seeded second"),
    );
    await settleEffects();

    expect(screen.getByDisplayValue("Letter A held")).toBeInTheDocument();
    expect(useCoverLetterStore.getState().coverLetter?.paragraphs[1]).toBe(
      "Held second",
    );

    // The next edit sends the whole letter, so whatever the editor adopted is
    // what the server is left with.
    await saveAndLeave(router, () =>
      useCoverLetterStore.getState().updateParagraph(0, "After the release"),
    );

    expect(stored.read().paragraphs).toEqual([
      "After the release",
      "Held second",
    ]);
  });

  // The guard above may not make the editor blind to the server: a letter
  // changed elsewhere still has to arrive once it is asked for after the save.
  it("opens a payload asked for after the save", async () => {
    const stored = server(makeLetter());
    mockGetLetter.mockImplementation(async () => stored.read());
    mockUpdateLetter.mockImplementation(
      async (_id: string, patch: Partial<CoverLetterDTO>) =>
        stored.patch(patch),
    );

    const router = renderRoute(`/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();
    await saveAndLeave(router, () =>
      useCoverLetterStore.getState().updateParagraph(1, "Saved second"),
    );

    stored.patch({ title: "Renamed on another device" });
    await requestAfterTheWrite();
    await navigateTo(router, `/app/cover-letters/${LETTER_ID}`);

    expect(
      await screen.findByDisplayValue("Renamed on another device"),
    ).toBeInTheDocument();
  });

  it("leaves the open letter alone while a save marks it stale", async () => {
    const stored = server(makeLetter());
    mockGetLetter.mockImplementation(async () => stored.read());
    mockUpdateLetter.mockImplementation(
      async (_id: string, patch: Partial<CoverLetterDTO>) =>
        stored.patch(patch),
    );

    renderRoute(`/app/cover-letters/${LETTER_ID}`);
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Typed here");
    });
    await waitFor(() => expect(mockUpdateLetter).toHaveBeenCalled(), {
      timeout: DEBOUNCED_SAVE_TIMEOUT,
    });

    // Stale for the next time the letter is opened, but not pulled out from
    // under the cursor: a refetch per save would reset the open document.
    expect(getQueryClient().getQueryState(detailKey)?.isInvalidated).toBe(true);
    expect(mockGetLetter).toHaveBeenCalledTimes(1);
    expect(screen.getByDisplayValue("Typed here")).toBeInTheDocument();
  });
});

describe("resume editor — reopening a resume that was saved", () => {
  const detailKey = resumeQueryKeys.builderDetail(RESUME_ID);

  function mockResumeServer() {
    const stored = server(makeResume());
    mockGetResume.mockImplementation(async () => stored.read());
    mockUpdateResume.mockImplementation(
      async (_id: string, patch: Partial<FullResumeDTO>) => stored.patch(patch),
    );
    mockUpsertSummary.mockImplementation(
      async (_id: string, summary: { content: string }) => {
        stored.patch({ summary });
        return summary;
      },
    );
    return stored;
  }

  async function saveAndLeave(
    router: ReturnType<typeof createMemoryRouter>,
    edit: () => void,
  ) {
    const writesBefore = mockUpdateResume.mock.calls.length;
    act(edit);
    await navigateTo(router, "/app/resume-builder");
    await waitFor(() =>
      expect(mockUpdateResume.mock.calls.length).toBeGreaterThan(writesBefore),
    );
  }

  /** Edits the open resume and waits for auto-save to report it stored. */
  async function saveWhileOpen(edit: () => void) {
    const writesBefore = mockUpdateResume.mock.calls.length;
    act(edit);
    await waitFor(
      () =>
        expect(mockUpdateResume.mock.calls.length).toBeGreaterThan(
          writesBefore,
        ),
      { timeout: DEBOUNCED_SAVE_TIMEOUT },
    );
    await waitFor(() =>
      expect(useResumeBuilderStore.getState().saveStatus).toBe("saved"),
    );
  }

  it("shows the stored revision, not the payload the cache still holds", async () => {
    const stored = mockResumeServer();

    const router = renderRoute(`/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();

    await saveWhileOpen(() => {
      useResumeBuilderStore
        .getState()
        .updateDesign({ title: "Resume A saved" });
      useResumeBuilderStore
        .getState()
        .updateSummary({ content: "Saved summary" });
    });
    expect(stored.read().summary?.content).toBe("Saved summary");

    await navigateTo(router, "/app/resume-builder");
    expect(await screen.findByText("resumes list")).toBeInTheDocument();

    const refetch = deferred<FullResumeDTO>();
    mockGetResume.mockImplementationOnce(() => refetch.promise);
    await navigateTo(router, `/app/resume-builder/${RESUME_ID}`);

    expect(await screen.findByText("Resume A saved")).toBeInTheDocument();
    expect(getQueryClient().getQueryData<FullResumeDTO>(detailKey)?.title).toBe(
      "Resume A",
    );
    expect(useResumeBuilderStore.getState().resume?.summary?.content).toBe(
      "Saved summary",
    );

    await act(async () => {
      refetch.resolve(stored.read());
    });
    expect(screen.getByText("Resume A saved")).toBeInTheDocument();
  });

  it("does not save the pre-save payload back over the server", async () => {
    const stored = mockResumeServer();

    const router = renderRoute(`/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();
    await saveAndLeave(router, () =>
      useResumeBuilderStore
        .getState()
        .updateSummary({ content: "Saved summary" }),
    );

    await navigateTo(router, `/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();

    await saveAndLeave(router, () =>
      useResumeBuilderStore.getState().updateDesign({ title: "Second edit" }),
    );

    expect(stored.read().title).toBe("Second edit");
    expect(stored.read().summary?.content).toBe("Saved summary");
  });

  // The resume editor reads its cache entry the same way; see the cover letter
  // case above and D2 in `runtime/scripts/ui-cl-reopen-delayed-get.mjs`.
  it("keeps the stored edit when a pre-write payload arrives after it", async () => {
    const stored = mockResumeServer();

    const router = renderRoute(`/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();
    await saveWhileOpen(() =>
      useResumeBuilderStore
        .getState()
        .updateSummary({ content: "Seeded summary" }),
    );

    await navigateTo(router, "/app/resume-builder");
    expect(await screen.findByText("resumes list")).toBeInTheDocument();

    const held = heldRead(() => stored.read());
    mockGetResume.mockImplementationOnce(held.request);
    await requestAfterTheWrite();
    await navigateTo(router, `/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();

    await saveWhileOpen(() => {
      useResumeBuilderStore.getState().updateDesign({ title: "Resume A held" });
      useResumeBuilderStore
        .getState()
        .updateSummary({ content: "Held summary" });
    });
    expect(stored.read().summary?.content).toBe("Held summary");
    expect(held.payload()?.summary?.content).toBe("Seeded summary");

    await act(async () => {
      held.release();
    });
    await waitFor(() =>
      expect(
        getQueryClient().getQueryData<FullResumeDTO>(detailKey)?.summary
          ?.content,
      ).toBe("Seeded summary"),
    );
    await settleEffects();

    expect(screen.getByText("Resume A held")).toBeInTheDocument();
    expect(useResumeBuilderStore.getState().resume?.summary?.content).toBe(
      "Held summary",
    );

    await saveAndLeave(router, () =>
      useResumeBuilderStore.getState().updateDesign({ title: "After release" }),
    );

    expect(stored.read().title).toBe("After release");
    expect(stored.read().summary?.content).toBe("Held summary");
  });

  it("opens a payload asked for after the save", async () => {
    const stored = mockResumeServer();

    const router = renderRoute(`/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();
    await saveAndLeave(router, () =>
      useResumeBuilderStore
        .getState()
        .updateSummary({ content: "Saved summary" }),
    );

    stored.patch({ title: "Renamed on another device" });
    await requestAfterTheWrite();
    await navigateTo(router, `/app/resume-builder/${RESUME_ID}`);

    expect(
      await screen.findByText("Renamed on another device"),
    ).toBeInTheDocument();
  });

  it("leaves the open resume alone while a save marks it stale", async () => {
    mockResumeServer();

    renderRoute(`/app/resume-builder/${RESUME_ID}`);
    expect(await screen.findByText("Resume A")).toBeInTheDocument();

    act(() => {
      useResumeBuilderStore.getState().updateDesign({ title: "Typed here" });
    });
    await waitFor(() => expect(mockUpdateResume).toHaveBeenCalled(), {
      timeout: DEBOUNCED_SAVE_TIMEOUT,
    });

    expect(getQueryClient().getQueryState(detailKey)?.isInvalidated).toBe(true);
    expect(mockGetResume).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Typed here")).toBeInTheDocument();
  });
});
