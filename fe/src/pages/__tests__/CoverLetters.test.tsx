import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import CoverLettersPage from "../CoverLetters";
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
    company_name: "",
    company_address: "",
    greeting: "Dear Jane Smith,",
    paragraphs: ["First paragraph"],
    closing: "Sincerely,",
    font_family: "Georgia",
    font_size: 12,
    primary_color: "#2563eb",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-06-15T10:30:00Z",
    ...overrides,
  };
}

const mockNavigate = vi.fn();
const mockInvalidateQueries = vi.fn();
const mockCreateMutate = vi.fn();
const mockDeleteMutate = vi.fn();
const mockDuplicateMutate = vi.fn();

let mockCoverLetters: CoverLetterDTO[] = [];
let mockIsLoading = false;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
  getI18n: () => ({ language: "en" }),
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

vi.mock("@/services/coverLetterService", () => ({
  coverLetterService: {
    list: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
    duplicate: vi.fn(),
  },
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

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockCoverLetters,
    isLoading: mockIsLoading,
  }),
  useMutation: (opts: { mutationFn: (...args: unknown[]) => unknown }) => {
    // Distinguish create vs delete vs duplicate by inspecting mutationFn
    const fnStr = opts.mutationFn.toString();
    if (fnStr.includes("duplicate")) {
      return {
        mutate: mockDuplicateMutate,
        isPending: false,
      };
    }
    if (fnStr.includes("delete")) {
      return {
        mutate: mockDeleteMutate,
        isPending: false,
      };
    }
    return {
      mutate: mockCreateMutate,
      isPending: false,
    };
  },
  useQueryClient: () => ({
    invalidateQueries: mockInvalidateQueries,
  }),
}));

describe("CoverLettersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetters = [];
    mockIsLoading = false;
  });

  it("renders list of cover letters", () => {
    mockCoverLetters = [
      createMockCoverLetter({ id: "cl-1", title: "Letter A" }),
      createMockCoverLetter({ id: "cl-2", title: "Letter B" }),
    ];

    render(<CoverLettersPage />);

    expect(screen.getByText("Letter A")).toBeInTheDocument();
    expect(screen.getByText("Letter B")).toBeInTheDocument();
  });

  it("shows empty state when no cover letters exist", () => {
    mockCoverLetters = [];

    render(<CoverLettersPage />);

    expect(screen.getByText("coverLetter.empty")).toBeInTheDocument();
  });

  it("shows company_name on card when available", () => {
    mockCoverLetters = [
      createMockCoverLetter({
        id: "cl-1",
        title: "Letter A",
        company_name: "Acme Corp",
      }),
    ];

    render(<CoverLettersPage />);

    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
  });

  it("does not show company_name when empty", () => {
    mockCoverLetters = [
      createMockCoverLetter({
        id: "cl-1",
        title: "Letter A",
        company_name: "",
      }),
    ];

    render(<CoverLettersPage />);

    // The company_name paragraph should not be rendered
    expect(screen.queryByText("Acme Corp")).not.toBeInTheDocument();
  });

  it("duplicate button appears on card and calls duplicate", async () => {
    const user = userEvent.setup();
    mockCoverLetters = [
      createMockCoverLetter({ id: "cl-1", title: "Letter A" }),
    ];

    render(<CoverLettersPage />);

    const duplicateBtn = screen.getByLabelText("common.duplicate");
    await user.click(duplicateBtn);

    expect(mockDuplicateMutate).toHaveBeenCalledWith("cl-1");
  });

  it("delete confirmation dialog shows on delete click", async () => {
    const user = userEvent.setup();
    mockCoverLetters = [
      createMockCoverLetter({ id: "cl-1", title: "Letter A" }),
    ];

    render(<CoverLettersPage />);

    const deleteBtn = screen.getByLabelText("common.delete");
    await user.click(deleteBtn);

    expect(
      screen.getByText("coverLetter.deleteConfirmTitle"),
    ).toBeInTheDocument();
  });

  it("renders create button", () => {
    render(<CoverLettersPage />);

    expect(screen.getByText("coverLetter.create")).toBeInTheDocument();
  });

  it("renders page title", () => {
    render(<CoverLettersPage />);

    expect(screen.getByText("coverLetter.title")).toBeInTheDocument();
  });
});
