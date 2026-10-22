import { useId } from "react";
import { useTranslation } from "react-i18next";
import { EditableTextarea } from "../../../inline/EditableTextarea";
import type { SectionContentProps } from "./types";

/**
 * SummaryContent renders the body of the summary section (textarea or text).
 * The heading, controls, and hover highlight are handled by SectionHeader
 * in TemplateSections, just like all other sections.
 */
export function SummaryContent({
  setup,
  config,
  editable,
}: SectionContentProps) {
  const { summary } = setup;
  const { t } = useTranslation();
  const helpId = useId();
  const content = summary?.content ?? "";
  const maxLength = 5000;
  const tooLong = content.length > maxLength;

  if (editable) {
    return (
      <div>
        <EditableTextarea
          value={content}
          maxLength={maxLength}
          aria-describedby={helpId}
          onChange={(content) => setup.updateSummary({ content })}
          placeholder="Write a brief summary of your professional background..."
          className={`${config.textSize} ${config.leadingClass} text-gray-700`}
        />
        <p
          id={helpId}
          role={tooLong ? "alert" : undefined}
          className={`mt-1 text-xs ${tooLong ? "text-destructive" : "text-muted-foreground"}`}
        >
          {t(
            tooLong
              ? "resumeBuilder.summaryTooLong"
              : "resumeBuilder.summaryLimit",
            { count: content.length, max: maxLength },
          )}
        </p>
      </div>
    );
  }

  if (!summary?.content) return null;

  return (
    <p
      className={`whitespace-pre-line ${config.textSize} ${config.leadingClass} text-gray-700`}
    >
      {summary.content}
    </p>
  );
}
