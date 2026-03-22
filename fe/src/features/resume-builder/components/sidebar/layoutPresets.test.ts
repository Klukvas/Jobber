import { describe, it, expect } from "vitest";
import { LAYOUT_PRESETS } from "./layoutPresets";

describe("layoutPresets", () => {
  it("exports three layout presets", () => {
    expect(Object.keys(LAYOUT_PRESETS)).toHaveLength(3);
    expect(LAYOUT_PRESETS.single).toBeDefined();
    expect(LAYOUT_PRESETS["double-left"]).toBeDefined();
    expect(LAYOUT_PRESETS["double-right"]).toBeDefined();
  });

  it("single preset assigns all sections to main", () => {
    const { assignments } = LAYOUT_PRESETS.single;
    expect(assignments.contact).toBe("main");
    expect(assignments.experience).toBe("main");
    expect(assignments.skills).toBe("main");
  });

  it("double-left preset puts skills in sidebar", () => {
    const { assignments } = LAYOUT_PRESETS["double-left"];
    expect(assignments.skills).toBe("sidebar");
    expect(assignments.experience).toBe("main");
  });

  it("double-right preset has sidebar_width of 35", () => {
    expect(LAYOUT_PRESETS["double-right"].sidebar_width).toBe(35);
  });
});
