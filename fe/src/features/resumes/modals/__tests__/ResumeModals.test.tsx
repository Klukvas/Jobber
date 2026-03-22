import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { DeleteResumeModal } from "../DeleteResumeModal";
import type { ResumeDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params) {
        return Object.entries(params).reduce(
          (acc, [k, v]) => acc.replace(`{{${k}}}`, String(v)),
          key,
        );
      }
      return key;
    },
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/services/resumesService", () => ({
  resumesService: { delete: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showErrorNotification: vi.fn(),
}));

const mockResume: ResumeDTO = {
  id: "r1",
  title: "My Resume",
  file_url: "/files/resume.pdf",
  storage_type: "external",
  is_active: true,
  can_delete: true,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  applications_count: 0,
};

describe("DeleteResumeModal", () => {
  it("renders when open", () => {
    render(
      <DeleteResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    // Title appears in both header and button
    expect(screen.getAllByText("resumes.delete").length).toBeGreaterThanOrEqual(
      1,
    );
  });

  it("returns null when closed", () => {
    const { container } = render(
      <DeleteResumeModal
        open={false}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders warning when no applications", () => {
    render(
      <DeleteResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(screen.getByText("resumes.deleteWarning")).toBeInTheDocument();
  });

  it("renders cancel button", () => {
    render(
      <DeleteResumeModal
        open={true}
        onOpenChange={vi.fn()}
        resume={mockResume}
      />,
    );
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});
