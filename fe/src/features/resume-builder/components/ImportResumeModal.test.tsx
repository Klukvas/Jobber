import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ImportResumeModal } from "./ImportResumeModal";

const mockNavigate = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    importFromText: vi.fn().mockResolvedValue({ id: "new-resume-id" }),
    importFromPDF: vi.fn().mockResolvedValue({ id: "new-resume-id" }),
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

describe("ImportResumeModal", () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when open is false", () => {
    const { container } = render(
      <ImportResumeModal open={false} onOpenChange={vi.fn()} />,
    );
    // The Dialog component should not render content when open is false
    expect(screen.queryByText("resumeBuilder.import.title")).not.toBeInTheDocument();
  });

  it("renders the modal title when open", () => {
    render(<ImportResumeModal {...defaultProps} />);
    expect(
      screen.getByText("resumeBuilder.import.title"),
    ).toBeInTheDocument();
  });

  it("renders text and PDF tab buttons", () => {
    render(<ImportResumeModal {...defaultProps} />);
    expect(
      screen.getByText("resumeBuilder.import.pasteText"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.import.uploadPdf"),
    ).toBeInTheDocument();
  });

  it("renders title input", () => {
    render(<ImportResumeModal {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.import.resumeTitle"),
    ).toBeInTheDocument();
  });

  it("renders textarea for text import by default", () => {
    render(<ImportResumeModal {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.import.resumeText"),
    ).toBeInTheDocument();
  });

  it("switches to PDF tab when clicked", async () => {
    const user = userEvent.setup();
    render(<ImportResumeModal {...defaultProps} />);

    await user.click(screen.getByText("resumeBuilder.import.uploadPdf"));

    expect(
      screen.getByText("resumeBuilder.import.dropOrClick"),
    ).toBeInTheDocument();
    expect(
      screen.queryByLabelText("resumeBuilder.import.resumeText"),
    ).not.toBeInTheDocument();
  });

  it("disables import button when text is empty", () => {
    render(<ImportResumeModal {...defaultProps} />);
    const importBtn = screen.getByText("resumeBuilder.import.importButton");
    expect(importBtn).toBeDisabled();
  });

  it("enables import button when text has content", async () => {
    const user = userEvent.setup();
    render(<ImportResumeModal {...defaultProps} />);

    const textarea = screen.getByLabelText("resumeBuilder.import.resumeText");
    await user.type(textarea, "Some resume content");

    const importBtn = screen.getByText("resumeBuilder.import.importButton");
    expect(importBtn).not.toBeDisabled();
  });

  it("renders cancel button", () => {
    render(<ImportResumeModal {...defaultProps} />);
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});
