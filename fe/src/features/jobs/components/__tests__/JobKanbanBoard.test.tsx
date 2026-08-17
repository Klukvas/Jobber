import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import type { DragEndEvent } from "@dnd-kit/core";
import { JobKanbanBoard } from "../JobKanbanBoard";
import type { JobDTO, StageTemplateDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));

// Capture the DndContext onDragEnd so tests can drive a drop.
let capturedOnDragEnd: ((e: DragEndEvent) => void) | undefined;
vi.mock("@dnd-kit/core", () => ({
  DndContext: ({
    children,
    onDragEnd,
  }: {
    children: React.ReactNode;
    onDragEnd: (e: DragEndEvent) => void;
  }) => {
    capturedOnDragEnd = onDragEnd;
    return <div>{children}</div>;
  },
  DragOverlay: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  PointerSensor: class {},
  KeyboardSensor: class {},
  useSensor: () => ({}),
  useSensors: () => [],
  useDroppable: () => ({ isOver: false, setNodeRef: vi.fn() }),
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
  closestCorners: vi.fn(),
}));

const moveMock = vi.fn();
vi.mock("@/services/jobsService", () => ({
  jobsService: {
    move: (...args: unknown[]) => moveMock(...args),
  },
}));

const templates: StageTemplateDTO[] = [
  { id: "t-offer", name: "Offer", order: 3, created_at: "" },
  { id: "t-wishlist", name: "Wishlist", order: 1, created_at: "" },
  { id: "t-screen", name: "Screening", order: 2, created_at: "" },
];

const cachedJobs: JobDTO[] = [];

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: vi.fn(),
  },
}));

// A hand-rolled query mock: templates come from useQuery, and the move
// mutation invokes jobsService.move and calls the provided mutationFn.
vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: { items: templates } }),
  useMutation: (opts: { mutationFn: (v: unknown) => unknown }) => ({
    mutate: (vars: unknown) => opts.mutationFn(vars),
    isPending: false,
  }),
  useQueryClient: () => ({
    invalidateQueries: vi.fn(),
    cancelQueries: vi.fn(),
    setQueryData: vi.fn(),
    getQueryData: () => ({ items: cachedJobs }),
  }),
}));

const job = (over: Partial<JobDTO>): JobDTO => ({
  id: "j1",
  title: "Job",
  is_archived: false,
  is_favorite: false,
  last_activity_at: "",
  created_at: "",
  updated_at: "",
  ...over,
});

function renderBoard(jobs: JobDTO[]) {
  cachedJobs.length = 0;
  cachedJobs.push(...jobs);
  render(
    <JobKanbanBoard
      jobs={jobs}
      queryKey={["jobs", "kanban"]}
      onAddComment={vi.fn()}
      onAddStage={vi.fn()}
      onDelete={vi.fn()}
    />,
  );
}

describe("JobKanbanBoard — single-axis columns", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnDragEnd = undefined;
  });

  it("renders one column per stage template, in `order`", () => {
    renderBoard([
      job({ id: "a", current_stage_template_id: "t-wishlist" }),
      job({ id: "b", current_stage_template_id: "t-screen" }),
    ]);
    // Column headers appear twice (mobile accordion + desktop board).
    const headings = screen
      .getAllByRole("heading", { level: 3 })
      .map((h) => h.textContent);
    // Desktop board order follows template order.
    expect(headings).toContain("Wishlist");
    expect(headings).toContain("Screening");
    expect(headings).toContain("Offer");
  });

  it("hides the No-stage column when every card sits in a column", () => {
    renderBoard([job({ id: "a", current_stage_template_id: "t-wishlist" })]);
    expect(screen.queryByText("jobs.board.noStage")).not.toBeInTheDocument();
  });

  it("shows the No-stage column for cards without a column", () => {
    renderBoard([job({ id: "a", current_stage_template_id: undefined })]);
    expect(screen.getAllByText("jobs.board.noStage").length).toBeGreaterThan(0);
  });

  it("moves a card to the dropped column via jobsService.move", () => {
    renderBoard([job({ id: "j1", current_stage_template_id: "t-wishlist" })]);
    expect(capturedOnDragEnd).toBeDefined();
    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "t-screen" },
    } as unknown as DragEndEvent);
    expect(moveMock).toHaveBeenCalledWith("j1", {
      stage_template_id: "t-screen",
    });
  });

  it("does not move when dropped on its current column", () => {
    renderBoard([job({ id: "j1", current_stage_template_id: "t-screen" })]);
    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "t-screen" },
    } as unknown as DragEndEvent);
    expect(moveMock).not.toHaveBeenCalled();
  });

  it("does not move when dropped on the No-stage column", () => {
    renderBoard([job({ id: "j1", current_stage_template_id: undefined })]);
    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "__no_stage__" },
    } as unknown as DragEndEvent);
    expect(moveMock).not.toHaveBeenCalled();
  });
});
