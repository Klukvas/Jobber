import { useDraggable } from "@dnd-kit/core";
import { useTranslation } from "react-i18next";
import {
  MoreVertical,
  Edit,
  Archive,
  Building2,
  ExternalLink,
} from "lucide-react";
import { useState, useEffect } from "react";
import type { JobDTO } from "@/shared/types/api";

interface KanbanJobCardProps {
  job: JobDTO;
  onEdit: (job: JobDTO) => void;
  onArchive: (jobId: string) => void;
}

export function KanbanJobCard({ job, onEdit, onArchive }: KanbanJobCardProps) {
  const { t } = useTranslation();
  const [menuOpen, setMenuOpen] = useState(false);

  const { attributes, listeners, setNodeRef, transform, isDragging } =
    useDraggable({
      id: job.id,
      data: { job },
    });

  const style = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
      }
    : undefined;

  // Close menu on outside click
  useEffect(() => {
    if (!menuOpen) return;
    const handler = () => setMenuOpen(false);
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, [menuOpen]);

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`rounded-lg border bg-card p-3 shadow-sm transition-all cursor-grab active:cursor-grabbing group ${
        isDragging
          ? "opacity-50 shadow-lg ring-2 ring-primary/20"
          : "hover:shadow-md"
      }`}
      {...listeners}
      {...attributes}
    >
      <div className="flex items-start justify-between gap-1">
        <h4 className="text-sm font-medium leading-tight line-clamp-2 flex-1">
          {job.title}
        </h4>
        <div className="relative flex-shrink-0">
          <button
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              setMenuOpen(!menuOpen);
            }}
            onPointerDown={(e) => e.stopPropagation()}
            className="p-1 rounded-md hover:bg-accent transition-colors opacity-0 group-hover:opacity-100"
            aria-label="Job actions"
          >
            <MoreVertical className="h-3.5 w-3.5" />
          </button>
          {menuOpen && (
            <div className="absolute right-0 mt-1 w-36 bg-popover border rounded-md shadow-lg z-50">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit(job);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
              >
                <Edit className="h-3.5 w-3.5" />
                {t("common.edit")}
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onArchive(job.id);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
              >
                <Archive className="h-3.5 w-3.5" />
                {t("jobs.archive")}
              </button>
            </div>
          )}
        </div>
      </div>

      {job.company_name && (
        <div className="flex items-center gap-1.5 mt-1.5 text-xs text-muted-foreground">
          <Building2 className="h-3 w-3 flex-shrink-0" />
          <span className="truncate">{job.company_name}</span>
        </div>
      )}

      <div className="flex items-center gap-2 mt-2">
        {job.source && (
          <span className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-xs text-secondary-foreground">
            {job.source}
          </span>
        )}
        {job.url && (
          <a
            href={job.url}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            onPointerDown={(e) => e.stopPropagation()}
            className="text-primary hover:underline"
            aria-label={t("jobs.viewPosting")}
          >
            <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        )}
      </div>
    </div>
  );
}
