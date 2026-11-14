import { useTranslation } from "react-i18next";
import { formatDistanceToNow } from "date-fns";
import { Calendar, ExternalLink } from "lucide-react";
import { FIELD_LIMITS } from "@/shared/lib/fieldLimits";
import { TOUCH_ICON_TARGET } from "@/shared/lib/touchTargets";
import { Card, CardContent, CardHeader } from "@/shared/ui/Card";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { CompanySelectWithQuickAdd } from "@/features/jobs/components/CompanySelectWithQuickAdd";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import type { CompanyDTO, JobDTO } from "@/shared/types/api";
import type { EditableFields } from "@/features/jobs/hooks/useJobDetailMutations";
import { isHttpUrl, isSaveableJobUrl } from "@/features/jobs/lib/jobUrl";
import { stageColor } from "@/features/jobs/lib/stageColors";

interface JobInfoCardProps {
  readonly job: JobDTO;
  readonly fields: EditableFields;
  readonly companies: CompanyDTO[];
  readonly onFieldChange: (key: keyof EditableFields, value: string) => void;
}

export function JobInfoCard({
  job,
  fields,
  companies,
  onFieldChange,
}: JobInfoCardProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const urlIsSaveable = isSaveableJobUrl(fields.url);

  return (
    <Card>
      <CardHeader>
        {/* Stacked on a phone: side by side, the stage badge took half the row
            and cut an ordinary title down to its first word. */}
        <div className="flex flex-col items-start gap-2 sm:flex-row sm:justify-between sm:gap-4">
          <Input
            aria-label={t("jobs.title_field")}
            aria-invalid={!fields.title.trim()}
            aria-describedby={!fields.title.trim() ? "detail-title-error" : undefined}
            value={fields.title}
            onChange={(e) => onFieldChange("title", e.target.value)}
            maxLength={FIELD_LIMITS.title}
            className="w-full min-w-0 text-xl font-bold sm:text-2xl border-transparent hover:border-input focus:border-input bg-transparent h-auto py-1 px-2 -ml-2"
            placeholder={t("jobs.titlePlaceholder")}
          />
          {/* Same colours as the list and the board, so Rejected stays red. */}
          <span
            className={`inline-flex flex-shrink-0 items-center rounded-full px-3 py-1 text-sm font-medium ${
              stageColor(job.current_stage_name, job.is_archived).pill
            }`}
          >
            {job.is_archived
              ? t("jobs.archived")
              : (job.current_stage_name ?? t("jobs.board.noStage"))}
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {!fields.title.trim() && (
          <p id="detail-title-error" role="alert" className="text-sm text-destructive">{t("jobs.titleRequired")}</p>
        )}
        {/* These fields are dirty-tracked and committed with Save in the header,
            unlike the pipeline's resume selector, which writes immediately.
            Saying so is cheaper than making a customer guess. */}
        <p className="text-xs text-muted-foreground">
          {t("jobs.fieldsNeedSaveHint")}
        </p>

        <CompanySelectWithQuickAdd
          companies={companies}
          value={fields.company_id}
          onChange={(val) => onFieldChange("company_id", val)}
        />

        <div className="space-y-2">
          <Label htmlFor="source">{t("jobs.source")}</Label>
          <Input
            id="source"
            value={fields.source}
            onChange={(e) => onFieldChange("source", e.target.value)}
            placeholder={t("jobs.sourcePlaceholder")}
            maxLength={FIELD_LIMITS.source}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="url">{t("jobs.url")}</Label>
          <div className="flex items-center gap-2">
            <Input
              id="url"
              type="url"
              value={fields.url}
              onChange={(e) => onFieldChange("url", e.target.value)}
              placeholder="https://example.com/jobs/123"
              maxLength={FIELD_LIMITS.url}
              className="flex-1"
              aria-invalid={!urlIsSaveable}
              aria-describedby={!urlIsSaveable ? "detail-url-error" : undefined}
            />
            {fields.url && isHttpUrl(fields.url) && (
              <a
                href={fields.url}
                target="_blank"
                rel="noopener noreferrer"
                aria-label={t("jobs.openUrl")}
                title={t("jobs.openUrl")}
                className={`${TOUCH_ICON_TARGET} shrink-0 rounded-md text-primary-ink hover:text-primary-ink/80`}
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            )}
          </div>
          {!urlIsSaveable && (
            <p
              id="detail-url-error"
              role="alert"
              className="text-sm text-destructive"
            >
              {t("errors.invalidUrl")}
            </p>
          )}
        </div>

        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Calendar className="h-4 w-4" />
          <span>
            {t("jobs.createdDate")}{" "}
            {formatDistanceToNow(new Date(job.created_at), {
              addSuffix: true,
              locale: dateLocale,
            })}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}
