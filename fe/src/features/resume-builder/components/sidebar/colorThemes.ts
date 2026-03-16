export interface ColorTheme {
  readonly nameKey: string;
  readonly primary: string;
  readonly text: string;
}

export const COLOR_THEMES: readonly ColorTheme[] = [
  // Corporate / Professional
  { nameKey: "resumeBuilder.themes.navy", primary: "#1e3a5f", text: "#1e3a5f" },
  { nameKey: "resumeBuilder.themes.royalBlue", primary: "#1d4ed8", text: "#1e293b" },
  { nameKey: "resumeBuilder.themes.slate", primary: "#334155", text: "#1e293b" },
  { nameKey: "resumeBuilder.themes.steel", primary: "#475569", text: "#0f172a" },

  // Warm / Bold
  { nameKey: "resumeBuilder.themes.crimson", primary: "#dc2626", text: "#7f1d1d" },
  { nameKey: "resumeBuilder.themes.coral", primary: "#ea580c", text: "#78350f" },
  { nameKey: "resumeBuilder.themes.rose", primary: "#e11d48", text: "#881337" },
  { nameKey: "resumeBuilder.themes.amber", primary: "#d97706", text: "#78350f" },

  // Nature / Earthy
  { nameKey: "resumeBuilder.themes.emerald", primary: "#059669", text: "#064e3b" },
  { nameKey: "resumeBuilder.themes.teal", primary: "#0d9488", text: "#134e4a" },
  { nameKey: "resumeBuilder.themes.forest", primary: "#15803d", text: "#14532d" },
  { nameKey: "resumeBuilder.themes.olive", primary: "#65a30d", text: "#365314" },

  // Creative / Vibrant
  { nameKey: "resumeBuilder.themes.purple", primary: "#9333ea", text: "#581c87" },
  { nameKey: "resumeBuilder.themes.magenta", primary: "#db2777", text: "#831843" },
  { nameKey: "resumeBuilder.themes.indigo", primary: "#4f46e5", text: "#312e81" },
  { nameKey: "resumeBuilder.themes.violet", primary: "#7c3aed", text: "#4c1d95" },

  // Minimal / Monochrome
  { nameKey: "resumeBuilder.themes.charcoal", primary: "#171717", text: "#171717" },
  { nameKey: "resumeBuilder.themes.graphite", primary: "#44403c", text: "#1c1917" },
  { nameKey: "resumeBuilder.themes.coolGray", primary: "#6b7280", text: "#111827" },
  { nameKey: "resumeBuilder.themes.warmGray", primary: "#78716c", text: "#292524" },
] as const;
