import { useState, useCallback, useMemo, memo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { ChevronDown, ChevronRight } from "lucide-react";
import { JobCardBase } from "./JobCardBase";
import { STATUS_LEFT_BORDER_COLORS } from "../lib/jobStatusColors";
import type { JobDTO, JobStatus } from "@/shared/types/api";

export interface MobileColumnData {
  id: string;
  label: string;
  jobs: JobDTO[];
}

interface AccordionProps {
  columns: MobileColumnData[];
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onChangeStatus: (job: JobDTO) => void;
  onStatusSelect?: (job: JobDTO, status: JobStatus) => void;
  onCompleteStage?: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

interface MobileCardProps {
  job: JobDTO;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onChangeStatus: (job: JobDTO) => void;
  onStatusSelect?: (job: JobDTO, status: JobStatus) => void;
  onCompleteStage?: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

const MobileCard = memo(function MobileCard({
  job,
  onAddComment,
  onAddStage,
  onChangeStatus,
  onStatusSelect,
  onCompleteStage,
  onDelete,
}: MobileCardProps) {
  const navigate = useNavigate();
  return (
    <JobCardBase
      job={job}
      onTitleClick={() => navigate(`/app/jobs/${job.id}`)}
      onAddComment={onAddComment}
      onAddStage={onAddStage}
      onChangeStatus={onChangeStatus}
      onStatusSelect={onStatusSelect}
      onCompleteStage={onCompleteStage}
      onDelete={onDelete}
    />
  );
});

function defaultOpenIds(columns: MobileColumnData[]): Set<string> {
  const first = columns.find((c) => c.jobs.length > 0) ?? columns[0];
  return first ? new Set([first.id]) : new Set();
}

export function JobMobileAccordion({
  columns,
  onAddComment,
  onAddStage,
  onChangeStatus,
  onStatusSelect,
  onCompleteStage,
  onDelete,
}: AccordionProps) {
  const { t } = useTranslation();
  const [openIds, setOpenIds] = useState<Set<string>>(() =>
    defaultOpenIds(columns),
  );

  // Derive effective open IDs at render time — avoids setState-in-effect lint rule.
  // Falls back to the default open column if none of the stored IDs are valid.
  const effectiveOpenIds = useMemo(() => {
    const currentIds = new Set(columns.map((c) => c.id));
    const valid = new Set([...openIds].filter((id) => currentIds.has(id)));
    return valid.size > 0 ? valid : defaultOpenIds(columns);
  }, [openIds, columns]);

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
        const isOpen = effectiveOpenIds.has(col.id);
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
                {col.jobs.length}
              </span>
            </button>

            {isOpen && (
              <div
                id={contentId}
                role="region"
                aria-labelledby={headerId}
                className="px-2 pb-2 space-y-2"
              >
                {col.jobs.length === 0 ? (
                  <p className="text-xs text-muted-foreground text-center py-4">
                    {t("jobs.board.emptyColumn")}
                  </p>
                ) : (
                  col.jobs.map((job) => (
                    <MobileCard
                      key={job.id}
                      job={job}
                      onAddComment={onAddComment}
                      onAddStage={onAddStage}
                      onChangeStatus={onChangeStatus}
                      onStatusSelect={onStatusSelect}
                      onCompleteStage={onCompleteStage}
                      onDelete={onDelete}
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
