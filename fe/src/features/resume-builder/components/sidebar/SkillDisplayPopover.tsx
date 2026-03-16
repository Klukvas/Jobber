import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { cn } from "@/shared/lib/utils";
import { SidebarPopover } from "./SidebarPopover";

interface SkillDisplayPopoverProps {
  readonly isOpen: boolean;
  readonly onClose: () => void;
  readonly fullscreen?: boolean;
}

const SKILL_DISPLAY_OPTIONS = [
  { value: "", labelKey: "templateDefault", descKey: "templateDefaultDesc" },
  { value: "text-level", labelKey: "textLevel", descKey: "textLevelDesc" },
  { value: "pill", labelKey: "pill", descKey: "pillDesc" },
  { value: "grid-level", labelKey: "gridLevel", descKey: "gridLevelDesc" },
  { value: "vertical", labelKey: "vertical", descKey: "verticalDesc" },
  { value: "text-only", labelKey: "textOnly", descKey: "textOnlyDesc" },
  { value: "dots", labelKey: "dots", descKey: "dotsDesc" },
  { value: "bar", labelKey: "bar", descKey: "barDesc" },
  { value: "square", labelKey: "square", descKey: "squareDesc" },
  { value: "star", labelKey: "star", descKey: "starDesc" },
  { value: "circle", labelKey: "circle", descKey: "circleDesc" },
  { value: "segmented", labelKey: "segmented", descKey: "segmentedDesc" },
  { value: "bubble", labelKey: "bubble", descKey: "bubbleDesc" },
] as const;

/** Tiny visual preview for each skill display mode. */
function ModePreview({
  mode,
  color,
}: {
  readonly mode: string;
  readonly color: string;
}) {
  switch (mode) {
    case "text-level":
      return (
        <div className="flex flex-wrap gap-1">
          <span className="text-[8px] text-gray-600">React (Expert)</span>
          <span className="text-[8px] text-gray-600">Node (Advanced)</span>
        </div>
      );
    case "pill":
      return (
        <div className="flex gap-1">
          <span
            className="rounded-full px-1.5 py-0.5 text-[7px] text-white"
            style={{ backgroundColor: color }}
          >
            React
          </span>
          <span
            className="rounded-full px-1.5 py-0.5 text-[7px] text-white"
            style={{ backgroundColor: color }}
          >
            Node
          </span>
        </div>
      );
    case "grid-level":
      return (
        <div className="grid grid-cols-3 gap-x-2">
          <span className="text-[7px] text-gray-600">React (Expert)</span>
          <span className="text-[7px] text-gray-600">Node (Adv)</span>
          <span className="text-[7px] text-gray-600">TS (Expert)</span>
        </div>
      );
    case "vertical":
      return (
        <div className="flex gap-3">
          <div>
            <div className="text-[8px] font-medium">React</div>
            <div className="text-[7px] opacity-60">Expert</div>
          </div>
          <div>
            <div className="text-[8px] font-medium">Node</div>
            <div className="text-[7px] opacity-60">Advanced</div>
          </div>
        </div>
      );
    case "text-only":
      return (
        <span className="text-[8px] text-gray-600">
          React &middot; Node &middot; TS
        </span>
      );
    case "dots":
      return (
        <div className="flex items-center gap-2">
          <span className="text-[8px] font-medium">React</span>
          <div className="flex gap-0.5">
            {Array.from({ length: 5 }, (_, i) => (
              <span
                key={i}
                className={`inline-block h-1.5 w-1.5 rounded-full ${i >= 4 ? "bg-gray-200" : ""}`}
                style={i < 4 ? { backgroundColor: color } : undefined}
              />
            ))}
          </div>
        </div>
      );
    case "bar":
      return (
        <div className="flex items-center gap-2">
          <span className="text-[8px] font-medium">React</span>
          <div className="h-1.5 w-12 rounded-full bg-gray-200">
            <div
              className="h-full rounded-full"
              style={{ width: "75%", backgroundColor: color }}
            />
          </div>
        </div>
      );
    case "square":
      return (
        <div className="flex items-center gap-2">
          <span className="text-[8px] font-medium">React</span>
          <div className="flex gap-0.5">
            {Array.from({ length: 5 }, (_, i) => (
              <span
                key={i}
                className={`inline-block h-1.5 w-1.5 rounded-sm ${i >= 4 ? "bg-gray-200" : ""}`}
                style={i < 4 ? { backgroundColor: color } : undefined}
              />
            ))}
          </div>
        </div>
      );
    case "star":
      return (
        <div className="flex items-center gap-2">
          <span className="text-[8px] font-medium">React</span>
          <div className="flex gap-0.5">
            {Array.from({ length: 5 }, (_, i) => (
              <span
                key={i}
                className="text-[10px] leading-none"
                style={{ color: i < 4 ? color : "#d1d5db" }}
              >
                {"\u2605"}
              </span>
            ))}
          </div>
        </div>
      );
    case "circle":
      return (
        <div className="flex items-center gap-3">
          <div className="flex flex-col items-center">
            <svg width="20" height="20" viewBox="0 0 20 20">
              <circle
                cx="10"
                cy="10"
                r="7"
                fill="none"
                stroke="#e5e7eb"
                strokeWidth="2"
              />
              <circle
                cx="10"
                cy="10"
                r="7"
                fill="none"
                stroke={color}
                strokeWidth="2"
                strokeDasharray={2 * Math.PI * 7}
                strokeDashoffset={2 * Math.PI * 7 * 0.25}
                strokeLinecap="round"
                transform="rotate(-90 10 10)"
              />
            </svg>
            <span className="text-[6px] font-medium">React</span>
          </div>
          <div className="flex flex-col items-center">
            <svg width="20" height="20" viewBox="0 0 20 20">
              <circle
                cx="10"
                cy="10"
                r="7"
                fill="none"
                stroke="#e5e7eb"
                strokeWidth="2"
              />
              <circle
                cx="10"
                cy="10"
                r="7"
                fill="none"
                stroke={color}
                strokeWidth="2"
                strokeDasharray={2 * Math.PI * 7}
                strokeDashoffset={2 * Math.PI * 7 * 0.5}
                strokeLinecap="round"
                transform="rotate(-90 10 10)"
              />
            </svg>
            <span className="text-[6px] font-medium">Node</span>
          </div>
        </div>
      );
    case "segmented":
      return (
        <div className="flex items-center gap-2">
          <span className="text-[8px] font-medium">React</span>
          <div className="flex flex-1 gap-0.5">
            {Array.from({ length: 4 }, (_, i) => (
              <div
                key={i}
                className={`h-1.5 w-3 rounded-sm ${i >= 3 ? "bg-gray-200" : ""}`}
                style={i < 3 ? { backgroundColor: color } : undefined}
              />
            ))}
          </div>
        </div>
      );
    case "bubble":
      return (
        <div className="flex gap-1">
          <span
            className="rounded-md px-1.5 py-0.5 text-[7px] font-medium"
            style={{
              backgroundColor: `color-mix(in srgb, ${color} 25%, transparent)`,
              color,
              border: `1px solid color-mix(in srgb, ${color} 30%, transparent)`,
            }}
          >
            React
          </span>
          <span
            className="rounded-md px-1.5 py-0.5 text-[7px] font-medium"
            style={{
              backgroundColor: `color-mix(in srgb, ${color} 18%, transparent)`,
              color,
              border: `1px solid color-mix(in srgb, ${color} 30%, transparent)`,
            }}
          >
            Node
          </span>
        </div>
      );
    default:
      return (
        <span className="text-[8px] italic text-muted-foreground">Auto</span>
      );
  }
}

