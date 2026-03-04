import { memo } from "react";
import { useDroppable } from "@dnd-kit/core";
import { useTranslation } from "react-i18next";
import { ApplicationKanbanCard } from "./ApplicationKanbanCard";
import { STATUS_TOP_BORDER_COLORS } from "../lib/applicationStatusColors";
import type { ApplicationDTO } from "@/shared/types/api";

interface ApplicationKanbanColumnProps {
  columnId: string;
  label: string;
  applications: ApplicationDTO[];
  onAddComment: (application: ApplicationDTO) => void;
  onAddStage: (application: ApplicationDTO) => void;
  onChangeStatus: (application: ApplicationDTO) => void;
}

export const ApplicationKanbanColumn = memo(function ApplicationKanbanColumn({
  columnId,
  label,
  applications,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: ApplicationKanbanColumnProps) {
  const { t } = useTranslation();
  const { isOver, setNodeRef } = useDroppable({ id: columnId });

  const colorClass =
    STATUS_TOP_BORDER_COLORS[columnId] ?? "border-t-purple-500";

  return (
    <div
      ref={setNodeRef}
      className={`flex flex-col rounded-lg border border-t-4 bg-muted/30 min-w-[280px] flex-1 flex-shrink-0 ${colorClass} ${
        isOver ? "ring-2 ring-primary/30 bg-primary/5" : ""
      }`}
    >
      <div className="flex items-center justify-between px-3 py-2.5 border-b">
        <h3 className="text-sm font-semibold">{label}</h3>
        <span className="inline-flex items-center justify-center rounded-full bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground min-w-[1.5rem]">
          {applications.length}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-2 min-h-[200px]">
        {applications.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-8">
            {t("applications.board.emptyColumn")}
          </p>
        ) : (
          applications.map((app) => (
            <ApplicationKanbanCard
              key={app.id}
              application={app}
              onAddComment={onAddComment}
              onAddStage={onAddStage}
              onChangeStatus={onChangeStatus}
            />
          ))
        )}
      </div>
    </div>
  );
});
