import { useCallback, useEffect, useMemo, useState } from "react";
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
import { useTranslation } from "react-i18next";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import { stageTemplatesService } from "@/services/stageTemplatesService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { JobKanbanColumn } from "./JobKanbanColumn";
import { JobKanbanCard } from "./JobKanbanCard";
import {
  JobMobileAccordion,
  type MobileColumnData,
} from "./JobMobileAccordion";
import type {
  JobDTO,
  PaginatedResponse,
  StageTemplateDTO,
} from "@/shared/types/api";

export const JOBS_KANBAN_QUERY_KEY = ["jobs", "kanban"] as const;

// Cards with no column assigned yet gather in a leading "No stage" column.
export const NO_STAGE_COLUMN_ID = "__no_stage__";

interface JobKanbanBoardProps {
  jobs: JobDTO[];
  // Exact React Query key the board's data lives under, so optimistic move
  // updates and invalidation target the currently-filtered board.
  queryKey: QueryKey;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

// Places a card into the column matching its current stage-template id. Cards
// without one land in the leading "No stage" column.
function columnIdForJob(job: JobDTO): string {
  return job.current_stage_template_id ?? NO_STAGE_COLUMN_ID;
}

export function JobKanbanBoard({
  jobs,
  queryKey,
  onAddComment,
  onAddStage,
  onDelete,
}: JobKanbanBoardProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [activeJob, setActiveJob] = useState<JobDTO | null>(null);
  // Card to briefly flash right after it lands in a new column.
  const [flashId, setFlashId] = useState<string | null>(null);

  useEffect(() => {
    if (!flashId) return;
    const tid = setTimeout(() => setFlashId(null), 700);
    return () => clearTimeout(tid);
  }, [flashId]);

  const { data: stageTemplatesData } = useQuery({
    queryKey: ["stage-templates"],
    queryFn: () => stageTemplatesService.list({ limit: 100, offset: 0 }),
    staleTime: 5 * 60 * 1000,
  });

  const stageTemplates = useMemo(
    () => stageTemplatesData?.items ?? [],
    [stageTemplatesData?.items],
  );

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 8 },
    }),
    useSensor(KeyboardSensor),
  );

  // The single write path: move the card to a pipeline column. The optimistic
  // cache write happens synchronously in handleDragEnd (the drop animation
  // depends on its timing) — onMutate only stops in-flight refetches from
  // overwriting it and stashes the rollback snapshot.
  const { mutate: moveJob } = useMutation({
    mutationFn: ({ id, stageTemplateId }: MoveVars) =>
      jobsService.move(id, { stage_template_id: stageTemplateId }),
    onMutate: async ({ previous }) => {
      await queryClient.cancelQueries({ queryKey });
      return { previous };
    },
    onSuccess: (data) => {
      // Fold in the authoritative card so the optimistic guess converges to
      // the server truth before the reconciling refetch lands.
      queryClient.setQueryData<PaginatedResponse<JobDTO>>(
        queryKey,
        (oldData) =>
          oldData
            ? {
                ...oldData,
                items: oldData.items.map((j) => (j.id === data.id ? data : j)),
              }
            : oldData,
      );
      showSuccessNotification(t("jobs.board.moveSuccess"));
      setFlashId(data.id);
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(queryKey, context.previous);
      }
      showErrorNotification(t("jobs.board.moveError"));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });

  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const draggedJob = jobs.find((j) => j.id === event.active.id);
      setActiveJob(draggedJob ?? null);
    },
    [jobs],
  );

  const handleDragCancel = useCallback(() => {
    setActiveJob(null);
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveJob(null);
      const { active, over } = event;
      if (!over) return;

      const jobId = active.id as string;
      const targetColumn = over.id as string;

      // The "No stage" column is not a move target — cards leave it by being
      // dropped into a real stage column.
      if (targetColumn === NO_STAGE_COLUMN_ID) return;

      const template = stageTemplates.find((tpl) => tpl.id === targetColumn);
      if (!template) return;

      const previous =
        queryClient.getQueryData<PaginatedResponse<JobDTO>>(queryKey);
      const currentJob = previous?.items.find((j) => j.id === jobId);
      if (!currentJob) return;
      if (currentJob.current_stage_template_id === template.id) return;

      // Write the move into the cache synchronously, before dnd-kit measures
      // the drop-animation target: the overlay then glides into the card's
      // slot in the new column instead of flying back to the old one. Doing
      // this in onMutate is too late — its `await cancelQueries` defers the
      // write past the measurement.
      queryClient.setQueryData<PaginatedResponse<JobDTO>>(
        queryKey,
        (oldData) =>
          oldData
            ? {
                ...oldData,
                items: oldData.items.map((j) =>
                  j.id === jobId
                    ? {
                        ...j,
                        current_stage_template_id: template.id,
                        current_stage_name: template.name,
                      }
                    : j,
                ),
              }
            : oldData,
      );

      moveJob({
        id: jobId,
        stageTemplateId: template.id,
        previous,
      });
    },
    [stageTemplates, queryClient, moveJob, queryKey],
  );

  const columns = useMemo(
    () => buildColumns(jobs, stageTemplates, t),
    [jobs, stageTemplates, t],
  );

  return (
    <div className="space-y-4">
      {/* Mobile accordion — replaces horizontal board on small screens */}
      <div className="md:hidden">
        <JobMobileAccordion
          columns={columns}
          onAddComment={onAddComment}
          onAddStage={onAddStage}
          onDelete={onDelete}
        />
      </div>

      {/* Desktop kanban with drag-and-drop */}
      <div className="hidden md:block">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCorners}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          <div className="relative flex gap-4 min-h-[calc(100vh-16rem)] overflow-x-auto pb-2">
            {columns.map((col) => (
              <JobKanbanColumn
                key={col.id}
                columnId={col.id}
                label={col.label}
                jobs={col.jobs}
                flashId={flashId}
                onAddComment={onAddComment}
                onAddStage={onAddStage}
                onDelete={onDelete}
              />
            ))}
          </div>

          <DragOverlay>
            {activeJob ? (
              <div className="w-[240px]">
                <JobKanbanCard
                  job={activeJob}
                  onAddComment={() => {}}
                  onAddStage={() => {}}
                  onDelete={() => {}}
                />
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      </div>
    </div>
  );
}

interface MoveVars {
  id: string;
  stageTemplateId: string;
  // Cache snapshot taken before the optimistic write in handleDragEnd, kept
  // for rollback when the server rejects the move.
  previous?: PaginatedResponse<JobDTO>;
}

// ColumnData is the same shape as MobileColumnData — re-export for consumers
export type { MobileColumnData as ColumnData };

// One column per stage template (in `order`), preceded by a "No stage" column
// for cards not yet placed in any column. The "No stage" column is hidden when
// empty so it does not clutter a fully-triaged board.
function buildColumns(
  jobs: JobDTO[],
  stageTemplates: StageTemplateDTO[],
  t: (key: string) => string,
): MobileColumnData[] {
  const ordered = [...stageTemplates].sort((a, b) => a.order - b.order);

  const noStageJobs = jobs.filter(
    (j) => columnIdForJob(j) === NO_STAGE_COLUMN_ID,
  );

  const stageColumns: MobileColumnData[] = ordered.map((template) => ({
    id: template.id,
    label: template.name,
    jobs: jobs.filter((j) => j.current_stage_template_id === template.id),
  }));

  if (noStageJobs.length === 0) return stageColumns;

  return [
    {
      id: NO_STAGE_COLUMN_ID,
      label: t("jobs.board.noStage"),
      jobs: noStageJobs,
    },
    ...stageColumns,
  ];
}
