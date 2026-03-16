import { describe, it, expect } from "vitest";

// Test the pure helper functions by importing the module and checking behavior.
// We test getSectionTitle indirectly through SECTION_TITLES and VARIANT_TITLE_OVERRIDES.
// Since these are not exported, we test the contract through the configs.

import { TEMPLATE_CONFIGS } from "./configs";
import type { TemplateConfig } from "./templateConfig";

/**
 * Replicate getSectionTitle logic to test independently.
 * (The actual function is internal to TemplateSections.tsx, but we verify the
 * config-driven behavior here to ensure correctness.)
 */
const SECTION_TITLES: Record<string, string> = {
  experience: "Work Experience",
  education: "Education",
  skills: "Skills",
  languages: "Languages",
  certifications: "Certifications",
  projects: "Projects",
  volunteering: "Volunteering",
  custom: "Custom Sections",
  custom_sections: "Custom Sections",
};

const VARIANT_TITLE_OVERRIDES: Partial<
  Record<string, Partial<Record<string, string>>>
> = {
  minimal: { experience: "Experience" },
  modern: { experience: "Experience" },
};

function getSectionTitle(key: string, config: TemplateConfig): string {
  const override =
    VARIANT_TITLE_OVERRIDES[config.variant]?.[key];
  const base = override ?? SECTION_TITLES[key] ?? key;
  if (!config.sectionTitlePrefix) return base;
  return `${config.sectionTitlePrefix}${base}`;
}

describe("getSectionTitle", () => {
  it("returns 'Work Experience' for professional template", () => {
    expect(
      getSectionTitle("experience", TEMPLATE_CONFIGS.professional),
    ).toBe("Work Experience");
  });

  it("returns 'Experience' for minimal template (override)", () => {
    expect(
      getSectionTitle("experience", TEMPLATE_CONFIGS.minimal),
    ).toBe("Experience");
  });

  it("returns 'Experience' for modern template (override)", () => {
    expect(
      getSectionTitle("experience", TEMPLATE_CONFIGS.modern),
    ).toBe("Experience");
  });

  it("prepends diamond for elegant template", () => {
    expect(
      getSectionTitle("skills", TEMPLATE_CONFIGS.elegant),
    ).toBe("\u25C6 Skills");
  });

  it("returns section key for unknown keys", () => {
    expect(
      getSectionTitle("unknown_key", TEMPLATE_CONFIGS.professional),
    ).toBe("unknown_key");
  });

  it("handles custom_sections key", () => {
    expect(
      getSectionTitle("custom_sections", TEMPLATE_CONFIGS.professional),
    ).toBe("Custom Sections");
  });
});
