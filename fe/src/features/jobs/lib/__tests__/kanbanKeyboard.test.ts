import { describe, it, expect } from "vitest";
import type { KeyboardCoordinateGetter } from "@dnd-kit/core";
import { kanbanCoordinateGetter } from "../kanbanKeyboard";

type Args = Parameters<KeyboardCoordinateGetter>[1];

function rect(left: number, width: number) {
  return { left, top: 0, width, height: 600, right: left + width, bottom: 600 };
}

// Three 280px columns with 16px gaps; the 240px card sits in the middle one.
const columnRects = new Map([
  ["a", rect(0, 280)],
  ["b", rect(296, 280)],
  ["c", rect(592, 280)],
]);

function args(cardLeft: number): Args {
  const collisionRect = { ...rect(cardLeft, 240), top: 100 };
  return {
    active: "job-1",
    currentCoordinates: { x: collisionRect.left, y: collisionRect.top },
    context: {
      collisionRect,
      droppableRects: columnRects,
      droppableContainers: {
        // Deliberately unsorted: the getter must order columns by position.
        getEnabled: () => [{ id: "c" }, { id: "a" }, { id: "b" }],
      },
    },
  } as unknown as Args;
}

function press(code: string, a: Args) {
  return kanbanCoordinateGetter({ code } as KeyboardEvent, a);
}

describe("kanbanCoordinateGetter", () => {
  it("ArrowRight jumps to the next column, centring the card in it", () => {
    // Card in column b (left 316) -> column c, centred: 592 + (280-240)/2.
    expect(press("ArrowRight", args(316))).toEqual({ x: 612, y: 100 });
  });

  it("ArrowLeft jumps to the previous column", () => {
    expect(press("ArrowLeft", args(316))).toEqual({ x: 20, y: 100 });
  });

  it("stays put past the first and last column", () => {
    expect(press("ArrowLeft", args(20))).toBeUndefined();
    expect(press("ArrowRight", args(612))).toBeUndefined();
  });

  it("keeps the default nudge for ArrowUp / ArrowDown", () => {
    expect(press("ArrowDown", args(316))).toEqual({ x: 316, y: 125 });
    expect(press("ArrowUp", args(316))).toEqual({ x: 316, y: 75 });
  });

  it("ignores other keys", () => {
    expect(press("KeyA", args(316))).toBeUndefined();
  });
});
