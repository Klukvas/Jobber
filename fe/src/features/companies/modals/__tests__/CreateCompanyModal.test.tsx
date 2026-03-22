import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreateCompanyModal } from "../CreateCompanyModal";

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

vi.mock("@/services/companiesService", () => ({
  companiesService: { create: vi.fn(), update: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

describe("CreateCompanyModal", () => {
  it("renders when open", () => {
    render(
      <CreateCompanyModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(screen.getByText("companies.create")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <CreateCompanyModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(
      <CreateCompanyModal open={true} onOpenChange={vi.fn()} />,
    );
    expect(screen.getByText(/companies.name/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});
