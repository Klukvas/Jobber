import { describe, expect, it } from "vitest";
import { routes } from "@/app/router";
import type { RouteObject } from "react-router-dom";
import { cleanPath, cleanUrl } from "../analyticsUrl";

const { origin } = window.location;

function routePaths(tree: RouteObject[], prefix = ""): string[] {
  return tree.flatMap((route) => {
    const path = route.path?.replace(/^\//, "") ?? "";
    const full = [prefix, path].filter(Boolean).join("/");
    return [full, ...routePaths(route.children ?? [], full)];
  });
}

describe("cleanUrl", () => {
  it("removes the email and one-time code from a reset link", () => {
    expect(cleanUrl("/reset-password?email=a%40b.co&code=123456")).toBe(
      "/reset-password",
    );
    expect(cleanUrl(`${origin}/reset-password?email=a%40b.co&code=1`)).toBe(
      `${origin}/reset-password`,
    );
  });

  it("removes what was typed into a search", () => {
    expect(cleanUrl("/app/jobs?search=Acme%20Corp&page=2")).toBe("/app/jobs");
  });

  it("collapses the share token and private record ids in paths", () => {
    expect(cleanUrl("/s/abc123")).toBe("/s/:token");
    expect(cleanUrl("/app/jobs/9f1c")).toBe("/app/jobs/:id");
    expect(cleanUrl("/app/resume-builder/9f1c")).toBe(
      "/app/resume-builder/:id",
    );
    expect(cleanUrl("/app/cover-letters/9f1c")).toBe("/app/cover-letters/:id");
    expect(cleanUrl("/api/v1/public/shares/abc123/preview")).toBe(
      "/api/v1/public/shares/:token/preview",
    );
  });

  it("keeps public pages readable", () => {
    expect(cleanUrl("/blog/how-to-track-applications")).toBe(
      "/blog/how-to-track-applications",
    );
  });

  it("keeps campaign tags, which is what attribution reads", () => {
    expect(
      cleanUrl("/?utm_source=google&utm_medium=cpc&email=a%40b.co#top"),
    ).toBe("/?utm_source=google&utm_medium=cpc");
  });

  it("drops a campaign tag that is not a plain label", () => {
    expect(cleanUrl("/?utm_source=a%40b.co")).toBe("/");
  });

  it("reduces another site's address to its origin", () => {
    expect(cleanUrl("https://mail.example.com/inbox/42?id=7")).toBe(
      "https://mail.example.com",
    );
  });

  it("has nothing to say about an empty referrer or a non-web address", () => {
    expect(cleanUrl("")).toBe("");
    expect(cleanUrl("chrome-extension://abc/panel.html")).toBe("");
  });
});

describe("cleanPath", () => {
  it("leaves a route pattern as it is", () => {
    expect(cleanPath("/app/jobs/:id")).toBe("/app/jobs/:id");
    expect(cleanPath("/s/:token")).toBe("/s/:token");
  });

  // A new route with an id in its path has to be a decision, not an accident.
  it("covers every route with a parameter in its path", () => {
    const parameterised = routePaths(routes).filter((path) =>
      path.includes(":"),
    );

    expect(parameterised.length).toBeGreaterThan(0);
    for (const path of parameterised) {
      if (path.startsWith("blog/")) continue;
      const sample = `/${path.replace(/:\w+/g, "sample-value")}`;
      expect(cleanPath(sample), path).not.toContain("sample-value");
    }
  });
});
