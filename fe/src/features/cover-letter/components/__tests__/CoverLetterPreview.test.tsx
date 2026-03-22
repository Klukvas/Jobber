import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  CoverLetterPreview,
  CoverLetterFullscreenPreview,
} from "../CoverLetterPreview";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

// ResizeObserver is not available in jsdom
beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
});

const mockCoverLetterRef = {
  current: null as CoverLetterDTO | null,
};

const mockUpdateFieldFn = vi.fn();
const mockUpdateParagraphFn = vi.fn();
const mockAddParagraphFn = vi.fn();
const mockRemoveParagraphFn = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
  getI18n: () => ({ language: "en" }),
}));

vi.mock("@/stores/coverLetterStore", () => ({
  useCoverLetterStore: (
    selector: (state: Record<string, unknown>) => unknown,
  ) =>
    selector({
      coverLetter: mockCoverLetterRef.current,
      updateField: mockUpdateFieldFn,
      updateParagraph: mockUpdateParagraphFn,
      addParagraph: mockAddParagraphFn,
      removeParagraph: mockRemoveParagraphFn,
      saveStatus: "idle",
    }),
}));

// Mock the inline editing components to simplify rendering
vi.mock("@/features/resume-builder/components/inline/EditableField", () => ({
  EditableField: ({
    value,
    placeholder,
  }: {
    value: string;
    placeholder?: string;
  }) => <span data-testid="editable-field">{value || placeholder}</span>,
}));

vi.mock("@/features/resume-builder/components/inline/EditableTextarea", () => ({
  EditableTextarea: ({
    value,
    placeholder,
  }: {
    value: string;
    placeholder?: string;
  }) => <span data-testid="editable-textarea">{value || placeholder}</span>,
}));

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
    greeting: "Dear Hiring Manager,",
    paragraphs: ["First paragraph content", "Second paragraph content"],
    closing: "Sincerely,",
    font_family: "Georgia",
    font_size: 12,
    primary_color: "#2563eb",
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
    ...overrides,
  };
}

describe("CoverLetterPreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetterRef.current = createMockCoverLetter();
  });

  it("renders nothing when coverLetter is null", () => {
    mockCoverLetterRef.current = null;
    const { container } = render(<CoverLetterPreview />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the preview container when coverLetter is provided", () => {
    render(<CoverLetterPreview />);
    // The preview renders editable fields for various parts of the cover letter
    const fields = screen.getAllByTestId("editable-field");
    expect(fields.length).toBeGreaterThan(0);
  });

  it("renders greeting text", () => {
    render(<CoverLetterPreview />);
    expect(screen.getByText("Dear Hiring Manager,")).toBeInTheDocument();
  });

  it("renders paragraph content", () => {
    render(<CoverLetterPreview />);
    expect(screen.getByText("First paragraph content")).toBeInTheDocument();
    expect(screen.getByText("Second paragraph content")).toBeInTheDocument();
  });

  it("renders with editable=false by default (no add paragraph button)", () => {
    render(<CoverLetterPreview />);
    expect(
      screen.queryByText("coverLetter.addParagraph"),
    ).not.toBeInTheDocument();
  });

  it("renders with editable=true showing add paragraph button", () => {
    render(<CoverLetterPreview editable />);
    expect(screen.getByText("coverLetter.addParagraph")).toBeInTheDocument();
  });
});

describe("CoverLetterFullscreenPreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetterRef.current = createMockCoverLetter();
  });

  it("renders nothing when open is false", () => {
    const { container } = render(
      <CoverLetterFullscreenPreview open={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when coverLetter is null", () => {
    mockCoverLetterRef.current = null;
    const { container } = render(
      <CoverLetterFullscreenPreview open={true} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the close button when open with cover letter", () => {
    render(<CoverLetterFullscreenPreview open={true} onClose={vi.fn()} />);
    expect(screen.getByLabelText("common.close")).toBeInTheDocument();
  });

  it("renders cover letter content when open", () => {
    render(<CoverLetterFullscreenPreview open={true} onClose={vi.fn()} />);
    expect(screen.getByText("Dear Hiring Manager,")).toBeInTheDocument();
  });
});
