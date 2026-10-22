import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { apiClient } from "@/services/api";
import { resumeQueryKeys } from "@/shared/lib/resumeQueryKeys";
import { QueryClientProvider } from "@tanstack/react-query";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import ResumeBuilderEditor from "../ResumeBuilderEditor";
import CoverLetterEditor from "../CoverLetterEditor";
import { getQueryClient } from "@/shared/lib/queryClient";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useCoverLetterStore } from "@/stores/coverLetterStore";
import { resetResumeDocumentSync } from "@/features/resume-builder/lib/resumeDocumentSync";
import { resetCoverLetterDocumentSync } from "@/features/cover-letter/lib/coverLetterDocumentSync";
import type { FullResumeDTO } from "@/shared/types/resume-builder";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

/**
 * Opening another document from the editor changes the route parameter while
 * React Router keeps the same element on screen. No unmount, so the auto-save
 * unmount flush never runs — and the debounce timer is cleared the moment the
 * store holds the new document. Whatever had been typed went with it.
 *
 * These tests drive the real router, the real stores and the real document sync
 * queues; only the services and the heavy panels are stood in for.
 */

const { mockGetResume, mockUpdateResume, mockGetLetter, mockUpdateLetter } =
  vi.hoisted(() => ({
    mockGetResume: vi.fn(),
    mockUpdateResume: vi.fn(),
    mockGetLetter: vi.fn(),
    mockUpdateLetter: vi.fn(),
  }));

/** Counts how often each editor page is mounted, to prove the page is reused. */
const { pageMounts } = vi.hoisted(() => ({
  pageMounts: { counts: {} as Record<string, number> },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: "en" },
  }),
}));

// Called unconditionally at the top of both editors, so its mount effect is a
// faithful count of how often the page itself was mounted.
vi.mock("@/shared/lib/usePageMeta", async () => {
  const { useEffect } = await import("react");
  return {
    usePageMeta: ({ titleKey }: { titleKey: string }) => {
      useEffect(() => {
        pageMounts.counts[titleKey] = (pageMounts.counts[titleKey] ?? 0) + 1;
      }, [titleKey]);
    },
  };
});

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
  EditorToolbar: ({
    onToggleAI,
    onToggleATS,
  }: {
    onToggleAI: () => void;
    onToggleATS: () => void;
  }) => (
    <div data-testid="editor-toolbar">
      <button onClick={onToggleAI}>Open AI</button>
      <button onClick={onToggleATS}>Open ATS</button>
    </div>
  ),
}));

vi.mock("@/features/resume-builder/components/sidebar/DesignSidebar", () => ({
  DesignSidebar: () => <div data-testid="design-sidebar" />,
}));

vi.mock("@/services/api", async (original) => ({
  ...(await original<typeof import("@/services/api")>()),
  apiClient: { post: vi.fn() },
}));
vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    canCreate: () => true,
    limits: { max_ai_requests: 1 },
  }),
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

function makeResume(id: string, title: string): FullResumeDTO {
  return {
    id,
    title,
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
  };
}

function makeLetter(id: string, title: string): CoverLetterDTO {
  return {
    id,
    resume_builder_id: null,
    job_id: null,
    title,
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
  };
}

