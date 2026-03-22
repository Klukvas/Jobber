import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import ResumeBuilderEditor from "../ResumeBuilderEditor";
import ResumeBuilderPrint from "../ResumeBuilderPrint";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({ id: "resume-1" }),
  useLocation: () => ({ pathname: "/", search: "" }),
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  Navigate: () => null,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
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

vi.mock("@/shared/ui/Tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// Mock the zustand store with temporal middleware
const mockResume = {
  id: "resume-1",
  title: "Test Resume",
  template_id: "professional",
  font_family: "Inter",
  font_size: 12,
  primary_color: "#000",
  text_color: "#333",
  spacing: 150,
  margin_top: 40,
  margin_bottom: 40,
  margin_left: 40,
  margin_right: 40,
  layout_mode: "single",
  sidebar_width: 35,
  skill_display: "",
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
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: Object.assign(
    (selector: (state: Record<string, unknown>) => unknown) =>
      selector({
        resume: mockResume,
        setResume: vi.fn(),
        activeSection: "contact",
        setActiveSection: vi.fn(),
        saveStatus: "idle",
        setSaveStatus: vi.fn(),
        isDirty: false,
        markDirty: vi.fn(),
        markClean: vi.fn(),
        updateContact: vi.fn(),
        updateSummary: vi.fn(),
        updateDesign: vi.fn(),
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

vi.mock("zustand", async () => {
  const actual = await vi.importActual("zustand");
  return {
    ...actual,
    useStore: () => ({ clear: vi.fn() }),
  };
});

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockResume,
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    getById: vi.fn(),
  },
}));

vi.mock("@/features/resume-builder/hooks/useAutoSave", () => ({
  useAutoSave: vi.fn(),
}));

vi.mock("@/features/resume-builder/hooks/useSectionPersistence", () => ({
  initServerIds: vi.fn(),
}));

vi.mock("@/features/resume-builder/components/preview/PreviewPanel", () => ({
  PreviewPanel: () => <div data-testid="preview-panel" />,
}));

vi.mock(
  "@/features/resume-builder/components/PreviewErrorBoundary",
  () => ({
    PreviewErrorBoundary: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
  }),
);

vi.mock("@/features/resume-builder/components/SaveIndicator", () => ({
  SaveIndicator: () => <div data-testid="save-indicator" />,
}));

vi.mock("@/features/resume-builder/components/EditorToolbar", () => ({
  EditorToolbar: () => <div data-testid="editor-toolbar" />,
}));

vi.mock(
  "@/features/resume-builder/components/sidebar/DesignSidebar",
  () => ({
    DesignSidebar: () => <div data-testid="design-sidebar" />,
  }),
);

vi.mock("@/features/resume-builder/components/AIAssistantPanel", () => ({
  AIAssistantPanel: () => null,
}));

vi.mock("@/features/resume-builder/components/ATSCheckerPanel", () => ({
  ATSCheckerPanel: () => null,
}));

vi.mock("@/features/resume-builder/components/ContentLibraryPanel", () => ({
  ContentLibraryPanel: () => null,
}));

vi.mock("@/features/resume-builder/lib/templateRegistry", () => ({
  TEMPLATE_MAP: {},
}));

vi.mock(
  "@/features/resume-builder/components/preview/ProfessionalTemplate",
  () => ({
    ProfessionalTemplate: () => <div data-testid="professional-template" />,
  }),
);

describe("ResumeBuilderEditor", () => {
  it("renders without crash", () => {
    render(<ResumeBuilderEditor />);
    expect(screen.getByTestId("preview-panel")).toBeInTheDocument();
  });

  it("shows resume title", () => {
    render(<ResumeBuilderEditor />);
    expect(screen.getByText("Test Resume")).toBeInTheDocument();
  });

  it("renders toolbar", () => {
    render(<ResumeBuilderEditor />);
    expect(screen.getByTestId("editor-toolbar")).toBeInTheDocument();
  });

  it("renders design sidebar on desktop", () => {
    render(<ResumeBuilderEditor />);
    expect(screen.getByTestId("design-sidebar")).toBeInTheDocument();
  });

  it("renders save indicator", () => {
    render(<ResumeBuilderEditor />);
    expect(screen.getByTestId("save-indicator")).toBeInTheDocument();
  });

  it("renders back button", () => {
    render(<ResumeBuilderEditor />);
    expect(
      screen.getByLabelText("resumeBuilder.backToList"),
    ).toBeInTheDocument();
  });
});

describe("ResumeBuilderPrint", () => {
  it("renders loading state when no injected data", () => {
    render(<ResumeBuilderPrint />);
    expect(document.getElementById("print-loading")).toBeInTheDocument();
  });
});
