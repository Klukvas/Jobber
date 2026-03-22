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
      { id: "c1", name: "Acme Corp" },
      { id: "c2", name: "Beta Inc" },
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
