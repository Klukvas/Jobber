import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ApplicationKanbanColumn } from "../ApplicationKanbanColumn";
import type { ApplicationDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@dnd-kit/core", () => ({
  useDroppable: () => ({ isOver: false, setNodeRef: vi.fn() }),
}));

vi.mock("../ApplicationKanbanCard", () => ({
  ApplicationKanbanCard: ({ application }: { application: ApplicationDTO }) => (
    <div data-testid={`kanban-card-${application.id}`}>{application.name}</div>
  ),
}));

const makeApp = (overrides: Partial<ApplicationDTO> = {}): ApplicationDTO => ({
  id: "app-1",
  name: "Test Application",
  status: "active",
  applied_at: "2025-01-01T00:00:00Z",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
  ...overrides,
});

describe("ApplicationKanbanColumn", () => {
  const defaultProps = {
    columnId: "active",
    label: "Active",
    applications: [] as ApplicationDTO[],
    onAddComment: vi.fn(),
    onAddStage: vi.fn(),
    onChangeStatus: vi.fn(),
  };

  it("renders column label", () => {
    render(<ApplicationKanbanColumn {...defaultProps} />);
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("renders application count", () => {
    render(<ApplicationKanbanColumn {...defaultProps} />);
    expect(screen.getByText("0")).toBeInTheDocument();
  });

  it("shows empty message when no applications", () => {
    render(<ApplicationKanbanColumn {...defaultProps} />);
    expect(
      screen.getByText("applications.board.emptyColumn"),
    ).toBeInTheDocument();
  });

  it("renders application cards when applications exist", () => {
    const apps = [
      makeApp({ id: "1", name: "App One" }),
      makeApp({ id: "2", name: "App Two" }),
    ];
    render(
      <ApplicationKanbanColumn {...defaultProps} applications={apps} />,
    );
    expect(screen.getByTestId("kanban-card-1")).toBeInTheDocument();
    expect(screen.getByTestId("kanban-card-2")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument(); // count badge
  });

  it("does not show empty message when applications exist", () => {
    const apps = [makeApp()];
    render(
      <ApplicationKanbanColumn {...defaultProps} applications={apps} />,
    );
    expect(
      screen.queryByText("applications.board.emptyColumn"),
    ).not.toBeInTheDocument();
  });
});
