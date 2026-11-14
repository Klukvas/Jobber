import { describe, expect, it } from "vitest";
import { loginPathFor, readReturnTo } from "@/shared/lib/loginReturn";

describe("loginReturn", () => {
  it("round-trips an in-app path with its query", () => {
    const path = loginPathFor("/app/jobs?view=board");
    expect(path).toBe("/login?next=%2Fapp%2Fjobs%3Fview%3Dboard");
    expect(readReturnTo(path.slice("/login".length))).toBe(
      "/app/jobs?view=board",
    );
  });

  it("drops anything that is not an in-app path", () => {
    for (const unsafe of [
      "https://evil.example",
      "//evil.example/app",
      "/apple",
      "/",
      "",
      null,
    ]) {
      expect(loginPathFor(unsafe)).toBe("/login");
    }
    expect(readReturnTo("?next=https%3A%2F%2Fevil.example")).toBeNull();
    expect(readReturnTo("?next=%2F%2Fevil.example%2Fapp")).toBeNull();
    expect(readReturnTo("")).toBeNull();
  });
});
