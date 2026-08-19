import { memo } from "react";
import { useDroppable } from "@dnd-kit/core";
import { useTranslation } from "react-i18next";
import { JobKanbanCard } from "./JobKanbanCard";
import type { JobDTO } from "@/shared/types/api";
import { stageColor } from "../lib/stageColors";

interface JobKanbanColumnProps {
  columnId: string;
  label: string;
  jobs: JobDTO[];
  /** Card id to flash (just moved into this board) */
  flashId?: string | null;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

export const JobKanbanColumn = memo(function JobKanbanColumn({
  columnId,
  label,
  jobs,
  flashId,
  onAddComment,
  onAddStage,
  onDelete,
}: JobKanbanColumnProps) {
  const { t } = useTranslation();
  const { isOver, setNodeRef } = useDroppable({ id: columnId });
  const stage = stageColor(label);

  return (
    <div
      ref={setNodeRef}
      className={`flex flex-col rounded-lg border border-t-4 border-t-primary/60 bg-muted/30 min-w-[280px] flex-1 flex-shrink-0 ${
        isOver ? "ring-2 ring-primary/30 bg-primary/5" : ""
      }`}
    >
      <div className="flex items-center justify-between px-3 py-2.5 border-b">
        <h3 className="flex items-center gap-2 text-sm font-semibold">
          <span className={`h-2 w-2 rounded-full ${stage.dot}`} aria-hidden />
          {label}
        </h3>
        <span className="inline-flex items-center justify-center rounded-full bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground min-w-[1.5rem]">
          {jobs.length}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-2 min-h-[200px]">
        {jobs.length === 0 ? (
          <div
            className={`flex items-center justify-center rounded-md border border-dashed py-8 text-xs transition-colors ${
              isOver
                ? "border-primary/50 bg-primary/5 text-primary"
                : "border-muted-foreground/25 text-muted-foreground"
            }`}
          >
            {t("jobs.board.emptyColumn")}
          </div>
        ) : (
          jobs.map((job) => (
            <JobKanbanCard
              key={job.id}
              job={job}
              flash={job.id === flashId}
              onAddComment={onAddComment}
              onAddStage={onAddStage}
              onDelete={onDelete}
            />
          ))
        )}
      </div>
    </div>
  );
});
