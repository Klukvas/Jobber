import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { EditorToolbar } from "../EditorToolbar";
import { createMockStoreState } from "./testHelpers";

const mockState = createMockStoreState();
const mockExportPDFMutate = vi.hoisted(() => vi.fn());
const mockExportDOCXMutate = vi.hoisted(() => vi.fn());
let pdfPending = false;
let docxPending = false;
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

vi.mock("../../hooks/useExportPDF", () => ({
  useExportPDF: () => ({ mutate: mockExportPDFMutate, isPending: pdfPending }),
}));
vi.mock("../../hooks/useExportDOCX", () => ({
  useExportDOCX: () => ({
    mutate: mockExportDOCXMutate,
    isPending: docxPending,
  }),
}));
vi.mock("../../hooks/useUndoRedo", () => ({
  useUndoRedo: () => ({
    undo: vi.fn(),
    redo: vi.fn(),
    canUndo: true,
    canRedo: true,
  }),
}));
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: mockShowSuccess,
  showErrorNotification: mockShowError,
}));

const defaultProps = {
  showAI: false,
  onToggleAI: vi.fn(),
  showATS: false,
  onToggleATS: vi.fn(),
  showContentLibrary: false,
  onToggleContentLibrary: vi.fn(),
};

let clickSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  vi.clearAllMocks();
  Object.assign(mockState, createMockStoreState());
  pdfPending = false;
  docxPending = false;
  // jsdom's anchor.click() is a no-op but throws if it navigates; spy to be safe
  clickSpy = vi
    .spyOn(HTMLAnchorElement.prototype, "click")
    .mockImplementation(() => {});
  // Object URL helpers are absent in jsdom
  Object.defineProperty(URL, "createObjectURL", {
    value: vi.fn(() => "blob:mock"),
    configurable: true,
  });
  Object.defineProperty(URL, "revokeObjectURL", {
    value: vi.fn(),
    configurable: true,
  });
});

afterEach(() => {
  clickSpy.mockRestore();
});

function getExportToggle() {
  // The export toggle carries the export label and a chevron.
  return screen
    .getAllByRole("button")
    .find(
      (b) =>
        b.textContent?.includes("resumeBuilder.toolbar.export") &&
        !b.textContent?.includes("Pdf") &&
        !b.textContent?.includes("Docx"),
    ) as HTMLButtonElement;
}

describe("EditorToolbar export flow", () => {
  it("opens the export dropdown with PDF and DOCX options", async () => {
    const user = userEvent.setup();
    render(<EditorToolbar {...defaultProps} />);

    await user.click(getExportToggle());

    expect(
      screen.getByText("resumeBuilder.toolbar.exportPdf"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.toolbar.exportDocx"),
    ).toBeInTheDocument();
  });

  it("exports as PDF, triggers a download and a success toast", async () => {
    const user = userEvent.setup();
    // mutate immediately invokes the success callback with a blob
    mockExportPDFMutate.mockImplementation((_id, opts) => {
      opts.onSuccess(new Blob(["pdf"], { type: "application/pdf" }));
    });
    render(<EditorToolbar {...defaultProps} />);

    await user.click(getExportToggle());
    await user.click(screen.getByText("resumeBuilder.toolbar.exportPdf"));

    expect(mockExportPDFMutate).toHaveBeenCalledTimes(1);
    expect(mockExportPDFMutate.mock.calls[0][0]).toBe("resume-1");
    expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
    expect(clickSpy).toHaveBeenCalledTimes(1);
    expect(mockShowSuccess).toHaveBeenCalledWith(
      "resumeBuilder.toolbar.exportSuccess",
    );
  });

  it("shows an error toast when PDF export fails", async () => {
    const user = userEvent.setup();
    mockExportPDFMutate.mockImplementation((_id, opts) => {
      opts.onError(new Error("pdf boom"));
    });
    render(<EditorToolbar {...defaultProps} />);

    await user.click(getExportToggle());
    await user.click(screen.getByText("resumeBuilder.toolbar.exportPdf"));

    expect(mockShowError).toHaveBeenCalledWith(
      "resumeBuilder.toolbar.exportError",
    );
  });

  it("exports as DOCX with a success toast", async () => {
    const user = userEvent.setup();
    mockExportDOCXMutate.mockImplementation((_id, opts) => {
      opts.onSuccess(new Blob(["docx"]));
    });
    render(<EditorToolbar {...defaultProps} />);

    await user.click(getExportToggle());
    await user.click(screen.getByText("resumeBuilder.toolbar.exportDocx"));

    expect(mockExportDOCXMutate).toHaveBeenCalledTimes(1);
    expect(mockShowSuccess).toHaveBeenCalledWith(
      "resumeBuilder.toolbar.exportDocxSuccess",
    );
  });

  it("shows an error toast when DOCX export fails", async () => {
    const user = userEvent.setup();
    mockExportDOCXMutate.mockImplementation((_id, opts) => {
      opts.onError(new Error("docx boom"));
    });
    render(<EditorToolbar {...defaultProps} />);

    await user.click(getExportToggle());
    await user.click(screen.getByText("resumeBuilder.toolbar.exportDocx"));

    expect(mockShowError).toHaveBeenCalledWith(
      "resumeBuilder.toolbar.exportDocxError",
    );
  });

  it("disables the export toggle and shows the exporting state while pending", () => {
    pdfPending = true;
    render(<EditorToolbar {...defaultProps} />);
    const toggle = screen
      .getAllByRole("button")
      .find((b) => b.textContent?.includes("resumeBuilder.toolbar.exporting"));
    expect(toggle).toBeDefined();
    expect(toggle).toBeDisabled();
  });

  it("does not export when there is no resume in the store", async () => {
    const user = userEvent.setup();
    // @ts-expect-error deliberately clearing resume
    mockState.resume = null;
    render(<EditorToolbar {...defaultProps} />);

    // export toggle is disabled without a resume
    const toggle = getExportToggle();
    expect(toggle).toBeDisabled();
    // even if forced, the handler guards on resume
    await user.click(toggle).catch(() => {});
    expect(mockExportPDFMutate).not.toHaveBeenCalled();
  });
});
