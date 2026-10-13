import { describe, it, expect } from "vitest";
import { nextStageOrder, sequentialStageOrders } from "../stageOrder";

describe("nextStageOrder", () => {
  it("starts at 0 for an empty pipeline (API orders are 0-based)", () => {
    expect(nextStageOrder([])).toBe(0);
  });

  it("continues after the highest existing order", () => {
    const seeded = [0, 1, 2, 3, 4, 5, 6].map((order) => ({ order }));
    expect(nextStageOrder(seeded)).toBe(7);
  });

  it("uses the max, not the count, when orders have gaps", () => {
    expect(nextStageOrder([{ order: 2 }, { order: 9 }, { order: 4 }])).toBe(10);
  });
});

describe("sequentialStageOrders", () => {
  it("returns consecutive orders after the existing ones", () => {
    expect(sequentialStageOrders([{ order: 0 }, { order: 6 }], 3)).toEqual([
      7, 8, 9,
    ]);
  });

  it("returns an empty list when nothing is added", () => {
    expect(sequentialStageOrders([{ order: 1 }], 0)).toEqual([]);
  });
});
