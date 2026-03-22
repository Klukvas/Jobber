import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { DeleteCompanyDialog } from "../DeleteCompanyDialog";
import type { CompanyDTO } from "@/shared/types/api";

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
  useMutation: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  useQuery: () => ({
    data: { jobs_count: 0, applications_count: 0 },
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
  }),
}));

vi.mock("@/services/companiesService", () => ({
  companiesService: {
    delete: vi.fn(),
    getRelatedCounts: vi.fn(),
  },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showErrorNotification: vi.fn(),
}));

const mockCompany: CompanyDTO = {
  id: "c1",
  name: "Acme Corp",
  is_favorite: false,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  applications_count: 0,
  active_applications_count: 0,
  derived_status: "idle",
};

describe("DeleteCompanyDialog", () => {
  it("renders when open", () => {
    render(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={vi.fn()}
        company={mockCompany}
      />,
    );
    expect(screen.getByText("companies.delete")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <DeleteCompanyDialog
        open={false}
        onOpenChange={vi.fn()}
        company={mockCompany}
      />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders safe delete message when no related data", () => {
    render(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={vi.fn()}
        company={mockCompany}
      />,
    );
    expect(screen.getByText("companies.deleteSafe")).toBeInTheDocument();
  });

  it("renders cancel and delete buttons", () => {
    render(
      <DeleteCompanyDialog
        open={true}
        onOpenChange={vi.fn()}
        company={mockCompany}
      />,
    );
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
    expect(screen.getByText("common.delete")).toBeInTheDocument();
  });
});
