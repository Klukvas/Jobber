import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { Label } from "@/shared/ui/Label";
import type { StageTemplateDTO } from "@/shared/types/api";

interface StageSelectorProps {
  readonly templates: readonly StageTemplateDTO[];
  /** The stage-template id the card currently sits in, if any */
  readonly currentStageTemplateId?: string;
  readonly onSelect: (stageTemplateId: string) => void;
  readonly isMoving: boolean;
}

// Primary pipeline control on the detail page: a dropdown listing the user's
// stages in `order`. Selecting one moves the card to that column.
export function StageSelector({
  templates,
  currentStageTemplateId,
  onSelect,
  isMoving,
}: StageSelectorProps) {
  const { t } = useTranslation();
  const ordered = [...templates].sort((a, b) => a.order - b.order);

  return (
    <div className="space-y-2">
      <Label htmlFor="stage-selector">{t("jobs.moveToStage")}</Label>
      <div className="flex items-center gap-2">
        <select
          id="stage-selector"
          value={currentStageTemplateId ?? ""}
          onChange={(e) => {
            const next = e.target.value;
            if (next && next !== currentStageTemplateId) onSelect(next);
          }}
          disabled={isMoving || ordered.length === 0}
          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        >
          {!currentStageTemplateId && (
            <option value="">{t("jobs.board.noStage")}</option>
          )}
          {ordered.map((tpl) => (
            <option key={tpl.id} value={tpl.id}>
              {tpl.name}
            </option>
          ))}
        </select>
        {isMoving && (
          <Loader2 className="h-4 w-4 flex-shrink-0 animate-spin text-muted-foreground" />
        )}
      </div>
      {ordered.length === 0 && (
        <p className="text-xs text-muted-foreground">
          {t("jobs.noStageTemplates")}
        </p>
      )}
    </div>
  );
}
