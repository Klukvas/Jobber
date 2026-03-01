import { useDraggable } from "@dnd-kit/core";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import {
  MoreVertical,
  MessageSquare,
  GitBranch,
  Archive,
  Building2,
  Briefcase,
} from "lucide-react";
import { memo, useState, useEffect, useRef, useCallback } from "react";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import type { ApplicationDTO } from "@/shared/types/api";

interface ApplicationKanbanCardProps {
  application: ApplicationDTO;
  onAddComment: (application: ApplicationDTO) => void;
  onAddStage: (application: ApplicationDTO) => void;
  onChangeStatus: (application: ApplicationDTO) => void;
}

export const ApplicationKanbanCard = memo(function ApplicationKanbanCard({
  application,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: ApplicationKanbanCardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);

  const { attributes, listeners, setNodeRef, transform, isDragging } =
    useDraggable({
      id: application.id,
      data: { application },
    });

  const style = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
      }
    : undefined;

  const closeMenu = useCallback((e: MouseEvent) => {
    // Don't close if clicking the toggle button itself (it handles its own toggle)
    if (toggleRef.current?.contains(e.target as Node)) return;
    // Close if clicking outside the menu
    if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
      setMenuOpen(false);
    }
  }, []);

  useEffect(() => {
    if (!menuOpen) return;
    document.addEventListener("mousedown", closeMenu);
    return () => document.removeEventListener("mousedown", closeMenu);
  }, [menuOpen, closeMenu]);

  const handleClick = () => {
    if (!isDragging) {
      navigate(`/app/applications/${application.id}`);
    }
  };

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
        <button className="text-left flex-1 min-w-0" onClick={handleClick}>
          <h4 className="text-sm font-medium leading-tight line-clamp-2">
            {application.name}
          </h4>
        </button>
        <div className="relative flex-shrink-0" ref={menuRef}>
          <button
            ref={toggleRef}
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              setMenuOpen((prev) => !prev);
            }}
            onPointerDown={(e) => e.stopPropagation()}
            className="p-1 rounded-md hover:bg-accent transition-colors text-muted-foreground"
            aria-label="Application actions"
          >
            <MoreVertical className="h-3.5 w-3.5" />
          </button>
          {menuOpen && (
            <div className="absolute right-0 mt-1 w-44 bg-popover border rounded-md shadow-lg z-50">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onAddComment(application);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
              >
                <MessageSquare className="h-3.5 w-3.5" />
                {t("applications.addComment")}
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onAddStage(application);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
              >
                <GitBranch className="h-3.5 w-3.5" />
                {t("applications.addStage")}
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onChangeStatus(application);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
              >
                <Archive className="h-3.5 w-3.5" />
                {t("applications.changeStatus")}
              </button>
            </div>
          )}
        </div>
      </div>

      {application.job?.company?.name && (
        <div className="flex items-center gap-1.5 mt-1.5 text-xs text-muted-foreground">
          <Building2 className="h-3 w-3 flex-shrink-0" />
          <span className="truncate">{application.job.company.name}</span>
        </div>
      )}

      {application.job?.title && (
        <div className="flex items-center gap-1.5 mt-1 text-xs text-muted-foreground">
          <Briefcase className="h-3 w-3 flex-shrink-0" />
          <span className="truncate">{application.job.title}</span>
        </div>
      )}

      <div className="mt-2">
        <StatusBadge status={application.status} size="sm" />
      </div>
    </div>
  );
});
