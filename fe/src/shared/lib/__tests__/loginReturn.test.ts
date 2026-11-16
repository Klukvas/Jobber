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

it("preserves next and independent close context through every auth screen", async () => {
  const { authPathFor, readAuthCloseReturn } = await import("../loginReturn");
  let url = "/login?next=%2Fapp%2Fanalytics%3Fview%3Dall&returnTo=%2Ffeatures%2Fresume-builder%23faq";
  for (const path of ["/register", "/forgot-password", "/login"] as const) {
    url = authPathFor(path, url.slice(url.indexOf("?")));
    expect(readReturnTo(url.slice(url.indexOf("?")))).toBe("/app/analytics?view=all");
    expect(readAuthCloseReturn(url.slice(url.indexOf("?")))).toBe("/features/resume-builder#faq");
  }
});

it("rejects escaped, external and normalized traversal destinations", async () => {
  const { authPathFor, readAuthCloseReturn } = await import("../loginReturn");
  for (const value of ["//evil.test", "https://evil.test", "/app/../../evil", "/app/%2e%2e/evil", "/app/\\evil", "/app/\n/evil"]) {
    expect(readReturnTo(`?next=${encodeURIComponent(value)}`)).toBeNull();
    expect(readAuthCloseReturn(`?returnTo=${encodeURIComponent(value)}`)).toBe("/");
  }
  expect(authPathFor("/register", "?returnTo=%2Flogin")).toBe("/register");
});
