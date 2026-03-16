import { describe, it, expect } from "vitest";
import { TEMPLATE_MAP, TEMPLATE_LIST } from "../../lib/templateRegistry";
import { TEMPLATE_CONFIGS } from "./shared/configs";
import type { TemplateVariant } from "./shared/templateConfig";
import { TEMPLATE_RECOMMENDED_PALETTE } from "../sidebar/colorPalettes";

/**
 * Integration tests verifying that all 12 templates are consistently registered
 * across every layer: registry, configs, picker list, color palettes.
 */

const NEW_TEMPLATES: { id: string; variant: TemplateVariant; nameKey: string }[] = [
  { id: "00000000-0000-0000-0000-000000000009", variant: "bold", nameKey: "resumeBuilder.templates.bold" },
  { id: "00000000-0000-0000-0000-00000000000a", variant: "accent", nameKey: "resumeBuilder.templates.accent" },
  { id: "00000000-0000-0000-0000-00000000000b", variant: "timeline", nameKey: "resumeBuilder.templates.timeline" },
  { id: "00000000-0000-0000-0000-00000000000c", variant: "vivid", nameKey: "resumeBuilder.templates.vivid" },
];

const ALL_TEMPLATE_IDS = [
  "00000000-0000-0000-0000-000000000001",
  "00000000-0000-0000-0000-000000000002",
  "00000000-0000-0000-0000-000000000003",
  "00000000-0000-0000-0000-000000000004",
  "00000000-0000-0000-0000-000000000005",
  "00000000-0000-0000-0000-000000000006",
  "00000000-0000-0000-0000-000000000007",
  "00000000-0000-0000-0000-000000000008",
  "00000000-0000-0000-0000-000000000009",
  "00000000-0000-0000-0000-00000000000a",
  "00000000-0000-0000-0000-00000000000b",
  "00000000-0000-0000-0000-00000000000c",
];

describe("Template registry consistency", () => {
  it("TEMPLATE_MAP has 12 entries with React components", () => {
    expect(Object.keys(TEMPLATE_MAP)).toHaveLength(12);
    for (const id of ALL_TEMPLATE_IDS) {
      expect(TEMPLATE_MAP[id]).toBeDefined();
      expect(typeof TEMPLATE_MAP[id]).toBe("function");
    }
  });

  it("TEMPLATE_LIST has 12 entries matching TEMPLATE_MAP", () => {
    expect(TEMPLATE_LIST).toHaveLength(12);
    for (const entry of TEMPLATE_LIST) {
      expect(TEMPLATE_MAP[entry.id]).toBeDefined();
      expect(entry.nameKey).toMatch(/^resumeBuilder\.templates\./);
    }
  });

  it("TEMPLATE_CONFIGS has 12 entries", () => {
    expect(Object.keys(TEMPLATE_CONFIGS)).toHaveLength(12);
  });

  it("every variant in TEMPLATE_CONFIGS has a recommended palette", () => {
    for (const variant of Object.keys(TEMPLATE_CONFIGS)) {
      expect(TEMPLATE_RECOMMENDED_PALETTE[variant]).toBeTruthy();
    }
  });
});

describe("New templates (bold, accent, timeline, vivid)", () => {
  for (const { id, variant, nameKey } of NEW_TEMPLATES) {
    describe(variant, () => {
      it(`is registered in TEMPLATE_MAP with UUID ${id}`, () => {
        expect(TEMPLATE_MAP[id]).toBeDefined();
        expect(typeof TEMPLATE_MAP[id]).toBe("function");
      });

      it("is listed in TEMPLATE_LIST with correct nameKey", () => {
        const entry = TEMPLATE_LIST.find((e) => e.id === id);
        expect(entry).toBeDefined();
        expect(entry!.nameKey).toBe(nameKey);
      });

      it("has a TemplateConfig", () => {
        const config = TEMPLATE_CONFIGS[variant];
        expect(config).toBeDefined();
        expect(config.variant).toBe(variant);
        expect(config.summaryTitle).toBeTruthy();
        expect(config.textSize).toBeTruthy();
        expect(config.leadingClass).toBeTruthy();
        expect(config.skills.renderAs).toBeTruthy();
        expect(config.languages.renderAs).toBeTruthy();
      });

      it("has a recommended color palette", () => {
        expect(TEMPLATE_RECOMMENDED_PALETTE[variant]).toBeTruthy();
      });
    });
  }
});

describe("Bold template config specifics", () => {
  it("uses pill skills and has section icons", () => {
    const config = TEMPLATE_CONFIGS["bold"];
    expect(config.skills.renderAs).toBe("pill");
    expect(config.sectionIcons).toBeDefined();
    expect(config.sectionIcons!["experience"]).toBeDefined();
    expect(config.sectionIcons!["education"]).toBeDefined();
  });
});

describe("Accent template config specifics", () => {
  it("uses text-level skills and no section icons", () => {
    const config = TEMPLATE_CONFIGS["accent"];
    expect(config.skills.renderAs).toBe("text-level");
    expect(config.sectionIcons).toBeUndefined();
  });
});

describe("Timeline template config specifics", () => {
  it("uses pill skills and has section icons", () => {
    const config = TEMPLATE_CONFIGS["timeline"];
    expect(config.skills.renderAs).toBe("pill");
    expect(config.sectionIcons).toBeDefined();
  });
});

describe("Vivid template config specifics", () => {
  it("uses pill skills and has section icons", () => {
    const config = TEMPLATE_CONFIGS["vivid"];
    expect(config.skills.renderAs).toBe("pill");
    expect(config.sectionIcons).toBeDefined();
    expect(config.sectionIcons!["summary"]).toBeDefined();
  });
});
