import { useState, type CSSProperties, type ReactNode } from "react";
import { Plus, X, ArrowUp, ArrowDown } from "lucide-react";
import { cn } from "@/shared/lib/utils";

interface SectionHeaderProps {
  readonly title: string;
  readonly color: string;
  readonly textColor?: string;
  readonly onAdd?: () => void;
  readonly onRemoveSection?: () => void;
  readonly onMoveUp?: () => void;
  readonly onMoveDown?: () => void;
  readonly isEmpty?: boolean;
  readonly emptyPlaceholder?: string;
  readonly editable?: boolean;
  readonly children?: ReactNode;
  readonly className?: string;
  readonly style?: CSSProperties;
  readonly variant?:
    | "professional"
    | "modern"
    | "minimal"
    | "executive"
    | "creative"
    | "compact"
    | "elegant"
    | "iconic"
    | "bold"
    | "accent"
    | "timeline"
    | "vivid";
}

const headerClasses: Record<string, string> = {
  professional:
    "mb-2 border-b-2 pb-1 text-sm font-bold uppercase tracking-wider",
  modern: "mb-2 border-b-2 pb-1 text-sm font-bold uppercase tracking-wider",
  minimal: "mb-3 border-b pb-2 text-xs font-semibold uppercase tracking-widest",
  executive:
    "mb-2 border-b pb-1 text-xs font-semibold uppercase tracking-[0.2em]",
  creative: "mb-2 border-l-4 pl-2 text-sm font-bold",
  compact:
    "mb-1 border-b pb-0.5 text-[10px] font-bold uppercase tracking-wider",
  elegant: "mb-1.5 border-b pb-1 flex items-center gap-1.5 text-sm font-bold",
  iconic: "mb-2 border-b-2 pb-1 text-sm font-bold",
  bold: "mb-2 rounded px-2 py-1 text-sm font-bold uppercase tracking-wider",
  accent: "mb-2 border-l-4 bg-gray-50 px-3 py-1 text-sm font-semibold",
  timeline: "mb-2 border-b-2 pb-1 text-sm font-bold uppercase tracking-wider",
  vivid:
    "mb-2 inline-block rounded-full px-3 py-1 text-xs font-bold uppercase tracking-wider",
};

export function SectionHeader({
  title,
  color,
  textColor,
  onAdd,
  onRemoveSection,
  onMoveUp,
  onMoveDown,
  isEmpty = false,
  emptyPlaceholder,
  editable = true,
  children,
  className,
  style,
  variant = "professional",
}: SectionHeaderProps) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <div
      className={cn(
        "group/section relative mb-4",
        editable && "-ml-10 pl-10",
        className,
      )}
      style={style}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* Left-side section controls (novoresume-style) */}
      {editable && (
        <div
          className={cn(
            "absolute left-0 top-0 flex flex-col items-center gap-0.5 rounded-md bg-white py-1 shadow-md border border-gray-200 transition-opacity",
            isHovered ? "opacity-100" : "opacity-0 pointer-events-none",
            "group-focus-within/section:opacity-100 group-focus-within/section:pointer-events-auto",
          )}
        >
          {onMoveUp && (
            <button
              onClick={onMoveUp}
              className="flex h-5 w-5 items-center justify-center text-gray-400 transition-colors hover:text-gray-700"
              aria-label="Move section up"
            >
              <ArrowUp className="h-3 w-3" />
            </button>
          )}
          {onMoveDown && (
            <button
              onClick={onMoveDown}
              className="flex h-5 w-5 items-center justify-center text-gray-400 transition-colors hover:text-gray-700"
              aria-label="Move section down"
            >
              <ArrowDown className="h-3 w-3" />
            </button>
          )}
          {onRemoveSection && (
            <button
              onClick={onRemoveSection}
              className="flex h-5 w-5 items-center justify-center text-gray-400 transition-colors hover:text-red-500"
              aria-label={`Remove ${title.toLowerCase()} section`}
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </div>
      )}

      <div className="flex items-center justify-between">
        <h2
          className={headerClasses[variant]}
          style={
            variant === "bold" || variant === "vivid"
              ? {
                  backgroundColor: color,
                  color:
                    textColor && textColor !== color ? textColor : "#ffffff",
                }
              : { borderColor: color, color: textColor ?? color }
          }
        >
          {title}
        </h2>
        {editable && onAdd && (
          <button
            onClick={onAdd}
            className={cn(
              "flex h-5 w-5 items-center justify-center rounded-full transition-all hover:bg-blue-100",
              isHovered ? "opacity-100" : "opacity-0 pointer-events-none",
              "group-focus-within/section:opacity-100 group-focus-within/section:pointer-events-auto",
            )}
            style={{ color }}
            aria-label={`Add ${title.toLowerCase()}`}
          >
            <Plus className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {children}

      {isEmpty && editable && onAdd && (
        <button
          onClick={onAdd}
          className="mt-1 flex items-center gap-1 text-xs italic text-gray-400 hover:text-gray-600 transition-colors"
          aria-label={emptyPlaceholder ?? `Add ${title.toLowerCase()}`}
        >
          <Plus className="h-3 w-3" />
          {emptyPlaceholder ?? `Add ${title.toLowerCase()}`}
        </button>
      )}
    </div>
  );
}
