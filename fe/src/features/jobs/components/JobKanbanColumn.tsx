import { memo } from "react";
import { useDroppable } from "@dnd-kit/core";
import { useTranslation } from "react-i18next";
import { JobKanbanCard } from "./JobKanbanCard";
import { STATUS_TOP_BORDER_COLORS } from "../lib/jobStatusColors";
import type { JobDTO } from "@/shared/types/api";

interface JobKanbanColumnProps {
  columnId: string;
  label: string;
  jobs: JobDTO[];
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onChangeStatus: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

export const JobKanbanColumn = memo(function JobKanbanColumn({
  columnId,
  label,
  jobs,
  onAddComment,
  onAddStage,
  onChangeStatus,
  onDelete,
}: JobKanbanColumnProps) {
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
          {jobs.length}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-2 min-h-[200px]">
        {jobs.length === 0 ? (
          <p className="text-xs text-muted-foreground text-center py-8">
            {t("jobs.board.emptyColumn")}
          </p>
        ) : (
          jobs.map((job) => (
            <JobKanbanCard
              key={job.id}
              job={job}
              onAddComment={onAddComment}
              onAddStage={onAddStage}
              onChangeStatus={onChangeStatus}
              onDelete={onDelete}
            />
          ))
        )}
      </div>
    </div>
  );
});
