import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ApplicationKanbanBoard } from "../ApplicationKanbanBoard";
import type { ApplicationDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DragOverlay: () => null,
  PointerSensor: class {},
  KeyboardSensor: class {},
  useSensor: () => ({}),
  useSensors: () => [],
  closestCorners: vi.fn(),
  useDroppable: () => ({ isOver: false, setNodeRef: vi.fn() }),
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQuery: () => ({ data: { items: [] } }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/services/applicationsService", () => ({
  applicationsService: { update: vi.fn() },
}));

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: { list: vi.fn() },
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("../ApplicationCardBase", () => ({
  ApplicationCardBase: ({ application }: { application: ApplicationDTO }) => (
    <div data-testid={`card-${application.id}`}>{application.name}</div>
  ),
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

describe("ApplicationKanbanBoard", () => {
  const defaultProps = {
    applications: [] as ApplicationDTO[],
    onAddComment: vi.fn(),
    onAddStage: vi.fn(),
    onChangeStatus: vi.fn(),
  };

  it("renders column headers", () => {
    render(<ApplicationKanbanBoard {...defaultProps} />);
    // The mobile accordion renders labels like applications.board.active
    expect(
      screen.getAllByText("applications.board.active").length,
    ).toBeGreaterThanOrEqual(1);
    expect(
      screen.getAllByText("applications.board.onHold").length,
    ).toBeGreaterThanOrEqual(1);
  });

  it("renders applications", () => {
    const apps = [makeApp({ id: "1", name: "Active App", status: "active" })];
    render(<ApplicationKanbanBoard {...defaultProps} applications={apps} />);
    // The card is rendered via ApplicationCardBase mock
    expect(screen.getAllByText("Active App").length).toBeGreaterThanOrEqual(1);
  });

  it("renders group-by toggle", () => {
    render(<ApplicationKanbanBoard {...defaultProps} />);
    expect(
      screen.getByText("applications.board.groupByStatus"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("applications.board.groupByStage"),
    ).toBeInTheDocument();
  });
});
