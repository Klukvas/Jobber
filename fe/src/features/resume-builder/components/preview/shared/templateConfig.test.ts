import { describe, it, expect } from "vitest";
import type {
  TemplateConfig,
  TemplateVariant,
  SkillRenderMode,
  LanguageRenderMode,
} from "./templateConfig";

describe("TemplateConfig types", () => {
  it("TemplateVariant includes all 12 variants", () => {
    // Type-level test: these assignments should compile without errors
    const variants: TemplateVariant[] = [
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
    expect(variants).toHaveLength(12);
  });

  it("SkillRenderMode includes all render modes", () => {
    const modes: SkillRenderMode[] = [
      "text-level",
      "pill",
      "grid-level",
      "vertical",
      "text-only",
    ];
    expect(modes).toHaveLength(5);
  });

  it("LanguageRenderMode includes all render modes", () => {
    const modes: LanguageRenderMode[] = ["flex", "grid"];
    expect(modes).toHaveLength(2);
  });

  it("TemplateConfig shape is valid with minimal required fields", () => {
    const config: TemplateConfig = {
      variant: "professional",
      summaryTitle: "Test",
      textSize: "text-xs",
      leadingClass: "leading-relaxed",
      skills: {
        renderAs: "text-level",
        containerClass: "flex",
      },
      languages: {
        renderAs: "flex",
        containerClass: "flex",
      },
    };
    expect(config.variant).toBe("professional");
    expect(config.sectionTitlePrefix).toBeUndefined();
    expect(config.sectionIcons).toBeUndefined();
    expect(config.renderContactInSwitch).toBeUndefined();
  });
});
