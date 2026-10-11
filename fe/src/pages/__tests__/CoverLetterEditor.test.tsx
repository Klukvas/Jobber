import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import CoverLetterEditorPage from "../CoverLetterEditor";
import { ApiError } from "@/services/api";
import { showErrorNotification } from "@/shared/lib/notifications";
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

const mockNavigate = vi.fn();
let mockCoverLetter: CoverLetterDTO | null = createMockCoverLetter();
/** Drives the editor's failure branch from the tests below. */
let mockQueryError: unknown = null;
let mockCanUndo = false;
let mockCanRedo = false;
const mockUndo = vi.fn();
const mockRedo = vi.fn();
const mockExportPDFMutate = vi.fn();
const mockExportDOCXMutate = vi.fn();
const mockFlush = vi.fn();
const mockSetCoverLetter = vi.fn();
const mockUpdateField = vi.fn();
const mockUpdateFields = vi.fn();
const mockMarkDirty = vi.fn();
const mockSetSaveStatus = vi.fn();
// Hoisted: the mock factory below runs while the page module is imported.
const mockReconcileLoaded = vi.hoisted(() => vi.fn());

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useParams: () => ({ id: "cl-1" }),
  useNavigate: () => mockNavigate,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

vi.mock("@/shared/hooks/useMediaQuery", () => ({
  useMediaQuery: () => true,
}));

vi.mock("@/shared/ui/Sheet", () => ({
  Sheet: () => null,
}));

vi.mock("@/services/coverLetterService", () => ({
  coverLetterService: {
    getById: vi.fn().mockResolvedValue(createMockCoverLetter()),
  },
}));

vi.mock("@/stores/coverLetterStore", () => ({
  useCoverLetterStore: Object.assign(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        coverLetter: mockCoverLetter,
        setCoverLetter: mockSetCoverLetter,
        updateField: mockUpdateField,
        updateFields: mockUpdateFields,
        markDirty: mockMarkDirty,
        setSaveStatus: mockSetSaveStatus,
      }),
    {
      temporal: {
        getState: () => ({
          clear: vi.fn(),
        }),
      },
    },
  ),
}));

vi.mock("@/features/cover-letter/lib/coverLetterDocumentSync", () => ({
  reconcileLoadedCoverLetter: mockReconcileLoaded,
  enqueueOutgoingCoverLetterSave: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockCoverLetter,
    isLoading: false,
    error: mockQueryError,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/features/cover-letter/hooks/useAutoSaveCoverLetter", () => ({
  useAutoSaveCoverLetter: () => ({ save: vi.fn(), flush: mockFlush }),
}));

vi.mock("@/features/cover-letter/hooks/useExportCoverLetterPDF", () => ({
  useExportCoverLetterPDF: () => ({
    mutate: mockExportPDFMutate,
    isPending: false,
  }),
}));

vi.mock("@/features/cover-letter/hooks/useExportCoverLetterDOCX", () => ({
  useExportCoverLetterDOCX: () => ({
    mutate: mockExportDOCXMutate,
    isPending: false,
  }),
}));

vi.mock("@/features/cover-letter/hooks/useUndoRedoCoverLetter", () => ({
  useUndoRedoCoverLetter: () => ({
    undo: mockUndo,
    redo: mockRedo,
    canUndo: mockCanUndo,
    canRedo: mockCanRedo,
  }),
}));

vi.mock("@/features/cover-letter/components/CoverLetterPreview", () => ({
  CoverLetterPreview: () => <div data-testid="cover-letter-preview" />,
  CoverLetterFullscreenPreview: () => (
    <div data-testid="cover-letter-fullscreen-preview" />
  ),
}));

vi.mock("@/features/cover-letter/components/CoverLetterSaveIndicator", () => ({
  CoverLetterSaveIndicator: () => (
    <div data-testid="cover-letter-save-indicator" />
  ),
}));

vi.mock("@/features/cover-letter/components/CoverLetterAIPanel", () => ({
  CoverLetterAIPanel: () => <div data-testid="cover-letter-ai-panel" />,
}));

vi.mock(
  "@/features/cover-letter/components/CoverLetterTemplateThumbnail",
  () => ({
    CoverLetterTemplateThumbnail: () => (
      <div data-testid="cover-letter-template-thumbnail" />
    ),
  }),
);

vi.mock(
  "@/features/cover-letter/components/CoverLetterTemplatePickerModal",
  () => ({
    CoverLetterTemplatePickerModal: () => (
      <div data-testid="cover-letter-template-picker" />
    ),
    COVER_LETTER_TEMPLATES: [
      { id: "professional", labelKey: "coverLetter.templates.professional" },
    ],
  }),
);

// The failure branch is opt-in per test; every other test renders the editor.
beforeEach(() => {
  mockQueryError = null;
});

