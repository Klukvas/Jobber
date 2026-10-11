import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { formatDistanceToNow } from "date-fns";
import {
  Calendar,
  Clock,
  GitBranch,
  Link2,
  MessageSquare,
  MoreVertical,
  Trash2,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { CompanyAvatar } from "@/shared/ui/CompanyAvatar";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { stageColor } from "@/features/jobs/lib/stageColors";
import { TOUCH_ICON_TARGET } from "@/features/jobs/lib/touchTargets";
import type { JobsListReturnState } from "@/features/jobs/lib/jobsListReturn";
import type { JobDTO } from "@/shared/types/api";

interface JobListGridProps {
  jobs: JobDTO[];
  /** Dimmed while a new page/search/filter is in flight. */
  isFetching: boolean;
  openMenuId: string | null;
  onToggleMenu: (id: string | null) => void;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
  /** Carried by each card so the detail page's Back returns to this list. */
  returnState: JobsListReturnState;
}

/** The Jobs list view: one card per application, in the current sort order. */
export function JobListGrid({
  jobs,
  isFetching,
  openMenuId,
  onToggleMenu,
  onAddComment,
  onAddStage,
  onDelete,
  returnState,
}: JobListGridProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();

  return (
    <div
      className={`grid gap-4 md:grid-cols-2 lg:grid-cols-3 transition-opacity ${
        isFetching ? "opacity-60" : ""
      }`}
    >
      {jobs.map((job) => {
        const stage = stageColor(job.current_stage_name, job.is_archived);
        return (
          <div key={job.id} className="relative">
            <Link to={`/app/jobs/${job.id}`} state={returnState}>
              <Card
                className={`transition-all hover:shadow-md motion-safe:hover:-translate-y-0.5 h-full group border-l-4 ${stage.border}`}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="text-xl font-bold leading-tight mb-2 flex-1">
                      {job.title}
                    </CardTitle>
                    <div
                      className="relative"
                      onClick={(e) => e.preventDefault()}
                    >
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          e.preventDefault();
                          onToggleMenu(openMenuId === job.id ? null : job.id);
                        }}
                        className={`${TOUCH_ICON_TARGET} rounded-md hover:bg-accent transition-colors text-muted-foreground`}
                        aria-label={t("jobs.actionsMenu")}
                      >
                        <MoreVertical className="h-4 w-4" />
                      </button>
                      {openMenuId === job.id && (
                        <div className="absolute right-0 mt-1 w-48 bg-popover border rounded-md shadow-lg z-10">
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              onAddComment(job);
                            }}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <MessageSquare className="h-4 w-4" />
                            {t("jobs.addComment")}
                          </button>
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              onAddStage(job);
                            }}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                          >
                            <GitBranch className="h-4 w-4" />
                            {t("jobs.addStage")}
                          </button>
                          <div className="my-1 border-t" role="separator" />
                          <button
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              onDelete(job);
                            }}
                            className="flex items-center gap-2 w-full px-3 py-2 text-sm text-destructive hover:bg-destructive/10 text-left"
                          >
                            <Trash2 className="h-4 w-4" />
                            {t("jobs.delete")}
                          </button>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Secondary: company (with monogram) and source.
                      The stage lives in the coloured pill below, so it is
                      not repeated here. */}
                  <div className="space-y-1">
                    {job.company_name && (
                      <div className="flex items-center gap-2 text-base font-medium text-foreground">
                        <CompanyAvatar name={job.company_name} size="sm" />
                        <span>{job.company_name}</span>
                      </div>
                    )}
                    {job.source && (
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Link2 className="h-4 w-4" />
                        <span>{job.source}</span>
                      </div>
                    )}
                  </div>
                </CardHeader>

                <CardContent className="space-y-3 pt-0">
                  {/* Pipeline column — the card's state */}
                  <div className="flex items-center justify-between">
                    <span
                      className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-medium ${stage.pill}`}
                    >
                      <span
                        className={`h-1.5 w-1.5 rounded-full ${stage.dot}`}
                        aria-hidden
                      />
                      {job.is_archived
                        ? t("jobs.archived")
                        : (job.current_stage_name ?? t("jobs.board.noStage"))}
                    </span>
                  </div>

                  {/* Meta Information */}
                  <div className="space-y-2 text-sm text-muted-foreground border-t pt-3">
                    {job.resume?.name && (
                      <div className="flex items-center gap-2">
                        <span className="font-medium">
                          {t("jobs.resumeLabel")}:
                        </span>
                        <span>{job.resume.name}</span>
                      </div>
                    )}
                    <div className="flex items-center gap-2">
                      <Calendar className="h-3.5 w-3.5" />
                      <span>
                        {job.applied_at
                          ? `${t("jobs.applied")} ${formatDistanceToNow(
                              new Date(job.applied_at),
                              { addSuffix: true, locale: dateLocale },
                            )}`
                          : `${t("jobs.createdDate")} ${formatDistanceToNow(
                              new Date(job.created_at),
                              { addSuffix: true, locale: dateLocale },
                            )}`}
                      </span>
                    </div>
                    {job.last_activity_at && (
                      <div className="flex items-center gap-2">
                        <Clock className="h-3.5 w-3.5" />
                        <span>
                          {t("jobs.updated")}{" "}
                          {formatDistanceToNow(new Date(job.last_activity_at), {
                            addSuffix: true,
                            locale: dateLocale,
                          })}
                        </span>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </Link>
          </div>
        );
      })}
    </div>
  );
}
