import { useState, useCallback, useEffect, memo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { ChevronDown, ChevronRight } from "lucide-react";
import { ApplicationCardBase } from "./ApplicationCardBase";
import { STATUS_LEFT_BORDER_COLORS } from "../lib/applicationStatusColors";
import type { ApplicationDTO } from "@/shared/types/api";

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
  const navigate = useNavigate();
  return (
    <ApplicationCardBase
      application={application}
      onTitleClick={() => navigate(`/app/applications/${application.id}`)}
      onAddComment={onAddComment}
      onAddStage={onAddStage}
      onChangeStatus={onChangeStatus}
    />
  );
});

function defaultOpenIds(columns: MobileColumnData[]): Set<string> {
  const first = columns.find((c) => c.applications.length > 0) ?? columns[0];
  return first ? new Set([first.id]) : new Set();
}

export function ApplicationMobileAccordion({
  columns,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: AccordionProps) {
  const { t } = useTranslation();
  const [openIds, setOpenIds] = useState<Set<string>>(() =>
    defaultOpenIds(columns),
  );

  // Re-sync when columns change (e.g. stage templates finish loading) and
  // the current openIds no longer intersect with the available column IDs.
  useEffect(() => {
    const currentIds = new Set(columns.map((c) => c.id));
    setOpenIds((prev) => {
      const hasValidOpen = [...prev].some((id) => currentIds.has(id));
      return hasValidOpen ? prev : defaultOpenIds(columns);
    });
  }, [columns]);

  const toggle = useCallback((id: string) => {
    setOpenIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  return (
    <div className="space-y-2">
      {columns.map((col) => {
        const isOpen = openIds.has(col.id);
        const colorClass =
          STATUS_LEFT_BORDER_COLORS[col.id] ?? "border-l-purple-500";
        const headerId = `accordion-header-${col.id}`;
        const contentId = `accordion-content-${col.id}`;

        return (
          <div
            key={col.id}
            className={`rounded-lg border border-l-4 bg-muted/30 ${colorClass}`}
          >
            <button
              id={headerId}
              aria-expanded={isOpen}
              aria-controls={contentId}
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
              <div
                id={contentId}
                role="region"
                aria-labelledby={headerId}
                className="px-2 pb-2 space-y-2"
              >
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
