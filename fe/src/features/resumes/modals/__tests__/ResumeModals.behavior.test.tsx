import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ResumeDTO } from "@/shared/types/api";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";
import { CreateResumeModal } from "../CreateResumeModal";
import { EditResumeModal } from "../EditResumeModal";
import { DeleteResumeModal } from "../DeleteResumeModal";
import { RenameBuilderResumeModal } from "../RenameBuilderResumeModal";

const mockCreate = vi.hoisted(() => vi.fn());
const mockUpdate = vi.hoisted(() => vi.fn());
const mockDelete = vi.hoisted(() => vi.fn());
const mockUploadResume = vi.hoisted(() => vi.fn());
const mockBuilderUpdate = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());
const mockShowError = vi.hoisted(() => vi.fn());

vi.mock("@/services/resumesService", () => ({
  resumesService: {
    create: mockCreate,
    update: mockUpdate,
    delete: mockDelete,
    uploadResume: mockUploadResume,
  },
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    update: mockBuilderUpdate,
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: mockShowSuccess,
  showErrorNotification: mockShowError,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) =>
      opts?.name ? `${key}:${opts.name}` : key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

function renderWithClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

function makeResume(overrides?: Partial<ResumeDTO>): ResumeDTO {
  return {
    id: "r-1",
    title: "My Resume",
    file_url: "https://example.com/r.pdf",
    storage_type: "external",
    is_active: true,
    can_delete: true,
    applications_count: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...overrides,
  };
}

function makeBuilder(): ResumeBuilderDTO {
  return {
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
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("CreateResumeModal — URL mode", () => {
  it("keeps submit disabled until title and url are provided", async () => {
    const user = userEvent.setup();
    renderWithClient(<CreateResumeModal open={true} onOpenChange={vi.fn()} />);
    const submit = screen.getByRole("button", { name: "common.create" });
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText(/resumes\.titleLabel/), "CV");
    expect(submit).toBeDisabled();

    await user.type(
      screen.getByLabelText(/resumes\.fileUrlLabel/),
      "https://x.test/cv.pdf",
    );
    expect(submit).toBeEnabled();
  });

  it("submits a URL-based create and calls onCreated + success", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onOpenChange = vi.fn();
    const created = makeResume({ id: "new" });
    mockCreate.mockResolvedValue(created);
    renderWithClient(
      <CreateResumeModal
        open={true}
        onOpenChange={onOpenChange}
        onCreated={onCreated}
      />,
    );

    await user.type(screen.getByLabelText(/resumes\.titleLabel/), "CV");
    await user.type(
      screen.getByLabelText(/resumes\.fileUrlLabel/),
      "https://x.test/cv.pdf",
    );
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      title: "CV",
      file_url: "https://x.test/cv.pdf",
      is_active: true,
    });
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("resumes.createSuccess"),
    );
    expect(onCreated).toHaveBeenCalledWith(created);
  });

  it("shows a create error notification on failure", async () => {
    const user = userEvent.setup();
    mockCreate.mockRejectedValue(new Error("create failed"));
    renderWithClient(<CreateResumeModal open={true} onOpenChange={vi.fn()} />);

    await user.type(screen.getByLabelText(/resumes\.titleLabel/), "CV");
    await user.type(
      screen.getByLabelText(/resumes\.fileUrlLabel/),
      "https://x.test/cv.pdf",
    );
    await user.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("create failed"),
    );
  });
});

describe("CreateResumeModal — file mode", () => {
  it("rejects non-PDF files with an error", async () => {
    const user = userEvent.setup();
    renderWithClient(<CreateResumeModal open={true} onOpenChange={vi.fn()} />);
    await user.click(
      screen.getByRole("button", { name: "resumes.uploadPdfOption" }),
    );

    const input = screen.getByLabelText(/resumes\.pdfFileLabel/);
    const badFile = new File(["hi"], "notes.txt", { type: "text/plain" });
    // fireEvent bypasses the input's accept filter so the component's own
    // validation path runs (user-event honors `accept` and drops the file).
    fireEvent.change(input, { target: { files: [badFile] } });

    expect(mockShowError).toHaveBeenCalledWith("resumes.onlyPdfAllowed");
  });

  it("accepts a PDF, shows its name, and uploads on submit", async () => {
    const user = userEvent.setup();
    mockUploadResume.mockResolvedValue(makeResume({ id: "up-1" }));
    mockUpdate.mockResolvedValue(makeResume({ id: "up-1" }));
    const { container } = renderWithClient(
      <CreateResumeModal open={true} onOpenChange={vi.fn()} />,
    );
    await user.click(
      screen.getByRole("button", { name: "resumes.uploadPdfOption" }),
    );

    const input = screen.getByLabelText(/resumes\.pdfFileLabel/);
    const pdf = new File(["%PDF-1.4"], "resume.pdf", {
      type: "application/pdf",
    });
    fireEvent.change(input, { target: { files: [pdf] } });

    // filename shown (auto-filled title / selected file line)
    expect(screen.getByText(/resumes\.selectedFile/)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "resumes.upload" }),
    ).toBeEnabled();

    // Submit the form directly — a file set via fireEvent doesn't reliably
    // submit through a synthetic button click in jsdom.
    fireEvent.submit(container.querySelector("form")!);

    await waitFor(() => expect(mockUploadResume).toHaveBeenCalledTimes(1));
    expect(mockUploadResume.mock.calls[0][0]).toBe(pdf);
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("resumes.uploadSuccess"),
    );
  });
});

