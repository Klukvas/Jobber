import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { Label } from "@/shared/ui/Label";
import { Textarea } from "@/shared/ui/Textarea";

const MAX_CHARS = 600;

export function SummaryEditor() {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateSummary = useResumeBuilderStore((s) => s.updateSummary);

  const content = resume?.summary?.content ?? "";

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">
        {t("resumeBuilder.sections.summary")}
      </h2>

      <div className="space-y-1.5">
        <Label htmlFor="summary">{t("resumeBuilder.summary.label")}</Label>
        <Textarea
          id="summary"
          value={content}
          onChange={(e) => updateSummary({ content: e.target.value })}
          placeholder={t("resumeBuilder.summary.placeholder")}
          rows={6}
          maxLength={MAX_CHARS}
        />
        <p className="text-right text-xs text-muted-foreground">
          {content.length}/{MAX_CHARS}
        </p>
      </div>
    </div>
  );
}
