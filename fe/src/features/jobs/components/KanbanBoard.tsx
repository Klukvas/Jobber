import { useCallback, useRef } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  closestCorners,
  type DragStartEvent,
  type DragEndEvent,
} from "@dnd-kit/core";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { KanbanColumn } from "./KanbanColumn";
import { KanbanJobCard } from "./KanbanJobCard";
import type {
  BoardColumn,
  JobDTO,
  PaginatedResponse,
} from "@/shared/types/api";

export const KANBAN_QUERY_KEY = ["jobs"] as const;

const COLUMNS: BoardColumn[] = [
  "wishlist",
  "applied",
  "interview",
  "offer",
  "rejected",
];

interface KanbanBoardProps {
  jobs: JobDTO[];
  onEditJob: (job: JobDTO) => void;
  onArchiveJob: (jobId: string) => void;
}

export function KanbanBoard({
  jobs,
  onEditJob,
  onArchiveJob,
}: KanbanBoardProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [activeJob, setActiveJob] = useState<JobDTO | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
    useSensor(KeyboardSensor),
  );

  const { mutate: moveJob } = useMutation({
    mutationFn: ({
      id,
      board_column,
    }: {
      id: string;
      board_column: BoardColumn;
    }) => jobsService.moveToColumn(id, board_column),
    onSuccess: () => {
      showSuccessNotification(t("jobs.board.moveSuccess"));
    },
    onError: () => {
      // Revert optimistic update by refetching
      queryClient.invalidateQueries({ queryKey: KANBAN_QUERY_KEY });
      showErrorNotification(t("jobs.board.moveError"));
    },
  });

  // Stable ref so handleDragEnd doesn't depend on mutate identity
  const moveJobRef = useRef(moveJob);
  moveJobRef.current = moveJob;

  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const draggedJob = jobs.find((j) => j.id === event.active.id);
      setActiveJob(draggedJob ?? null);
    },
    [jobs],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveJob(null);
      const { active, over } = event;

      if (!over) return;

      const jobId = active.id as string;
      const targetColumn = over.id as BoardColumn;

      // Read current board_column from cache to avoid stale prop
      const cachedData = queryClient.getQueryData<PaginatedResponse<JobDTO>>(
        KANBAN_QUERY_KEY,
      );
      const currentJob = cachedData?.items.find((j) => j.id === jobId);
      if (!currentJob || currentJob.board_column === targetColumn) return;

      // Optimistic update - update the cache immediately
      queryClient.setQueryData<PaginatedResponse<JobDTO>>(
        KANBAN_QUERY_KEY,
        (oldData) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            items: oldData.items.map((j) =>
              j.id === jobId ? { ...j, board_column: targetColumn } : j,
            ),
          };
        },
      );

      // Fire API call via stable ref
      moveJobRef.current({ id: jobId, board_column: targetColumn });
    },
    [queryClient],
  );

  // Group jobs by column
  const jobsByColumn = COLUMNS.reduce(
    (acc, col) => {
      acc[col] = jobs.filter((j) => j.board_column === col);
      return acc;
    },
    {} as Record<BoardColumn, JobDTO[]>,
  );

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
    >
      <div className="grid grid-cols-5 gap-4 min-h-[calc(100vh-16rem)]">
        {COLUMNS.map((column) => (
          <KanbanColumn
            key={column}
            column={column}
            jobs={jobsByColumn[column]}
            onEditJob={onEditJob}
            onArchiveJob={onArchiveJob}
          />
        ))}
      </div>

      <DragOverlay>
        {activeJob ? (
          <div className="w-[240px]">
            <KanbanJobCard
              job={activeJob}
              onEdit={() => {}}
              onArchive={() => {}}
            />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}
