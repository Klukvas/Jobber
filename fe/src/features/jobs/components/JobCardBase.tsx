import {
  memo,
  useState,
  useEffect,
  useRef,
  useCallback,
  type CSSProperties,
  type HTMLAttributes,
} from "react";
import { useTranslation } from "react-i18next";
import {
  MoreVertical,
  MessageSquare,
  GitBranch,
  Building2,
  Trash2,
} from "lucide-react";
import type { JobDTO } from "@/shared/types/api";

interface JobCardBaseProps {
  job: JobDTO;
  onTitleClick: () => void;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
  /** Ref callback from useDraggable — omit for non-draggable cards */
  dragRef?: (element: HTMLElement | null) => void;
  /** Transform style from useDraggable */
  dragStyle?: CSSProperties;
  /** Spread of dnd-kit listeners + attributes */
  dragProps?: HTMLAttributes<HTMLDivElement>;
  /** Current drag state — undefined means the card is not in a drag context */
  isDragging?: boolean;
}

export const JobCardBase = memo(function JobCardBase({
  job,
  onTitleClick,
  onAddComment,
  onAddStage,
  onDelete,
  dragRef,
  dragStyle,
  dragProps,
  isDragging,
}: JobCardBaseProps) {
  const { t } = useTranslation();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const firstItemRef = useRef<HTMLButtonElement>(null);

  const isDraggable = isDragging !== undefined;

  // Close menu when clicking outside
  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (toggleRef.current?.contains(e.target as Node)) return;
    if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
      setMenuOpen(false);
    }
  }, []);

  // Close menu on Escape and return focus to toggle button
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape" && menuOpen) {
        setMenuOpen(false);
        toggleRef.current?.focus();
      }
    },
    [menuOpen],
  );

  useEffect(() => {
    if (!menuOpen) return;
    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [menuOpen, handleClickOutside, handleKeyDown]);

  // Move focus into the menu when it opens
  useEffect(() => {
    if (menuOpen) {
      firstItemRef.current?.focus();
    }
  }, [menuOpen]);

  const outerClassName = [
    "rounded-lg border bg-card p-3 shadow-sm transition-all",
    isDraggable ? "cursor-grab active:cursor-grabbing group" : "",
    isDragging
      ? "opacity-50 shadow-lg ring-2 ring-primary/20"
      : "hover:shadow-md",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      ref={dragRef}
      style={dragStyle}
      className={outerClassName}
      {...dragProps}
    >
      <div className="flex items-start justify-between gap-1">
        <button className="text-left flex-1 min-w-0" onClick={onTitleClick}>
          <h4 className="text-sm font-medium leading-tight line-clamp-2">
            {job.title}
          </h4>
        </button>

        <div className="relative flex-shrink-0" ref={menuRef}>
          <button
            ref={toggleRef}
            aria-label={t("jobs.actionsMenu")}
            aria-expanded={menuOpen}
            aria-haspopup="menu"
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              setMenuOpen(!menuOpen);
            }}
            // Prevent drag sensor from triggering on menu button press
            onPointerDown={(e) => e.stopPropagation()}
            className="p-2 rounded-md hover:bg-accent transition-colors text-muted-foreground"
          >
            <MoreVertical className="h-4 w-4" />
          </button>

          {menuOpen && (
            <div
              role="menu"
              aria-label={t("jobs.actionsMenu")}
              className="absolute right-0 mt-1 w-44 bg-popover border rounded-md shadow-lg z-50"
            >
              <button
                ref={firstItemRef}
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  onAddComment(job);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
              >
                <MessageSquare className="h-3.5 w-3.5" />
                {t("jobs.addComment")}
              </button>
              <button
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  onAddStage(job);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
              >
                <GitBranch className="h-3.5 w-3.5" />
                {t("jobs.addStage")}
              </button>
              <div className="my-1 border-t" role="separator" />
              <button
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(job);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm text-destructive hover:bg-destructive/10 text-left"
              >
                <Trash2 className="h-3.5 w-3.5" />
                {t("jobs.delete")}
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

      {job.current_stage_name && (
        <div className="flex items-center gap-1.5 mt-1 text-xs text-muted-foreground">
          <GitBranch className="h-3 w-3 flex-shrink-0" />
          <span className="truncate">{job.current_stage_name}</span>
        </div>
      )}
    </div>
  );
});
