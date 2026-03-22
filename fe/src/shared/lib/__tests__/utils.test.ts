import { describe, it, expect } from "vitest";
import { cn } from "../utils";

describe("cn", () => {
  it("merges simple class names", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("returns empty string for no arguments", () => {
    expect(cn()).toBe("");
  });

  it("filters out falsy values", () => {
    expect(cn("foo", false, null, undefined, "bar")).toBe("foo bar");
  });

  it("handles conditional classes via objects", () => {
    expect(cn({ foo: true, bar: false, baz: true })).toBe("foo baz");
  });

  it("handles arrays of classes", () => {
    expect(cn(["foo", "bar"])).toBe("foo bar");
  });

  it("merges tailwind classes and resolves conflicts", () => {
    expect(cn("px-2 py-1", "px-4")).toBe("py-1 px-4");
  });

  it("resolves tailwind color conflicts keeping last", () => {
    expect(cn("text-red-500", "text-blue-500")).toBe("text-blue-500");
  });

  it("merges mixed arguments: strings, objects, arrays", () => {
    const result = cn("base", ["arr-class"], { conditional: true });
    expect(result).toBe("base arr-class conditional");
  });

  it("handles empty strings gracefully", () => {
    expect(cn("", "foo", "")).toBe("foo");
  });

  it("resolves conflicting tailwind size utilities", () => {
    expect(cn("w-4", "w-8")).toBe("w-8");
  });
});
