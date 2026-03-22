import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ApplicationMobileAccordion } from "../ApplicationMobileAccordion";
import type { ApplicationDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("../ApplicationCardBase", () => ({
  ApplicationCardBase: ({
    application,
  }: {
    application: ApplicationDTO;
  }) => <div data-testid={`card-${application.id}`}>{application.name}</div>,
}));

const makeApp = (overrides: Partial<ApplicationDTO> = {}): ApplicationDTO => ({
  id: "app-1",
  name: "Test App",
  status: "active",
  applied_at: "2025-01-01T00:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  ...overrides,
});

describe("ApplicationMobileAccordion", () => {
  const defaultProps = {
    columns: [
      {
        id: "active",
        label: "Active",
        applications: [makeApp({ id: "1", name: "App One" })],
      },
      {
        id: "on_hold",
        label: "On Hold",
        applications: [],
      },
    ],
    onAddComment: vi.fn(),
    onAddStage: vi.fn(),
    onChangeStatus: vi.fn(),
  };

  it("renders column labels", () => {
    render(<ApplicationMobileAccordion {...defaultProps} />);
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("On Hold")).toBeInTheDocument();
  });

  it("renders application counts", () => {
    render(<ApplicationMobileAccordion {...defaultProps} />);
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("0")).toBeInTheDocument();
  });

  it("shows first column with apps expanded by default", () => {
    render(<ApplicationMobileAccordion {...defaultProps} />);
    // Active column has the application
    expect(screen.getByTestId("card-1")).toBeInTheDocument();
  });

  it("toggles accordion on click", () => {
    render(<ApplicationMobileAccordion {...defaultProps} />);
    // Click on the On Hold accordion
    const onHoldButton = screen.getByText("On Hold").closest("button")!;
    fireEvent.click(onHoldButton);
    // Should show empty message for On Hold
    expect(screen.getByText("applications.board.emptyColumn")).toBeInTheDocument();
  });

  it("renders aria attributes for accessibility", () => {
    render(<ApplicationMobileAccordion {...defaultProps} />);
    const activeButton = screen.getByText("Active").closest("button")!;
    expect(activeButton).toHaveAttribute("aria-expanded", "true");
  });
});