describe("CoverLetterEditorPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetter = createMockCoverLetter();
    mockCanUndo = false;
    mockCanRedo = false;
    mockFlush.mockResolvedValue(undefined);
    mockReconcileLoaded.mockImplementation((letter: CoverLetterDTO) => ({
      coverLetter: letter,
      unsaved: "none",
    }));
  });

  it("renders undo button", () => {
    render(<CoverLetterEditorPage />);

    const undoBtn = screen.getByLabelText("coverLetter.undo");
    expect(undoBtn).toBeInTheDocument();
  });

  it("renders redo button", () => {
    render(<CoverLetterEditorPage />);

    const redoBtn = screen.getByLabelText("coverLetter.redo");
    expect(redoBtn).toBeInTheDocument();
  });

  it("undo button is disabled when canUndo is false", () => {
    mockCanUndo = false;
    render(<CoverLetterEditorPage />);

    const undoBtn = screen.getByLabelText("coverLetter.undo");
    expect(undoBtn).toBeDisabled();
  });

  it("undo button is enabled when canUndo is true", () => {
    mockCanUndo = true;
    render(<CoverLetterEditorPage />);

    const undoBtn = screen.getByLabelText("coverLetter.undo");
    expect(undoBtn).not.toBeDisabled();
  });

  it("redo button is disabled when canRedo is false", () => {
    mockCanRedo = false;
    render(<CoverLetterEditorPage />);

    const redoBtn = screen.getByLabelText("coverLetter.redo");
    expect(redoBtn).toBeDisabled();
  });

  it("renders Export PDF button", () => {
    render(<CoverLetterEditorPage />);

    expect(screen.getByText("coverLetter.exportPDF")).toBeInTheDocument();
  });

  it("renders Export DOCX button", () => {
    render(<CoverLetterEditorPage />);

    expect(screen.getByText("coverLetter.exportDOCX")).toBeInTheDocument();
  });

  it("renders the cover letter title input", () => {
    render(<CoverLetterEditorPage />);

    const titleInput = screen.getByDisplayValue("My Cover Letter");
    expect(titleInput).toBeInTheDocument();
  });

  it("renders preview component", () => {
    render(<CoverLetterEditorPage />);

    expect(screen.getByTestId("cover-letter-preview")).toBeInTheDocument();
  });

  it("shows not-found state when coverLetter is null", () => {
    mockCoverLetter = null;
    render(<CoverLetterEditorPage />);

    expect(screen.getByText("coverLetter.notFound")).toBeInTheDocument();
  });

  // Opening a letter again refetches it. A refetch that fails next to a
  // payload the editor already has must not take the document away — the
  // customer would lose the screen their unsaved work is on.
  it("keeps the open letter when a refetch fails", () => {
    mockQueryError = new ApiError("boom", "INTERNAL_ERROR", 500);
    render(<CoverLetterEditorPage />);

    expect(screen.getByDisplayValue("My Cover Letter")).toBeInTheDocument();
    expect(screen.queryByText("coverLetter.notFound")).not.toBeInTheDocument();
  });

  it("reports a letter the server says is gone, payload or not", () => {
    mockQueryError = new ApiError("nope", "NOT_FOUND", 404);
    render(<CoverLetterEditorPage />);

    expect(screen.getByText("coverLetter.notFound")).toBeInTheDocument();
  });
});

// The export is rendered server-side from the stored letter, so a debounce
// window or a failed save would otherwise produce a file of the old version.
describe("CoverLetterEditorPage export", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetter = createMockCoverLetter();
    mockFlush.mockResolvedValue(undefined);
    mockReconcileLoaded.mockImplementation((letter: CoverLetterDTO) => ({
      coverLetter: letter,
      unsaved: "none",
    }));
  });

  it("stores pending edits before exporting", async () => {
    const user = userEvent.setup();
    let releaseFlush!: () => void;
    mockFlush.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          releaseFlush = resolve;
        }),
    );
    render(<CoverLetterEditorPage />);

    await user.click(screen.getByText("coverLetter.exportPDF"));

    expect(mockFlush).toHaveBeenCalledTimes(1);
    expect(mockExportPDFMutate).not.toHaveBeenCalled();

    releaseFlush();

    await waitFor(() => expect(mockExportPDFMutate).toHaveBeenCalledTimes(1));
  });

  it("refuses to export a letter it could not store", async () => {
    const user = userEvent.setup();
    mockFlush.mockRejectedValue(new Error("offline"));
    render(<CoverLetterEditorPage />);

    await user.click(screen.getByText("coverLetter.exportDOCX"));

    await waitFor(() =>
      expect(showErrorNotification).toHaveBeenCalledWith(
        "coverLetter.saveFailed",
      ),
    );
    expect(mockExportDOCXMutate).not.toHaveBeenCalled();
  });
});

// Opening the fetched letter as a saved document is how unsaved text used to
// disappear: coming back from another letter replaced it with the stored one.
describe("CoverLetterEditorPage document recovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetter = createMockCoverLetter();
    mockFlush.mockResolvedValue(undefined);
  });

  it("opens the recovered letter, keeps it dirty and reports the failed write", async () => {
    mockReconcileLoaded.mockReturnValue({
      coverLetter: createMockCoverLetter({ title: "Never stored" }),
      unsaved: "failed",
    });
    render(<CoverLetterEditorPage />);

    await waitFor(() =>
      expect(mockSetCoverLetter).toHaveBeenCalledWith(
        expect.objectContaining({ title: "Never stored" }),
      ),
    );
    expect(mockMarkDirty).toHaveBeenCalledTimes(1);
    expect(mockSetSaveStatus).toHaveBeenCalledWith("error");
  });

  it("keeps edits typed during the fetch dirty without an error", async () => {
    mockReconcileLoaded.mockReturnValue({
      coverLetter: createMockCoverLetter({ title: "Typed during the fetch" }),
      unsaved: "pending",
    });
    render(<CoverLetterEditorPage />);

    await waitFor(() => expect(mockMarkDirty).toHaveBeenCalledTimes(1));
    expect(mockSetSaveStatus).not.toHaveBeenCalledWith("error");
  });

  it("leaves a fully stored letter clean", async () => {
    mockReconcileLoaded.mockImplementation((letter: CoverLetterDTO) => ({
      coverLetter: letter,
      unsaved: "none",
    }));
    render(<CoverLetterEditorPage />);

    await waitFor(() => expect(mockSetCoverLetter).toHaveBeenCalledTimes(1));
    expect(mockMarkDirty).not.toHaveBeenCalled();
    expect(mockSetSaveStatus).not.toHaveBeenCalled();
  });
});
