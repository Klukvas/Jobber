import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ImportResumeModal } from "./ImportResumeModal";
import { ApiError } from "@/services/api";
import { resumeBuilderService } from "@/services/resumeBuilderService";
import { showErrorNotification } from "@/shared/lib/notifications";

const mockNavigate = vi.fn();
const mockInvalidate = vi.fn();

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: mockInvalidate }),
}));

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
    render(<ImportResumeModal open={false} onOpenChange={vi.fn()} />);
    // The Dialog component should not render content when open is false
    expect(
      screen.queryByText("resumeBuilder.import.title"),
    ).not.toBeInTheDocument();
  });

  it("renders the modal title when open", () => {
    render(<ImportResumeModal {...defaultProps} />);
    expect(screen.getByText("resumeBuilder.import.title")).toBeInTheDocument();
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

  it("refreshes the subscription limits after a successful import", async () => {
    const user = userEvent.setup();
    render(<ImportResumeModal {...defaultProps} />);

    await user.type(
      screen.getByLabelText("resumeBuilder.import.resumeText"),
      "Some resume content",
    );
    await user.click(screen.getByText("resumeBuilder.import.importButton"));

    await waitFor(() =>
      expect(mockInvalidate).toHaveBeenCalledWith({
        queryKey: ["subscription"],
      }),
    );
    expect(mockNavigate).toHaveBeenCalledWith("/app/resume-builder/new-resume-id");
  });
});

for (const error of [new Error(""), new ApiError("", "UNKNOWN_ERROR", 404)]) {
  it(`preserves entered text and explains ${error instanceof ApiError ? "unavailable AI import" : "an empty transport message"}`, async () => {
    vi.mocked(resumeBuilderService.importFromText).mockRejectedValueOnce(error);
    const close = vi.fn();
    render(<ImportResumeModal open onOpenChange={close} />);
    await userEvent.type(
      screen.getByLabelText("resumeBuilder.import.resumeText"),
      "Synthetic engineer",
    );
    await userEvent.click(
      screen.getByText("resumeBuilder.import.importButton"),
    );
    await waitFor(() =>
      expect(showErrorNotification).toHaveBeenCalledWith(
        error instanceof ApiError
          ? "resumeBuilder.import.unavailable"
          : "resumeBuilder.import.error",
      ),
    );
    expect(
      screen.getByLabelText("resumeBuilder.import.resumeText"),
    ).toHaveValue("Synthetic engineer");
    expect(close).not.toHaveBeenCalled();
  });
}
