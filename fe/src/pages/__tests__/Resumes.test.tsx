import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ResumesPage from "../Resumes";
import type { UnifiedResumeItem } from "@/features/resumes/hooks/useUnifiedResumes";
import type { ResumeDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

// --- factories ---

function createUploadedItem(overrides?: Partial<ResumeDTO>): UnifiedResumeItem {
  return {
    kind: "uploaded",
    data: {
      id: "u-1",
      title: "Uploaded Resume",
      file_url: null,
      storage_type: "s3",
      is_active: true,
      applications_count: 0,
      can_delete: true,
      created_at: "2024-06-01T00:00:00Z",
      updated_at: "2024-06-15T00:00:00Z",
      ...overrides,
    },
  };
}

function createBuiltItem(
  overrides?: Partial<ResumeBuilderDTO>,
): UnifiedResumeItem {
  return {
    kind: "built",
    data: {
      id: "b-1",
      title: "Built Resume",
      template_id: "professional",
      font_family: "Inter",
      primary_color: "#000",
      text_color: "#333",
      spacing: 150,
      margin_top: 40,
      margin_bottom: 40,
      margin_left: 40,
      margin_right: 40,
      layout_mode: "single",
      sidebar_width: 35,
      font_size: 12,
      skill_display: "",
      created_at: "2024-07-01T00:00:00Z",
      updated_at: "2024-07-10T00:00:00Z",
      ...overrides,
    },
  };
}

// --- mock state ---

let mockItems: UnifiedResumeItem[] = [];
let mockIsLoading = false;
let mockIsError = false;
const mockSetKindFilter = vi.fn();
const mockToggleSort = vi.fn();
const mockRefetch = vi.fn();
const mockNavigate = vi.fn();
const mockCreateMutate = vi.fn();
const mockDuplicateMutate = vi.fn();
const mockDeleteMutate = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    canCreate: () => true,
  }),
}));

vi.mock("@/features/subscription/components/UpgradeBanner", () => ({
  UpgradeBanner: () => null,
}));

vi.mock("@/services/api", () => ({
  ApiError: class ApiError extends Error {
    code: string;
    status: number;
    constructor(message: string, code: string, status: number) {
      super(message);
      this.code = code;
      this.status = status;
    }
  },
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    create: vi.fn(),
    duplicate: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/features/resumes/hooks/useUnifiedResumes", () => ({
  useUnifiedResumes: () => ({
    items: mockItems,
    isLoading: mockIsLoading,
    isError: mockIsError,
    error: mockIsError ? new Error("test error") : null,
    refetch: mockRefetch,
    kindFilter: "all" as const,
    setKindFilter: mockSetKindFilter,
    sortBy: "updated_at" as const,
    sortDir: "desc" as const,
    toggleSort: mockToggleSort,
  }),
}));

vi.mock("@/features/resumes/modals/CreateResumeModal", () => ({
  CreateResumeModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="create-resume-modal" /> : null,
}));

vi.mock("@/features/resumes/modals/EditResumeModal", () => ({
  EditResumeModal: () => null,
}));

vi.mock("@/features/resumes/modals/DeleteResumeModal", () => ({
  DeleteResumeModal: () => null,
}));

vi.mock("@/features/resume-builder/components/ImportResumeModal", () => ({
  ImportResumeModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="import-resume-modal" /> : null,
}));

vi.mock("@/features/resumes/components/UploadedResumeCard", () => ({
  UploadedResumeCard: ({ resume }: { resume: ResumeDTO }) => (
    <div data-testid={`uploaded-card-${resume.id}`}>{resume.title}</div>
  ),
}));

vi.mock("@/features/resumes/components/BuilderResumeCard", () => ({
  BuilderResumeCard: ({ resume }: { resume: ResumeBuilderDTO }) => (
    <div data-testid={`builder-card-${resume.id}`}>{resume.title}</div>
  ),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: (opts: { mutationFn: (...args: unknown[]) => unknown }) => {
    const fnStr = opts.mutationFn.toString();
    if (fnStr.includes("duplicate")) {
      return { mutate: mockDuplicateMutate, isPending: false };
    }
    if (fnStr.includes("delete")) {
      return { mutate: mockDeleteMutate, isPending: false };
    }
    return { mutate: mockCreateMutate, isPending: false };
  },
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

describe("Resumes Page", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockItems = [];
    mockIsLoading = false;
    mockIsError = false;
  });

  it("renders page title", () => {
    render(<ResumesPage />);
    expect(screen.getByText("resumes.title")).toBeInTheDocument();
  });

  it("shows empty state when no items and filter is all", () => {
    mockItems = [];
    render(<ResumesPage />);
    expect(screen.getByText("resumes.noResumes")).toBeInTheDocument();
  });

  it("renders uploaded resume cards", () => {
    mockItems = [createUploadedItem({ id: "u-1", title: "My Upload" })];
    render(<ResumesPage />);
    expect(screen.getByTestId("uploaded-card-u-1")).toBeInTheDocument();
    expect(screen.getByText("My Upload")).toBeInTheDocument();
  });

  it("renders builder resume cards", () => {
    mockItems = [createBuiltItem({ id: "b-1", title: "My Builder" })];
    render(<ResumesPage />);
    expect(screen.getByTestId("builder-card-b-1")).toBeInTheDocument();
    expect(screen.getByText("My Builder")).toBeInTheDocument();
  });

  it("renders both card types in unified grid", () => {
    mockItems = [
      createUploadedItem({ id: "u-1" }),
      createBuiltItem({ id: "b-1" }),
    ];
    render(<ResumesPage />);
    expect(screen.getByTestId("uploaded-card-u-1")).toBeInTheDocument();
    expect(screen.getByTestId("builder-card-b-1")).toBeInTheDocument();
  });

  it("renders filter buttons", () => {
    mockItems = [createUploadedItem()];
    render(<ResumesPage />);
    expect(screen.getByText("resumes.filterAll")).toBeInTheDocument();
    expect(screen.getByText("resumes.filterUploaded")).toBeInTheDocument();
    expect(screen.getByText("resumes.filterBuilt")).toBeInTheDocument();
  });

  it("renders sort buttons", () => {
    mockItems = [createUploadedItem()];
    render(<ResumesPage />);
    expect(screen.getByText("resumes.sortLastModified")).toBeInTheDocument();
    expect(screen.getByText("resumes.sortCreatedDate")).toBeInTheDocument();
    expect(screen.getByText("resumes.sortTitle")).toBeInTheDocument();
  });

  it("shows create dropdown with three options when clicked", async () => {
    const user = userEvent.setup();
    mockItems = [createUploadedItem()];
    render(<ResumesPage />);

    await user.click(screen.getByText("resumes.create"));

    expect(screen.getByText("resumes.uploadResume")).toBeInTheDocument();
    expect(screen.getByText("resumes.buildResume")).toBeInTheDocument();
    expect(screen.getByText("resumes.importResume")).toBeInTheDocument();
  });

  it("shows error state when isError", () => {
    mockIsError = true;
    render(<ResumesPage />);
    expect(screen.getByText("common.tryAgain")).toBeInTheDocument();
  });

  it("shows loading skeleton when isLoading", () => {
    mockIsLoading = true;
    render(<ResumesPage />);
    // Loading renders title + SkeletonList, but no cards or empty state
    expect(screen.getByText("resumes.title")).toBeInTheDocument();
    expect(screen.queryByText("resumes.noResumes")).not.toBeInTheDocument();
  });
});
