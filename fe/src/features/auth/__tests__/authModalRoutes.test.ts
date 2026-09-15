import { describe, it, expect } from "vitest";

import { getAuthModalRoute } from "../authModalRoutes";

/**
 * The router treats `/login` and `/login/` as the same route, so both render
 * the landing page — but the lookup here was an exact map read, and the
 * trailing-slash form fell through to `null`. The visitor got the landing page
 * with no modal on it, and a signed-in visitor was not redirected away either,
 * because the layout asks this same question.
 */
describe("getAuthModalRoute", () => {
  it.each([
    ["/login", "login"],
    ["/register", "register"],
    ["/forgot-password", "forgot-password"],
  ] as const)("recognises %s", (pathname, expected) => {
    expect(getAuthModalRoute(pathname)).toBe(expected);
  });

  it.each([
    ["/login/", "login"],
    ["/register/", "register"],
    ["/forgot-password/", "forgot-password"],
  ] as const)("recognises the trailing-slash form %s", (pathname, expected) => {
    expect(getAuthModalRoute(pathname)).toBe(expected);
  });

  it("recognises a path a redirect chain left with several slashes", () => {
    expect(getAuthModalRoute("/login//")).toBe("login");
  });

  it.each(["/", "", "/dashboard", "/logins", "/auth/login", "/login/extra"])(
    "returns null for %s",
    (pathname) => {
      expect(getAuthModalRoute(pathname)).toBeNull();
    },
  );
});
