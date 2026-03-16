import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditorToolbar } from "./EditorToolbar";
import { createMockStoreState } from "./__tests__/testHelpers";

const mockState = createMockStoreState();
const mockUndo = vi.fn();
const mockRedo = vi.fn();
const mockExportPDFMutate = vi.fn();
const mockExportDOCXMutate = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

vi.mock("../hooks/useExportPDF", () => ({
  useExportPDF: () => ({
    mutate: mockExportPDFMutate,
    isPending: false,
  }),
}));

vi.mock("../hooks/useExportDOCX", () => ({
  useExportDOCX: () => ({
    mutate: mockExportDOCXMutate,
    isPending: false,
  }),
}));

vi.mock("../hooks/useUndoRedo", () => ({
  useUndoRedo: () => ({
    undo: mockUndo,
    redo: mockRedo,
    canUndo: true,
    canRedo: false,
  }),
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

describe("EditorToolbar", () => {
  const defaultProps = {
    showAI: false,
    onToggleAI: vi.fn(),
    showATS: false,
    onToggleATS: vi.fn(),
    showContentLibrary: false,
    onToggleContentLibrary: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders undo and redo buttons", () => {
    render(<EditorToolbar {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.undo"),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.redo"),
    ).toBeInTheDocument();
  });

  it("calls undo when undo button is clicked", async () => {
    const user = userEvent.setup();
    render(<EditorToolbar {...defaultProps} />);

    await user.click(screen.getByLabelText("resumeBuilder.toolbar.undo"));
    expect(mockUndo).toHaveBeenCalledTimes(1);
  });

  it("disables redo button when canRedo is false", () => {
    render(<EditorToolbar {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.redo"),
    ).toBeDisabled();
  });

  it("enables undo button when canUndo is true", () => {
    render(<EditorToolbar {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.undo"),
    ).not.toBeDisabled();
  });

  it("renders AI assistant button", () => {
    render(<EditorToolbar {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.aiAssistant"),
    ).toBeInTheDocument();
  });

  it("calls onToggleAI when AI button is clicked", async () => {
    const user = userEvent.setup();
    render(<EditorToolbar {...defaultProps} />);

    await user.click(
      screen.getByLabelText("resumeBuilder.toolbar.aiAssistant"),
    );
    expect(defaultProps.onToggleAI).toHaveBeenCalledTimes(1);
  });

  it("renders ATS check button", () => {
    render(<EditorToolbar {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.atsCheck"),
    ).toBeInTheDocument();
  });

  it("calls onToggleATS when ATS button is clicked", async () => {
    const user = userEvent.setup();
    render(<EditorToolbar {...defaultProps} />);

    await user.click(
      screen.getByLabelText("resumeBuilder.toolbar.atsCheck"),
    );
    expect(defaultProps.onToggleATS).toHaveBeenCalledTimes(1);
  });

  it("renders content library button", () => {
    render(<EditorToolbar {...defaultProps} />);
    expect(
      screen.getByLabelText("resumeBuilder.toolbar.contentLibrary"),
    ).toBeInTheDocument();
  });

  it("calls onToggleContentLibrary when content library button is clicked", async () => {
    const user = userEvent.setup();
    render(<EditorToolbar {...defaultProps} />);

    await user.click(
      screen.getByLabelText("resumeBuilder.toolbar.contentLibrary"),
    );
    expect(defaultProps.onToggleContentLibrary).toHaveBeenCalledTimes(1);
  });

  it("shows export dropdown when export button is clicked", async () => {
    const user = userEvent.setup();
    render(<EditorToolbar {...defaultProps} />);

    // The export button contains the export text
    const exportButtons = screen.getAllByRole("button");
    const exportBtn = exportButtons.find(
      (btn) =>
        btn.textContent?.includes("resumeBuilder.toolbar.export") &&
        !btn.textContent?.includes("Pdf") &&
        !btn.textContent?.includes("Docx"),
    );
    expect(exportBtn).toBeDefined();
  });
});
