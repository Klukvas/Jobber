import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import type { DragEndEvent } from "@dnd-kit/core";
import { JobKanbanBoard } from "../JobKanbanBoard";
import {
  jobBoardQueryKey,
  type JobBoardData,
} from "@/features/jobs/hooks/useJobBoardQuery";
import type { JobDTO, StageTemplateDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockShowError = vi.hoisted(() => vi.fn());
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: mockShowError,
}));

// Capture the DndContext onDragEnd so tests can drive a drop. Everything else
// about dnd-kit needs a real pointer and a real layout, neither of which jsdom
// has.
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
    setActivatorNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
  closestCorners: vi.fn(),
  defaultKeyboardCoordinateGetter: vi.fn(),
}));

const moveMock = vi.hoisted(() => vi.fn());
vi.mock("@/services/jobsService", () => ({
  jobsService: {
    move: (...args: unknown[]) => moveMock(...args),
  },
}));

// Hoisted so the service mock below can serve the very same columns.
const templates: StageTemplateDTO[] = vi.hoisted(() => [
  { id: "t-offer", name: "Offer", order: 3, kind: "offer", created_at: "" },
  {
    id: "t-wishlist",
    name: "Wishlist",
    order: 1,
    kind: "saved",
    created_at: "",
  },
  {
    id: "t-screen",
    name: "Screening",
    order: 2,
    kind: "response",
    created_at: "",
  },
]);

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: vi.fn().mockResolvedValue({
      items: templates,
      pagination: { total: templates.length, limit: 100, offset: 0 },
    }),
  },
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

const BOARD_KEY = jobBoardQueryKey({
  archived: "",
  sort: "last_activity:desc",
  search: "",
  companyId: "",
});

/** The board's cache as the infinite query leaves it: one entry per page. */
function boardData(pages: JobDTO[][]): JobBoardData {
  const total = pages.reduce((n, items) => n + items.length, 0);
  return {
    pages: pages.map((items, index) => ({
      items,
      pagination: { total, limit: 100, offset: index * 100 },
    })),
    pageParams: pages.map((_, index) => index * 100),
  };
}

/**
 * The real query client, seeded the way the board's own infinite query would
 * leave it — the optimistic move is then asserted on the cache itself rather
 * than on a spy.
 */