describe("EditResumeModal", () => {
  it("prefills fields and submits update with title/is_active/file_url", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockUpdate.mockResolvedValue(makeResume());
    renderWithClient(
      <EditResumeModal
        open={true}
        onOpenChange={onOpenChange}
        resume={makeResume({ title: "Old Title" })}
      />,
    );

    const titleInput = screen.getByLabelText(/resumes\.titleLabel/);
    expect(titleInput).toHaveValue("Old Title");
    await user.clear(titleInput);
    await user.type(titleInput, "New Title");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith("r-1", {
        title: "New Title",
        is_active: true,
        file_url: "https://example.com/r.pdf",
      }),
    );
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("resumes.updateSuccess"),
    );
  });

  it("omits file_url for s3-stored resumes", async () => {
    const user = userEvent.setup();
    mockUpdate.mockResolvedValue(makeResume());
    renderWithClient(
      <EditResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeResume({ storage_type: "s3", file_url: null })}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() => expect(mockUpdate).toHaveBeenCalledTimes(1));
    expect(mockUpdate.mock.calls[0][1]).not.toHaveProperty("file_url");
  });

  it("shows an update error notification", async () => {
    const user = userEvent.setup();
    mockUpdate.mockRejectedValue(new Error("update failed"));
    renderWithClient(
      <EditResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeResume()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("update failed"),
    );
  });
});

describe("DeleteResumeModal", () => {
  it("shows the in-use warning when applications_count > 0", () => {
    renderWithClient(
      <DeleteResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeResume({ applications_count: 3 })}
      />,
    );
    expect(screen.getByText("resumes.inUseWarning")).toBeInTheDocument();
  });

  it("shows the plain delete warning when unused", () => {
    renderWithClient(
      <DeleteResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeResume({ applications_count: 0 })}
      />,
    );
    expect(screen.getByText("resumes.deleteWarning")).toBeInTheDocument();
  });

  it("deletes and closes on confirm", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockDelete.mockResolvedValue(undefined);
    renderWithClient(
      <DeleteResumeModal
        open={true}
        onOpenChange={onOpenChange}
        resume={makeResume()}
      />,
    );

    // the destructive confirm button carries the resumes.delete label
    const buttons = screen.getAllByRole("button", { name: "resumes.delete" });
    await user.click(buttons[buttons.length - 1]);

    await waitFor(() => expect(mockDelete).toHaveBeenCalledWith("r-1"));
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("surfaces a delete error", async () => {
    const user = userEvent.setup();
    mockDelete.mockRejectedValue(new Error("delete failed"));
    renderWithClient(
      <DeleteResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeResume()}
      />,
    );

    const buttons = screen.getAllByRole("button", { name: "resumes.delete" });
    await user.click(buttons[buttons.length - 1]);

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("delete failed"),
    );
  });
});

describe("RenameBuilderResumeModal", () => {
  it("prefills the title and renames on submit", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockBuilderUpdate.mockResolvedValue(makeBuilder());
    renderWithClient(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={onOpenChange}
        resume={makeBuilder()}
      />,
    );

    const input = screen.getByLabelText(/resumes\.titleLabel/);
    expect(input).toHaveValue("Built Resume");
    await user.clear(input);
    await user.type(input, "Renamed Builder");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockBuilderUpdate).toHaveBeenCalledWith("b-1", {
        title: "Renamed Builder",
      }),
    );
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("resumes.updateSuccess"),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not submit when the title is only whitespace", async () => {
    const user = userEvent.setup();
    mockBuilderUpdate.mockResolvedValue(makeBuilder());
    renderWithClient(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeBuilder()}
      />,
    );

    const input = screen.getByLabelText(/resumes\.titleLabel/);
    await user.clear(input);
    await user.type(input, "   ");
    await user.click(screen.getByRole("button", { name: "common.save" }));

    // trimmed empty -> mutation must not fire
    expect(mockBuilderUpdate).not.toHaveBeenCalled();
  });

  it("surfaces a rename error", async () => {
    const user = userEvent.setup();
    mockBuilderUpdate.mockRejectedValue(new Error("rename failed"));
    renderWithClient(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={makeBuilder()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("rename failed"),
    );
  });
});
