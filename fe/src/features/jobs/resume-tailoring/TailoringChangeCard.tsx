import { useId } from "react";
import { useTranslation } from "react-i18next";
import type { TailoringChange } from "@/services/resumeTailoringService";
import { Checkbox } from "@/shared/ui/Checkbox";
import { cn } from "@/shared/lib/utils";

type Props = {
  change: TailoringChange;
  checked: boolean;
  disabled: boolean;
  onCheckedChange: (checked: boolean) => void;
};

export function TailoringChangeCard({ change, checked, disabled, onCheckedChange }: Props) {
  const { t } = useTranslation();
  const id = useId();
  const title = change.kind === "experience"
    ? [change.position, change.company].filter(Boolean).join(" — ") || t("jobs.tailoring.experience")
    : t(`jobs.tailoring.${change.kind === "summary" ? "summary" : "skillOrder"}`);
  const before = change.kind === "skill_order" ? change.before.map((skill) => skill.name).join(" → ") : change.before;
  const after = change.kind === "skill_order" ? change.after.map((skill) => skill.name).join(" → ") : change.after;

  return (
    <article className={cn("min-w-0 rounded-lg border", checked && "border-primary/60 bg-primary/5")}>
      <label htmlFor={id} className="flex min-h-12 cursor-pointer items-center gap-3 border-b p-4 text-sm font-semibold">
        <Checkbox id={id} checked={checked} disabled={disabled} onCheckedChange={onCheckedChange} />
        <span className="min-w-0 break-words">{title}</span>
      </label>
      <div className="grid min-w-0 divide-y sm:grid-cols-2 sm:divide-x sm:divide-y-0">
        <div className="min-w-0 p-4">
          <p className="mb-2 text-xs font-medium text-muted-foreground">{t("jobs.tailoring.before")}</p>
          <p className="whitespace-pre-wrap break-words text-sm [overflow-wrap:anywhere]">{before || t("jobs.tailoring.emptyContent")}</p>
        </div>
        <div className="min-w-0 p-4">
          <p className="mb-2 text-xs font-medium text-foreground">{t("jobs.tailoring.after")}</p>
          <p className="whitespace-pre-wrap break-words text-sm [overflow-wrap:anywhere]">{after}</p>
        </div>
      </div>
      <div className="space-y-2 border-t p-4 text-sm">
        <p className="break-words"><span className="font-medium">{t("jobs.tailoring.reason")} </span>{change.reason}</p>
        {change.evidence.length > 0 && (
          <details className="text-muted-foreground">
            <summary className="cursor-pointer rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">{t("jobs.tailoring.evidence")}</summary>
            <ul className="mt-2 space-y-2 border-l-2 pl-3">
              {change.evidence.map((item, index) => <li className="whitespace-pre-wrap break-words [overflow-wrap:anywhere]" key={`${item.section}:${item.entry_id ?? ""}:${index}`}>{item.quote}</li>)}
            </ul>
          </details>
        )}
      </div>
    </article>
  );
}
