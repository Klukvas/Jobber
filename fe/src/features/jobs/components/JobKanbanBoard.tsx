import { useCallback, useMemo, useState } from "react";
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
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
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
import type { JobDTO, JobStatus, PaginatedResponse } from "@/shared/types/api";

export const JOBS_KANBAN_QUERY_KEY = ["jobs", "kanban"] as const;

const STATUS_COLUMNS: JobStatus[] = [
  "saved",
  "applied",
  "on_hold",
  "offer",
  "rejected",
  "archived",
];

const NO_STAGE_COLUMN_ID = "__no_stage__";

type GroupBy = "status" | "stage";

interface JobKanbanBoardProps {
  jobs: JobDTO[];
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onChangeStatus: (job: JobDTO) => void;
  onStatusSelect?: (job: JobDTO, status: JobStatus) => void;
  onCompleteStage?: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

export function JobKanbanBoard({
  jobs,
  onAddComment,
  onAddStage,
  onChangeStatus,
  onStatusSelect,
  onCompleteStage,
  onDelete,
}: JobKanbanBoardProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [activeJob, setActiveJob] = useState<JobDTO | null>(null);
  const [groupBy, setGroupBy] = useState<GroupBy>("status");

  const { data: stageTemplatesData } = useQuery({
    queryKey: ["stage-templates"],
    queryFn: () => stageTemplatesService.list({ limit: 100, offset: 0 }),
    enabled: groupBy === "stage",
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

  const { mutate: updateStatus } = useMutation({
    mutationFn: ({ id, status }: { id: string; status: JobStatus }) =>
      jobsService.update(id, { status }),
    onMutate: async ({ id, status }) => {
      await queryClient.cancelQueries({ queryKey: JOBS_KANBAN_QUERY_KEY });
      const previous = queryClient.getQueryData<PaginatedResponse<JobDTO>>(
        JOBS_KANBAN_QUERY_KEY,
      );

      queryClient.setQueryData<PaginatedResponse<JobDTO>>(
        JOBS_KANBAN_QUERY_KEY,
        (oldData) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            items: oldData.items.map((j) =>
              j.id === id ? { ...j, status } : j,
            ),
          };
        },
      );

      return { previous };
    },
    onSuccess: () => {
      showSuccessNotification(t("jobs.board.moveSuccess"));
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(JOBS_KANBAN_QUERY_KEY, context.previous);
      }
      showErrorNotification(t("jobs.board.moveError"));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: JOBS_KANBAN_QUERY_KEY });
    },
  });

  const { mutate: addStageToJob } = useMutation({
    mutationFn: ({
      id,
      stage_template_id,
    }: {
      id: string;
      stage_template_id: string;
      targetStageName: string;
    }) => jobsService.addStage(id, { stage_template_id }),
    onMutate: async ({ id, targetStageName }) => {
      await queryClient.cancelQueries({ queryKey: JOBS_KANBAN_QUERY_KEY });
      const previous = queryClient.getQueryData<PaginatedResponse<JobDTO>>(
        JOBS_KANBAN_QUERY_KEY,
      );

      queryClient.setQueryData<PaginatedResponse<JobDTO>>(
        JOBS_KANBAN_QUERY_KEY,
        (oldData) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            items: oldData.items.map((j) =>
              j.id === id ? { ...j, current_stage_name: targetStageName } : j,
            ),
          };
        },
      );

      return { previous };
    },
    onSuccess: () => {
      showSuccessNotification(t("jobs.board.moveSuccess"));
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(JOBS_KANBAN_QUERY_KEY, context.previous);
      }
      showErrorNotification(t("jobs.board.moveError"));
    },
    onSettled: () => {
      // Reconcile the optimistic write with the server, which also updates
      // current_stage_id and last_activity_at when a stage is added.
      queryClient.invalidateQueries({ queryKey: JOBS_KANBAN_QUERY_KEY });
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

      if (groupBy === "status") {
        const cachedData = queryClient.getQueryData<PaginatedResponse<JobDTO>>(
          JOBS_KANBAN_QUERY_KEY,
        );
        const currentJob = cachedData?.items.find((j) => j.id === jobId);
        if (!currentJob || currentJob.status === targetColumn) return;

        updateStatus({ id: jobId, status: targetColumn as JobStatus });
      } else {
        // Stage mode: targetColumn is template.id
        if (targetColumn === NO_STAGE_COLUMN_ID) return;

        const template = stageTemplates.find((st) => st.id === targetColumn);
        if (!template) return;

        const cachedData = queryClient.getQueryData<PaginatedResponse<JobDTO>>(
          JOBS_KANBAN_QUERY_KEY,
        );
        const currentJob = cachedData?.items.find((j) => j.id === jobId);
        if (!currentJob) return;

        if (currentJob.current_stage_name === template.name) return;

        addStageToJob({
          id: jobId,
          stage_template_id: template.id,
          targetStageName: template.name,
        });
      }
    },
    [groupBy, queryClient, stageTemplates, updateStatus, addStageToJob],
  );

  const columns = useMemo(
    () =>
      groupBy === "status"
        ? buildStatusColumns(jobs, t)
        : buildStageColumns(jobs, stageTemplates, t),
    [groupBy, jobs, stageTemplates, t],
  );

  return (
    <div className="space-y-4">
      {/* Grouping toggle */}
      <div className="flex items-center gap-2">
        <div className="flex items-center rounded-lg border bg-muted p-0.5">
          <button
            onClick={() => setGroupBy("status")}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              groupBy === "status"
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t("jobs.board.groupByStatus")}
          </button>
          <button
            onClick={() => setGroupBy("stage")}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              groupBy === "stage"
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t("jobs.board.groupByStage")}
          </button>
        </div>
      </div>

      {/* Mobile accordion — replaces horizontal board on small screens */}
      <div className="md:hidden">
        <JobMobileAccordion
          key={groupBy}
          columns={columns}
          onAddComment={onAddComment}
          onAddStage={onAddStage}
          onChangeStatus={onChangeStatus}
          onStatusSelect={onStatusSelect}
          onCompleteStage={onCompleteStage}
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
                onAddComment={onAddComment}
                onAddStage={onAddStage}
                onChangeStatus={onChangeStatus}
                onStatusSelect={onStatusSelect}
                onCompleteStage={onCompleteStage}
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
                  onChangeStatus={() => {}}
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

// ColumnData is the same shape as MobileColumnData — re-export for consumers
export type { MobileColumnData as ColumnData };

const STATUS_I18N_MAP: Record<JobStatus, string> = {
  saved: "saved",
  applied: "applied",
  on_hold: "onHold",
  offer: "offer",
  rejected: "rejected",
  archived: "archived",
};

function buildStatusColumns(
  jobs: JobDTO[],
  t: (key: string) => string,
): MobileColumnData[] {
  return STATUS_COLUMNS.map((status) => ({
    id: status,
    label: t(`jobs.board.${STATUS_I18N_MAP[status]}`),
    jobs: jobs.filter((j) => j.status === status),
  }));
}

function buildStageColumns(
  jobs: JobDTO[],
  stageTemplates: { id: string; name: string }[],
  t: (key: string) => string,
): MobileColumnData[] {
  return [
    {
      id: NO_STAGE_COLUMN_ID,
      label: t("jobs.board.noStage"),
      jobs: jobs.filter((j) => !j.current_stage_name),
    },
    ...stageTemplates.map((template) => ({
      id: template.id,
      label: template.name,
      jobs: jobs.filter((j) => j.current_stage_name === template.name),
    })),
  ];
}
