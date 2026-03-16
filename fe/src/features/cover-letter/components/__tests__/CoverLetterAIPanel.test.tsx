import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CoverLetterAIPanel } from "../CoverLetterAIPanel";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

const {
  mockUpdateFields,
  mockGenerateMutate,
  mockGenerateReset,
  mockCoverLetterRef,
  mockGenerateStateRef,
  mockJobsList,
} = vi.hoisted(() => {
  const mockUpdateFields = vi.fn();
  const mockGenerateMutate = vi.fn();
  const mockGenerateReset = vi.fn();

  const mockJobsList = [
    {
      id: "job-1",
      title: "Software Engineer",
      company_name: "Acme Corp",
      description: "Build awesome software",
    },
    {
      id: "job-2",
      title: "Product Manager",
      company_name: "Beta Inc",
      description: "Lead product",
    },
  ];

  const mockCoverLetterRef = { current: null as CoverLetterDTO | null };
  const mockGenerateStateRef = {
    current: {
      isPending: false,
      isError: false,
      data: null as null | {
        greeting: string;
        paragraphs: string[];
        closing: string;
      },
      mutate: mockGenerateMutate,
      reset: mockGenerateReset,
    },
  };

  return {
    mockUpdateFields,
    mockGenerateMutate,
    mockGenerateReset,
    mockCoverLetterRef,
    mockGenerateStateRef,
    mockJobsList,
  };
});

function createMockCoverLetter(
  overrides?: Partial<CoverLetterDTO>,
): CoverLetterDTO {
  return {
    id: "cl-1",
    resume_builder_id: null,
    job_id: null,
    title: "My Cover Letter",
    template: "professional",
    recipient_name: "",
    recipient_title: "",
    company_name: "",
    company_address: "",
    greeting: "Dear Hiring Manager,",
    paragraphs: [""],
    closing: "Sincerely,",
    font_family: "Georgia",
    font_size: 12,
    primary_color: "#2563eb",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/coverLetterStore", () => ({
  useCoverLetterStore: (
    selector: (state: Record<string, unknown>) => unknown,
  ) =>
    selector({
      coverLetter: mockCoverLetterRef.current,
      updateFields: mockUpdateFields,
    }),
}));

vi.mock("../hooks/useCoverLetterAI", () => ({
  useCoverLetterAI: () => ({ generate: mockGenerateStateRef.current }),
}));

vi.mock("@/services/jobsService", () => ({
  jobsService: {
    list: vi.fn().mockResolvedValue({ items: mockJobsList }),
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: { items: mockJobsList },
    isLoading: false,
  }),
  useMutation: () => mockGenerateStateRef.current,
}));

describe("CoverLetterAIPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetterRef.current = createMockCoverLetter();
    mockGenerateStateRef.current = {
      isPending: false,
      isError: false,
      data: null,
      mutate: mockGenerateMutate,
      reset: mockGenerateReset,
    };
  });

  it("renders the generate button", () => {
    render(<CoverLetterAIPanel />);

    expect(
      screen.getByText("coverLetter.ai.generate"),
    ).toBeInTheDocument();
  });

  it("renders the AI title heading", () => {
    render(<CoverLetterAIPanel />);

    expect(screen.getByText("coverLetter.ai.title")).toBeInTheDocument();
  });

  it("renders job description textarea", () => {
    render(<CoverLetterAIPanel />);

    expect(
      screen.getByPlaceholderText(
        "coverLetter.ai.jobDescriptionPlaceholder",
      ),
    ).toBeInTheDocument();
  });

  it("when a job is selected, updateFields is called with job_id", async () => {
    const user = userEvent.setup();
    render(<CoverLetterAIPanel />);

    const searchInput = screen.getByPlaceholderText(
      "coverLetter.ai.selectJobPlaceholder",
    );
    await user.click(searchInput);

    const jobOption = screen.getByText("Software Engineer (Acme Corp)");
    await user.click(jobOption);

    expect(mockUpdateFields).toHaveBeenCalledWith(
      expect.objectContaining({ job_id: "job-1" }),
    );
  });

  it("when job is selected and cover letter has no company_name, updates company_name too", async () => {
    const user = userEvent.setup();
    mockCoverLetterRef.current = createMockCoverLetter({ company_name: "" });
    render(<CoverLetterAIPanel />);

    const searchInput = screen.getByPlaceholderText(
      "coverLetter.ai.selectJobPlaceholder",
    );
    await user.click(searchInput);

    const jobOption = screen.getByText("Software Engineer (Acme Corp)");
    await user.click(jobOption);

    expect(mockUpdateFields).toHaveBeenCalledWith(
      expect.objectContaining({
        job_id: "job-1",
        company_name: "Acme Corp",
      }),
    );
  });

  it("calls generate mutate when generate button is clicked", async () => {
    const user = userEvent.setup();
    render(<CoverLetterAIPanel />);

    const generateBtn = screen.getByText("coverLetter.ai.generate");
    await user.click(generateBtn);

    expect(mockGenerateMutate).toHaveBeenCalledWith({
      cover_letter_id: "cl-1",
      job_description: undefined,
    });
  });
});
