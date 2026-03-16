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

  if (editable) {
    return (
      <EditableTextarea
        value={summary?.content ?? ""}
        onChange={(content) => setup.updateSummary({ content })}
        placeholder="Write a brief summary of your professional background..."
        className={`${config.textSize} ${config.leadingClass} text-gray-700`}
      />
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
