import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import CoverLetterEditorPage from "../CoverLetterEditor";
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
let mockCanUndo = false;
let mockCanRedo = false;
const mockUndo = vi.fn();
const mockRedo = vi.fn();
const mockExportPDFMutate = vi.fn();
const mockExportDOCXMutate = vi.fn();
const mockSetCoverLetter = vi.fn();
const mockUpdateField = vi.fn();
const mockUpdateFields = vi.fn();

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

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockCoverLetter,
    isLoading: false,
    error: null,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/features/cover-letter/hooks/useAutoSaveCoverLetter", () => ({
  useAutoSaveCoverLetter: () => ({ save: vi.fn() }),
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

describe("CoverLetterEditorPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetter = createMockCoverLetter();
    mockCanUndo = false;
    mockCanRedo = false;
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
});
