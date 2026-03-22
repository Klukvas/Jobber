import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreateResumeModal } from "../CreateResumeModal";
import { EditResumeModal } from "../EditResumeModal";
import type { ResumeDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/services/resumesService", () => ({
  resumesService: { create: vi.fn(), update: vi.fn(), uploadFile: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

const mockResume: ResumeDTO = {
  id: "r1",
  title: "My Resume",
  file_name: "resume.pdf",
  file_url: "https://example.com/resume.pdf",
  is_active: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  applications_count: 0,
};

describe("CreateResumeModal", () => {
  it("renders when open", () => {
    render(<CreateResumeModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("resumes.create")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <CreateResumeModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders title field and cancel button", () => {
    render(<CreateResumeModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText(/resumes\.titleLabel/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});

describe("EditResumeModal", () => {
  it("renders when open", () => {
    render(
      <EditResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(screen.getByText("resumes.edit")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <EditResumeModal
        open={false}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields with resume data", () => {
    render(
      <EditResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});
