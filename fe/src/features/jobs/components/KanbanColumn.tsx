import { useDroppable } from "@dnd-kit/core";
import { useTranslation } from "react-i18next";
import { KanbanJobCard } from "./KanbanJobCard";
import type { BoardColumn, JobDTO } from "@/shared/types/api";

const COLUMN_COLORS: Record<BoardColumn, string> = {
  wishlist: "border-t-blue-500",
  applied: "border-t-yellow-500",
  interview: "border-t-purple-500",
  offer: "border-t-green-500",
  rejected: "border-t-red-500",
};

interface KanbanColumnProps {
  column: BoardColumn;
  jobs: JobDTO[];
  onEditJob: (job: JobDTO) => void;
  onArchiveJob: (jobId: string) => void;
}

export function KanbanColumn({
  column,
  jobs,
  onEditJob,
  onArchiveJob,
}: KanbanColumnProps) {
  const { t } = useTranslation();
  const { isOver, setNodeRef } = useDroppable({ id: column });

  return (
    <div
      ref={setNodeRef}
      className={`flex flex-col rounded-lg border border-t-4 bg-muted/30 ${COLUMN_COLORS[column]} ${
        isOver ? "ring-2 ring-primary/30 bg-primary/5" : ""
      }`}
    >
      <div className="flex items-center justify-between px-3 py-2.5 border-b">
        <h3 className="text-sm font-semibold">
          {t(`jobs.board.${column}`)}
        </h3>
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
            <KanbanJobCard
              key={job.id}
              job={job}
              onEdit={onEditJob}
              onArchive={onArchiveJob}
            />
          ))
        )}
      </div>
    </div>
  );
}
