import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CoverLetterTemplatePickerModal } from "../CoverLetterTemplatePickerModal";
import type { CoverLetterDTO } from "@/shared/types/cover-letter";

const mockUpdateField = vi.fn();
const mockCoverLetterRef = vi.hoisted(() => ({
  current: null as CoverLetterDTO | null,
}));

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
      updateField: mockUpdateField,
    }),
}));

vi.mock("../CoverLetterTemplateThumbnail", () => ({
  CoverLetterTemplateThumbnail: ({ templateId }: { templateId: string }) => (
    <div data-testid={`thumbnail-${templateId}`}>Thumbnail</div>
  ),
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

describe("CoverLetterTemplatePickerModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCoverLetterRef.current = createMockCoverLetter();
  });

  it("renders nothing when isOpen is false", () => {
    const { container } = render(
      <CoverLetterTemplatePickerModal isOpen={false} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when coverLetter is null", () => {
    mockCoverLetterRef.current = null;
    const { container } = render(
      <CoverLetterTemplatePickerModal isOpen={true} onClose={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the modal heading when open", () => {
    render(<CoverLetterTemplatePickerModal isOpen={true} onClose={vi.fn()} />);
    expect(screen.getByText("coverLetter.chooseTemplate")).toBeInTheDocument();
  });

  it("renders close button", () => {
    render(<CoverLetterTemplatePickerModal isOpen={true} onClose={vi.fn()} />);
    expect(screen.getByLabelText("common.close")).toBeInTheDocument();
  });

  it("renders template thumbnails", () => {
    render(<CoverLetterTemplatePickerModal isOpen={true} onClose={vi.fn()} />);
    expect(screen.getByTestId("thumbnail-professional")).toBeInTheDocument();
    expect(screen.getByTestId("thumbnail-modern")).toBeInTheDocument();
    expect(screen.getByTestId("thumbnail-minimal")).toBeInTheDocument();
  });

  it("renders template names", () => {
    render(<CoverLetterTemplatePickerModal isOpen={true} onClose={vi.fn()} />);
    expect(
      screen.getByText("coverLetter.templates.professional"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("coverLetter.templates.modern"),
    ).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(<CoverLetterTemplatePickerModal isOpen={true} onClose={onClose} />);

    await user.click(screen.getByLabelText("common.close"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls updateField and onClose when a template is selected", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(<CoverLetterTemplatePickerModal isOpen={true} onClose={onClose} />);

    // Click the modern template button
    await user.click(screen.getByText("coverLetter.templates.modern"));
    expect(mockUpdateField).toHaveBeenCalledWith("template", "modern");
    expect(onClose).toHaveBeenCalledOnce();
  });
});
