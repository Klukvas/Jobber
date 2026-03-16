import { describe, it, expect } from "vitest";
import { COLOR_PALETTES, TEMPLATE_RECOMMENDED_PALETTE } from "./colorPalettes";

describe("COLOR_PALETTES", () => {
  it("has 5 palette groups", () => {
    expect(COLOR_PALETTES).toHaveLength(5);
  });

  it("each palette has a nameKey and at least 3 colors", () => {
    for (const palette of COLOR_PALETTES) {
      expect(palette.nameKey).toBeTruthy();
      expect(palette.colors.length).toBeGreaterThanOrEqual(3);
    }
  });

  it("all colors are valid hex format", () => {
    const hexRegex = /^#[0-9a-fA-F]{6}$/;
    for (const palette of COLOR_PALETTES) {
      for (const color of palette.colors) {
        expect(color).toMatch(hexRegex);
      }
    }
  });

  it("has no duplicate colors within a palette", () => {
    for (const palette of COLOR_PALETTES) {
      const unique = new Set(palette.colors);
      expect(unique.size).toBe(palette.colors.length);
    }
  });

  it("has unique nameKeys", () => {
    const keys = COLOR_PALETTES.map((p) => p.nameKey);
    expect(new Set(keys).size).toBe(keys.length);
  });
});

describe("TEMPLATE_RECOMMENDED_PALETTE", () => {
  const variants = [
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

  it("maps every template variant to a palette", () => {
    for (const variant of variants) {
      expect(TEMPLATE_RECOMMENDED_PALETTE[variant]).toBeTruthy();
    }
  });

  it("every recommended palette exists in COLOR_PALETTES", () => {
    const paletteKeys = new Set(COLOR_PALETTES.map((p) => p.nameKey));
    for (const key of Object.values(TEMPLATE_RECOMMENDED_PALETTE)) {
      expect(paletteKeys.has(key)).toBe(true);
    }
  });
});
