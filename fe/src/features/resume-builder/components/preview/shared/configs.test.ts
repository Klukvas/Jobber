import { describe, it, expect } from "vitest";
import {
  TEMPLATE_CONFIGS,
  professionalConfig,
  modernConfig,
  minimalConfig,
  executiveConfig,
  creativeConfig,
  compactConfig,
  elegantConfig,
  iconicConfig,
  boldConfig,
  accentConfig,
  timelineConfig,
  vividConfig,
} from "./configs";
import type { TemplateVariant } from "./templateConfig";

const ALL_VARIANTS: TemplateVariant[] = [
  "professional",
  "modern",
  "minimal",
  "executive",
  "creative",
  "compact",
  "elegant",
  "iconic",
  "bold",
  "accent",
  "timeline",
  "vivid",
];

describe("TEMPLATE_CONFIGS", () => {
  it("contains all 12 template variants", () => {
    expect(Object.keys(TEMPLATE_CONFIGS)).toHaveLength(12);
    for (const variant of ALL_VARIANTS) {
      expect(TEMPLATE_CONFIGS[variant]).toBeDefined();
    }
  });

  it("each config has required fields", () => {
    for (const variant of ALL_VARIANTS) {
      const config = TEMPLATE_CONFIGS[variant];
      expect(config.variant).toBe(variant);
      expect(config.summaryTitle).toBeTruthy();
      expect(config.textSize).toBeTruthy();
      expect(config.leadingClass).toBeTruthy();
      expect(config.skills).toBeDefined();
      expect(config.skills.renderAs).toBeTruthy();
      expect(config.skills.containerClass).toBeTruthy();
      expect(config.languages).toBeDefined();
      expect(config.languages.renderAs).toBeTruthy();
      expect(config.languages.containerClass).toBeTruthy();
    }
  });
});

describe("individual configs", () => {
  it("professional uses 'Professional Summary'", () => {
    expect(professionalConfig.summaryTitle).toBe("Professional Summary");
    expect(professionalConfig.skills.renderAs).toBe("text-level");
  });

  it("modern uses 'About Me' and vertical skills", () => {
    expect(modernConfig.summaryTitle).toBe("About Me");
    expect(modernConfig.skills.renderAs).toBe("vertical");
    expect(modernConfig.renderContactInSwitch).toBe(true);
  });

  it("minimal uses 'Summary' and text-only skills", () => {
    expect(minimalConfig.summaryTitle).toBe("Summary");
    expect(minimalConfig.skills.renderAs).toBe("text-only");
  });

  it("executive uses 'Executive Summary'", () => {
    expect(executiveConfig.summaryTitle).toBe("Executive Summary");
    expect(executiveConfig.skills.renderAs).toBe("text-level");
  });

  it("creative uses pill skills", () => {
    expect(creativeConfig.summaryTitle).toBe("About Me");
    expect(creativeConfig.skills.renderAs).toBe("pill");
  });

  it("compact uses smaller text and grid skills", () => {
    expect(compactConfig.textSize).toBe("text-[10px]");
    expect(compactConfig.skills.renderAs).toBe("grid-level");
    expect(compactConfig.languages.renderAs).toBe("grid");
  });

  it("elegant uses diamond prefix", () => {
    expect(elegantConfig.sectionTitlePrefix).toBe("\u25C6 ");
  });

  it("iconic has section icons for all sections", () => {
    expect(iconicConfig.sectionIcons).toBeDefined();
    const iconKeys = Object.keys(iconicConfig.sectionIcons!);
    expect(iconKeys).toContain("experience");
    expect(iconKeys).toContain("education");
    expect(iconKeys).toContain("skills");
    expect(iconKeys).toContain("summary");
  });

  it("bold uses pill skills and has section icons", () => {
    expect(boldConfig.summaryTitle).toBe("About Me");
    expect(boldConfig.skills.renderAs).toBe("pill");
    expect(boldConfig.sectionIcons).toBeDefined();
  });

  it("accent uses text-level skills and no icons", () => {
    expect(accentConfig.summaryTitle).toBe("Professional Summary");
    expect(accentConfig.skills.renderAs).toBe("text-level");
    expect(accentConfig.sectionIcons).toBeUndefined();
  });

  it("timeline uses pill skills and has section icons", () => {
    expect(timelineConfig.summaryTitle).toBe("Summary");
    expect(timelineConfig.skills.renderAs).toBe("pill");
    expect(timelineConfig.sectionIcons).toBeDefined();
  });

  it("vivid uses pill skills and has section icons", () => {
    expect(vividConfig.summaryTitle).toBe("About Me");
    expect(vividConfig.skills.renderAs).toBe("pill");
    expect(vividConfig.sectionIcons).toBeDefined();
  });
});
