import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ResumeBuilderEditorPage from "../ResumeBuilderEditor";
import { createMockResume } from "@/features/resume-builder/components/__tests__/testHelpers";
import type { LoadedResume } from "@/features/resume-builder/lib/resumeDocumentSync";

const {
  mockClear,
  mockFlush,
  mockMarkDirty,
  mockReconcileLoaded,
  mockSetResume,
  mockSetSaveStatus,
  mockState,
} = vi.hoisted(() => ({
  mockClear: vi.fn(),
  mockFlush: vi.fn(),
  mockMarkDirty: vi.fn(),
  mockReconcileLoaded: vi.fn(),
  mockSetResume: vi.fn(),
  mockSetSaveStatus: vi.fn(),
  mockState: {
    resume: null as ReturnType<typeof createMockResume> | null,
    isDirty: true,
    saveStatus: "error" as const,
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("react-router-dom", () => ({
  useParams: () => ({ id: "resume-1" }),
  useNavigate: () => vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: createMockResume({ title: "Stored on the server" }),
    isLoading: false,
    error: null,
  }),
}));

vi.mock("zustand", async (importOriginal) => ({
  ...(await importOriginal<typeof import("zustand")>()),
  useStore: () => ({ clear: mockClear }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: Object.assign(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        ...mockState,
        setResume: mockSetResume,
        setSaveStatus: mockSetSaveStatus,
        markDirty: mockMarkDirty,
      }),
    { temporal: {} },
  ),
}));

vi.mock("@/features/resume-builder/hooks/useAutoSave", () => ({
  useAutoSave: () => ({ flush: mockFlush, save: mockFlush }),
}));

vi.mock("@/features/resume-builder/lib/resumeDocumentSync", () => ({
  reconcileLoadedResume: mockReconcileLoaded,
  enqueueOutgoingResumeSave: vi.fn(),
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/hooks/useMediaQuery", () => ({
  useMediaQuery: () => true,
}));
vi.mock("@/shared/ui/Tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("@/features/resume-builder/components/preview/PreviewPanel", () => ({
  PreviewPanel: () => null,
}));
vi.mock("@/features/resume-builder/components/PreviewErrorBoundary", () => ({
  PreviewErrorBoundary: ({ children }: { children: React.ReactNode }) =>
    children,
}));
vi.mock("@/features/resume-builder/components/EditorToolbar", () => ({
  EditorToolbar: () => null,
}));
vi.mock("@/features/resume-builder/components/sidebar/DesignSidebar", () => ({
  DesignSidebar: () => null,
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

function loadedResume(
  title: string,
  unsaved: LoadedResume["unsaved"],
): LoadedResume {
  return { resume: createMockResume({ title }), unsaved };
}

describe("ResumeBuilderEditorPage document recovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockState.resume = createMockResume();
    mockFlush.mockResolvedValue(undefined);
    mockReconcileLoaded.mockReturnValue(
      loadedResume("Recovered pending section", "failed"),
    );
  });

  it("opens the reconciled document, not the raw query payload", async () => {
    render(<ResumeBuilderEditorPage />);

    await waitFor(() => expect(mockReconcileLoaded).toHaveBeenCalledTimes(1));
    expect(mockReconcileLoaded).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Stored on the server" }),
    );
    expect(mockSetResume).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Recovered pending section" }),
    );
  });

  // The query payload lacks the work that never reached the server, so opening
  // it as a saved document is how the edits used to disappear for good.
  it("keeps recovered work dirty and reports the failed write", async () => {
    render(<ResumeBuilderEditorPage />);

    await waitFor(() => expect(mockMarkDirty).toHaveBeenCalledTimes(1));
    expect(mockSetSaveStatus).toHaveBeenCalledWith("error");
  });

  it("offers retry for the failed write", async () => {
    const user = userEvent.setup();
    render(<ResumeBuilderEditorPage />);

    await user.click(
      screen.getByRole("button", { name: "resumeBuilder.saveFailed" }),
    );

    expect(mockFlush).toHaveBeenCalledTimes(1);
  });

  it("keeps edits made while the payload was in flight dirty without an error", async () => {
    mockReconcileLoaded.mockReturnValue(
      loadedResume("Typed during the refetch", "pending"),
    );
    render(<ResumeBuilderEditorPage />);

    await waitFor(() => expect(mockMarkDirty).toHaveBeenCalledTimes(1));
    expect(mockSetSaveStatus).not.toHaveBeenCalledWith("error");
  });

  it("leaves a fully stored document clean", async () => {
    mockReconcileLoaded.mockReturnValue(
      loadedResume("Nothing pending", "none"),
    );
    render(<ResumeBuilderEditorPage />);

    await waitFor(() => expect(mockSetResume).toHaveBeenCalledTimes(1));
    expect(mockMarkDirty).not.toHaveBeenCalled();
    expect(mockSetSaveStatus).not.toHaveBeenCalled();
  });
});
