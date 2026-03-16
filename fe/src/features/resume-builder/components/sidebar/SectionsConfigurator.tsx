import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Check } from "lucide-react";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import type {
  LayoutMode,
  ColumnPlacement,
  SectionOrderDTO,
} from "@/shared/types/resume-builder";
import { LAYOUT_PRESETS } from "./layoutPresets";
import { cn } from "@/shared/lib/utils";
import { SECTION_LABEL_KEYS } from "../../constants/sectionLabels";

type Tab = "predefined" | "custom";

/** Top-level section toggles (shown as checkboxes above tabs) */
const TOGGLEABLE_SECTIONS = ["summary", "contact"] as const;

/** Sections shown in preset cards */
const ALL_SECTIONS = [
  "education",
  "experience",
  "skills",
  "projects",
  "certifications",
  "volunteering",
  "languages",
  "custom",
] as const;

interface PresetLayout {
  label: string;
  mode: LayoutMode;
  main: string[];
  sidebar: string[];
}

const PRESETS: PresetLayout[] = [
  {
    label: "1",
    mode: "double-left",
    main: ["summary", "experience", "education"],
    sidebar: [
      "contact",
      "skills",
      "languages",
      "certifications",
      "projects",
      "custom",
    ],
  },
  {
    label: "2",
    mode: "double-left",
    main: ["summary", "experience", "education", "projects"],
    sidebar: [
      "contact",
      "skills",
      "certifications",
      "languages",
      "volunteering",
      "custom",
    ],
  },
  {
    label: "3",
    mode: "double-right",
    main: ["summary", "experience", "education", "volunteering"],
    sidebar: [
      "contact",
      "skills",
      "projects",
      "certifications",
      "languages",
      "custom",
    ],
  },
];

