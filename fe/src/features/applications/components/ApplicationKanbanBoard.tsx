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
import { applicationsService } from "@/services/applicationsService";
import { stageTemplatesService } from "@/services/stageTemplatesService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { ApplicationKanbanColumn } from "./ApplicationKanbanColumn";
import { ApplicationKanbanCard } from "./ApplicationKanbanCard";
import type {
  ApplicationDTO,
  ApplicationStatus,
  PaginatedResponse,
} from "@/shared/types/api";

export const APPLICATIONS_KANBAN_QUERY_KEY = [
  "applications",
  "kanban",
] as const;

const STATUS_COLUMNS: ApplicationStatus[] = [
  "active",
  "on_hold",
  "offer",
  "rejected",
  "archived",
];

const NO_STAGE_COLUMN_ID = "__no_stage__";

type GroupBy = "status" | "stage";

interface ApplicationKanbanBoardProps {
  applications: ApplicationDTO[];
  onAddComment: (application: ApplicationDTO) => void;
  onAddStage: (application: ApplicationDTO) => void;
  onChangeStatus: (application: ApplicationDTO) => void;
}

export function ApplicationKanbanBoard({
  applications,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: ApplicationKanbanBoardProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [activeApp, setActiveApp] = useState<ApplicationDTO | null>(null);
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
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      applicationsService.update(id, { status }),
    onMutate: async ({ id, status }) => {
      await queryClient.cancelQueries({
        queryKey: APPLICATIONS_KANBAN_QUERY_KEY,
      });
      const previous = queryClient.getQueryData<
        PaginatedResponse<ApplicationDTO>
      >(APPLICATIONS_KANBAN_QUERY_KEY);

      queryClient.setQueryData<PaginatedResponse<ApplicationDTO>>(
        APPLICATIONS_KANBAN_QUERY_KEY,
        (oldData) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            items: oldData.items.map((a) =>
              a.id === id ? { ...a, status: status as ApplicationStatus } : a,
            ),
          };
        },
      );

      return { previous };
    },
    onSuccess: () => {
      showSuccessNotification(t("applications.board.moveSuccess"));
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(
          APPLICATIONS_KANBAN_QUERY_KEY,
          context.previous,
        );
      }
      showErrorNotification(t("applications.board.moveError"));
    },
  });

  const { mutate: addStageToApp } = useMutation({
    mutationFn: ({
      id,
      stage_template_id,
    }: {
      id: string;
      stage_template_id: string;
      targetStageName: string;
    }) => applicationsService.addStage(id, { stage_template_id }),
    onMutate: async ({ id, targetStageName }) => {
      await queryClient.cancelQueries({
        queryKey: APPLICATIONS_KANBAN_QUERY_KEY,
      });
      const previous = queryClient.getQueryData<
        PaginatedResponse<ApplicationDTO>
      >(APPLICATIONS_KANBAN_QUERY_KEY);

      queryClient.setQueryData<PaginatedResponse<ApplicationDTO>>(
        APPLICATIONS_KANBAN_QUERY_KEY,
        (oldData) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            items: oldData.items.map((a) =>
              a.id === id ? { ...a, current_stage_name: targetStageName } : a,
            ),
          };
        },
      );

      return { previous };
    },
    onSuccess: () => {
      showSuccessNotification(t("applications.board.moveSuccess"));
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(
          APPLICATIONS_KANBAN_QUERY_KEY,
          context.previous,
        );
      }
      showErrorNotification(t("applications.board.moveError"));
    },
  });

  const handleDragStart = useCallback(
    (event: DragStartEvent) => {
      const draggedApp = applications.find((a) => a.id === event.active.id);
      setActiveApp(draggedApp ?? null);
    },
    [applications],
  );

  const handleDragCancel = useCallback(() => {
    setActiveApp(null);
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      setActiveApp(null);
      const { active, over } = event;

      if (!over) return;

      const appId = active.id as string;
      const targetColumn = over.id as string;

      if (groupBy === "status") {
        const cachedData = queryClient.getQueryData<
          PaginatedResponse<ApplicationDTO>
        >(APPLICATIONS_KANBAN_QUERY_KEY);
        const currentApp = cachedData?.items.find((a) => a.id === appId);
        if (!currentApp || currentApp.status === targetColumn) return;

        updateStatus({ id: appId, status: targetColumn });
      } else {
        // Stage mode: targetColumn is template.id
        if (targetColumn === NO_STAGE_COLUMN_ID) return;

        const template = stageTemplates.find((st) => st.id === targetColumn);
        if (!template) return;

        const cachedData = queryClient.getQueryData<
          PaginatedResponse<ApplicationDTO>
        >(APPLICATIONS_KANBAN_QUERY_KEY);
        const currentApp = cachedData?.items.find((a) => a.id === appId);
        if (!currentApp) return;

        if (currentApp.current_stage_name === template.name) return;

        addStageToApp({
          id: appId,
          stage_template_id: template.id,
          targetStageName: template.name,
        });
      }
    },
    [groupBy, queryClient, stageTemplates, updateStatus, addStageToApp],
  );

  const columns = useMemo(
    () =>
      groupBy === "status"
        ? buildStatusColumns(applications, t)
        : buildStageColumns(applications, stageTemplates, t),
    [groupBy, applications, stageTemplates, t],
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
            {t("applications.board.groupByStatus")}
          </button>
          <button
            onClick={() => setGroupBy("stage")}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              groupBy === "stage"
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t("applications.board.groupByStage")}
          </button>
        </div>
      </div>

      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
        onDragCancel={handleDragCancel}
      >
        <div className="relative flex gap-4 min-h-[calc(100vh-16rem)] overflow-x-auto pb-2 snap-x snap-mandatory md:snap-none scroll-pl-0">
          {columns.map((col) => (
            <ApplicationKanbanColumn
              key={col.id}
              columnId={col.id}
              label={col.label}
              applications={col.applications}
              onAddComment={onAddComment}
              onAddStage={onAddStage}
              onChangeStatus={onChangeStatus}
            />
          ))}
        </div>

        <DragOverlay>
          {activeApp ? (
            <div className="w-[240px]">
              <ApplicationKanbanCard
                application={activeApp}
                onAddComment={() => {}}
                onAddStage={() => {}}
                onChangeStatus={() => {}}
              />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  );
}

interface ColumnData {
  id: string;
  label: string;
  applications: ApplicationDTO[];
}

const STATUS_I18N_MAP: Record<ApplicationStatus, string> = {
  active: "active",
  on_hold: "onHold",
  offer: "offer",
  rejected: "rejected",
  archived: "archived",
};

function buildStatusColumns(
  applications: ApplicationDTO[],
  t: (key: string) => string,
): ColumnData[] {
  return STATUS_COLUMNS.map((status) => ({
    id: status,
    label: t(`applications.board.${STATUS_I18N_MAP[status]}`),
    applications: applications.filter((a) => a.status === status),
  }));
}

function buildStageColumns(
  applications: ApplicationDTO[],
  stageTemplates: { id: string; name: string }[],
  t: (key: string) => string,
): ColumnData[] {
  const noStageApps = applications.filter((a) => !a.current_stage_name);

  const columns: ColumnData[] = [
    {
      id: NO_STAGE_COLUMN_ID,
      label: t("applications.board.noStage"),
      applications: noStageApps,
    },
  ];

  for (const template of stageTemplates) {
    columns.push({
      id: template.id,
      label: template.name,
      applications: applications.filter(
        (a) => a.current_stage_name === template.name,
      ),
    });
  }

  return columns;
}
