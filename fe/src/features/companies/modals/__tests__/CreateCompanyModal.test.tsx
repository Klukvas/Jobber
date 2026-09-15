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
    render(<CreateCompanyModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText("companies.create")).toBeInTheDocument();
  });

  it("returns null when closed", () => {
    const { container } = render(
      <CreateCompanyModal open={false} onOpenChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders form fields", () => {
    render(<CreateCompanyModal open={true} onOpenChange={vi.fn()} />);
    expect(screen.getByText(/companies.name/)).toBeInTheDocument();
    expect(screen.getByText("common.cancel")).toBeInTheDocument();
  });
});

/**
 * A dialog with no accessible name is announced as just "dialog". Every one of
 * these draws a heading; the shared `Dialog`/`DialogTitle` pair is what turns
 * that heading into the name, and this is the consumer-side half of that
 * contract.
 */
function expectDialogNamedBy(headingText: string) {
  const dialog = screen.getByRole("dialog");
  const labelledBy = dialog.getAttribute("aria-labelledby");

  expect(labelledBy).toBeTruthy();
  expect(document.getElementById(labelledBy ?? "")?.textContent).toBe(
    headingText,
  );
}

describe("CreateCompanyModal — accessible name", () => {
  it("is named by its own heading", () => {
    render(<CreateCompanyModal open={true} onOpenChange={vi.fn()} />);

    expectDialogNamedBy("companies.create");
  });
});