export function SectionsConfigurator() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<Tab>("predefined");

  const resume = useResumeBuilderStore((s) => s.resume);
  const updateDesign = useResumeBuilderStore((s) => s.updateDesign);
  const setSectionOrder = useResumeBuilderStore((s) => s.setSectionOrder);

  if (!resume) return null;

  const sectionOrder = resume.section_order;

  const isSectionVisible = (key: string) =>
    sectionOrder.find((s) => s.section_key === key)?.is_visible ?? false;

  const normalizeSortOrder = (entries: SectionOrderDTO[]): SectionOrderDTO[] =>
    [...entries]
      .sort((a, b) => a.sort_order - b.sort_order)
      .map((entry, idx) => ({ ...entry, sort_order: idx }));

  const toggleSectionVisibility = (key: string) => {
    const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
      if (entry.section_key !== key) return entry;
      return { ...entry, is_visible: !entry.is_visible };
    });
    setSectionOrder(normalizeSortOrder(updated));
  };

  const applyPreset = (preset: PresetLayout) => {
    const layoutPreset = LAYOUT_PRESETS[preset.mode];
    if (!layoutPreset) return;

    updateDesign({
      layout_mode: preset.mode,
      sidebar_width: layoutPreset.sidebar_width,
    });

    // Build assignments from preset
    const assignments: Record<string, ColumnPlacement> = {};
    for (const key of preset.main) {
      assignments[key] = "main";
    }
    for (const key of preset.sidebar) {
      assignments[key] = "sidebar";
    }

    // Make all listed sections visible, preserve contact/summary state
    const presetSections = new Set([...preset.main, ...preset.sidebar]);
    const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
      if (presetSections.has(entry.section_key)) {
        const mainIdx = preset.main.indexOf(entry.section_key);
        const sidebarIdx = preset.sidebar.indexOf(entry.section_key);
        const newOrder =
          mainIdx >= 0
            ? mainIdx
            : preset.main.length + (sidebarIdx >= 0 ? sidebarIdx : 99);
        return {
          ...entry,
          is_visible: true,
          column: (assignments[entry.section_key] ?? "main") as ColumnPlacement,
          sort_order: newOrder + 2, // offset for contact(0) and summary(1)
        };
      }
      return entry;
    });
    setSectionOrder(normalizeSortOrder(updated));
  };

  const toggleCustomColumn = (key: string) => {
    const updated: SectionOrderDTO[] = sectionOrder.map((entry) => {
      if (entry.section_key !== key) return entry;
      const newCol: ColumnPlacement =
        entry.column === "sidebar" ? "main" : "sidebar";
      return { ...entry, column: newCol };
    });
    setSectionOrder(normalizeSortOrder(updated));
  };

  const currentMode = resume.layout_mode ?? "single";
  const sortedVisible = [...sectionOrder]
    .filter(
      (s) => s.is_visible && !["contact", "summary"].includes(s.section_key),
    )
    .sort((a, b) => a.sort_order - b.sort_order);

  const mainSections = sortedVisible.filter((s) => s.column !== "sidebar");
  const sidebarSections = sortedVisible.filter((s) => s.column === "sidebar");

  return (
    <div className="space-y-4">
      {/* Section visibility toggles */}
      <div className="flex flex-wrap gap-2">
        {TOGGLEABLE_SECTIONS.map((key) => {
          const visible = isSectionVisible(key);
          return (
            <button
              key={key}
              onClick={() => toggleSectionVisibility(key)}
              className={cn(
                "flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors",
                visible
                  ? "bg-primary/10 text-primary"
                  : "bg-muted text-muted-foreground",
              )}
            >
              <div
                className={cn(
                  "flex h-4 w-4 items-center justify-center rounded border transition-colors",
                  visible
                    ? "border-primary bg-primary text-white"
                    : "border-border bg-background",
                )}
              >
                {visible && <Check className="h-3 w-3" />}
              </div>
              {t(SECTION_LABEL_KEYS[key] ?? key)}
            </button>
          );
        })}
        {/* Toggles for other sections */}
        {ALL_SECTIONS.map((key) => {
          const visible = isSectionVisible(key);
          return (
            <button
              key={key}
              onClick={() => toggleSectionVisibility(key)}
              className={cn(
                "flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors",
                visible
                  ? "bg-primary/10 text-primary"
                  : "bg-muted text-muted-foreground",
              )}
            >
              <div
                className={cn(
                  "flex h-4 w-4 items-center justify-center rounded border transition-colors",
                  visible
                    ? "border-primary bg-primary text-white"
                    : "border-border bg-background",
                )}
              >
                {visible && <Check className="h-3 w-3" />}
              </div>
              {t(SECTION_LABEL_KEYS[key] ?? key)}
            </button>
          );
        })}
      </div>

      {/* Tabs: Predefined / Custom */}
      <div className="flex gap-1 rounded-lg bg-muted p-1">
        <button
          onClick={() => setActiveTab("predefined")}
          className={cn(
            "flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition-colors",
            activeTab === "predefined"
              ? "bg-primary text-white shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {t("resumeBuilder.sections.predefined")}
        </button>
        <button
          onClick={() => setActiveTab("custom")}
          className={cn(
            "flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition-colors",
            activeTab === "custom"
              ? "bg-primary text-white shadow-sm"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {t("resumeBuilder.sections.customTab")}
        </button>
      </div>

      {/* Predefined tab — preset cards */}
      {activeTab === "predefined" && (
        <div className="space-y-2">
          {PRESETS.map((preset) => (
            <button
              key={preset.label}
              onClick={() => applyPreset(preset)}
              className="w-full rounded-lg border-2 border-border p-3 transition-colors hover:border-primary/50"
            >
              <div className="flex gap-2">
                {/* Main column */}
                <div className="flex flex-1 flex-col gap-1">
                  {preset.main.map((key) => (
                    <div
                      key={key}
                      className="rounded bg-gray-600 px-2 py-1.5 text-[10px] font-medium text-white"
                    >
                      {t(SECTION_LABEL_KEYS[key] ?? key)}
                    </div>
                  ))}
                </div>
                {/* Sidebar column */}
                <div className="flex w-[45%] flex-col gap-1">
                  {preset.sidebar.map((key) => (
                    <div
                      key={key}
                      className="rounded bg-gray-500 px-2 py-1 text-[10px] font-medium text-white"
                    >
                      {t(SECTION_LABEL_KEYS[key] ?? key)}
                    </div>
                  ))}
                </div>
              </div>
            </button>
          ))}
        </div>
      )}

      {/* Custom tab — interactive section assignment */}
      {activeTab === "custom" && (
        <div className="space-y-3">
          {/* Sidebar width slider */}
          {currentMode !== "single" && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                {t("resumeBuilder.layout.sidebarWidth")}:{" "}
                {resume.sidebar_width ?? 35}%
              </label>
              <input
                type="range"
                min={25}
                max={50}
                value={resume.sidebar_width ?? 35}
                onChange={(e) =>
                  updateDesign({
                    sidebar_width: parseInt(e.target.value, 10),
                  })
                }
                className="w-full accent-primary"
              />
            </div>
          )}

          {/* Two-column arrangement */}
          <div className="flex gap-2">
            {/* Main column */}
            <div className="flex-1 space-y-1">
              <div className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {t("resumeBuilder.layout.mainColumn")}
              </div>
              {mainSections.map((entry) => (
                <button
                  key={entry.section_key}
                  onClick={() => toggleCustomColumn(entry.section_key)}
                  className="flex w-full items-center gap-1.5 rounded bg-gray-600 px-2 py-1.5 text-left text-[10px] font-medium text-white transition-colors hover:bg-gray-500"
                >
                  {t(
                    SECTION_LABEL_KEYS[entry.section_key] ?? entry.section_key,
                  )}
                  <span className="ml-auto text-[8px] text-gray-300">→</span>
                </button>
              ))}
            </div>
            {/* Sidebar column */}
            <div className="w-[45%] space-y-1">
              <div className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {t("resumeBuilder.layout.sidebarColumn")}
              </div>
              {sidebarSections.map((entry) => (
                <button
                  key={entry.section_key}
                  onClick={() => toggleCustomColumn(entry.section_key)}
                  className="flex w-full items-center gap-1.5 rounded bg-primary/80 px-2 py-1.5 text-left text-[10px] font-medium text-white transition-colors hover:bg-primary/70"
                >
                  <span className="text-[8px] text-white/60">←</span>
                  {t(
                    SECTION_LABEL_KEYS[entry.section_key] ?? entry.section_key,
                  )}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
