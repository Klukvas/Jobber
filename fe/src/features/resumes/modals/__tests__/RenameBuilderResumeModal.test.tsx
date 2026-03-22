import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { RenameBuilderResumeModal } from "../RenameBuilderResumeModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    update: vi.fn().mockResolvedValue({}),
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

const mockResume = {
  id: "resume-1",
  title: "My Resume",
  template_id: "tmpl-1",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

describe("RenameBuilderResumeModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the modal with title when open", () => {
    render(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(screen.getByText("common.rename")).toBeInTheDocument();
  });

  it("renders the title input with current resume title", () => {
    render(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    const input = screen.getByLabelText(/resumes.titleLabel/);
    expect(input).toHaveValue("My Resume");
  });

  it("renders cancel and save buttons", () => {
    render(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
    expect(screen.getByText("common.save")).toBeInTheDocument();
  });

  it("renders description text", () => {
    render(
      <RenameBuilderResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(
      screen.getByText("resumes.editDescription"),
    ).toBeInTheDocument();
  });
});