function renderRoute(path: string) {
  const router = createMemoryRouter(
    [
      { path: "/app/resume-builder/:id", element: <ResumeBuilderEditor /> },
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
  pageMounts.counts = {};
  mockUpdateResume.mockResolvedValue({});
  mockUpdateLetter.mockResolvedValue({});
  mockGetResume.mockImplementation(async (id: string) =>
    makeResume(id, id === "resume-a" ? "Resume A" : "Resume B"),
  );
  mockGetLetter.mockImplementation(async (id: string) =>
    makeLetter(id, id === "cl-a" ? "Letter A" : "Letter B"),
  );
  getQueryClient().clear();
  resetResumeDocumentSync();
  resetCoverLetterDocumentSync();
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

describe("resume editor — the route id changes under the same page", () => {
  it("stores the open resume before the next one replaces it", async () => {
    const router = renderRoute("/app/resume-builder/resume-a");
    expect(await screen.findByText("Resume A")).toBeInTheDocument();

    // A keystroke the debounce has not fired for yet.
    act(() => {
      useResumeBuilderStore
        .getState()
        .updateDesign({ title: "Resume A typed" });
    });

    await navigateTo(router, "/app/resume-builder/resume-b");

    expect(await screen.findByText("Resume B")).toBeInTheDocument();
    expect(mockUpdateResume).toHaveBeenCalledWith(
      "resume-a",
      expect.objectContaining({ title: "Resume A typed" }),
    );
    // The page itself was never unmounted, so nothing else could have saved it.
    expect(pageMounts.counts["resumeBuilder.editor"]).toBe(1);
  });

  it("hands the typed title back when its save failed on the way out", async () => {
    mockUpdateResume.mockRejectedValue(new Error("offline"));
    const router = renderRoute("/app/resume-builder/resume-a");
    expect(await screen.findByText("Resume A")).toBeInTheDocument();

    act(() => {
      useResumeBuilderStore
        .getState()
        .updateDesign({ title: "Resume A typed" });
    });
    await navigateTo(router, "/app/resume-builder/resume-b");
    expect(await screen.findByText("Resume B")).toBeInTheDocument();

    await navigateTo(router, "/app/resume-builder/resume-a");

    expect(await screen.findByText("Resume A typed")).toBeInTheDocument();
    expect(useResumeBuilderStore.getState().isDirty).toBe(true);
    expect(useResumeBuilderStore.getState().saveStatus).toBe("error");
  });
});

describe("cover letter editor — the route id changes under the same page", () => {
  it("stores the open letter before the next one replaces it", async () => {
    const router = renderRoute("/app/cover-letters/cl-a");
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Letter A typed");
    });

    await navigateTo(router, "/app/cover-letters/cl-b");

    expect(await screen.findByDisplayValue("Letter B")).toBeInTheDocument();
    expect(mockUpdateLetter).toHaveBeenCalledWith(
      "cl-a",
      expect.objectContaining({ title: "Letter A typed" }),
    );
    expect(pageMounts.counts["coverLetter.editor"]).toBe(1);
  });

  it("hands the typed title back when its save failed on the way out", async () => {
    mockUpdateLetter.mockRejectedValue(new Error("offline"));
    const router = renderRoute("/app/cover-letters/cl-a");
    expect(await screen.findByDisplayValue("Letter A")).toBeInTheDocument();

    act(() => {
      useCoverLetterStore.getState().updateField("title", "Letter A typed");
    });
    await navigateTo(router, "/app/cover-letters/cl-b");
    expect(await screen.findByDisplayValue("Letter B")).toBeInTheDocument();

    await navigateTo(router, "/app/cover-letters/cl-a");

    expect(
      await screen.findByDisplayValue("Letter A typed"),
    ).toBeInTheDocument();
    expect(useCoverLetterStore.getState().isDirty).toBe(true);
    expect(useCoverLetterStore.getState().saveStatus).toBe("error");
  });
});

for (const panel of ["AI", "ATS"] as const) {
  it(`ignores a late ${panel} response for A after navigating to cached B`, async () => {
    let resolveResponse: (value: unknown) => void = () => {};
    vi.mocked(apiClient.post).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveResponse = resolve;
        }),
    );
    // B already exists in the detail cache: route reuse does not pass through
    // a loading screen, so a reset in an unmount/loading branch cannot help.
    getQueryClient().setQueryData(
      resumeQueryKeys.builderDetail("resume-b"),
      makeResume("resume-b", "Resume B"),
    );
    const router = renderRoute("/app/resume-builder/resume-a");
    await screen.findByText("Resume A");
    await userEvent.click(screen.getByText(`Open ${panel}`));
    await userEvent.click(
      screen.getByRole("button", {
        name:
          panel === "AI"
            ? "resumeBuilder.ai.suggestSummary"
            : "resumeBuilder.ats.check",
      }),
    );
    await waitFor(() => expect(apiClient.post).toHaveBeenCalledTimes(1));
    await navigateTo(router, "/app/resume-builder/resume-b");
    await screen.findByText("Resume B");
    await act(async () =>
      resolveResponse(
        panel === "AI"
          ? { summary: "Late summary from A" }
          : {
              score: 83,
              issues: [],
              suggestions: ["Late ATS from A"],
              keywords_found: [],
            },
      ),
    );
    expect(
      screen.queryByText(/Late (summary|ATS) from A/),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "resumeBuilder.ai.apply" }),
    ).not.toBeInTheDocument();
    expect(useResumeBuilderStore.getState().resume?.id).toBe("resume-b");
    expect(useResumeBuilderStore.getState().resume?.summary).toBeNull();
    expect(pageMounts.counts["resumeBuilder.editor"]).toBe(1);
  });
}