export function SkillDisplayPopover({
  isOpen,
  onClose,
  fullscreen = false,
}: SkillDisplayPopoverProps) {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);

  const currentValue = resume?.skill_display ?? "";
  const accentColor = resume?.primary_color ?? "#2563eb";

  return (
    <SidebarPopover
      isOpen={isOpen}
      onClose={onClose}
      title={t("resumeBuilder.skillDisplay.title")}
      fullscreen={fullscreen}
    >
      <div className="grid gap-2">
        {SKILL_DISPLAY_OPTIONS.map(({ value, labelKey, descKey }) => {
          const isActive = currentValue === value;
          return (
            <button
              key={value || "__default__"}
              onClick={() => updateDesign({ skill_display: value })}
              className={cn(
                "flex flex-col gap-1.5 rounded-lg border p-3 text-left transition-colors",
                isActive
                  ? "border-primary bg-primary/5"
                  : "border-border hover:bg-muted",
              )}
            >
              <div className="flex items-center justify-between">
                <span
                  className={cn(
                    "text-xs font-medium",
                    isActive ? "text-primary" : "text-foreground",
                  )}
                >
                  {t(`resumeBuilder.skillDisplay.${labelKey}`)}
                </span>
                {isActive && (
                  <span
                    className="h-2 w-2 rounded-full"
                    style={{ backgroundColor: accentColor }}
                  />
                )}
              </div>
              <p className="text-[10px] text-muted-foreground">
                {t(`resumeBuilder.skillDisplay.${descKey}`)}
              </p>
              <div className="mt-0.5 rounded border border-gray-100 bg-gray-50 px-2 py-1.5">
                <ModePreview mode={value} color={accentColor} />
              </div>
            </button>
          );
        })}
      </div>
    </SidebarPopover>
  );
}
