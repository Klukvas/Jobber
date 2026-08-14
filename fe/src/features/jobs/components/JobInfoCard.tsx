import { useTranslation } from "react-i18next";
import { formatDistanceToNow } from "date-fns";
import { Calendar, ExternalLink } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/shared/ui/Card";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { StatusBadge } from "@/shared/ui/StatusBadge";
import { CompanySelectWithQuickAdd } from "@/features/jobs/components/CompanySelectWithQuickAdd";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import type { CompanyDTO, JobDTO } from "@/shared/types/api";
import type { EditableFields } from "@/features/jobs/hooks/useJobDetailMutations";

interface JobInfoCardProps {
  readonly job: JobDTO;
  readonly fields: EditableFields;
  readonly companies: CompanyDTO[];
  readonly onFieldChange: (key: keyof EditableFields, value: string) => void;
}

function isValidUrl(url: string): boolean {
  try {
    const protocol = new URL(url).protocol;
    return protocol === "http:" || protocol === "https:";
  } catch {
    return false;
  }
}

export function JobInfoCard({
  job,
  fields,
  companies,
  onFieldChange,
}: JobInfoCardProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <Input
            value={fields.title}
            onChange={(e) => onFieldChange("title", e.target.value)}
            className="text-2xl font-bold border-transparent hover:border-input focus:border-input bg-transparent h-auto py-1 px-2 -ml-2"
            placeholder={t("jobs.titlePlaceholder")}
          />
          <StatusBadge status={job.status} />
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
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
              className="flex-1"
            />
            {fields.url && isValidUrl(fields.url) && (
              <a
                href={fields.url}
                target="_blank"
                rel="noopener noreferrer"
                className="shrink-0 text-primary hover:text-primary/80"
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            )}
          </div>
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
