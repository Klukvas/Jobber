import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { Copy, Trash2, PenTool, Pencil } from "lucide-react";
import { ResumeThumbnail } from "@/features/resume-builder/components/ResumeThumbnail";
import { Button } from "@/shared/ui/Button";
import { Card, CardContent } from "@/shared/ui/Card";
import type { ResumeBuilderDTO } from "@/shared/types/resume-builder";

interface BuilderResumeCardProps {
  readonly resume: ResumeBuilderDTO;
  readonly limitReached: boolean;
  readonly onDuplicate: (id: string) => void;
  readonly onDelete: (resume: ResumeBuilderDTO) => void;
  readonly onRename?: (resume: ResumeBuilderDTO) => void;
}

export function BuilderResumeCard({
  resume,
  limitReached,
  onDuplicate,
  onDelete,
  onRename,
}: BuilderResumeCardProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const dateLocale = useDateLocale();

  return (
    <Card
      className="group cursor-pointer transition-shadow hover:shadow-md h-full"
      onClick={() => navigate(`/app/resume-builder/${resume.id}`)}
    >
      <CardContent className="p-4">
        <div className="mb-3">
          <ResumeThumbnail
            resumeId={resume.id}
            templateId={resume.template_id}
          />
        </div>

        {/* Type badge */}
        <div className="mb-2">
          <div className="inline-flex items-center gap-1 text-xs font-medium text-purple-600 bg-purple-50 dark:bg-purple-900/30 dark:text-purple-400 px-2 py-1 rounded">
            <PenTool className="h-3 w-3" />
            {t("resumes.typeBuilt")}
          </div>
        </div>

        <div className="flex items-start justify-between">
          <div className="min-w-0 flex-1">
            <h3 className="truncate font-medium">{resume.title}</h3>
            <p className="text-sm text-muted-foreground">
              {t("resumeBuilder.lastEdited", {
                date: format(new Date(resume.updated_at), "PPP", {
                  locale: dateLocale,
                }),
              })}
            </p>
          </div>
          <div className="ml-2 flex gap-1 opacity-100 sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100">
            {onRename && (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={(e) => {
                  e.stopPropagation();
                  onRename(resume);
                }}
                aria-label={t("common.rename")}
              >
                <Pencil className="h-4 w-4" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              disabled={limitReached}
              onClick={(e) => {
                e.stopPropagation();
                onDuplicate(resume.id);
              }}
              aria-label={t("resumeBuilder.duplicate")}
              title={
                limitReached
                  ? t("settings.subscription.limitReached")
                  : undefined
              }
            >
              <Copy className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive"
              onClick={(e) => {
                e.stopPropagation();
                onDelete(resume);
              }}
              aria-label={t("common.delete")}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
