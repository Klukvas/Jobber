import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { format } from "date-fns";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import { resumesService } from "@/services/resumesService";
import { showErrorNotification } from "@/shared/lib/notifications";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import {
  ExternalLink,
  CheckCircle,
  XCircle,
  MoreVertical,
  Edit,
  Trash2,
  Briefcase,
  Download,
  Cloud,
  Link as LinkIcon,
  Upload,
} from "lucide-react";
import { Button } from "@/shared/ui/Button";
import type { ResumeDTO } from "@/shared/types/api";

interface UploadedResumeCardProps {
  readonly resume: ResumeDTO;
  readonly isMenuOpen: boolean;
  readonly onToggleMenu: () => void;
  readonly onEdit: (resume: ResumeDTO) => void;
  readonly onDelete: (resume: ResumeDTO) => void;
}

export function UploadedResumeCard({
  resume,
  isMenuOpen,
  onToggleMenu,
  onEdit,
  onDelete,
}: UploadedResumeCardProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();

  const downloadMutation = useMutation({
    mutationFn: resumesService.generateDownloadURL,
    onSuccess: (data) => {
      window.open(data.download_url, "_blank");
    },
    onError: (error: Error) => {
      showErrorNotification(error?.message || t("resumes.downloadError"));
    },
  });

  return (
    <Card
      className={`transition-all hover:shadow-md h-full group relative ${
        !resume.is_active ? "opacity-60" : ""
      }`}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-2">
          <CardTitle className="text-lg font-bold leading-tight flex-1">
            {resume.title}
          </CardTitle>
          {/* Context Menu */}
          <div className="relative" onClick={(e) => e.preventDefault()}>
            <button
              onClick={(e) => {
                e.stopPropagation();
                e.preventDefault();
                onToggleMenu();
              }}
              className="p-1 rounded-md hover:bg-accent transition-colors text-muted-foreground"
              aria-label={t("resumes.actionsMenu")}
              aria-haspopup="menu"
              aria-expanded={isMenuOpen}
            >
              <MoreVertical className="h-4 w-4" />
            </button>
            {isMenuOpen && (
              <div
                role="menu"
                className="absolute right-0 mt-1 w-48 bg-popover border rounded-md shadow-lg z-10"
                onKeyDown={(e) => {
                  if (e.key === "Escape") onToggleMenu();
                }}
              >
                <button
                  role="menuitem"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onEdit(resume);
                  }}
                  className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
                >
                  <Edit className="h-4 w-4" />
                  {t("common.edit")}
                </button>
                <button
                  role="menuitem"
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onDelete(resume);
                  }}
                  disabled={resume.can_delete === false}
                  className={`flex items-center gap-2 w-full px-3 py-2 text-sm text-left ${
                    resume.can_delete !== false
                      ? "hover:bg-accent text-destructive"
                      : "opacity-50 cursor-not-allowed"
                  }`}
                  title={
                    resume.can_delete === false
                      ? t("resumes.cannotDeleteInUse")
                      : ""
                  }
                >
                  <Trash2 className="h-4 w-4" />
                  {t("common.delete")}
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Badges */}
        <div className="flex items-center gap-2 mt-2">
          {resume.is_active ? (
            <div className="flex items-center gap-1 text-xs font-medium text-green-600 bg-green-50 dark:bg-green-900/30 dark:text-green-400 px-2 py-1 rounded">
              <CheckCircle className="h-3 w-3" />
              {t("common.active")}
            </div>
          ) : (
            <div className="flex items-center gap-1 text-xs font-medium text-muted-foreground bg-muted px-2 py-1 rounded">
              <XCircle className="h-3 w-3" />
              {t("common.inactive")}
            </div>
          )}
          <div className="flex items-center gap-1 text-xs font-medium text-blue-600 bg-blue-50 dark:bg-blue-900/30 dark:text-blue-400 px-2 py-1 rounded">
            <Upload className="h-3 w-3" />
            {t("resumes.typeUploaded")}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {/* File Access Section */}
        <div className="space-y-2">
          {resume.storage_type === "s3" ? (
            <>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Cloud className="h-3 w-3" />
                <span>{t("resumes.cloudStorage")}</span>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => downloadMutation.mutate(resume.id)}
                disabled={downloadMutation.isPending}
                className="w-full"
              >
                <Download className="h-3 w-3 mr-2" />
                {downloadMutation.isPending
                  ? t("resumes.generatingLink")
                  : t("resumes.downloadResume")}
              </Button>
            </>
          ) : resume.file_url ? (
            <>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <LinkIcon className="h-3 w-3" />
                <span>{t("resumes.externalUrl")}</span>
              </div>
              <a
                href={resume.file_url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-center gap-2 text-sm hover:underline w-full px-4 py-2 border rounded-md hover:bg-accent transition-colors"
              >
                <ExternalLink className="h-3 w-3" />
                {t("resumes.viewResume")}
              </a>
            </>
          ) : (
            <div className="text-sm text-muted-foreground italic">
              {t("resumes.noFileAttached")}
            </div>
          )}
        </div>

        {/* Usage Indicator */}
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Briefcase className="h-4 w-4" />
          <span>
            {(resume.applications_count ?? 0) === 0
              ? t("resumes.notUsedYet")
              : t("resumes.usedInApplications", {
                  count: resume.applications_count,
                })}
          </span>
        </div>

        <div className="text-sm text-muted-foreground">
          {t("resumes.created")}{" "}
          {format(new Date(resume.created_at), "PPP", {
            locale: dateLocale,
          })}
        </div>
      </CardContent>
    </Card>
  );
}
