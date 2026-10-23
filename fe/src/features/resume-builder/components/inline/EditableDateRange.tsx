import { useTranslation } from "react-i18next";
import {
  useState,
  useId,
  useRef,
  useCallback,
  useEffect,
  type CSSProperties,
} from "react";
import { cn } from "@/shared/lib/utils";

interface EditableDateRangeProps {
  readonly startDate: string;
  readonly endDate: string;
  readonly isCurrent: boolean;
  readonly onStartDateChange: (value: string) => void;
  readonly onEndDateChange: (value: string) => void;
  readonly onIsCurrentChange: (value: boolean) => void;
  readonly style?: CSSProperties;
  readonly className?: string;
  readonly editable?: boolean;
  readonly currentLabel?: string;
}

function formatDate(date: string): string {
  if (!date) return "";
  const d = new Date(date + "T00:00:00");
  if (isNaN(d.getTime())) return date;
  return d.toLocaleDateString("en-US", { month: "short", year: "numeric" });
}

export function EditableDateRange({
  startDate,
  endDate,
  isCurrent,
  onStartDateChange,
  onEndDateChange,
  onIsCurrentChange,
  style,
  className,
  editable = true,
  currentLabel = "Present",
}: EditableDateRangeProps) {
  const { t } = useTranslation();
  const inputId = useId();
  const triggerRef = useRef<HTMLSpanElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  const displayStart = formatDate(startDate);
  const displayEnd = isCurrent ? currentLabel : formatDate(endDate);
  const displayText =
    displayStart || displayEnd
      ? `${displayStart}${displayStart && displayEnd ? " — " : ""}${displayEnd}`
      : "";

  const toggle = useCallback(() => {
    if (!editable) return;
    setIsOpen((prev) => !prev);
  }, [editable]);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;
    function handleClick(e: MouseEvent) {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(e.target as Node)
      ) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [isOpen]);

  if (!editable && !displayText) return null;

  return (
    <span data-date-range className="relative inline-block" style={style}>
      <span
        ref={triggerRef}
        data-date-trigger
        role={editable ? "button" : undefined}
        tabIndex={editable ? 0 : undefined}
        aria-expanded={editable ? isOpen : undefined}
        aria-label={editable ? (displayText || t("resumeBuilder.inline.addDates")) : undefined}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") { event.preventDefault(); toggle(); }
        }}
        onClick={toggle}
        className={cn(
          className,
          editable &&
            "cursor-text transition-colors rounded hover:bg-blue-50/50",
          !displayText && editable && "italic text-gray-400",
        )}
      >
        {displayText || t("resumeBuilder.inline.addDates")}
      </span>

      {isOpen && (
        <div
          ref={popoverRef}
          data-date-popover
          onKeyDown={(event) => { if (event.key === "Escape") { event.stopPropagation(); setIsOpen(false); triggerRef.current?.focus(); } }}
          className="absolute right-0 top-full z-50 mt-1 w-56 rounded-md border bg-white p-3 shadow-lg"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="space-y-2.5">
            <div>
              <label htmlFor={`${inputId}-start`} className="mb-0.5 block text-[10px] font-medium text-gray-600">
                {t("resumeBuilder.inline.startDate")}
              </label>
              <input
                id={`${inputId}-start`}
                type="date"
                value={startDate}
                onChange={(e) => onStartDateChange(e.target.value)}
                className="w-full rounded border border-gray-200 px-2 py-1 text-xs"
              />
            </div>
            {!isCurrent && (
              <div>
                <label htmlFor={`${inputId}-end`} className="mb-0.5 block text-[10px] font-medium text-gray-600">
                  {t("resumeBuilder.inline.endDate")}
                </label>
                <input
                  id={`${inputId}-end`}
                  type="date"
                  value={endDate}
                  onChange={(e) => onEndDateChange(e.target.value)}
                  className="w-full rounded border border-gray-200 px-2 py-1 text-xs"
                />
              </div>
            )}
            <label className="flex items-center gap-1.5 text-[10px]">
              <input
                type="checkbox"
                checked={isCurrent}
                onChange={(e) => {
                  onIsCurrentChange(e.target.checked);
                  if (e.target.checked) onEndDateChange("");
                }}
                className="rounded"
              />
              <span className="text-gray-700">{t("resumeBuilder.inline.currentlyHere")}</span>
            </label>
          </div>
        </div>
      )}
    </span>
  );
}
