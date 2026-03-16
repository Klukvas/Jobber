import {
  FileText,
  Briefcase,
  GraduationCap,
  Wrench,
  Languages as LanguagesIcon,
  Award,
  FolderOpen,
  Heart,
  LayoutList,
} from "lucide-react";
import type { TemplateConfig } from "./templateConfig";

export const professionalConfig: TemplateConfig = {
  variant: "professional",
  summaryTitle: "Professional Summary",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "text-level",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const modernConfig: TemplateConfig = {
  variant: "modern",
  summaryTitle: "About Me",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "vertical",
    containerClass: "space-y-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "space-y-1",
  },
  renderContactInSwitch: true,
  inputClassName: "text-white placeholder:text-white/60",
  summaryMb: "mb-5",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1.5",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const minimalConfig: TemplateConfig = {
  variant: "minimal",
  summaryTitle: "Summary",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "text-only",
    containerClass: "flex flex-wrap gap-x-2 gap-y-1",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-2 gap-y-1",
  },
  summaryMb: "mb-5",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: undefined,
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const executiveConfig: TemplateConfig = {
  variant: "executive",
  summaryTitle: "Executive Summary",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "text-level",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  summaryMb: "mb-5",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const creativeConfig: TemplateConfig = {
  variant: "creative",
  summaryTitle: "About Me",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "pill",
    containerClass: "flex flex-wrap gap-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const compactConfig: TemplateConfig = {
  variant: "compact",
  summaryTitle: "Summary",
  textSize: "text-[10px]",
  leadingClass: "leading-snug",
  skills: {
    renderAs: "grid-level",
    containerClass: "grid grid-cols-3 gap-x-4 gap-y-0.5",
  },
  languages: {
    renderAs: "grid",
    containerClass: "grid grid-cols-3 gap-x-4 gap-y-0.5",
  },
  summaryMb: "mb-2",
  entrySpacing: {
    experience: "mb-1.5",
    education: "mb-1",
    certification: "mb-0.5",
    project: "mb-1",
    volunteering: "mb-1",
    customSection: "mb-1",
  },
};

export const elegantConfig: TemplateConfig = {
  variant: "elegant",
  summaryTitle: "Summary",
  sectionTitlePrefix: "\u25C6 ",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "pill",
    containerClass: "flex flex-wrap gap-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

const SHARED_SECTION_ICONS = {
  summary: FileText,
  experience: Briefcase,
  education: GraduationCap,
  skills: Wrench,
  languages: LanguagesIcon,
  certifications: Award,
  projects: FolderOpen,
  volunteering: Heart,
  custom: LayoutList,
  custom_sections: LayoutList,
} as const;

export const iconicConfig: TemplateConfig = {
  variant: "iconic",
  summaryTitle: "About Me",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "pill",
    containerClass: "flex flex-wrap gap-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  sectionIcons: SHARED_SECTION_ICONS,
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const boldConfig: TemplateConfig = {
  variant: "bold",
  summaryTitle: "About Me",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "pill",
    containerClass: "flex flex-wrap gap-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  sectionIcons: SHARED_SECTION_ICONS,
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const accentConfig: TemplateConfig = {
  variant: "accent",
  summaryTitle: "Professional Summary",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "text-level",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const timelineConfig: TemplateConfig = {
  variant: "timeline",
  summaryTitle: "Summary",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "pill",
    containerClass: "flex flex-wrap gap-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  sectionIcons: SHARED_SECTION_ICONS,
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

export const vividConfig: TemplateConfig = {
  variant: "vivid",
  summaryTitle: "About Me",
  textSize: "text-xs",
  leadingClass: "leading-relaxed",
  skills: {
    renderAs: "pill",
    containerClass: "flex flex-wrap gap-1.5",
  },
  languages: {
    renderAs: "flex",
    containerClass: "flex flex-wrap gap-x-4 gap-y-1",
  },
  sectionIcons: SHARED_SECTION_ICONS,
  summaryMb: "mb-4",
  entrySpacing: {
    experience: "mb-3",
    education: "mb-2",
    certification: "mb-1",
    project: "mb-2",
    volunteering: "mb-2",
    customSection: "mb-2",
  },
};

/** Lookup map for all template configs by variant name. */
export const TEMPLATE_CONFIGS: Readonly<
  Record<TemplateConfig["variant"], TemplateConfig>
> = {
  professional: professionalConfig,
  modern: modernConfig,
  minimal: minimalConfig,
  executive: executiveConfig,
  creative: creativeConfig,
  compact: compactConfig,
  elegant: elegantConfig,
  iconic: iconicConfig,
  bold: boldConfig,
  accent: accentConfig,
  timeline: timelineConfig,
  vivid: vividConfig,
};
