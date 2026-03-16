export interface ColorPalette {
  readonly nameKey: string; // i18n key
  readonly colors: readonly string[];
}

export const COLOR_PALETTES: readonly ColorPalette[] = [
  {
    nameKey: "resumeBuilder.colors.corporate",
    colors: ["#1e3a5f", "#1d4ed8", "#2563eb", "#1e293b", "#334155"],
  },
  {
    nameKey: "resumeBuilder.colors.creative",
    colors: ["#e11d48", "#9333ea", "#ea580c", "#db2777", "#7c3aed"],
  },
  {
    nameKey: "resumeBuilder.colors.minimal",
    colors: ["#ffffff", "#000000", "#171717", "#292524", "#44403c", "#64748b"],
  },
  {
    nameKey: "resumeBuilder.colors.nature",
    colors: ["#059669", "#0d9488", "#16a34a", "#15803d", "#14532d"],
  },
  {
    nameKey: "resumeBuilder.colors.warm",
    colors: ["#dc2626", "#ca8a04", "#78716c", "#b91c1c", "#92400e"],
  },
];

/** Maps template variant name to recommended palette nameKey */
export const TEMPLATE_RECOMMENDED_PALETTE: Readonly<Record<string, string>> = {
  professional: "resumeBuilder.colors.corporate",
  modern: "resumeBuilder.colors.creative",
  minimal: "resumeBuilder.colors.minimal",
  executive: "resumeBuilder.colors.corporate",
  creative: "resumeBuilder.colors.creative",
  compact: "resumeBuilder.colors.minimal",
  elegant: "resumeBuilder.colors.minimal",
  iconic: "resumeBuilder.colors.creative",
  bold: "resumeBuilder.colors.creative",
  accent: "resumeBuilder.colors.corporate",
  timeline: "resumeBuilder.colors.corporate",
  vivid: "resumeBuilder.colors.creative",
};
