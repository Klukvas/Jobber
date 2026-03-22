import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreateJobModal } from "../CreateJobModal";

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
  useQuery: () => ({
    data: { items: [] },
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/services/jobsService", () => ({
  jobsService: { create: vi.fn(), update: vi.fn() },
}));

vi.mock("@/services/companiesService", () => ({
  companiesService: { list: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("@/features/jobs/components/CompanySelectWithQuickAdd", () => ({
  CompanySelectWithQuickAdd: () => <select data-testid="company-select" />,
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    canCreate: () => true,
  }),
}));

vi.mock("@/features/subscription/components/UpgradeBanner", () => ({
  UpgradeBanner: () => null,
}));

describe("CreateJobModal", () => {
  it("renders when open", () => {
    render(<CreateJobModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("jobs.create")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <CreateJobModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(<CreateJobModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText(/jobs\.title_field/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});
