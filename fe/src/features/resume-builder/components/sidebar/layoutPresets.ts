import type { LayoutMode, ColumnPlacement } from "@/shared/types/resume-builder";

export interface LayoutPreset {
  layout_mode: LayoutMode;
  sidebar_width: number;
  assignments: Record<string, ColumnPlacement>;
}

const SIDEBAR_SECTIONS: Record<string, ColumnPlacement> = {
  contact: "sidebar",
  skills: "sidebar",
  languages: "sidebar",
  certifications: "sidebar",
};

const MAIN_SECTIONS: Record<string, ColumnPlacement> = {
  summary: "main",
  experience: "main",
  education: "main",
  projects: "main",
  volunteering: "main",
  custom: "main",
};

const ALL_MAIN: Record<string, ColumnPlacement> = {
  contact: "main",
  summary: "main",
  experience: "main",
  education: "main",
  skills: "main",
  languages: "main",
  certifications: "main",
  projects: "main",
  volunteering: "main",
  custom: "main",
};

export const LAYOUT_PRESETS: Record<string, LayoutPreset> = {
  single: {
    layout_mode: "single",
    sidebar_width: 35,
    assignments: ALL_MAIN,
  },
  "double-left": {
    layout_mode: "double-left",
    sidebar_width: 35,
    assignments: { ...MAIN_SECTIONS, ...SIDEBAR_SECTIONS },
  },
  "double-right": {
    layout_mode: "double-right",
    sidebar_width: 35,
    assignments: { ...MAIN_SECTIONS, ...SIDEBAR_SECTIONS },
  },
};
