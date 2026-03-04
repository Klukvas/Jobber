import { useState, useCallback, useEffect, useRef, memo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import {
  ChevronDown,
  ChevronRight,
  Building2,
  Briefcase,
  MoreVertical,
  MessageSquare,
  GitBranch,
  Archive,
} from "lucide-react";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import type { ApplicationDTO } from "@/shared/types/api";

const SECTION_BORDER_COLORS: Record<string, string> = {
  active: "border-l-green-500",
  on_hold: "border-l-yellow-500",
  offer: "border-l-blue-500",
  rejected: "border-l-red-500",
  archived: "border-l-gray-500",
};

export interface MobileColumnData {
  id: string;
  label: string;
  applications: ApplicationDTO[];
}

interface AccordionProps {
  columns: MobileColumnData[];
  onAddComment: (app: ApplicationDTO) => void;
  onAddStage: (app: ApplicationDTO) => void;
  onChangeStatus: (app: ApplicationDTO) => void;
}

interface MobileCardProps {
  application: ApplicationDTO;
  onAddComment: (app: ApplicationDTO) => void;
  onAddStage: (app: ApplicationDTO) => void;
  onChangeStatus: (app: ApplicationDTO) => void;
}

const MobileCard = memo(function MobileCard({
  application,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: MobileCardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);

  const closeMenu = useCallback((e: MouseEvent) => {
    if (toggleRef.current?.contains(e.target as Node)) return;
    if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
      setMenuOpen(false);
    }
  }, []);

  useEffect(() => {
    if (!menuOpen) return;
    document.addEventListener("mousedown", closeMenu);
    return () => document.removeEventListener("mousedown", closeMenu);
  }, [menuOpen, closeMenu]);

  return (
    <div className="rounded-lg border bg-card p-3 shadow-sm">
      <div className="flex items-start justify-between gap-1">
        <button
          className="text-left flex-1 min-w-0"
          onClick={() => navigate(`/app/applications/${application.id}`)}
        >
          <h4 className="text-sm font-medium leading-tight line-clamp-2">
            {application.name}
          </h4>
        </button>

        <div className="relative flex-shrink-0" ref={menuRef}>
          <button
            ref={toggleRef}
            onClick={(e) => {
              e.stopPropagation();
              setMenuOpen((prev) => !prev);
            }}
            className="p-1.5 rounded-md hover:bg-accent transition-colors text-muted-foreground"
            aria-label="Application actions"
          >
            <MoreVertical className="h-4 w-4" />
          </button>

          {menuOpen && (
            <div className="absolute right-0 mt-1 w-44 bg-popover border rounded-md shadow-lg z-50">
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onAddComment(application);
                  setMenuOpen(false);
                }}
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
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
                className="flex items-center gap-2 w-full px-3 py-2.5 text-sm hover:bg-accent text-left"
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

export function ApplicationMobileAccordion({
  columns,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: AccordionProps) {
  const { t } = useTranslation();

  // First non-empty column open by default, fall back to first column
  const [openIds, setOpenIds] = useState<Set<string>>(() => {
    const firstWithItems = columns.find((c) => c.applications.length > 0);
    const first = firstWithItems ?? columns[0];
    return first ? new Set([first.id]) : new Set();
  });

  const toggle = (id: string) => {
    setOpenIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <div className="space-y-2">
      {columns.map((col) => {
        const isOpen = openIds.has(col.id);
        const colorClass =
          SECTION_BORDER_COLORS[col.id] ?? "border-l-purple-500";

        return (
          <div
            key={col.id}
            className={`rounded-lg border border-l-4 bg-muted/30 ${colorClass}`}
          >
            <button
              className="flex items-center justify-between w-full px-3 py-3 text-left"
              onClick={() => toggle(col.id)}
            >
              <div className="flex items-center gap-2">
                {isOpen ? (
                  <ChevronDown className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                )}
                <span className="text-sm font-semibold">{col.label}</span>
              </div>
              <span className="inline-flex items-center justify-center rounded-full bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground min-w-[1.5rem]">
                {col.applications.length}
              </span>
            </button>

            {isOpen && (
              <div className="px-2 pb-2 space-y-2">
                {col.applications.length === 0 ? (
                  <p className="text-xs text-muted-foreground text-center py-4">
                    {t("applications.board.emptyColumn")}
                  </p>
                ) : (
                  col.applications.map((app) => (
                    <MobileCard
                      key={app.id}
                      application={app}
                      onAddComment={onAddComment}
                      onAddStage={onAddStage}
                      onChangeStatus={onChangeStatus}
                    />
                  ))
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
