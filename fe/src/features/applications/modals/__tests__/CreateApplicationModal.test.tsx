import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreateApplicationModal } from "../CreateApplicationModal";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQuery: () => ({ data: { items: [] } }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/services/applicationsService", () => ({
  applicationsService: { create: vi.fn() },
}));
vi.mock("@/services/resumesService", () => ({
  resumesService: { list: vi.fn() },
}));
vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: { list: vi.fn() },
}));
vi.mock("@/services/jobsService", () => ({
  jobsService: { list: vi.fn() },
}));
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/features/jobs/modals/CreateJobModal", () => ({
  CreateJobModal: () => null,
}));
vi.mock("@/features/resumes/modals/CreateResumeModal", () => ({
  CreateResumeModal: () => null,
}));

describe("CreateApplicationModal", () => {
  it("renders when open", () => {
    render(<CreateApplicationModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("applications.create")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <CreateApplicationModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(<CreateApplicationModal open={true} onOpenChange={vi.fn()} />);
    expect(
      screen.getByText("applications.applicationName"),
    ).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});
