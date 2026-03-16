import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type {
  LayoutMode,
  ColumnPlacement,
  SectionOrderDTO,
} from "@/shared/types/resume-builder";
import { LAYOUT_PRESETS } from "./layoutPresets";
import { LayoutPresetThumbnail } from "./LayoutPresetThumbnail";
import { SECTION_LABEL_KEYS } from "../../constants/sectionLabels";

const PRESET_MODES: { mode: LayoutMode; labelKey: string }[] = [
  { mode: "single", labelKey: "resumeBuilder.layout.singleColumn" },
  { mode: "double-left", labelKey: "resumeBuilder.layout.twoColumnLeft" },
  { mode: "double-right", labelKey: "resumeBuilder.layout.twoColumnRight" },
  { mode: "custom", labelKey: "resumeBuilder.layout.custom" },
];

export function LayoutConfigurator() {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);
  const setSectionOrder = useResumeBuilderStore((s) => s.setSectionOrder);

  if (!resume) return null;

  const currentMode = resume.layout_mode ?? "single";
  const sidebarWidth = resume.sidebar_width ?? 35;

  const applyPreset = (mode: LayoutMode) => {
    if (mode === "custom") {
      updateDesign({ layout_mode: "custom" });
      return;
    }

    const preset = LAYOUT_PRESETS[mode];
    if (!preset) return;

    updateDesign({
      layout_mode: preset.layout_mode,
      sidebar_width: preset.sidebar_width,
    });

    if (resume.section_order.length > 0) {
      const updatedOrder: SectionOrderDTO[] = resume.section_order.map(
        (entry) => ({
          ...entry,
          column: (preset.assignments[entry.section_key] ??
            "main") as ColumnPlacement,
        }),
      );
      setSectionOrder(updatedOrder);
    }
  };

  const toggleSectionColumn = (sectionKey: string) => {
    const updatedOrder: SectionOrderDTO[] = resume.section_order.map(
      (entry) => {
        if (entry.section_key !== sectionKey) return entry;
        const newCol: ColumnPlacement =
          entry.column === "sidebar" ? "main" : "sidebar";
        return { ...entry, column: newCol };
      },
    );
    setSectionOrder(updatedOrder);
  };

  const isTwoColumn = currentMode !== "single";

  return (
    <div className="space-y-4">
      {/* Layout Presets */}
      <div className="space-y-1.5">
        <label className="text-xs font-medium text-muted-foreground">
          {t("resumeBuilder.layout.layoutPresets")}
        </label>
        <div className="grid grid-cols-4 gap-1.5">
          {PRESET_MODES.map(({ mode, labelKey }) => (
            <LayoutPresetThumbnail
              key={mode}
              mode={mode}
              isActive={currentMode === mode}
              onClick={() => applyPreset(mode)}
              label={t(labelKey)}
            />
          ))}
        </div>
      </div>

      {/* Sidebar Width Slider */}
      {isTwoColumn && (
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.layout.sidebarWidth")}: {sidebarWidth}%
          </label>
          <input
            type="range"
            min={25}
            max={50}
            value={sidebarWidth}
            onChange={(e) =>
              updateDesign({ sidebar_width: parseInt(e.target.value, 10) })
            }
            className="w-full accent-primary"
          />
          <div className="flex justify-between text-[10px] text-muted-foreground/60">
            <span>25%</span>
            <span>50%</span>
          </div>
        </div>
      )}

      {/* Custom Section Assignment */}
      {currentMode === "custom" && (
        <div className="space-y-1.5">
          <label className="text-xs font-medium text-muted-foreground">
            {t("resumeBuilder.layout.sectionAssignment")}
          </label>
          <div className="space-y-1">
            {[...resume.section_order]
              .sort((a, b) => a.sort_order - b.sort_order)
              .map((entry) => (
                <div
                  key={entry.section_key}
                  className="flex items-center justify-between rounded border px-2.5 py-1.5"
                >
                  <span className="text-xs text-foreground">
                    {t(
                      SECTION_LABEL_KEYS[entry.section_key] ??
                        entry.section_key,
                    )}
                  </span>
                  <button
                    type="button"
                    onClick={() => toggleSectionColumn(entry.section_key)}
                    className={`rounded-full px-2.5 py-0.5 text-[10px] font-medium transition-colors ${
                      entry.column === "sidebar"
                        ? "bg-primary/10 text-primary"
                        : "bg-muted text-muted-foreground"
                    }`}
                  >
                    {entry.column === "sidebar"
                      ? t("resumeBuilder.layout.sidebarColumn")
                      : t("resumeBuilder.layout.mainColumn")}
                  </button>
                </div>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}
