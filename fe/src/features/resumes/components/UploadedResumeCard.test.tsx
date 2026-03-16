import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UploadedResumeCard } from "./UploadedResumeCard";
import type { ResumeDTO } from "@/shared/types/api";

function createResume(overrides?: Partial<ResumeDTO>): ResumeDTO {
  return {
    id: "r-1",
    title: "My Resume",
    file_url: null,
    storage_type: "s3",
    is_active: true,
    applications_count: 0,
    can_delete: true,
    created_at: "2024-06-01T00:00:00Z",
    updated_at: "2024-06-15T00:00:00Z",
    ...overrides,
  };
}

const mockMutate = vi.fn();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) => {
      if (opts && "count" in opts) return `${key}:${opts.count}`;
      return key;
    },
  }),
}));

vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));

vi.mock("@/services/resumesService", () => ({
  resumesService: { generateDownloadURL: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showErrorNotification: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
}));

describe("UploadedResumeCard", () => {
  const defaultProps = {
    resume: createResume(),
    isMenuOpen: false,
    onToggleMenu: vi.fn(),
    onEdit: vi.fn(),
    onDelete: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders resume title", () => {
    render(<UploadedResumeCard {...defaultProps} />);
    expect(screen.getByText("My Resume")).toBeInTheDocument();
  });

  it("shows active badge when resume is active", () => {
    render(<UploadedResumeCard {...defaultProps} />);
    expect(screen.getByText("common.active")).toBeInTheDocument();
  });

  it("shows inactive badge when resume is inactive", () => {
    render(
      <UploadedResumeCard
        {...defaultProps}
        resume={createResume({ is_active: false })}
      />,
    );
    expect(screen.getByText("common.inactive")).toBeInTheDocument();
  });

  it("shows type badge 'Uploaded'", () => {
    render(<UploadedResumeCard {...defaultProps} />);
    expect(screen.getByText("resumes.typeUploaded")).toBeInTheDocument();
  });

  it("shows cloud storage download button for s3 resumes", () => {
    render(<UploadedResumeCard {...defaultProps} />);
    expect(screen.getByText("resumes.downloadResume")).toBeInTheDocument();
    expect(screen.getByText("resumes.cloudStorage")).toBeInTheDocument();
  });

  it("shows external URL link for url-type resumes", () => {
    render(
      <UploadedResumeCard
        {...defaultProps}
        resume={createResume({
          storage_type: "external",
          file_url: "https://example.com/resume.pdf",
        })}
      />,
    );
    expect(screen.getByText("resumes.viewResume")).toBeInTheDocument();
    expect(screen.getByText("resumes.externalUrl")).toBeInTheDocument();
  });

  it("shows 'no file attached' when no file_url and not s3", () => {
    render(
      <UploadedResumeCard
        {...defaultProps}
        resume={createResume({ storage_type: "external", file_url: null })}
      />,
    );
    expect(screen.getByText("resumes.noFileAttached")).toBeInTheDocument();
  });

  it("shows applications count when > 0", () => {
    render(
      <UploadedResumeCard
        {...defaultProps}
        resume={createResume({ applications_count: 3 })}
      />,
    );
    expect(
      screen.getByText("resumes.usedInApplications:3"),
    ).toBeInTheDocument();
  });

  it("shows 'not used yet' when applications_count is 0", () => {
    render(<UploadedResumeCard {...defaultProps} />);
    expect(screen.getByText("resumes.notUsedYet")).toBeInTheDocument();
  });

  it("calls onToggleMenu when menu button clicked", async () => {
    const user = userEvent.setup();
    const onToggleMenu = vi.fn();
    render(
      <UploadedResumeCard {...defaultProps} onToggleMenu={onToggleMenu} />,
    );

    await user.click(screen.getByLabelText("resumes.actionsMenu"));
    expect(onToggleMenu).toHaveBeenCalledOnce();
  });

  it("shows context menu when isMenuOpen is true", () => {
    render(<UploadedResumeCard {...defaultProps} isMenuOpen={true} />);
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByText("common.edit")).toBeInTheDocument();
    expect(screen.getByText("common.delete")).toBeInTheDocument();
  });

  it("does not show context menu when isMenuOpen is false", () => {
    render(<UploadedResumeCard {...defaultProps} isMenuOpen={false} />);
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("calls onEdit when edit menu item clicked", async () => {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    render(
      <UploadedResumeCard
        {...defaultProps}
        isMenuOpen={true}
        onEdit={onEdit}
      />,
    );

    await user.click(screen.getByText("common.edit"));
    expect(onEdit).toHaveBeenCalledWith(defaultProps.resume);
  });

  it("calls onDelete when delete menu item clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(
      <UploadedResumeCard
        {...defaultProps}
        isMenuOpen={true}
        onDelete={onDelete}
      />,
    );

    await user.click(screen.getByText("common.delete"));
    expect(onDelete).toHaveBeenCalledWith(defaultProps.resume);
  });

  it("disables delete when can_delete is false", () => {
    render(
      <UploadedResumeCard
        {...defaultProps}
        isMenuOpen={true}
        resume={createResume({ can_delete: false })}
      />,
    );

    const deleteBtn = screen.getByText("common.delete").closest("button");
    expect(deleteBtn).toBeDisabled();
  });

  it("has aria-haspopup and aria-expanded on menu trigger", () => {
    render(<UploadedResumeCard {...defaultProps} isMenuOpen={false} />);

    const trigger = screen.getByLabelText("resumes.actionsMenu");
    expect(trigger).toHaveAttribute("aria-haspopup", "menu");
    expect(trigger).toHaveAttribute("aria-expanded", "false");
  });

  it("sets aria-expanded=true when menu is open", () => {
    render(<UploadedResumeCard {...defaultProps} isMenuOpen={true} />);

    const trigger = screen.getByLabelText("resumes.actionsMenu");
    expect(trigger).toHaveAttribute("aria-expanded", "true");
  });
});
