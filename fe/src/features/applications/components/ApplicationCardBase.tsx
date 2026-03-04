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
  Archive,
  Building2,
  Briefcase,
} from "lucide-react";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import type { ApplicationDTO } from "@/shared/types/api";

interface ApplicationCardBaseProps {
  application: ApplicationDTO;
  onTitleClick: () => void;
  onAddComment: (app: ApplicationDTO) => void;
  onAddStage: (app: ApplicationDTO) => void;
  onChangeStatus: (app: ApplicationDTO) => void;
  /** Ref callback from useDraggable — omit for non-draggable cards */
  dragRef?: (element: HTMLElement | null) => void;
  /** Transform style from useDraggable */
  dragStyle?: CSSProperties;
  /** Spread of dnd-kit listeners + attributes */
  dragProps?: HTMLAttributes<HTMLDivElement>;
  /** Current drag state — undefined means the card is not in a drag context */
  isDragging?: boolean;
}

export const ApplicationCardBase = memo(function ApplicationCardBase({
  application,
  onTitleClick,
  onAddComment,
  onAddStage,
  onChangeStatus,
  dragRef,
  dragStyle,
  dragProps,
  isDragging,
}: ApplicationCardBaseProps) {
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
            {application.name}
          </h4>
        </button>

        <div className="relative flex-shrink-0" ref={menuRef}>
          <button
            ref={toggleRef}
            aria-label={t("applications.actionsMenu")}
            aria-expanded={menuOpen}
            aria-haspopup="menu"
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              setMenuOpen((prev) => !prev);
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
              aria-label={t("applications.actionsMenu")}
              className="absolute right-0 mt-1 w-44 bg-popover border rounded-md shadow-lg z-50"
            >
              <button
                ref={firstItemRef}
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  onAddComment(application);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
              >
                <MessageSquare className="h-3.5 w-3.5" />
                {t("applications.addComment")}
              </button>
              <button
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  onAddStage(application);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
              >
                <GitBranch className="h-3.5 w-3.5" />
                {t("applications.addStage")}
              </button>
              <button
                role="menuitem"
                onClick={(e) => {
                  e.stopPropagation();
                  onChangeStatus(application);
                  setMenuOpen(false);
                }}
                onPointerDown={(e) => e.stopPropagation()}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
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