function renderBoard(pages: JobDTO[][]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  queryClient.setQueryData(BOARD_KEY, boardData(pages));
  queryClient.setQueryData(["stage-templates"], {
    items: templates,
    pagination: { total: templates.length, limit: 100, offset: 0 },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/app/jobs"]}>
        <JobKanbanBoard
          jobs={pages.flat()}
          queryKey={BOARD_KEY}
          onAddComment={vi.fn()}
          onAddStage={vi.fn()}
          onDelete={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return queryClient;
}

function cachedJob(queryClient: QueryClient, id: string): JobDTO | undefined {
  return queryClient
    .getQueryData<JobBoardData>(BOARD_KEY)
    ?.pages.flatMap((page) => page.items)
    .find((item) => item.id === id);
}

describe("JobKanbanBoard — single-axis columns", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnDragEnd = undefined;
  });

  it("renders one column per stage template, in `order`", () => {
    renderBoard([
      [
        job({ id: "a", current_stage_template_id: "t-wishlist" }),
        job({ id: "b", current_stage_template_id: "t-screen" }),
      ],
    ]);
    // Column headers appear twice (mobile accordion + desktop board).
    const headings = screen
      .getAllByRole("heading", { level: 3 })
      .map((h) => h.textContent);
    expect(headings).toContain("Wishlist");
    expect(headings).toContain("Screening");
    expect(headings).toContain("Offer");
  });

  it("hides the No-stage column when every card sits in a column", () => {
    renderBoard([[job({ id: "a", current_stage_template_id: "t-wishlist" })]]);
    expect(screen.queryByText("jobs.board.noStage")).not.toBeInTheDocument();
  });

  it("shows the No-stage column for cards without a column", () => {
    renderBoard([[job({ id: "a", current_stage_template_id: undefined })]]);
    expect(screen.getAllByText("jobs.board.noStage").length).toBeGreaterThan(0);
  });

  it("moves a card to the dropped column via jobsService.move", async () => {
    moveMock.mockResolvedValue(
      job({ id: "j1", current_stage_template_id: "t-screen" }),
    );
    renderBoard([[job({ id: "j1", current_stage_template_id: "t-wishlist" })]]);
    expect(capturedOnDragEnd).toBeDefined();
    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "t-screen" },
    } as unknown as DragEndEvent);
    // The mutation itself runs a tick later than the optimistic cache write.
    await waitFor(() =>
      expect(moveMock).toHaveBeenCalledWith("j1", {
        stage_template_id: "t-screen",
      }),
    );
  });

  // Regression: the optimistic cache write must happen synchronously inside
  // the drag-end handler — dnd-kit measures the drop-animation target right
  // after it runs, and a deferred write (e.g. in onMutate after an awaited
  // cancelQueries) makes the overlay animate back to the old column.
  it("writes the move into the cache synchronously on drop", () => {
    moveMock.mockResolvedValue(
      job({ id: "j1", current_stage_template_id: "t-screen" }),
    );
    const queryClient = renderBoard([
      [job({ id: "j1", current_stage_template_id: "t-wishlist" })],
    ]);

    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "t-screen" },
    } as unknown as DragEndEvent);

    // No await: the cache already holds the move.
    const moved = cachedJob(queryClient, "j1");
    expect(moved?.current_stage_template_id).toBe("t-screen");
    expect(moved?.current_stage_name).toBe("Screening");
  });

  // Every loaded card has to be draggable, not just the ones from page one.
  it("moves a card that came from a later page", async () => {
    moveMock.mockResolvedValue(
      job({ id: "page2", current_stage_template_id: "t-offer" }),
    );
    const queryClient = renderBoard([
      [job({ id: "page1", current_stage_template_id: "t-wishlist" })],
      [job({ id: "page2", current_stage_template_id: "t-screen" })],
    ]);

    capturedOnDragEnd!({
      active: { id: "page2" },
      over: { id: "t-offer" },
    } as unknown as DragEndEvent);

    await waitFor(() =>
      expect(moveMock).toHaveBeenCalledWith("page2", {
        stage_template_id: "t-offer",
      }),
    );
    expect(cachedJob(queryClient, "page2")?.current_stage_template_id).toBe(
      "t-offer",
    );
    // The card on the other page is untouched.
    expect(cachedJob(queryClient, "page1")?.current_stage_template_id).toBe(
      "t-wishlist",
    );
  });

  it("rolls the card back into its old column when the server refuses", async () => {
    moveMock.mockRejectedValue(new Error("nope"));
    const queryClient = renderBoard([
      [job({ id: "j1", current_stage_template_id: "t-wishlist" })],
    ]);

    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "t-screen" },
    } as unknown as DragEndEvent);

    await waitFor(() =>
      expect(mockShowError).toHaveBeenCalledWith("jobs.board.moveError"),
    );
    expect(cachedJob(queryClient, "j1")?.current_stage_template_id).toBe(
      "t-wishlist",
    );
  });

  it("does not move when dropped on its current column", () => {
    renderBoard([[job({ id: "j1", current_stage_template_id: "t-screen" })]]);
    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "t-screen" },
    } as unknown as DragEndEvent);
    expect(moveMock).not.toHaveBeenCalled();
  });

  it("does not move when dropped on the No-stage column", () => {
    renderBoard([[job({ id: "j1", current_stage_template_id: undefined })]]);
    capturedOnDragEnd!({
      active: { id: "j1" },
      over: { id: "__no_stage__" },
    } as unknown as DragEndEvent);
    expect(moveMock).not.toHaveBeenCalled();
  });
});
