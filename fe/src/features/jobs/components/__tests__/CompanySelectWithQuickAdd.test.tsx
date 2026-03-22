import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CompanySelectWithQuickAdd } from "../CompanySelectWithQuickAdd";

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

vi.mock("@/services/companiesService", () => ({
  companiesService: { create: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

describe("CompanySelectWithQuickAdd", () => {
  it("renders select with companies", () => {
    const companies = [
      {
        id: "c1",
        name: "Acme Corp",
        is_favorite: false,
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-01T00:00:00Z",
        applications_count: 0,
        active_applications_count: 0,
        derived_status: "idle" as const,
      },
      {
        id: "c2",
        name: "Beta Inc",
        is_favorite: false,
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-01T00:00:00Z",
        applications_count: 0,
        active_applications_count: 0,
        derived_status: "idle" as const,
      },
    ];
    render(
      <CompanySelectWithQuickAdd
        companies={companies}
        value=""
        onChange={vi.fn()}
      />,
    );
    // Should render a select or combobox-like element
    expect(screen.getByText("jobs.selectCompany")).toBeInTheDocument();
  });

  it("renders add new company button", () => {
    render(
      <CompanySelectWithQuickAdd companies={[]} value="" onChange={vi.fn()} />,
    );
    expect(screen.getByTitle("jobs.quickAddCompany")).toBeInTheDocument();
  });
});
