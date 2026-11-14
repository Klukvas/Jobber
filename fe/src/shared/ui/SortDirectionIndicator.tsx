import { useTranslation } from "react-i18next";
import { ArrowDown, ArrowUp } from "lucide-react";

interface SortDirectionIndicatorProps {
  readonly direction: "asc" | "desc";
}

/**
 * The arrow on the active sort button, plus the same fact in words for a
 * screen reader, which got "Last Activity" with no hint of the direction.
 */
export function SortDirectionIndicator({
  direction,
}: SortDirectionIndicatorProps) {
  const { t } = useTranslation();
  const Icon = direction === "desc" ? ArrowDown : ArrowUp;
  return (
    <>
      <Icon className="h-3 w-3 ml-1" aria-hidden="true" />
      <span className="sr-only">
        {`, ${t(direction === "desc" ? "common.sortedDescending" : "common.sortedAscending")}`}
      </span>
    </>
  );
}
